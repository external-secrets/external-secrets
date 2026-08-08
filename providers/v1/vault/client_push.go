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

// MergePolicy defines how metadata should be merged when pushing secrets.
type MergePolicy string

const (
	// MergePolicyMerge indicates that metadata should be merged.
	MergePolicyMerge MergePolicy = "Merge"
	// MergePolicyReplace indicates that metadata should be replaced entirely.
	MergePolicyReplace MergePolicy = "Replace"
)

const (
	managedByKey   = "managed-by"
	managedByValue = "external-secrets"
)

const customMetadataKey = "custom_metadata"

// errNotManagedByESO is returned when touching a secret not managed by ESO.
var errNotManagedByESO = errors.New("secret not managed by external-secrets")

// isManagedByESO checks if metadata carries the ESO ownership tag
// (managed-by=external-secrets) to prevent clobbering unmanaged secrets.
func isManagedByESO[V any](meta map[string]V) bool {
	return any(meta[managedByKey]) == managedByValue
}

// PushSecretMetadataSpec defines the metadata specification for pushed secrets.
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
	if err != nil && !errors.Is(err, esv1.NoSecretError{}) {
		return err
	}

	path := c.buildPath(data.GetRemoteKey())
	switch c.store.Version {
	case esv1.VaultKVStoreV1:
		return c.pushSecretKV1(ctx, path, vaultSecret, value, data)
	case esv1.VaultKVStoreV2:
		return c.pushSecretKV2(ctx, path, vaultSecret, value, data)
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
	// Deleting an already-absent secret is a no-op.
	if err != nil && errors.Is(err, esv1.NoSecretError{}) {
		return nil
	}
	// Any other read error must be propagated, not ignored.
	if err != nil {
		return err
	}
	remoteMeta, err := c.readSecretMetadata(ctx, remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}
	if !isManagedByESO(remoteMeta) {
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
		// KV1 has no metadata endpoint; the ownership tag rides inside the
		// secret data under the custom_metadata key.
		cmMap, _ := vaultSecret[customMetadataKey].(map[string]any)
		if !isManagedByESO(cmMap) {
			return errNotManagedByESO
		}

		delete(vaultSecret, customMetadataKey)
	}

	mergedData, dataNeedsUpdate, err := reconcileData(vaultSecret, value, data.GetProperty())
	if err != nil {
		return err
	}

	if vaultSecret != nil && !dataNeedsUpdate {
		return nil // secret exists and is already up-to-date
	}

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

func (c *client) pushSecretKV2(ctx context.Context, path string, vaultSecret map[string]any, value []byte, data esv1.PushSecretData) error {
	secretExists := vaultSecret != nil

	metaPath, err := c.buildMetadataPath(data.GetRemoteKey())
	if err != nil {
		return err
	}

	mergedData, dataNeedsUpdate, err := reconcileData(vaultSecret, value, data.GetProperty())
	if err != nil {
		return err
	}

	var remoteMeta map[string]string
	if secretExists {
		remoteMeta, err = c.readSecretMetadata(ctx, data.GetRemoteKey())
		if err != nil {
			return err
		}
	}

	suppliedMeta, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](data.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to parse push secret metadata: %w", err)
	}

	finalCustomMeta, metadataNeedsUpdate, err := reconcileKV2Metadata(secretExists, remoteMeta, suppliedMeta)
	if err != nil {
		return err
	}

	if !dataNeedsUpdate && !metadataNeedsUpdate {
		return nil
	}

	// Metadata is written before data intentionally: for new secrets this establishes
	// the "managed-by" ownership tag first. If the data write subsequently fails the
	// reconciler will retry; because managed-by is already present it will proceed
	// straight to updating the data without re-checking ownership.
	if metadataNeedsUpdate {
		if err := c.writeKV2Metadata(ctx, metaPath, finalCustomMeta); err != nil {
			return err
		}
	}

	if dataNeedsUpdate {
		if err := c.writeKV2Data(ctx, path, mergedData, data.GetRemoteKey(), secretExists); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) writeKV2Metadata(ctx context.Context, metaPath string, finalCustomMeta map[string]string) error {
	metaPayload := map[string]any{customMetadataKey: finalCustomMeta}
	_, err := c.logical.WriteWithContext(ctx, metaPath, metaPayload)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
	if err != nil {
		return fmt.Errorf("failed to write secret metadata: %w", err)
	}
	return nil
}

func (c *client) writeKV2Data(ctx context.Context, path string, mergedData map[string]any, remoteKey string, secretExists bool) error {
	secretToPush := map[string]any{"data": mergedData}

	if c.store.CheckAndSet != nil && c.store.CheckAndSet.Required {
		casVersion, casErr := c.getCASVersion(ctx, remoteKey, secretExists)
		if casErr != nil {
			return fmt.Errorf("failed to get CAS version: %w", casErr)
		}
		secretToPush["options"] = map[string]any{"cas": casVersion}
	}

	_, err := c.logical.WriteWithContext(ctx, path, secretToPush)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
	if err != nil {
		return fmt.Errorf("failed to write secret data: %w", err)
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
			d, isStr := existingVal.(string)
			if !isStr {
				// Refuse to overwrite a non-string remote value (e.g. a nested
				// object) with a string; the caller must resolve the conflict.
				return nil, false, fmt.Errorf("error converting %s to string", property)
			}
			if bytes.Equal([]byte(d), value) {
				needsUpdate = false
			}
		}
	}
	secretVal[property] = string(value)

	return secretVal, needsUpdate, nil
}

// reconcileKV2Metadata computes the KV v2 custom_metadata map that PushSecret
// should end up with, and whether it differs from what Vault already has.
func reconcileKV2Metadata(secretExists bool, remoteMeta map[string]string, suppliedMeta *metadata.PushSecretMetadata[PushSecretMetadataSpec]) (map[string]string, bool, error) {
	desiredCustomMeta := map[string]string{managedByKey: managedByValue}
	mergePolicy := MergePolicyMerge // Default Policy

	if suppliedMeta != nil {
		if suppliedMeta.Spec.MergePolicy != "" {
			switch suppliedMeta.Spec.MergePolicy {
			case MergePolicyMerge, MergePolicyReplace:
				mergePolicy = suppliedMeta.Spec.MergePolicy
			default:
				return nil, false, fmt.Errorf("unsupported merge policy %q, must be one of: Merge, Replace", suppliedMeta.Spec.MergePolicy)
			}
		}
		if suppliedMeta.Spec.CustomMetadata != nil {
			maps.Copy(desiredCustomMeta, suppliedMeta.Spec.CustomMetadata)
		}
	}

	// Re-assert after merging user-supplied custom metadata so every downstream
	// path (the new-secret return below and all merge-policy branches) inherits
	// the correct ownership tag from this single source.
	desiredCustomMeta[managedByKey] = managedByValue

	if !secretExists {
		return desiredCustomMeta, true, nil
	}

	// Refuse to touch a secret ESO doesn't already own.
	if !isManagedByESO(remoteMeta) {
		return nil, false, errNotManagedByESO
	}

	finalCustomMeta := make(map[string]string)
	switch mergePolicy {
	case MergePolicyMerge:
		maps.Copy(finalCustomMeta, remoteMeta)
		maps.Copy(finalCustomMeta, desiredCustomMeta)
	case MergePolicyReplace:
		finalCustomMeta = maps.Clone(desiredCustomMeta)
	}

	needsUpdate := !maps.Equal(remoteMeta, finalCustomMeta)

	return finalCustomMeta, needsUpdate, nil
}
