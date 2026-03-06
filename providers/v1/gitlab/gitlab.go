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

// Package gitlab implements a GitLab provider for External Secrets.
package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	errList                         = "could not verify whether the gitlabClient is valid: %w"
	errProjectAuth                  = "gitlabClient is not allowed to get secrets for project id [%s]"
	errGroupAuth                    = "gitlabClient is not allowed to get secrets for group id [%s]"
	errUninitializedGitlabProvider  = "provider gitlab is not initialized"
	errNameNotDefined               = "'find.name' is mandatory"
	errEnvironmentIsConstricted     = "'find.tags' is constrained by 'environment_scope' of the store"
	errTagsOnlyEnvironmentSupported = "'find.tags' only supports 'environment_scope'"
	errPathNotImplemented           = "'find.path' is not implemented in the GitLab provider"
	errJSONSecretUnmarshal          = "unable to unmarshal secret from JSON: %w"
	errNotImplemented               = "not implemented"
	errNoProjectOrGroup             = "no projectID or groupIDs configured"
	errNotManagedByESO              = "variable %q is not managed by external-secrets (description does not contain 'managed-by: external-secrets')"
	managedByDescription            = "managed-by: external-secrets"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &gitlabBase{}
var _ esv1.Provider = &Provider{}

// ProjectsClient is an interface for interacting with GitLab project APIs.
type ProjectsClient interface {
	ListProjectsGroups(pid any, opt *gitlab.ListProjectGroupOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectGroup, *gitlab.Response, error)
}

// ProjectVariablesClient is an interface for managing GitLab project variables.
type ProjectVariablesClient interface {
	GetVariable(pid any, key string, opt *gitlab.GetProjectVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error)
	ListVariables(pid any, opt *gitlab.ListProjectVariablesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.ProjectVariable, *gitlab.Response, error)
	CreateVariable(pid any, opt *gitlab.CreateProjectVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error)
	UpdateVariable(pid any, key string, opt *gitlab.UpdateProjectVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.ProjectVariable, *gitlab.Response, error)
	RemoveVariable(pid any, key string, opt *gitlab.RemoveProjectVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error)
}

// GroupVariablesClient is an interface for managing GitLab group variables.
type GroupVariablesClient interface {
	GetVariable(gid any, key string, opts *gitlab.GetGroupVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error)
	ListVariables(gid any, opt *gitlab.ListGroupVariablesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.GroupVariable, *gitlab.Response, error)
	CreateVariable(gid any, opt *gitlab.CreateGroupVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error)
	UpdateVariable(gid any, key string, opt *gitlab.UpdateGroupVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.GroupVariable, *gitlab.Response, error)
	RemoveVariable(gid any, key string, opt *gitlab.RemoveGroupVariableOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error)
}

// ProjectGroupPathSorter implements sort.Interface for sorting project groups by path length.
type ProjectGroupPathSorter []*gitlab.ProjectGroup

func (a ProjectGroupPathSorter) Len() int           { return len(a) }
func (a ProjectGroupPathSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ProjectGroupPathSorter) Less(i, j int) bool { return len(a[i].FullPath) < len(a[j].FullPath) }

var log = ctrl.Log.WithName("provider").WithName("gitlab")

// getAuth retrieves the GitLab access token from the Kubernetes secret.
func (g *gitlabBase) getAuth(ctx context.Context) (string, error) {
	return resolvers.SecretKeyRef(
		ctx,
		g.kube,
		g.storeKind,
		g.namespace,
		&g.store.Auth.SecretRef.AccessToken)
}

// getVariables retrieves a project variable with automatic wildcard retry on 404.
func (g *gitlabBase) getVariables(ref esv1.ExternalSecretDataRemoteRef, vopts *gitlab.GetProjectVariableOptions) (*gitlab.ProjectVariable, error) {
	// First attempt to get the variable
	data, _, err := g.projectVariablesClient.GetVariable(g.store.ProjectID, ref.Key, vopts)
	metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableGet, err)

	// If successful, return immediately
	if err == nil {
		return data, nil
	}

	// If not a "not found" error or environment is already wildcard, return the error
	if !errors.Is(err, gitlab.ErrNotFound) || isEmptyOrWildcard(g.store.Environment) {
		return nil, err
	}

	// Retry with wildcard environment scope
	opts := &gitlab.GetProjectVariableOptions{Filter: &gitlab.VariableFilter{EnvironmentScope: "*"}}
	data, _, err = g.projectVariablesClient.GetVariable(g.store.ProjectID, ref.Key, opts)
	metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableGet, err)

	if err != nil {
		return nil, fmt.Errorf("error getting variable %s from GitLab (including wildcard retry): %w", ref.Key, err)
	}

	return data, nil
}

// Close closes the client (no-op for GitLab).
func (g *gitlabBase) Close(_ context.Context) error {
	return nil
}

// ResolveGroupIDs resolves group IDs when inheritFromGroups is enabled.
func (g *gitlabBase) ResolveGroupIDs() error {
	if g.store.InheritFromGroups {
		projectGroups, resp, err := g.projectsClient.ListProjectsGroups(g.store.ProjectID, nil)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabListProjectsGroups, err)
		if resp.StatusCode >= 400 && err != nil {
			return err
		}
		sort.Sort(ProjectGroupPathSorter(projectGroups))
		discoveredIDs := make([]string, len(projectGroups))
		for i, group := range projectGroups {
			discoveredIDs[i] = strconv.Itoa(group.ID)
		}
		g.store.GroupIDs = discoveredIDs
	}
	return nil
}

// Validate validates the gitlab provider credentials and permissions.
func (g *gitlabBase) Validate() (esv1.ValidationResult, error) {
	if g.store.ProjectID != "" {
		_, resp, err := g.projectVariablesClient.ListVariables(g.store.ProjectID, nil)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectListVariables, err)
		if err != nil {
			return esv1.ValidationResultError, fmt.Errorf(errList, err)
		} else if resp == nil || resp.StatusCode != http.StatusOK {
			return esv1.ValidationResultError, fmt.Errorf(errProjectAuth, g.store.ProjectID)
		}

		err = g.ResolveGroupIDs()
		if err != nil {
			return esv1.ValidationResultError, fmt.Errorf(errList, err)
		}
		log.V(1).Info("discovered project groups", "name", g.store.GroupIDs)
	}

	if len(g.store.GroupIDs) > 0 {
		for _, groupID := range g.store.GroupIDs {
			_, resp, err := g.groupVariablesClient.ListVariables(groupID, nil)
			metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupListVariables, err)
			if err != nil {
				return esv1.ValidationResultError, fmt.Errorf(errList, err)
			} else if resp == nil || resp.StatusCode != http.StatusOK {
				return esv1.ValidationResultError, fmt.Errorf(errGroupAuth, groupID)
			}
		}
	}

	return esv1.ValidationResultReady, nil
}
