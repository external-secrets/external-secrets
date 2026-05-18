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

package secretmanager

import (
	"context"
	"errors"
	"fmt"

	mwssdk "go.mws.cloud/go-sdk/mws"
	secretmanagerclient "go.mws.cloud/go-sdk/service/secretmanager/client"
	secretmanagersdk "go.mws.cloud/go-sdk/service/secretmanager/sdk"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var (
	errNotImplemented = errors.New("not implemented")
)

var _ esv1.SecretsClient = (*Client)(nil)

// Client implements the Secrets Client interface for MWS Secret Manager.
type Client struct {
	sdk           *mwssdk.SDK
	secretVersion *secretmanagersdk.SecretVersion

	project string
}

// GetSecret gets the secret from MWS Secret Manager.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	var (
		secretName     = ref.Key
		secretVersion  = ref.Version
		secretProperty = ref.Property
	)

	if secretVersion == "" {
		secretVersion = "current"
	}

	content, err := c.secretVersion.GetData(ctx, secretmanagerclient.GetDataRequest{
		Project: c.project,
		Name:    secretName,
		Version: secretVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret version data: %w", err)
	}

	if secretProperty == "" {
		buffer, err := content.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal secret version content: %w", err)
		}

		return buffer, nil
	}

	property, ok := content[secretProperty]
	if !ok {
		return nil, fmt.Errorf("secret version does not contain property %s", secretProperty)
	}

	return []byte(property), nil
}

// GetSecretMap gets the secret map from MWS Secret Manager.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	var (
		secretName    = ref.Key
		secretVersion = ref.Version
	)

	if secretVersion == "" {
		secretVersion = "current"
	}

	content, err := c.secretVersion.GetData(ctx, secretmanagerclient.GetDataRequest{
		Project: c.project,
		Name:    secretName,
		Version: secretVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret version data: %w", err)
	}

	secretMap := make(map[string][]byte, len(content))

	for key, value := range content {
		secretMap[key] = []byte(value)
	}

	return secretMap, nil
}

// GetAllSecrets gets all secrets from MWS Secret Manager.
func (c *Client) GetAllSecrets(context.Context, esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errNotImplemented
}

// PushSecret pushes the secret to MWS Secret Manager.
func (c *Client) PushSecret(context.Context, *corev1.Secret, esv1.PushSecretData) error {
	return errNotImplemented
}

// DeleteSecret deletes the secret from MWS Secret Manager.
func (c *Client) DeleteSecret(context.Context, esv1.PushSecretRemoteRef) error {
	return errNotImplemented
}

// SecretExists checks does the secret exist in MWS Secret Manager.
func (c *Client) SecretExists(context.Context, esv1.PushSecretRemoteRef) (bool, error) {
	return false, errNotImplemented
}

// Validate validates the client configuration.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultUnknown, nil
}

// Close closes the client.
func (c *Client) Close(ctx context.Context) error {
	if err := c.sdk.Close(ctx); err != nil {
		return fmt.Errorf("failed to close mws sdk: %w", err)
	}

	return nil
}
