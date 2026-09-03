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

package protonpass

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/protonpass/internal/client"
)

const (
	errGetSecret       = "protonpass client: failed to get secret: %w"
	errGetSecretMap    = "protonpass client: failed to get secret map: %w"
	errPropertyMissing = "protonpass client: property %q not found for item %q"
)

var _ esv1.SecretsClient = &Client{}

// Client is a Proton Pass secrets client.
type Client struct {
	client client.Interface
}

// GetSecret retrieves a single property from a Proton Pass item.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	proj, err := c.client.GetItem(ctx, ref.Key)
	if err != nil {
		if errors.Is(err, esv1.NoSecretErr) {
			return nil, esv1.NoSecretErr
		}
		return nil, fmt.Errorf(errGetSecret, err)
	}

	property := ref.Property
	if property == "" {
		property = "password"
	}
	value, ok := proj[property]
	if !ok {
		return nil, fmt.Errorf(errPropertyMissing, property, ref.Key)
	}
	return value, nil
}

// GetSecretMap returns the full projected key/value map for a Proton Pass item.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	proj, err := c.client.GetItem(ctx, ref.Key)
	if err != nil {
		if errors.Is(err, esv1.NoSecretErr) {
			return nil, esv1.NoSecretErr
		}
		return nil, fmt.Errorf(errGetSecretMap, err)
	}
	return proj, nil
}

// PushSecret is not implemented for the read-only Proton Pass provider.
func (c *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errors.New("protonpass provider does not support pushing secrets (read-only)")
}

// DeleteSecret is not implemented for the read-only Proton Pass provider.
func (c *Client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New("protonpass provider does not support deleting secrets (read-only)")
}

// SecretExists is not implemented for the read-only Proton Pass provider.
func (c *Client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("protonpass provider does not support checking secret existence (read-only)")
}

// GetAllSecrets is not implemented for the read-only Proton Pass provider.
func (c *Client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("protonpass provider does not support finding secrets (read-only)")
}

// Validate checks if the client is configured correctly by minting a session.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	if err := c.client.Validate(context.Background()); err != nil {
		return esv1.ValidationResultError, err
	}
	return esv1.ValidationResultReady, nil
}

// Close closes the client and any underlying connections.
func (c *Client) Close(_ context.Context) error {
	return nil
}
