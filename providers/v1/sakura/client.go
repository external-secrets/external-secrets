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

package sakura

import (
	"context"
	"strconv"

	"github.com/sacloud/secretmanager-api-go"
	v1 "github.com/sacloud/secretmanager-api-go/apis/v1"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type Client struct {
	api secretmanager.SecretAPI
}

// Check if the Client satisfies the esv1.SecretsClient interface.
var _ esv1.SecretsClient = &Client{}

func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	version := v1.OptNilInt{}

	if ref.Version != "" {
		versionInt, err := strconv.Atoi(ref.Version)
		if err != nil {
			return nil, err
		}

		version = v1.NewOptNilInt(versionInt)
	}

	res, err := c.api.Unveil(ctx, v1.Unveil{
		Name:    ref.Key,
		Version: version,
	})
	if err != nil {
		return nil, err
	}

	return []byte(res.GetValue()), nil
}

func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	// Implement the logic to push a secret to Sakura Cloud
	return nil
}

func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	// Implement the logic to delete a secret from Sakura Cloud
	return nil
}

func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	// Implement the logic to check if a secret exists in Sakura Cloud
	return false, nil
}

func (c *Client) Validate() (esv1.ValidationResult, error) {
	// Implement the logic to validate the client configuration
	return esv1.ValidationResultUnknown, nil
}

func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	// Implement the logic to get a secret map from Sakura Cloud
	return nil, nil
}

func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	// Implement the logic to get all secrets from Sakura Cloud
	return nil, nil
}

func (c *Client) Close(ctx context.Context) error {
	// Implement the logic to close the client connection
	return nil
}
