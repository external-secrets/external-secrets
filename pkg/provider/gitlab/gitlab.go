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
	"strings"

	"github.com/tidwall/gjson"
	gitlab "github.com/xanzy/go-gitlab"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/e2e/framework/log"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// Requires GITLAB_TOKEN and GITLAB_PROJECT_ID to be set in environment variables

const (
	errGitlabCredSecretName                   = "credentials are empty"
	errInvalidClusterStoreMissingSAKNamespace = "invalid clusterStore missing SAK namespace"
	errFetchSAKSecret                         = "couldn't find secret on cluster: %w"
	errMissingSAK                             = "missing credentials while setting auth"
	errList                                   = "could not verify if the client is valid: %w"
	errAuth                                   = "client is not allowed to get secrets"
	errUninitalizedGitlabProvider             = "provider gitlab is not initialized"
	errJSONSecretUnmarshal                    = "unable to unmarshal secret: %w"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Gitlab{}
var _ esv1beta1.Provider = &Gitlab{}

type Client interface {
	GetVariable(pid interface{}, key string, opt *gitlab.GetProjectVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error)
	ListVariables(pid interface{}, opt *gitlab.ListProjectVariablesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectVariable, *gitlab.Response, error)
}

// Gitlab Provider struct with reference to a GitLab client and a projectID.
type Gitlab struct {
	client    Client
	url       string
	projectID interface{}
}

// Client for interacting with kubernetes cluster...?
type gClient struct {
	kube        kclient.Client
	store       *esv1beta1.GitlabProvider
	namespace   string
	storeKind   string
	credentials []byte
}

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
	if (c.credentials == nil) || (len(c.credentials) == 0) {
		return fmt.Errorf(errMissingSAK)
	}
	// I don't know where ProjectID is being set
	// This line SHOULD set it, but instead just breaks everything :)
	// c.store.ProjectID = string(credentialsSecret.Data[c.store.ProjectID])
	return nil
}

// Function newGitlabProvider returns a reference to a new instance of a 'Gitlab' struct.
func NewGitlabProvider() *Gitlab {
	return &Gitlab{}
}

// Method on Gitlab Provider to set up client with credentials and populate projectID.
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

	// Create client options
	var opts []gitlab.ClientOptionFunc
	if cliStore.store.URL != "" {
		opts = append(opts, gitlab.WithBaseURL(cliStore.store.URL))
	}
	// ClientOptionFunc from the gitlab package can be mapped with the CRD
	// in a similar way to extend functionality of the provider

	// Create a new Gitlab client using credentials and options
	gitlabClient, err := gitlab.NewClient(string(cliStore.credentials), opts...)
	if err != nil {
		log.Logf("Failed to create client: %v", err)
	}

	g.client = gitlabClient.ProjectVariables
	g.projectID = cliStore.store.ProjectID
	g.url = cliStore.store.URL

	return g, nil
}

// Empty GetAllSecrets.
func (g *Gitlab) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

func (g *Gitlab) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(g.client) {
		return nil, fmt.Errorf(errUninitalizedGitlabProvider)
	}
	// Need to replace hyphens with underscores to work with Gitlab API
	ref.Key = strings.ReplaceAll(ref.Key, "-", "_")
	// Retrieves a gitlab variable in the form
	// {
	// 	"key": "TEST_VARIABLE_1",
	// 	"variable_type": "env_var",
	// 	"value": "TEST_1",
	// 	"protected": false,
	// 	"masked": true
	data, _, err := g.client.GetVariable(g.projectID, ref.Key, nil) // Optional 'filter' parameter could be added later
	if err != nil {
		return nil, err
	}

	if ref.Property == "" {
		if data.Value != "" {
			return []byte(data.Value), nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string for key: %s", ref.Key)
	}

	var payload string
	if data.Value != "" {
		payload = data.Value
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

func (g *Gitlab) Close(ctx context.Context) error {
	return nil
}

// Validate will use the gitlab client to validate the gitlab provider using the ListVariable call to ensure get permissions without needing a specific key.
func (g *Gitlab) Validate() (esv1beta1.ValidationResult, error) {
	_, resp, err := g.client.ListVariables(g.projectID, nil)
	if err != nil {
		return esv1beta1.ValidationResultError, fmt.Errorf(errList, err)
	} else if resp == nil || resp.StatusCode != http.StatusOK {
		return esv1beta1.ValidationResultError, fmt.Errorf(errAuth)
	}
	return esv1beta1.ValidationResultReady, nil
}

func (g *Gitlab) ValidateStore(store esv1beta1.GenericStore) error {
	return nil
}
