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
	"errors"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/find"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

// GetAllSecrets syncs all gitlab project and group variables into a single Kubernetes Secret.
func (g *gitlabBase) GetAllSecrets(_ context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if esutils.IsNil(g.projectVariablesClient) {
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

	err := g.ResolveGroupIDs()
	if err != nil {
		return nil, err
	}

	secretData, err := g.fetchSecretData(effectiveEnvironment, matcher)
	if err != nil {
		return nil, err
	}

	// _Note_: fetchProjectVariables alters secret data map
	if err := g.fetchProjectVariables(effectiveEnvironment, matcher, secretData); err != nil {
		return nil, err
	}

	return secretData, nil
}

func (g *gitlabBase) fetchProjectVariables(effectiveEnvironment string, matcher *find.Matcher, secretData map[string][]byte) error {
	var popts = &gitlab.ListProjectVariablesOptions{PerPage: 100}
	nonWildcardSet := make(map[string]bool)
	for projectPage := 1; ; projectPage++ {
		popts.Page = projectPage
		projectData, response, err := g.projectVariablesClient.ListVariables(g.store.ProjectID, popts)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectListVariables, err)
		if err != nil {
			return err
		}

		processProjectVariables(projectData, effectiveEnvironment, matcher, secretData, nonWildcardSet)
		if response.CurrentPage >= response.TotalPages {
			break
		}
	}

	return nil
}

func processProjectVariables(
	projectData []*gitlab.ProjectVariable,
	effectiveEnvironment string,
	matcher *find.Matcher,
	secretData map[string][]byte,
	nonWildcardSet map[string]bool,
) {
	for _, data := range projectData {
		matching, key, isWildcard := matchesFilter(effectiveEnvironment, data.EnvironmentScope, data.Key, matcher)
		if !matching {
			continue
		}
		if isWildcard && nonWildcardSet[key] {
			continue
		}
		secretData[key] = []byte(data.Value)
		if !isWildcard {
			nonWildcardSet[key] = true
		}
	}
}

func (g *gitlabBase) fetchSecretData(effectiveEnvironment string, matcher *find.Matcher) (map[string][]byte, error) {
	var gopts = &gitlab.ListGroupVariablesOptions{PerPage: 100}
	secretData := make(map[string][]byte)
	for _, groupID := range g.store.GroupIDs {
		if err := g.setVariablesForGroupID(effectiveEnvironment, matcher, gopts, groupID, secretData); err != nil {
			return nil, err
		}
	}

	return secretData, nil
}

func (g *gitlabBase) setVariablesForGroupID(
	effectiveEnvironment string,
	matcher *find.Matcher,
	gopts *gitlab.ListGroupVariablesOptions,
	groupID string,
	secretData map[string][]byte,
) error {
	for groupPage := 1; ; groupPage++ {
		gopts.Page = groupPage
		groupVars, response, err := g.groupVariablesClient.ListVariables(groupID, gopts)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupListVariables, err)
		if err != nil {
			return err
		}
		g.setGroupValues(effectiveEnvironment, matcher, groupVars, secretData)

		if response.CurrentPage >= response.TotalPages {
			break
		}
	}
	return nil
}

func (g *gitlabBase) setGroupValues(
	effectiveEnvironment string,
	matcher *find.Matcher,
	groupVars []*gitlab.GroupVariable,
	secretData map[string][]byte,
) {
	for _, data := range groupVars {
		matching, key, isWildcard := matchesFilter(effectiveEnvironment, data.EnvironmentScope, data.Key, matcher)
		if !matching {
			continue
		}
		// Check if a more specific variable already exists (project environment > project variable > group environment > group variable)
		_, exists := secretData[key]
		if exists && isWildcard {
			continue
		}
		secretData[key] = []byte(data.Value)
	}
}

// ExtractTag extracts the environment scope from the provided tags map.
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
