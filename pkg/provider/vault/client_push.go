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

package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

func (c *client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
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
		value, err = utils.JSONMarshal(secretStringVal)
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
	if err != nil && !errors.Is(err, esv1beta1.NoSecretError{}) {
		return err
	}
	// If the secret exists (err == nil), we should check if it is managed by external-secrets
	if err == nil {
		metadata, err := c.readSecretMetadata(ctx, data.GetRemoteKey())
		if err != nil {
			return err
		}
		manager, ok := metadata["managed-by"]
		if !ok || manager != "external-secrets" {
			return errors.New("secret not managed by external-secrets")
		}
	}
	// Remove the metadata map to check the reconcile difference
	if c.store.Version == esv1beta1.VaultKVStoreV1 {
		delete(vaultSecret, "custom_metadata")
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err = enc.Encode(vaultSecret)
	if err != nil {
		return fmt.Errorf("error encoding vault secret: %w", err)
	}
	vaultSecretValue := bytes.TrimSpace(buf.Bytes())
	if err != nil {
		return fmt.Errorf("error marshaling vault secret: %w", err)
	}
	if bytes.Equal(vaultSecretValue, value) {
		return nil
	}
	// If a Push of a property only, we should merge and add/update the property
	if data.GetProperty() != "" {
		if _, ok := vaultSecret[data.GetProperty()]; ok {
			d := vaultSecret[data.GetProperty()].(string)
			if err != nil {
				return fmt.Errorf("error marshaling vault secret: %w", err)
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
			return fmt.Errorf("error unmarshalling vault secret: %w", err)
		}
	}
	secretToPush := secretVal
	// Adding custom_metadata to the secret for KV v1
	if c.store.Version == esv1beta1.VaultKVStoreV1 {
		secretToPush["custom_metadata"] = label["custom_metadata"]
	}
	if c.store.Version == esv1beta1.VaultKVStoreV2 {
		secretToPush = map[string]any{
			"data": secretVal,
		}
	}
	if err != nil {
		return fmt.Errorf("failed to convert value to a valid JSON: %w", err)
	}
	// Secret metadata should be pushed separately only for KV2
	if c.store.Version == esv1beta1.VaultKVStoreV2 {
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

func (c *client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	path := c.buildPath(remoteRef.GetRemoteKey())
	metaPath, err := c.buildMetadataPath(remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}
	// Retrieve the secret map from vault and convert the secret value in string form.
	secretVal, err := c.readSecret(ctx, path, "")
	// If error is not of type secret not found, we should error
	if err != nil && errors.Is(err, esv1beta1.NoSecretError{}) {
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
		if c.store.Version == esv1beta1.VaultKVStoreV1 && len(secretVal) == 1 {
			delete(secretVal, "custom_metadata")
		}
		if len(secretVal) > 0 {
			secretToPush := secretVal
			if c.store.Version == esv1beta1.VaultKVStoreV2 {
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
	if c.store.Version == esv1beta1.VaultKVStoreV2 {
		_, err = c.logical.DeleteWithContext(ctx, metaPath)
		metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultDeleteSecret, err)
		if err != nil {
			return fmt.Errorf("could not delete secret metadata %v: %w", remoteRef.GetRemoteKey(), err)
		}
	}
	return nil
}
