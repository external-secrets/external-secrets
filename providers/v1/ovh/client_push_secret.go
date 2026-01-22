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

package ovh

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/ovh/okms-sdk-go/types"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// Create or update a secret.
//
// If updatePolicy is set to Replace, the secret will be written, possibly overwriting an existing secret.
// If set to IfNotExists, it will not overwrite an existing secret.
func (cl *ovhClient) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if secret == nil {
		return errors.New("nil secret")
	}
	if len(secret.Data) == 0 {
		return errors.New("cannot push empty secret")
	}

	// Check if the secret already exists.
	// This determines which method to use: create or update.
	remoteSecret, currentVersion, err := getSecretWithOvhSDK(ctx, cl.okmsClient, cl.okmsID, esv1.ExternalSecretDataRemoteRef{
		Key: data.GetRemoteKey(),
	})
	if err != nil && !errors.Is(err, esv1.NoSecretErr) {
		return err
	}
	secretExists := !errors.Is(err, esv1.NoSecretErr)

	// Build the secret to be pushed.
	secretToPush, err := buildSecretToPush(secret, data)
	if err != nil {
		return err
	}

	// Compare the data of secretToPush with that of remoteSecret.
	equal, err := compareSecretsData(secretToPush, remoteSecret)
	if err != nil {
		return err
	}
	if equal {
		return nil
	}

	// Set cas according to client configuration
	if !cl.cas {
		currentVersion = nil
	}

	// Push the secret.
	return pushNewSecret(
		ctx,
		cl.okmsClient,
		cl.okmsID,
		secretToPush,
		data.GetRemoteKey(),
		currentVersion,
		secretExists)
}

// Compare the secret to push with the remote secret.
// If they are equal, do not push the secret.
func compareSecretsData(secretToPush map[string]any, remoteSecret []byte) (bool, error) {
	if len(remoteSecret) == 0 {
		return false, nil
	}

	secretToPushByte, err := json.Marshal(secretToPush)
	if err != nil {
		return false, err
	}

	return bytes.Equal(secretToPushByte, remoteSecret), nil
}

// Build the secret to be pushed.
//
// If remoteProperty is defined, it will be used as the key to store the secret value.
// If secretKey is not defined, the entire secret value will be pushed.
// Otherwise, only the value of the specified secretKey will be pushed.
func buildSecretToPush(secret *corev1.Secret, data esv1.PushSecretData) (map[string]any, error) {
	// Retrieve the secret value to push based on secretKey.
	var secretValueToPush map[string]any
	var err error

	secretKey := data.GetSecretKey()
	if secretKey == "" {
		secretValueToPush, err = extractAllSecretValues(secret.Data)
	} else {
		secretValueToPush, err = extractSecretKeyValue(secret.Data, secretKey)
	}

	if err != nil {
		return map[string]any{}, err
	}

	// Build the secret to push using remoteProperty.
	secretToPush := make(map[string]any)
	property := data.GetProperty()

	if property == "" {
		secretToPush = secretValueToPush
	} else {
		secretToPush[property] = secretValueToPush
	}

	return secretToPush, nil
}

func extractAllSecretValues(data map[string][]byte) (map[string]any, error) {
	var err error
	secretValueToPush := make(map[string]any)

	for key, value := range data {
		var decoded any
		if json.Unmarshal(value, &decoded) != nil {
			secretValueToPush[key] = string(value)
			continue
		}
		var cleanJSON []byte
		if cleanJSON, err = json.Marshal(decoded); err != nil {
			return map[string]any{}, err
		}

		secretValueToPush[key] = json.RawMessage(cleanJSON)
	}

	return secretValueToPush, nil
}

func extractSecretKeyValue(data map[string][]byte, secretKey string) (map[string]any, error) {
	secretValueToPush := make(map[string]any)

	value, ok := data[secretKey]
	if !ok {
		return nil, errors.New("secretKey not found in secret data")
	}
	var decoded any
	if json.Unmarshal(value, &decoded) != nil {
		secretValueToPush[secretKey] = string(value)
	} else {
		secretValueToPush[secretKey] = json.RawMessage(value)
	}

	return secretValueToPush, nil
}

// This pushes the created/updated secret.
func pushNewSecret(ctx context.Context, okmsClient OkmsClient, okmsID uuid.UUID, secretToPush map[string]any, path string, cas *uint32, secretExists bool) error {
	var err error

	if !secretExists {
		// Create a secret.
		_, err = okmsClient.PostSecretV2(ctx, okmsID, types.PostSecretV2Request{
			Path: path,
			Version: types.SecretV2VersionShort{
				Data: &secretToPush,
			},
		})
	} else {
		// Update a secret.
		_, err = okmsClient.PutSecretV2(ctx, okmsID, path, cas, types.PutSecretV2Request{
			Version: &types.SecretV2VersionShort{
				Data: &secretToPush,
			},
		})
	}

	return err
}
