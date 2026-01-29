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

package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

func (c *client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	var (
		value []byte
		err   error
	)
	key := data.GetSecretKey()
	if key == "" {
		// Must convert secret values to string, otherwise data will be sent as base64 to Vault
		secretStringVal := make(map[string]string)
		for k, v := range secret.Data {
			secretStringVal[k] = string(v)
		}
		value, err = esutils.JSONMarshal(secretStringVal)
		if err != nil {
			return fmt.Errorf("failed to serialize secret content as JSON: %w", err)
		}
	} else {
		value = secret.Data[key]
	}
	label := map[string]any{
		"custom_metadata": map[string]string{
			"managed-by": "external-secrets",
		},
	}
	secretVal := make(map[string]any)
	path := c.buildPath(data.GetRemoteKey())
	metaPath, err := c.buildMetadataPath(data.GetRemoteKey())
	if err != nil {
		return err
	}

	// Retrieve the secret map from vault and convert the secret value in string form.
	vaultSecret, err := c.readSecret(ctx, path, "")
	// If error is not of type secret not found, we should error
	if err != nil && !errors.Is(err, esv1.NoSecretError{}) {
		return err
	}

	secretExists := err == nil
	// If the secret exists, we should check if it is managed by external-secrets
	if secretExists {
		metadata, err := c.readSecretMetadata(ctx, data.GetRemoteKey())
		if err != nil {
			return err
		}
		manager, ok := metadata["managed-by"]
		if !ok || manager != "external-secrets" {
			return errors.New("secret not managed by external-secrets")
		}
		// Remove the metadata map to check the reconcile difference
		if c.store.Version == esv1.VaultKVStoreV1 {
			delete(vaultSecret, "custom_metadata")
		}
		// Only compare the entire secret if we're pushing the whole secret (not a single property)
		if data.GetProperty() == "" {
			// Convert incoming value to map for proper JSON comparison
			var incomingSecretMap map[string]any
			err = json.Unmarshal(value, &incomingSecretMap)
			if err != nil {
				// Do not wrap the original error with %w as json.Unmarshal errors
				// may contain sensitive secret data in the error message
				return errors.New("error unmarshalling incoming secret value: invalid JSON format")
			}
			// Compare maps instead of raw bytes to handle JSON field ordering and formatting
			if maps.Equal(vaultSecret, incomingSecretMap) {
				return nil
			}
		}
	}
	// If a Push of a property only, we should merge and add/update the property
	if data.GetProperty() != "" {
		if _, ok := vaultSecret[data.GetProperty()]; ok {
			d, ok := vaultSecret[data.GetProperty()].(string)
			if !ok {
				return fmt.Errorf("error converting %s to string", data.GetProperty())
			}
			// If the property has the same value, don't update the secret
			if bytes.Equal([]byte(d), value) {
				return nil
			}
		}
		maps.Insert(secretVal, maps.All(vaultSecret))
		// Secret got from vault is already on map[string]string format
		secretVal[data.GetProperty()] = string(value)
	} else {
		err = json.Unmarshal(value, &secretVal)
		if err != nil {
			// Do not wrap the original error with %w as json.Unmarshal errors
			// may contain sensitive secret data in the error message
			return errors.New("error unmarshalling vault secret: invalid JSON format")
		}
	}
	secretToPush := secretVal
	// Adding custom_metadata to the secret for KV v1
	if c.store.Version == esv1.VaultKVStoreV1 {
		secretToPush["custom_metadata"] = label["custom_metadata"]
	}
	if c.store.Version == esv1.VaultKVStoreV2 {
		secretToPush = map[string]any{
			"data": secretVal,
		}

		// Add CAS options if required
		if c.store.CheckAndSet != nil && c.store.CheckAndSet.Required {
			casVersion, casErr := c.getCASVersion(ctx, data.GetRemoteKey(), secretExists)
			if casErr != nil {
				return fmt.Errorf("failed to get CAS version: %w", casErr)
			}

			secretToPush["options"] = map[string]any{
				"cas": casVersion,
			}
		}
	}
	// Secret metadata should be pushed separately only for KV2
	if c.store.Version == esv1.VaultKVStoreV2 {
		_, err = c.logical.WriteWithContext(ctx, metaPath, label)
		metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
		if err != nil {
			return err
		}
	}
	// Otherwise, create or update the version.
	_, err = c.logical.WriteWithContext(ctx, path, secretToPush)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
	return err
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
	metadata, err := c.readSecretMetadata(ctx, remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}
	manager, ok := metadata["managed-by"]
	if !ok || manager != "external-secrets" {
		return nil
	}
	// If Push for a Property, we need to delete the property and update the secret
	if remoteRef.GetProperty() != "" {
		delete(secretVal, remoteRef.GetProperty())
		// If the only key left in the remote secret is the reference of the metadata.
		if c.store.Version == esv1.VaultKVStoreV1 && len(secretVal) == 1 {
			delete(secretVal, "custom_metadata")
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
	var err error
	if currentVersion, ok := data["current_version"]; ok {
		switch v := currentVersion.(type) {
		case int:
			return v, nil
		case float64:
			return int(v), nil
		case json.Number:
			if intVal, err := v.Int64(); err == nil {
				return int(intVal), nil
			}
			return 0, fmt.Errorf("failed to convert json.Number to int: %w", err)
		default:
			return 0, fmt.Errorf("unexpected type for current_version: %T", currentVersion)
		}
	}

	// If metadata exists but no current_version found, assume this is version 1.
	// This handles edge cases with legacy secrets or incomplete metadata.
	// Vault KV v2 secrets start at version 1, so this is the safest assumption.
	return 1, nil
}
