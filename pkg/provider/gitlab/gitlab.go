/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	gitlab "github.com/xanzy/go-gitlab"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	errGitlabCredSecretName                   = "credentials are empty"
	errInvalidClusterStoreMissingSAKNamespace = "invalid clusterStore missing SAK namespace"
	errFetchSAKSecret                         = "couldn't find secret on cluster: %w"
	errMissingSAK                             = "missing credentials while setting auth"
	errList                                   = "could not verify whether the gilabClient is valid: %w"
	errProjectAuth                            = "gitlabClient is not allowed to get secrets for project id [%s]"
	errGroupAuth                              = "gitlabClient is not allowed to get secrets for group id [%s]"
	errUninitializedGitlabProvider            = "provider gitlab is not initialized"
	errNameNotDefined                         = "'find.name' is mandatory"
	errEnvironmentIsConstricted               = "'find.tags' is constrained by 'environment_scope' of the store"
	errTagsOnlyEnvironmentSupported           = "'find.tags' only supports 'environment_scope'"
	errPathNotImplemented                     = "'find.path' is not implemented in the Gitlab provider"
	errJSONSecretUnmarshal                    = "unable to unmarshal secret: %w"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Gitlab{}
var _ esv1beta1.Provider = &Gitlab{}

type ProjectsClient interface {
	ListProjectsGroups(pid interface{}, opt *gitlab.ListProjectGroupOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectGroup, *gitlab.Response, error)
}

type ProjectVariablesClient interface {
	GetVariable(pid interface{}, key string, opt *gitlab.GetProjectVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error)
	ListVariables(pid interface{}, opt *gitlab.ListProjectVariablesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectVariable, *gitlab.Response, error)
}

type GroupVariablesClient interface {
	GetVariable(gid interface{}, key string, options ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error)
	ListVariables(gid interface{}, opt *gitlab.ListGroupVariablesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.GroupVariable, *gitlab.Response, error)
}

// Gitlab Provider struct with reference to GitLab clients, a projectID and groupIDs.
type Gitlab struct {
	projectsClient         ProjectsClient
	projectVariablesClient ProjectVariablesClient
	groupVariablesClient   GroupVariablesClient
	url                    string
	projectID              string
	inheritFromGroups      bool
	groupIDs               []string
	environment            string
}

// gClient for interacting with kubernetes cluster...?
type gClient struct {
	kube        kclient.Client
	store       *esv1beta1.GitlabProvider
	namespace   string
	storeKind   string
	credentials []byte
}

type ProjectGroupPathSorter []*gitlab.ProjectGroup

func (a ProjectGroupPathSorter) Len() int           { return len(a) }
func (a ProjectGroupPathSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ProjectGroupPathSorter) Less(i, j int) bool { return len(a[i].FullPath) < len(a[j].FullPath) }

var log = ctrl.Log.WithName("provider").WithName("gitlab")

func init() {
	esv1beta1.Register(&Gitlab{}, &esv1beta1.SecretStoreProvider{
		Gitlab: &esv1beta1.GitlabProvider{},
	})
}

// Set gClient credentials to Access Token.
func (c *gClient) setAuth(ctx context.Context) error {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.SecretRef.AccessToken.Name
	if credentialsSecretName == "" {
		return fmt.Errorf(errGitlabCredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: c.namespace,
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if c.storeKind == esv1beta1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.AccessToken.Namespace == nil {
			return fmt.Errorf(errInvalidClusterStoreMissingSAKNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.AccessToken.Namespace
	}

	err := c.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return fmt.Errorf(errFetchSAKSecret, err)
	}

	c.credentials = credentialsSecret.Data[c.store.Auth.SecretRef.AccessToken.Key]
	if c.credentials == nil || len(c.credentials) == 0 {
		return fmt.Errorf(errMissingSAK)
	}
	return nil
}

// Function newGitlabProvider returns a reference to a new instance of a 'Gitlab' struct.
func NewGitlabProvider() *Gitlab {
	return &Gitlab{}
}

// Method on Gitlab Provider to set up projectVariablesClient with credentials, populate projectID and environment.
func (g *Gitlab) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Gitlab == nil {
		return nil, fmt.Errorf("no store type or wrong store type")
	}
	storeSpecGitlab := storeSpec.Provider.Gitlab

	cliStore := gClient{
		kube:      kube,
		store:     storeSpecGitlab,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	if err := cliStore.setAuth(ctx); err != nil {
		return nil, err
	}

	var err error

	// Create projectVariablesClient options
	var opts []gitlab.ClientOptionFunc
	if cliStore.store.URL != "" {
		opts = append(opts, gitlab.WithBaseURL(cliStore.store.URL))
	}

	// ClientOptionFunc from the gitlab package can be mapped with the CRD
	// in a similar way to extend functionality of the provider

	// Create a new Gitlab Client using credentials and options
	gitlabClient, err := gitlab.NewClient(string(cliStore.credentials), opts...)
	if err != nil {
		return nil, err
	}

	g.projectsClient = gitlabClient.Projects
	g.projectVariablesClient = gitlabClient.ProjectVariables
	g.groupVariablesClient = gitlabClient.GroupVariables
	g.projectID = cliStore.store.ProjectID
	g.inheritFromGroups = cliStore.store.InheritFromGroups
	g.groupIDs = cliStore.store.GroupIDs
	g.environment = cliStore.store.Environment
	g.url = cliStore.store.URL

	return g, nil
}

// GetAllSecrets syncs all gitlab project and group variables into a single Kubernetes Secret.
func (g *Gitlab) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if utils.IsNil(g.projectVariablesClient) {
		return nil, fmt.Errorf(errUninitializedGitlabProvider)
	}
	if ref.Tags != nil {
		environment, err := ExtractTag(ref.Tags)
		if err != nil {
			return nil, err
		}
		if !isEmptyOrWildcard(g.environment) && !isEmptyOrWildcard(environment) {
			return nil, fmt.Errorf(errEnvironmentIsConstricted)
		}
		g.environment = environment
	}
	if ref.Path != nil {
		return nil, fmt.Errorf(errPathNotImplemented)
	}
	if ref.Name == nil {
		return nil, fmt.Errorf(errNameNotDefined)
	}

	var matcher *find.Matcher
	if ref.Name != nil {
		m, err := find.New(*ref.Name)
		if err != nil {
			return nil, err
		}
		matcher = m
	}

	err := g.ResolveGroupIds()
	if err != nil {
		return nil, err
	}

	secretData := make(map[string][]byte)
	for _, groupID := range g.groupIDs {
		var groupVars []*gitlab.GroupVariable
		groupVars, _, err := g.groupVariablesClient.ListVariables(groupID, nil)
		if err != nil {
			return nil, err
		}
		for _, data := range groupVars {
			matching, key := matchesFilter(g.environment, data.EnvironmentScope, data.Key, matcher)
			if !matching {
				continue
			}
			secretData[key] = []byte(data.Value)
		}
	}

	var projectData []*gitlab.ProjectVariable
	projectData, _, err = g.projectVariablesClient.ListVariables(g.projectID, nil)
	if err != nil {
		return nil, err
	}

	for _, data := range projectData {
		matching, key := matchesFilter(g.environment, data.EnvironmentScope, data.Key, matcher)
		if !matching {
			continue
		}
		secretData[key] = []byte(data.Value)
	}

	return secretData, nil
}

func ExtractTag(tags map[string]string) (string, error) {
	var environmentScope string
	for tag, value := range tags {
		if tag != "environment_scope" {
			return "", fmt.Errorf(errTagsOnlyEnvironmentSupported)
		}
		environmentScope = value
	}
	return environmentScope, nil
}

func (g *Gitlab) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(g.projectVariablesClient) || utils.IsNil(g.groupVariablesClient) {
		return nil, fmt.Errorf(errUninitializedGitlabProvider)
	}

	// Need to replace hyphens with underscores to work with Gitlab API
	ref.Key = strings.ReplaceAll(ref.Key, "-", "_")
	// Retrieves a gitlab variable in the form
	// {
	// 	"key": "TEST_VARIABLE_1",
	// 	"variable_type": "env_var",
	// 	"value": "TEST_1",
	// 	"protected": false,
	// 	"masked": true,
	// 	"environment_scope": "*"
	// }
	var vopts *gitlab.GetProjectVariableOptions
	if g.environment != "" {
		vopts = &gitlab.GetProjectVariableOptions{Filter: &gitlab.VariableFilter{EnvironmentScope: g.environment}}
	}

	data, resp, err := g.projectVariablesClient.GetVariable(g.projectID, ref.Key, vopts)
	if resp.StatusCode >= 400 && resp.StatusCode != 404 && err != nil {
		return nil, err
	}

	err = g.ResolveGroupIds()
	if err != nil {
		return nil, err
	}

	var result []byte
	if resp.StatusCode < 300 {
		result, err = extractVariable(ref, data.Value)
	}

	for i := len(g.groupIDs) - 1; i >= 0; i-- {
		groupID := g.groupIDs[i]
		if result != nil {
			return result, nil
		}

		groupVar, resp, err := g.groupVariablesClient.GetVariable(groupID, ref.Key, nil)
		if resp.StatusCode >= 400 && resp.StatusCode != 404 && err != nil {
			return nil, err
		}
		if resp.StatusCode < 300 {
			result, _ = extractVariable(ref, groupVar.Value)
		}
	}

	if result != nil {
		return result, nil
	}
	return nil, err
}

func extractVariable(ref esv1beta1.ExternalSecretDataRemoteRef, value string) ([]byte, error) {
	if ref.Property == "" {
		if value != "" {
			return []byte(value), nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string for key: %s", ref.Key)
	}

	var payload string
	if value != "" {
		payload = value
	}

	val := gjson.Get(payload, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

func (g *Gitlab) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	// Gets a secret as normal, expecting secret value to be a json object
	data, err := g.GetSecret(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s: %w", ref.Key, err)
	}

	// Maps the json data to a string:string map
	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
	}

	// Converts values in K:V pairs into bytes, while leaving keys as strings
	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}
	return secretData, nil
}

func isEmptyOrWildcard(environment string) bool {
	return environment == "" || environment == "*"
}

func matchesFilter(environment, varEnvironment, key string, matcher *find.Matcher) (bool, string) {
	if !isEmptyOrWildcard(environment) {
		// as of now gitlab does not support filtering of EnvironmentScope through the api call
		if varEnvironment != environment {
			return false, ""
		}
	}

	if key == "" || (matcher != nil && !matcher.MatchName(key)) {
		return false, ""
	}

	return true, key
}

func (g *Gitlab) Close(ctx context.Context) error {
	return nil
}

// Validate will use the gitlab projectVariablesClient/groupVariablesClient to validate the gitlab provider using the ListVariable call to ensure get permissions without needing a specific key.
func (g *Gitlab) Validate() (esv1beta1.ValidationResult, error) {
	if g.projectID != "" {
		_, resp, err := g.projectVariablesClient.ListVariables(g.projectID, nil)
		if err != nil {
			return esv1beta1.ValidationResultError, fmt.Errorf(errList, err)
		} else if resp == nil || resp.StatusCode != http.StatusOK {
			return esv1beta1.ValidationResultError, fmt.Errorf(errProjectAuth, g.projectID)
		}

		err = g.ResolveGroupIds()
		if err != nil {
			return esv1beta1.ValidationResultError, fmt.Errorf(errList, err)
		}
		log.V(1).Info("discovered project groups", "name", g.groupIDs)
	}

	if len(g.groupIDs) > 0 {
		for _, groupID := range g.groupIDs {
			_, resp, err := g.groupVariablesClient.ListVariables(groupID, nil)
			if err != nil {
				return esv1beta1.ValidationResultError, fmt.Errorf(errList, err)
			} else if resp == nil || resp.StatusCode != http.StatusOK {
				return esv1beta1.ValidationResultError, fmt.Errorf(errGroupAuth, groupID)
			}
		}
	}

	return esv1beta1.ValidationResultReady, nil
}

func (g *Gitlab) ResolveGroupIds() error {
	if g.inheritFromGroups {
		projectGroups, resp, err := g.projectsClient.ListProjectsGroups(g.projectID, nil)
		if resp.StatusCode >= 400 && err != nil {
			return err
		}
		sort.Sort(ProjectGroupPathSorter(projectGroups))
		discoveredIds := make([]string, len(projectGroups))
		for i, group := range projectGroups {
			discoveredIds[i] = strconv.Itoa(group.ID)
		}
		g.groupIDs = discoveredIds
	}
	return nil
}

func (g *Gitlab) ValidateStore(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	gitlabSpec := storeSpec.Provider.Gitlab
	accessToken := gitlabSpec.Auth.SecretRef.AccessToken
	err := utils.ValidateSecretSelector(store, accessToken)
	if err != nil {
		return err
	}

	if gitlabSpec.ProjectID == "" && len(gitlabSpec.GroupIDs) == 0 {
		return fmt.Errorf("projectID and groupIDs must not both be empty")
	}

	if gitlabSpec.InheritFromGroups && len(gitlabSpec.GroupIDs) > 0 {
		return fmt.Errorf("defining groupIDs and inheritFromGroups = true is not allowed")
	}

	if accessToken.Key == "" {
		return fmt.Errorf("accessToken.key cannot be empty")
	}

	if accessToken.Name == "" {
		return fmt.Errorf("accessToken.name cannot be empty")
	}

	return nil
}
