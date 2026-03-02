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
	"fmt"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

// SecretExists checks if a secret exists in GitLab.
// Note: PushSecretRemoteRef doesn't include metadata, so this uses store configuration only.
func (g *gitlabBase) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	if esutils.IsNil(g.projectVariablesClient) && esutils.IsNil(g.groupVariablesClient) {
		return false, errors.New(errUninitializedGitlabProvider)
	}

	// Get the remote key and replace hyphens with underscores for GitLab API
	remoteKey := strings.ReplaceAll(remoteRef.GetRemoteKey(), "-", "_")

	environmentScope := g.store.Environment
	if environmentScope == "" {
		environmentScope = "*"
	}

	// Check project variable first if project ID is set
	if g.store.ProjectID != "" {
		exists, err := g.projectVariableExists(remoteKey, environmentScope)
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
	}

	// Check group variables if configured
	if len(g.store.GroupIDs) > 0 {
		exists, err := g.groupVariableExists(g.store.GroupIDs[0], remoteKey, environmentScope)
		if err != nil {
			return false, err
		}
		return exists, nil
	}

	return false, nil
}

// DeleteSecret deletes a variable from GitLab.
// TODO: Talk about this and how to handle this potentially.
// LIMITATION: This can only delete from the store's configured projectID or first groupID.
// Variables pushed with metadata-based groupID/projectID overrides cannot be automatically
// deleted because the SecretsClient.DeleteSecret interface doesn't provide metadata.
// Users must either:
// 1. Not use deletionPolicy: Delete with metadata overrides, OR
// 2. Manually clean up variables pushed to non-default locations
func (g *gitlabBase) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	if esutils.IsNil(g.projectVariablesClient) && esutils.IsNil(g.groupVariablesClient) {
		return errors.New(errUninitializedGitlabProvider)
	}

	// Get the remote key and replace hyphens with underscores for GitLab API
	remoteKey := strings.ReplaceAll(remoteRef.GetRemoteKey(), "-", "_")

	environmentScope := g.store.Environment
	if environmentScope == "" {
		environmentScope = "*"
	}

	// Only delete from the store's default location (no metadata available)
	if g.store.ProjectID != "" {
		opts := &gitlab.RemoveProjectVariableOptions{
			Filter: &gitlab.VariableFilter{EnvironmentScope: environmentScope},
		}
		_, err := g.projectVariablesClient.RemoveVariable(g.store.ProjectID, remoteKey, opts)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableDelete, err)

		if err == nil || errors.Is(err, gitlab.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("failed to delete from project %q: %w", g.store.ProjectID, err)
	}

	// Delete from first configured group only
	if len(g.store.GroupIDs) > 0 {
		opts := &gitlab.RemoveGroupVariableOptions{
			Filter: &gitlab.VariableFilter{EnvironmentScope: environmentScope},
		}
		_, err := g.groupVariablesClient.RemoveVariable(g.store.GroupIDs[0], remoteKey, opts)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupVariableDelete, err)

		if err == nil || errors.Is(err, gitlab.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("failed to delete from group %q: %w", g.store.GroupIDs[0], err)
	}

	return fmt.Errorf("%s for delete operation", errNoProjectOrGroup)
}
