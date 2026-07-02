/*
Copyright © The ESO Authors

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

package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

// MergePolicy controls how user-supplied custom metadata is reconciled with the
// metadata already present on a remote Vault KV v2 secret.
type MergePolicy string

const (
	// MergePolicyMerge merges the supplied custom metadata into the existing remote metadata.
	MergePolicyMerge MergePolicy = "Merge"
	// MergePolicyReplace replaces the existing remote metadata with the supplied custom metadata.
	MergePolicyReplace MergePolicy = "Replace"
)

const (
	managedByKey   = "managed-by"
	managedByValue = "external-secrets"
)

const customMetadataKey = "custom_metadata"

// PushSecretMetadataSpec is the Vault-specific spec carried by a PushSecretMetadata
// envelope. It defines the custom metadata to push and the policy used to merge it.
type PushSecretMetadataSpec struct {
	CustomMetadata map[string]string `json:"customMetadata,omitempty"`
	MergePolicy    MergePolicy       `json:"mergePolicy,omitempty"`
}

func (c *client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	value, err := esutils.ExtractSecretData(data, secret)
	if err != nil {
		return err
	}

	vaultSecret, err := c.readSecret(ctx, data.GetRemoteKey(), "")
	secretExists := true
	if err != nil {
		if errors.Is(err, esv1.NoSecretError{}) {
			secretExists = false
		} else {
			return err
		}
	}

	path := c.buildPath(data.GetRemoteKey())
	switch c.store.Version {
	case esv1.VaultKVStoreV1:
		return c.pushSecretKV1(ctx, path, vaultSecret, value, data)
	case esv1.VaultKVStoreV2:
		return c.pushSecretKV2(ctx, path, secretExists, vaultSecret, value, data)
	default:
		return fmt.Errorf("unsupported vault KV store version: %s", c.store.Version)
	}
}

func (c *client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	path := c.buildPath(remoteRef.GetRemoteKey())
	metaPath, err := c.buildMetadataPath(remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}
	// Retrieve the secret map from vault and convert the secret value in string form.
	secretVal, err := c.readSecret(ctx, path, "")
	// If error is not of type secret not found, we should error
	if err != nil && errors.Is(err, esv1.NoSecretError{}) {
		return nil
	}
	if err != nil {
		return err
	}
	remoteMeta, err := c.readSecretMetadata(ctx, remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}
	manager, ok := remoteMeta[managedByKey]
	if !ok || manager != managedByValue {
		return nil
	}
	// If Push for a Property, we need to delete the property and update the secret
	if remoteRef.GetProperty() != "" {
		delete(secretVal, remoteRef.GetProperty())
		// If the only key left in the remote secret is the reference of the metadata.
		if c.store.Version == esv1.VaultKVStoreV1 {
			if _, onlyMeta := secretVal[customMetadataKey]; onlyMeta && len(secretVal) == 1 {
				delete(secretVal, customMetadataKey)
			}
		}
		if len(secretVal) > 0 {
			secretToPush := secretVal
			if c.store.Version == esv1.VaultKVStoreV2 {
				secretToPush = map[string]any{
					"data": secretVal,
				}
			}
			_, err = c.logical.WriteWithContext(ctx, path, secretToPush)
			metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultDeleteSecret, err)
			return err
		}
	}
	_, err = c.logical.DeleteWithContext(ctx, path)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultDeleteSecret, err)
	if err != nil {
		return fmt.Errorf("could not delete secret %v: %w", remoteRef.GetRemoteKey(), err)
	}
	if c.store.Version == esv1.VaultKVStoreV2 {
		_, err = c.logical.DeleteWithContext(ctx, metaPath)
		metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultDeleteSecret, err)
		if err != nil {
			return fmt.Errorf("could not delete secret metadata %v: %w", remoteRef.GetRemoteKey(), err)
		}
	}
	return nil
}

func (c *client) pushSecretKV1(ctx context.Context, path string, vaultSecret map[string]any, value []byte, data esv1.PushSecretData) error {
	if vaultSecret != nil {
		// KV1 Metadata verification
		customMetaRaw, ok := vaultSecret[customMetadataKey]
		if !ok || customMetaRaw == nil {
			return errors.New("secret not managed by external-secrets (no custom_metadata found)")
		}

		cmMap, ok := customMetaRaw.(map[string]any)
		if !ok || cmMap[managedByKey] != managedByValue {
			return errors.New("secret not managed by external-secrets")
		}

		delete(vaultSecret, customMetadataKey)
	}

	var (
		mergedData      map[string]any
		dataNeedsUpdate bool
		err             error
	)
	mergedData, dataNeedsUpdate, err = reconcileData(vaultSecret, value, data.GetProperty())
	if err != nil {
		return err
	}

	if vaultSecret != nil && !dataNeedsUpdate {
		return nil // secret exists and is already up-to-date
	}

	// Inject hardcoded KV1 metadata
	mergedData[customMetadataKey] = map[string]string{
		managedByKey: managedByValue,
	}

	_, err = c.logical.WriteWithContext(ctx, path, mergedData)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
	if err != nil {
		return fmt.Errorf("failed to write secret data: %w", err)
	}

	return nil
}

func (c *client) pushSecretKV2(ctx context.Context, path string, secretExists bool, vaultSecret map[string]any, value []byte, data esv1.PushSecretData) error {
	metaPath, err := c.buildMetadataPath(data.GetRemoteKey())
	if err != nil {
		return err
	}

	// 1. Resolve Data State
	var mergedData map[string]any
	var dataNeedsUpdate bool
	mergedData, dataNeedsUpdate, err = reconcileData(vaultSecret, value, data.GetProperty())
	if err != nil {
		return err
	}

	// 2. Fetch Remote Metadata (if the secret exists)
	var remoteMeta map[string]string
	if secretExists {
		remoteMeta, err = c.readSecretMetadata(ctx, data.GetRemoteKey())
		if err != nil {
			return err
		}
	}

	// 3. Resolve Metadata State
	parsedMetadata, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](data.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to parse push secret metadata: %w", err)
	}

	finalCustomMeta, metadataNeedsUpdate, err := reconcileKV2Metadata(secretExists, remoteMeta, parsedMetadata)
	if err != nil {
		return err
	}

	// 4. Unified Early Exit
	if !dataNeedsUpdate && !metadataNeedsUpdate {
		return nil
	}

	// 5. Execute Writes
	// Metadata is written before data intentionally: for new secrets this establishes
	// the "managed-by" ownership tag first. If the data write subsequently fails the
	// reconciler will retry; because managed-by is already present it will proceed
	// straight to updating the data without re-checking ownership.
	if metadataNeedsUpdate {
		metaPayload := map[string]any{customMetadataKey: finalCustomMeta}
		_, err = c.logical.WriteWithContext(ctx, metaPath, metaPayload)
		metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
		if err != nil {
			return fmt.Errorf("failed to write secret metadata: %w", err)
		}
	}

	if dataNeedsUpdate {
		secretToPush := map[string]any{"data": mergedData}

		if c.store.CheckAndSet != nil && c.store.CheckAndSet.Required {
			casVersion, casErr := c.getCASVersion(ctx, data.GetRemoteKey(), secretExists)
			if casErr != nil {
				return fmt.Errorf("failed to get CAS version: %w", casErr)
			}
			secretToPush["options"] = map[string]any{"cas": casVersion}
		}

		_, err = c.logical.WriteWithContext(ctx, path, secretToPush)
		metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
		if err != nil {
			return fmt.Errorf("failed to write secret data: %w", err)
		}
	}

	return nil
}

// getCASVersion retrieves the current version of the secret for check-and-set operations.
// Returns:
//   - 0 for new secrets (CAS version 0 means "create only if doesn't exist")
//   - N for existing secrets (CAS version N means "update only if current version is N")
func (c *client) getCASVersion(ctx context.Context, remoteKey string, secretExists bool) (int, error) {
	// For new secrets, use CAS version 0 (create only if doesn't exist)
	if !secretExists {
		return 0, nil
	}

	// For existing secrets, read the full metadata to get current version
	metaPath, err := c.buildMetadataPath(remoteKey)
	if err != nil {
		return 0, fmt.Errorf("failed to build metadata path: %w", err)
	}

	secret, err := c.logical.ReadWithDataWithContext(ctx, metaPath, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to read metadata: %w", err)
	}

	if secret == nil || secret.Data == nil {
		// If no metadata found for an existing secret, assume this is version 1.
		// This can happen with older secrets that were created before version tracking.
		// Vault KV v2 secrets start at version 1 (not 0) when first created.
		return 1, nil
	}

	return getCurrentVersionFromMetadata(secret.Data)
}

func getCurrentVersionFromMetadata(data map[string]any) (int, error) {
	if currentVersion, ok := data["current_version"]; ok {
		switch v := currentVersion.(type) {
		case int:
			return v, nil
		case float64:
			return int(v), nil
		case json.Number:
			intVal, err := v.Int64()
			if err != nil {
				return 0, fmt.Errorf("failed to convert json.Number to int: %w", err)
			}
			return int(intVal), nil
		default:
			return 0, fmt.Errorf("unexpected type for current_version: %T", currentVersion)
		}
	}

	// If metadata exists but no current_version found, assume this is version 1.
	// This handles edge cases with legacy secrets or incomplete metadata.
	// Vault KV v2 secrets start at version 1, so this is the safest assumption.
	return 1, nil
}

func reconcileData(vaultSecret map[string]any, value []byte, property string) (map[string]any, bool, error) {
	if property == "" {
		return reconcileWholeSecret(vaultSecret, value)
	}
	return reconcileSecretProperty(vaultSecret, value, property)
}

func reconcileWholeSecret(vaultSecret map[string]any, value []byte) (map[string]any, bool, error) {
	incomingSecretMap := make(map[string]any)
	if err := json.Unmarshal(value, &incomingSecretMap); err != nil {
		if vaultSecret == nil {
			return nil, false, errors.New("error unmarshalling vault secret: invalid JSON format")
		}
		return nil, false, errors.New("error unmarshalling incoming secret value: invalid JSON format")
	}

	needsUpdate := !reflect.DeepEqual(vaultSecret, incomingSecretMap)
	return incomingSecretMap, needsUpdate, nil
}

func reconcileSecretProperty(vaultSecret map[string]any, value []byte, property string) (map[string]any, bool, error) {
	secretVal := make(map[string]any)
	needsUpdate := true

	if vaultSecret != nil {
		maps.Copy(secretVal, vaultSecret)

		// Check if the specific property matches to avoid unnecessary writes
		if existingVal, ok := vaultSecret[property]; ok {
			if d, isStr := existingVal.(string); isStr && bytes.Equal([]byte(d), value) {
				needsUpdate = false
			}
		}
	}
	secretVal[property] = string(value)

	return secretVal, needsUpdate, nil
}

func reconcileKV2Metadata(secretExists bool, remoteMeta map[string]string, parsedMeta *metadata.PushSecretMetadata[PushSecretMetadataSpec]) (map[string]string, bool, error) {
	desiredCustomMeta := map[string]string{managedByKey: managedByValue}
	mergePolicy := MergePolicyMerge // Default Policy

	if parsedMeta != nil {
		if parsedMeta.Spec.MergePolicy != "" {
			switch parsedMeta.Spec.MergePolicy {
			case MergePolicyMerge, MergePolicyReplace:
				mergePolicy = parsedMeta.Spec.MergePolicy
			default:
				return nil, false, fmt.Errorf("unsupported merge policy %q, must be one of: Merge, Replace", parsedMeta.Spec.MergePolicy)
			}
		}
		if parsedMeta.Spec.CustomMetadata != nil {
			maps.Copy(desiredCustomMeta, parsedMeta.Spec.CustomMetadata)
		}
	}

	// The managed-by key is reserved and must never be user-overridable. Re-assert it
	// here, after user-supplied custom metadata has been merged in, so every downstream
	// path (the new-secret return below and all merge-policy branches) inherits the
	// correct ownership tag from this single source.
	desiredCustomMeta[managedByKey] = managedByValue

	if !secretExists {
		return desiredCustomMeta, true, nil
	}

	if manager, ok := remoteMeta[managedByKey]; !ok || manager != managedByValue {
		return nil, false, errors.New("secret not managed by external-secrets")
	}

	finalCustomMeta := make(map[string]string)
	switch mergePolicy {
	case MergePolicyMerge:
		if remoteMeta != nil {
			maps.Copy(finalCustomMeta, remoteMeta)
		}
		maps.Copy(finalCustomMeta, desiredCustomMeta)
	case MergePolicyReplace:
		maps.Copy(finalCustomMeta, desiredCustomMeta)
	}

	needsUpdate := !maps.Equal(remoteMeta, finalCustomMeta)

	return finalCustomMeta, needsUpdate, nil
}
