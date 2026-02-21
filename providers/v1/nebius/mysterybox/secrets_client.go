// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package mysterybox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/mysterybox"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	errNotImplemented             = "not implemented"
	errJSONMarshal                = "failed to marshal JSON"
	errSecretNotFound             = "secret %q not found: %w"
	errSecretByKeyNotFound        = "key %q not found in secret %q: %w"
	errSecretVersionByKeyNotFound = "version %q of secret %q not found by key %q: %w"
	errSecretVersionNotFound      = "version %q of secret %q not found: %w"
)

// SecretsClient provides methods to interact with secrets in the Mysterybox service.
// It wraps a mysterybox.Client instance and uses a token for authentication.
type SecretsClient struct {
	mysteryboxClient mysterybox.Client
	token            string
}

// GetSecret retrieves the value of a secret from Mysterybox based on the provided reference.
func (c *SecretsClient) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secretKey := ref.Property
	if secretKey == "" {
		payload, err := c.mysteryboxClient.GetSecret(ctx, c.token, ref.Key, ref.Version)
		metrics.ObserveAPICall(constants.ProviderNebiusMysterybox, constants.CallNebiusMysteryboxGetSecret, err)
		if err != nil {
			return nil, handleGetSecretError(err, ref)
		}
		keyToValue := make(map[string]any, len(payload.Entries))
		for _, entry := range payload.Entries {
			value := getValue(&entry)
			keyToValue[entry.Key] = value
		}
		out, err := json.Marshal(keyToValue)
		if err != nil {
			return nil, errors.New(errJSONMarshal)
		}
		return out, nil
	}
	payloadEntry, err := c.mysteryboxClient.GetSecretByKey(ctx, c.token, ref.Key, ref.Version, secretKey)
	metrics.ObserveAPICall(constants.ProviderNebiusMysterybox, constants.CallNebiusMysteryboxGetSecretByKey, err)
	if err != nil {
		return nil, handleGetSecretByKeyError(err, ref)
	}
	return getValueAsBinary(&payloadEntry.Entry), nil
}

// GetSecretMap retrieves a map of secret key-value pairs from Mysterybox using the provided reference.
func (c *SecretsClient) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	payload, err := c.mysteryboxClient.GetSecret(ctx, c.token, ref.Key, ref.Version)
	metrics.ObserveAPICall(constants.ProviderNebiusMysterybox, constants.CallNebiusMysteryboxGetSecret, err)
	if err != nil {
		return nil, handleGetSecretError(err, ref)
	}
	secretMap := make(map[string][]byte, len(payload.Entries))
	for _, entry := range payload.Entries {
		value := getValueAsBinary(&entry)
		secretMap[entry.Key] = value
	}
	return secretMap, nil
}

// DeleteSecret not implemented for Nebius Mysterybox provider.
func (c *SecretsClient) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

// SecretExists not implemented for Nebius Mysterybox provider.
func (c *SecretsClient) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// PushSecret not implemented for Nebius Mysterybox provider.
func (c *SecretsClient) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

// Validate checks the configuration of the SecretsClient and returns its validation result.
func (c *SecretsClient) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

// GetAllSecrets not implemented for Nebius Mysterybox provider.
func (c *SecretsClient) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// Close cleans up resources when the provider is done being used.
func (c *SecretsClient) Close(_ context.Context) error {
	return nil
}

func getValueAsBinary(entry *mysterybox.Entry) []byte {
	if entry.BinaryValue != nil {
		return entry.BinaryValue
	}
	return []byte(entry.StringValue)
}

func getValue(entry *mysterybox.Entry) any {
	if entry.BinaryValue != nil {
		return entry.BinaryValue
	}
	return entry.StringValue
}

func handleGetSecretError(err error, ref esv1.ExternalSecretDataRemoteRef) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err // not a grpc error
	}
	if st.Code() == codes.NotFound {
		if ref.Version != "" {
			return fmt.Errorf(errSecretVersionNotFound, ref.Version, ref.Key, esv1.NoSecretErr)
		}
		return fmt.Errorf(errSecretNotFound, ref.Key, esv1.NoSecretErr)
	}
	return MapGrpcErrors("get secret", err)
}

func handleGetSecretByKeyError(err error, ref esv1.ExternalSecretDataRemoteRef) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err // not a grpc error
	}
	if st.Code() == codes.NotFound {
		if ref.Property != "" {
			if ref.Version != "" {
				return fmt.Errorf(errSecretVersionByKeyNotFound, ref.Version, ref.Key, ref.Property, esv1.NoSecretErr)
			}
			return fmt.Errorf(errSecretByKeyNotFound, ref.Property, ref.Key, esv1.NoSecretErr)
		}
	}
	return MapGrpcErrors("get secret by key", err)
}

var _ esv1.SecretsClient = &SecretsClient{}
