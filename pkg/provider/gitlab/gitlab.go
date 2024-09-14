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
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/xanzy/go-gitlab"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errGitlabCredSecretName                   = "credentials are empty"
	errInvalidClusterStoreMissingSAKNamespace = "invalid clusterStore missing SAK namespace"
	errFetchSAKSecret                         = "couldn't find secret on cluster: %w"
	errList                                   = "could not verify whether the gitlabClient is valid: %w"
	errProjectAuth                            = "gitlabClient is not allowed to get secrets for project id [%s]"
	errGroupAuth                              = "gitlabClient is not allowed to get secrets for group id [%s]"
	errUninitializedGitlabProvider            = "provider gitlab is not initialized"
	errNameNotDefined                         = "'find.name' is mandatory"
	errEnvironmentIsConstricted               = "'find.tags' is constrained by 'environment_scope' of the store"
	errTagsOnlyEnvironmentSupported           = "'find.tags' only supports 'environment_scope'"
	errPathNotImplemented                     = "'find.path' is not implemented in the GitLab provider"
	errJSONSecretUnmarshal                    = "unable to unmarshal secret: %w"
	errNotImplemented                         = "not implemented"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &gitlabBase{}
var _ esv1beta1.Provider = &Provider{}

type ProjectsClient interface {
	ListProjectsGroups(pid any, opt *gitlab.ListProjectGroupOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectGroup, *gitlab.Response, error)
}

type ProjectVariablesClient interface {
	GetVariable(pid any, key string, opt *gitlab.GetProjectVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error)
	ListVariables(pid any, opt *gitlab.ListProjectVariablesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectVariable, *gitlab.Response, error)
}

type GroupVariablesClient interface {
	GetVariable(gid any, key string, opts *gitlab.GetGroupVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error)
	ListVariables(gid any, opt *gitlab.ListGroupVariablesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.GroupVariable, *gitlab.Response, error)
}

type ProjectGroupPathSorter []*gitlab.ProjectGroup

func (a ProjectGroupPathSorter) Len() int           { return len(a) }
func (a ProjectGroupPathSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ProjectGroupPathSorter) Less(i, j int) bool { return len(a[i].FullPath) < len(a[j].FullPath) }

var log = ctrl.Log.WithName("provider").WithName("gitlab")

// Set gitlabBase credentials to Access Token.
func (g *gitlabBase) getAuth(ctx context.Context) (string, error) {
	return resolvers.SecretKeyRef(
		ctx,
		g.kube,
		g.storeKind,
		g.namespace,
		&g.store.Auth.SecretRef.AccessToken)
}

func (g *gitlabBase) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

func (g *gitlabBase) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

func (g *gitlabBase) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

// GetAllSecrets syncs all gitlab project and group variables into a single Kubernetes Secret.
func (g *gitlabBase) GetAllSecrets(_ context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if utils.IsNil(g.projectVariablesClient) {
		return nil, errors.New(errUninitializedGitlabProvider)
	}
	var effectiveEnvironment = g.store.Environment
	if ref.Tags != nil {
		environment, err := ExtractTag(ref.Tags)
		if err != nil {
			return nil, err
		}
		if !isEmptyOrWildcard(effectiveEnvironment) && !isEmptyOrWildcard(environment) {
			return nil, errors.New(errEnvironmentIsConstricted)
		}
		effectiveEnvironment = environment
	}
	if ref.Path != nil {
		return nil, errors.New(errPathNotImplemented)
	}
	if ref.Name == nil {
		return nil, errors.New(errNameNotDefined)
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

	var gopts = &gitlab.ListGroupVariablesOptions{PerPage: 100}
	secretData := make(map[string][]byte)
	for _, groupID := range g.store.GroupIDs {
		for groupPage := 1; ; groupPage++ {
			gopts.Page = groupPage
			groupVars, response, err := g.groupVariablesClient.ListVariables(groupID, gopts)
			metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupListVariables, err)
			if err != nil {
				return nil, err
			}
			for _, data := range groupVars {
				matching, key, isWildcard := matchesFilter(effectiveEnvironment, data.EnvironmentScope, data.Key, matcher)
				if !matching && !isWildcard {
					continue
				}
				secretData[key] = []byte(data.Value)
			}
			if response.CurrentPage >= response.TotalPages {
				break
			}
		}
	}

	var popts = &gitlab.ListProjectVariablesOptions{PerPage: 100}
	for projectPage := 1; ; projectPage++ {
		popts.Page = projectPage
		projectData, response, err := g.projectVariablesClient.ListVariables(g.store.ProjectID, popts)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectListVariables, err)
		if err != nil {
			return nil, err
		}

		for _, data := range projectData {
			matching, key, isWildcard := matchesFilter(effectiveEnvironment, data.EnvironmentScope, data.Key, matcher)

			if !matching {
				continue
			}
			_, exists := secretData[key]
			if exists && isWildcard {
				continue
			}
			secretData[key] = []byte(data.Value)
		}
		if response.CurrentPage >= response.TotalPages {
			break
		}
	}

	return secretData, nil
}

func ExtractTag(tags map[string]string) (string, error) {
	var environmentScope string
	for tag, value := range tags {
		if tag != "environment_scope" {
			return "", errors.New(errTagsOnlyEnvironmentSupported)
		}
		environmentScope = value
	}
	return environmentScope, nil
}

func (g *gitlabBase) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(g.projectVariablesClient) || utils.IsNil(g.groupVariablesClient) {
		return nil, errors.New(errUninitializedGitlabProvider)
	}

	// Need to replace hyphens with underscores to work with GitLab API
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
	if g.store.Environment != "" {
		vopts = &gitlab.GetProjectVariableOptions{Filter: &gitlab.VariableFilter{EnvironmentScope: g.store.Environment}}
	}

	data, resp, err := g.projectVariablesClient.GetVariable(g.store.ProjectID, ref.Key, vopts)
	metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableGet, err)
	if !isEmptyOrWildcard(g.store.Environment) && resp.StatusCode == http.StatusNotFound {
		vopts.Filter.EnvironmentScope = "*"
		data, resp, err = g.projectVariablesClient.GetVariable(g.store.ProjectID, ref.Key, vopts)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableGet, err)
	}

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound && err != nil {
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

	for i := len(g.store.GroupIDs) - 1; i >= 0; i-- {
		groupID := g.store.GroupIDs[i]
		if result != nil {
			return result, nil
		}

		groupVar, resp, err := g.groupVariablesClient.GetVariable(groupID, ref.Key, nil)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupGetVariable, err)
		if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound && err != nil {
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

func (g *gitlabBase) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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

func matchesFilter(environment, varEnvironment, key string, matcher *find.Matcher) (bool, string, bool) {
	isWildcard := isEmptyOrWildcard(varEnvironment)
	if !isWildcard && !isEmptyOrWildcard(environment) {
		// as of now gitlab does not support filtering of EnvironmentScope through the api call
		if varEnvironment != environment {
			return false, "", isWildcard
		}
	}

	if key == "" || (matcher != nil && !matcher.MatchName(key)) {
		return false, "", isWildcard
	}

	return true, key, isWildcard
}

func (g *gitlabBase) Close(_ context.Context) error {
	return nil
}

func (g *gitlabBase) ResolveGroupIds() error {
	if g.store.InheritFromGroups {
		projectGroups, resp, err := g.projectsClient.ListProjectsGroups(g.store.ProjectID, nil)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabListProjectsGroups, err)
		if resp.StatusCode >= 400 && err != nil {
			return err
		}
		sort.Sort(ProjectGroupPathSorter(projectGroups))
		discoveredIds := make([]string, len(projectGroups))
		for i, group := range projectGroups {
			discoveredIds[i] = strconv.Itoa(group.ID)
		}
		g.store.GroupIDs = discoveredIds
	}
	return nil
}

// Validate will use the gitlab projectVariablesClient/groupVariablesClient to validate the gitlab provider using the ListVariable call to ensure get permissions without needing a specific key.
func (g *gitlabBase) Validate() (esv1beta1.ValidationResult, error) {
	if g.store.ProjectID != "" {
		_, resp, err := g.projectVariablesClient.ListVariables(g.store.ProjectID, nil)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectListVariables, err)
		if err != nil {
			return esv1beta1.ValidationResultError, fmt.Errorf(errList, err)
		} else if resp == nil || resp.StatusCode != http.StatusOK {
			return esv1beta1.ValidationResultError, fmt.Errorf(errProjectAuth, g.store.ProjectID)
		}

		err = g.ResolveGroupIds()
		if err != nil {
			return esv1beta1.ValidationResultError, fmt.Errorf(errList, err)
		}
		log.V(1).Info("discovered project groups", "name", g.store.GroupIDs)
	}

	if len(g.store.GroupIDs) > 0 {
		for _, groupID := range g.store.GroupIDs {
			_, resp, err := g.groupVariablesClient.ListVariables(groupID, nil)
			metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupListVariables, err)
			if err != nil {
				return esv1beta1.ValidationResultError, fmt.Errorf(errList, err)
			} else if resp == nil || resp.StatusCode != http.StatusOK {
				return esv1beta1.ValidationResultError, fmt.Errorf(errGroupAuth, groupID)
			}
		}
	}

	return esv1beta1.ValidationResultReady, nil
}
