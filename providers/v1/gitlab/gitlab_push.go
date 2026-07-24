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

	gitlab "gitlab.com/gitlab-org/api/client-go"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

// PushSecret creates or updates a variable in GitLab.
func (g *gitlabBase) PushSecret(ctx context.Context, secret *corev1.Secret, psd esv1.PushSecretData) error {
	if esutils.IsNil(g.projectVariablesClient) && esutils.IsNil(g.groupVariablesClient) {
		return errors.New(errUninitializedGitlabProvider)
	}

	value, err := esutils.ExtractSecretData(psd, secret)
	if err != nil {
		return fmt.Errorf("failed to extract secret data: %w", err)
	}

	// Get the remote key and replace hyphens with underscores for GitLab API
	remoteKey := strings.ReplaceAll(psd.GetRemoteKey(), "-", "_")

	// Extract metadata to check for groupID or projectID overrides
	metadata := psd.GetMetadata()
	groupID, projectID := extractTargetFromMetadata(metadata)

	// Determine which type of variable to create/update
	// Priority: metadata projectID > metadata groupID > store projectID > store groupIDs
	if projectID != "" {
		return g.pushProjectVariableWithID(projectID, remoteKey, string(value), psd)
	}

	if groupID != "" {
		err := g.pushGroupVariable(groupID, remoteKey, string(value), psd)
		if err != nil {
			return fmt.Errorf("failed to push to group %q: %w", groupID, err)
		}
		return nil
	}

	// Fall back to store configuration
	if g.store.ProjectID != "" {
		return g.pushProjectVariable(remoteKey, string(value), psd)
	}

	// If no project ID is set, try to push to the first group
	if len(g.store.GroupIDs) > 0 {
		err := g.pushGroupVariable(g.store.GroupIDs[0], remoteKey, string(value), psd)
		if err != nil {
			return fmt.Errorf("failed to push to group %q: %w", g.store.GroupIDs[0], err)
		}
		return nil
	}

	return fmt.Errorf("%s for push operation", errNoProjectOrGroup)
}

// extractTargetFromMetadata extracts groupID and projectID from PushSecret metadata.
// Returns empty strings if metadata is nil or doesn't contain the fields.
func extractTargetFromMetadata(metadata *apiextensionsv1.JSON) (groupID, projectID string) {
	if metadata == nil {
		return "", ""
	}

	var metadataMap map[string]interface{}
	if err := json.Unmarshal(metadata.Raw, &metadataMap); err != nil {
		return "", ""
	}

	if gid, ok := metadataMap["groupID"].(string); ok {
		groupID = gid
	}
	if pid, ok := metadataMap["projectID"].(string); ok {
		projectID = pid
	}

	return groupID, projectID
}

// pushProjectVariableWithID creates or updates a project-level variable with a specific project ID.
func (g *gitlabBase) pushProjectVariableWithID(projectID, key, value string, psd esv1.PushSecretData) error {
	environmentScope := g.store.Environment
	if environmentScope == "" {
		environmentScope = "*"
	}

	// Extract metadata if provided
	metadata := psd.GetMetadata()

	// Check if variable exists
	exists, err := g.projectVariableExistsWithID(projectID, key, environmentScope)
	if err != nil {
		return fmt.Errorf("failed to check if variable exists: %w", err)
	}

	if exists {
		// Check if variable is managed by ESO before updating
		vopts := &gitlab.GetProjectVariableOptions{
			Filter: &gitlab.VariableFilter{EnvironmentScope: environmentScope},
		}
		existingVar, _, getErr := g.projectVariablesClient.GetVariable(projectID, key, vopts)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableGet, getErr)
		if getErr != nil {
			return fmt.Errorf("failed to get existing variable: %w", getErr)
		}
		if existingVar != nil && !isManagedByESO(existingVar.Description) {
			return fmt.Errorf(errNotManagedByESO, key)
		}

		// Update existing variable
		opts := &gitlab.UpdateProjectVariableOptions{
			Value:            gitlab.Ptr(value),
			EnvironmentScope: gitlab.Ptr(environmentScope),
		}

		// Apply metadata options if provided
		applyMetadataToUpdateOptions(metadata, opts)

		_, _, err := g.projectVariablesClient.UpdateVariable(projectID, key, opts, gitlab.WithContext(context.Background()))
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableUpdate, err)
		if err != nil {
			// If update fails with 404, the variable might have a different environment scope
			// Try to get and update with the correct scope
			if errors.Is(err, gitlab.ErrNotFound) {
				existingVar, _, getErr := g.projectVariablesClient.GetVariable(projectID, key, vopts)
				if getErr == nil && existingVar != nil {
					opts.Filter = &gitlab.VariableFilter{EnvironmentScope: existingVar.EnvironmentScope}
					_, _, err = g.projectVariablesClient.UpdateVariable(projectID, key, opts)
					metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableUpdate, err)
				}
			}
		}
		return err
	}

	// Create new variable
	opts := &gitlab.CreateProjectVariableOptions{
		Key:              gitlab.Ptr(key),
		Value:            gitlab.Ptr(value),
		EnvironmentScope: gitlab.Ptr(environmentScope),
		Description:      gitlab.Ptr(managedByDescription),
	}

	// Apply metadata options if provided
	applyMetadataToCreateOptions(metadata, opts)

	_, _, err = g.projectVariablesClient.CreateVariable(projectID, opts)
	metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableCreate, err)
	return err
}

// pushProjectVariable creates or updates a project-level variable.
func (g *gitlabBase) pushProjectVariable(key, value string, psd esv1.PushSecretData) error {
	environmentScope := g.store.Environment
	if environmentScope == "" {
		environmentScope = "*"
	}

	// Extract metadata if provided
	metadata := psd.GetMetadata()

	// Check if variable exists
	exists, err := g.projectVariableExists(key, environmentScope)
	if err != nil {
		return fmt.Errorf("failed to check if variable exists: %w", err)
	}

	if exists {
		// Check if variable is managed by ESO before updating
		vopts := &gitlab.GetProjectVariableOptions{
			Filter: &gitlab.VariableFilter{EnvironmentScope: environmentScope},
		}
		existingVar, _, getErr := g.projectVariablesClient.GetVariable(g.store.ProjectID, key, vopts)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableGet, getErr)
		if getErr != nil {
			return fmt.Errorf("failed to get existing variable: %w", getErr)
		}
		if existingVar != nil && !isManagedByESO(existingVar.Description) {
			return fmt.Errorf(errNotManagedByESO, key)
		}

		// Update existing variable
		opts := &gitlab.UpdateProjectVariableOptions{
			Value:            gitlab.Ptr(value),
			EnvironmentScope: gitlab.Ptr(environmentScope),
		}

		// Apply metadata options if provided
		applyMetadataToUpdateOptions(metadata, opts)

		_, _, err := g.projectVariablesClient.UpdateVariable(g.store.ProjectID, key, opts, gitlab.WithContext(context.Background()))
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableUpdate, err)
		if err != nil {
			// If update fails with 404, the variable might have a different environment scope
			// Try to get and update with the correct scope
			if errors.Is(err, gitlab.ErrNotFound) {
				existingVar, _, getErr := g.projectVariablesClient.GetVariable(g.store.ProjectID, key, vopts)
				if getErr == nil && existingVar != nil {
					if !isManagedByESO(existingVar.Description) {
						return fmt.Errorf(errNotManagedByESO, key)
					}
					opts.Filter = &gitlab.VariableFilter{EnvironmentScope: existingVar.EnvironmentScope}
					_, _, err = g.projectVariablesClient.UpdateVariable(g.store.ProjectID, key, opts)
					metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableUpdate, err)
				}
			}
		}
		return err
	}

	// Create new variable
	opts := &gitlab.CreateProjectVariableOptions{
		Key:              gitlab.Ptr(key),
		Value:            gitlab.Ptr(value),
		EnvironmentScope: gitlab.Ptr(environmentScope),
		Description:      gitlab.Ptr(managedByDescription),
	}

	// Apply metadata options if provided
	applyMetadataToCreateOptions(metadata, opts)

	_, _, err = g.projectVariablesClient.CreateVariable(g.store.ProjectID, opts)
	metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableCreate, err)
	return err
}

// pushGroupVariable creates or updates a group-level variable.
func (g *gitlabBase) pushGroupVariable(groupID, key, value string, psd esv1.PushSecretData) error {
	environmentScope := g.store.Environment
	if environmentScope == "" {
		environmentScope = "*"
	}

	// Extract metadata if provided
	metadata := psd.GetMetadata()

	// Check if variable exists
	exists, err := g.groupVariableExists(groupID, key, environmentScope)
	if err != nil {
		return fmt.Errorf("failed to check if variable %q exists in group %q: %w", key, groupID, err)
	}

	if exists {
		// Check if variable is managed by ESO before updating
		vopts := &gitlab.GetGroupVariableOptions{
			Filter: &gitlab.VariableFilter{EnvironmentScope: environmentScope},
		}
		existingVar, _, getErr := g.groupVariablesClient.GetVariable(groupID, key, vopts)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupGetVariable, getErr)
		if getErr != nil {
			return fmt.Errorf("failed to get existing variable: %w", getErr)
		}
		if existingVar != nil && !isManagedByESO(existingVar.Description) {
			return fmt.Errorf(errNotManagedByESO, key)
		}

		// Update existing variable
		opts := &gitlab.UpdateGroupVariableOptions{
			Value:            gitlab.Ptr(value),
			EnvironmentScope: gitlab.Ptr(environmentScope),
		}

		// Apply metadata options if provided
		applyMetadataToGroupUpdateOptions(metadata, opts)

		_, _, err := g.groupVariablesClient.UpdateVariable(groupID, key, opts)
		metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupVariableUpdate, err)
		if err != nil {
			return fmt.Errorf("failed to update variable %q in group %q (env: %q): %w", key, groupID, environmentScope, err)
		}
		return nil
	}

	// Create new variable
	opts := &gitlab.CreateGroupVariableOptions{
		Key:              gitlab.Ptr(key),
		Value:            gitlab.Ptr(value),
		EnvironmentScope: gitlab.Ptr(environmentScope),
		Description:      gitlab.Ptr(managedByDescription),
	}

	// Apply metadata options if provided
	applyMetadataToGroupCreateOptions(metadata, opts)

	_, _, err = g.groupVariablesClient.CreateVariable(groupID, opts)
	metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupVariableCreate, err)
	if err != nil {
		return fmt.Errorf("failed to create variable %q in group %q (env: %q): %w", key, groupID, environmentScope, err)
	}
	return nil
}

// projectVariableExists checks if a project variable exists.
// projectVariableExists checks if a project variable exists in the default project.
// Uses the project ID from the store configuration.
func (g *gitlabBase) projectVariableExists(key, environmentScope string) (bool, error) {
	return g.projectVariableExistsWithID(g.store.ProjectID, key, environmentScope)
}

// projectVariableExistsWithID checks if a project variable exists with a specific project ID.
// Returns true if the variable exists, false if not found, or an error for other API failures.
func (g *gitlabBase) projectVariableExistsWithID(projectID, key, environmentScope string) (bool, error) {
	opts := &gitlab.GetProjectVariableOptions{
		Filter: &gitlab.VariableFilter{EnvironmentScope: environmentScope},
	}

	_, resp, err := g.projectVariablesClient.GetVariable(projectID, key, opts)
	metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabProjectVariableGet, err)

	if err != nil {
		if errors.Is(err, gitlab.ErrNotFound) || (resp != nil && resp.StatusCode == http.StatusNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// groupVariableExists checks if a group variable exists for the specified group.
// Returns true if the variable exists, false if not found, or an error for other API failures.
func (g *gitlabBase) groupVariableExists(groupID, key, environmentScope string) (bool, error) {
	opts := &gitlab.GetGroupVariableOptions{
		Filter: &gitlab.VariableFilter{EnvironmentScope: environmentScope},
	}

	_, resp, err := g.groupVariablesClient.GetVariable(groupID, key, opts)
	metrics.ObserveAPICall(constants.ProviderGitLab, constants.CallGitLabGroupGetVariable, err)

	if err != nil {
		if errors.Is(err, gitlab.ErrNotFound) || (resp != nil && resp.StatusCode == http.StatusNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// metadataOptions holds common GitLab variable options extracted from metadata.
type metadataOptions struct {
	Masked       *bool
	Protected    *bool
	Raw          *bool
	Description  *string
	VariableType *gitlab.VariableTypeValue
}

// parseMetadata extracts common GitLab variable options from PushSecret metadata.
func parseMetadata(metadata *apiextensionsv1.JSON) *metadataOptions {
	if metadata == nil {
		return nil
	}

	var metadataMap map[string]interface{}
	if err := json.Unmarshal(metadata.Raw, &metadataMap); err != nil {
		return nil
	}

	opts := &metadataOptions{}

	if masked, ok := metadataMap["masked"].(bool); ok {
		opts.Masked = gitlab.Ptr(masked)
	}
	if protected, ok := metadataMap["protected"].(bool); ok {
		opts.Protected = gitlab.Ptr(protected)
	}
	if raw, ok := metadataMap["raw"].(bool); ok {
		opts.Raw = gitlab.Ptr(raw)
	}
	if description, ok := metadataMap["description"].(string); ok {
		opts.Description = gitlab.Ptr(description)
	}
	if variableType, ok := metadataMap["variable_type"].(string); ok {
		if variableType == "file" {
			opts.VariableType = gitlab.Ptr(gitlab.FileVariableType)
		} else {
			opts.VariableType = gitlab.Ptr(gitlab.EnvVariableType)
		}
	}

	return opts
}

// applyMetadataToCreateOptions applies metadata to project variable create options.
func applyMetadataToCreateOptions(metadata *apiextensionsv1.JSON, opts *gitlab.CreateProjectVariableOptions) {
	parsed := parseMetadata(metadata)
	if parsed == nil {
		return
	}

	if parsed.Masked != nil {
		opts.Masked = parsed.Masked
	}
	if parsed.Protected != nil {
		opts.Protected = parsed.Protected
	}
	if parsed.Raw != nil {
		opts.Raw = parsed.Raw
	}
	if parsed.Description != nil {
		opts.Description = parsed.Description
	}
	if parsed.VariableType != nil {
		opts.VariableType = parsed.VariableType
	}
}

// applyMetadataToUpdateOptions applies metadata to project variable update options.
func applyMetadataToUpdateOptions(metadata *apiextensionsv1.JSON, opts *gitlab.UpdateProjectVariableOptions) {
	parsed := parseMetadata(metadata)
	if parsed == nil {
		return
	}

	if parsed.Masked != nil {
		opts.Masked = parsed.Masked
	}
	if parsed.Protected != nil {
		opts.Protected = parsed.Protected
	}
	if parsed.Raw != nil {
		opts.Raw = parsed.Raw
	}
	if parsed.Description != nil {
		opts.Description = parsed.Description
	}
	if parsed.VariableType != nil {
		opts.VariableType = parsed.VariableType
	}
}

// applyMetadataToGroupCreateOptions applies metadata to group variable create options.
func applyMetadataToGroupCreateOptions(metadata *apiextensionsv1.JSON, opts *gitlab.CreateGroupVariableOptions) {
	parsed := parseMetadata(metadata)
	if parsed == nil {
		return
	}

	if parsed.Masked != nil {
		opts.Masked = parsed.Masked
	}
	if parsed.Protected != nil {
		opts.Protected = parsed.Protected
	}
	if parsed.Raw != nil {
		opts.Raw = parsed.Raw
	}
	if parsed.Description != nil {
		opts.Description = parsed.Description
	}
	if parsed.VariableType != nil {
		opts.VariableType = parsed.VariableType
	}
}

// applyMetadataToGroupUpdateOptions applies metadata to group variable update options.
func applyMetadataToGroupUpdateOptions(metadata *apiextensionsv1.JSON, opts *gitlab.UpdateGroupVariableOptions) {
	parsed := parseMetadata(metadata)
	if parsed == nil {
		return
	}

	if parsed.Masked != nil {
		opts.Masked = parsed.Masked
	}
	if parsed.Protected != nil {
		opts.Protected = parsed.Protected
	}
	if parsed.Raw != nil {
		opts.Raw = parsed.Raw
	}
	if parsed.Description != nil {
		opts.Description = parsed.Description
	}
	if parsed.VariableType != nil {
		opts.VariableType = parsed.VariableType
	}
}

// isManagedByESO checks if a variable is managed by external-secrets.
// Returns true if the description contains "managed-by: external-secrets".
func isManagedByESO(description string) bool {
	return strings.Contains(description, managedByDescription)
}
