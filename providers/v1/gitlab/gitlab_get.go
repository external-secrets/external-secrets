/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

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
	"strings"

	"github.com/tidwall/gjson"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

func (g *gitlabBase) getGroupVariables(groupID string, ref esv1.ExternalSecretDataRemoteRef, gopts *gitlab.GetGroupVariableOptions) (*gitlab.GroupVariable, *gitlab.Response, error) {
	groupVar, resp, err := g.groupVariablesClient.GetVariable(groupID, ref.Key, gopts)
	metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupGetVariable, err)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound && !isEmptyOrWildcard(g.store.Environment) {
			if gopts == nil {
				gopts = &gitlab.GetGroupVariableOptions{}
			}
			if gopts.Filter == nil {
				gopts.Filter = &gitlab.VariableFilter{}
			}
			gopts.Filter.EnvironmentScope = "*"
			groupVar, resp, err = g.groupVariablesClient.GetVariable(groupID, ref.Key, gopts)
			metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupGetVariable, err)
			if err != nil || resp == nil {
				return nil, resp, fmt.Errorf("error getting group variable %s from GitLab: %w", ref.Key, err)
			}
		} else {
			return nil, resp, err
		}
	}

	return groupVar, resp, nil
}

func (g *gitlabBase) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if esutils.IsNil(g.projectVariablesClient) || esutils.IsNil(g.groupVariablesClient) {
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
	var gopts *gitlab.GetGroupVariableOptions
	var vopts *gitlab.GetProjectVariableOptions
	if g.store.Environment != "" {
		gopts = &gitlab.GetGroupVariableOptions{Filter: &gitlab.VariableFilter{EnvironmentScope: g.store.Environment}}
		vopts = &gitlab.GetProjectVariableOptions{Filter: &gitlab.VariableFilter{EnvironmentScope: g.store.Environment}}
	}

	data, err := g.getVariables(ref, vopts)
	if err == nil {
		return extractVariable(ref, data.Value)
	}

	// If project variable not found, try group variables
	if errors.Is(err, gitlab.ErrNotFound) {
		return g.tryGroupVariables(ref, gopts, err)
	}

	return nil, err
}

// tryGroupVariables attempts to retrieve the secret from group variables when project lookup fails.
func (g *gitlabBase) tryGroupVariables(ref esv1.ExternalSecretDataRemoteRef, gopts *gitlab.GetGroupVariableOptions, originalErr error) ([]byte, error) {
	// Load groupIds from the `InheritFromGroups` property
	if err := g.ResolveGroupIDs(); err != nil {
		return nil, err
	}

	for i := len(g.store.GroupIDs) - 1; i >= 0; i-- {
		groupID := g.store.GroupIDs[i]
		groupVar, _, err := g.getGroupVariables(groupID, ref, gopts)
		if err == nil {
			return extractVariable(ref, groupVar.Value)
		}

		// If a 404 error, continue to the next stage, otherwise exit early with error
		if errors.Is(err, gitlab.ErrNotFound) {
			continue
		}
		return nil, err
	}

	// No group variables found, return the original project error
	return nil, originalErr
}

func extractVariable(ref esv1.ExternalSecretDataRemoteRef, value string) ([]byte, error) {
	// If no property specified, return the raw value
	if ref.Property == "" {
		if value == "" {
			return nil, fmt.Errorf("invalid secret received. no secret string for key: %s", ref.Key)
		}
		return []byte(value), nil
	}

	// Extract property from JSON value
	val := gjson.Get(value, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

func (g *gitlabBase) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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
