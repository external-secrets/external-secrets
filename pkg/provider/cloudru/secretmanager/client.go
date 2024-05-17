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

package secretmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	smsV1 "github.com/cloudru-tech/secret-manager-sdk/api/v1"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/cloudru/secretmanager/adapter"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// SecretProvider is an API client for the Cloud.ru Secret Manager.
type SecretProvider interface {
	// ListSecrets lists secrets by the given request.
	ListSecrets(ctx context.Context, req *adapter.ListSecretsRequest) ([]*smsV1.Secret, error)
	// AccessSecretVersion gets the secret by the given request.
	AccessSecretVersion(ctx context.Context, id, version string) ([]byte, error)
}

// Client is a client for the Cloud.ru Secret Manager.
type Client struct {
	apiClient SecretProvider

	productInstanceID string
}

// GetSecret gets the secret by the remote reference.
func (c *Client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := c.accessSecret(ctx, ref.Key, ref.Version)
	if err != nil {
		return nil, err
	}

	prop := strings.TrimSpace(ref.Property)
	if prop == "" {
		return secret, nil
	}

	// For more obvious behavior, we return an error if we are dealing with invalid JSON
	// this is needed, because the gjson library works fine with value for `key`, for example:
	//
	// {"key": "value", another: "value"}
	//
	// but it will return "" when accessing to a property `another` (no quotes)
	if err = json.Unmarshal(secret, &map[string]interface{}{}); err != nil {
		return nil, fmt.Errorf("expecting the secret %q in JSON format, could not access property %q", ref.Key, ref.Property)
	}

	result := gjson.Parse(string(secret)).Get(prop)
	if !result.Exists() {
		return nil, fmt.Errorf("the requested property %q does not exist in secret %q", prop, ref.Key)
	}

	return []byte(result.Str), nil
}

func (c *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := c.accessSecret(ctx, ref.Key, ref.Version)
	if err != nil {
		return nil, err
	}

	secretMap := make(map[string]json.RawMessage)
	if err = json.Unmarshal(secret, &secretMap); err != nil {
		return nil, fmt.Errorf("expecting the secret %q in JSON format", ref.Key)
	}

	out := make(map[string][]byte)
	for k, v := range secretMap {
		out[k] = []byte(strings.Trim(string(v), "\""))
	}

	return out, nil
}

// GetAllSecrets gets all secrets by the remote reference.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if len(ref.Tags) == 0 && ref.Name == nil {
		return nil, fmt.Errorf("at least one of the following fields must be set: tags, name")
	}

	var nameFilter string
	if ref.Name != nil {
		nameFilter = ref.Name.RegExp
	}

	var totalSecrets []*smsV1.Secret
	searchReq := &adapter.ListSecretsRequest{
		ParentID:  c.productInstanceID,
		Labels:    ref.Tags,
		NameRegex: nameFilter,
		Offset:    0,
	}
	for {
		secrets, err := c.apiClient.ListSecrets(ctx, searchReq)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}
		if len(secrets) == 0 {
			break
		}

		totalSecrets = append(totalSecrets, secrets...)
		searchReq.Offset += int32(len(secrets))
	}

	out := make(map[string][]byte)
	for _, s := range totalSecrets {
		secret, err := c.GetSecretMap(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: s.GetId()})
		if err != nil {
			return nil, err
		}

		for k, v := range secret {
			out[k] = v
		}
	}

	return utils.ConvertKeys(ref.ConversionStrategy, out)
}

func (c *Client) accessSecret(ctx context.Context, key, version string) ([]byte, error) {
	if version == "" {
		version = "latest"
	}

	// check if the secret key is UUID
	// The uuid value means that the provided `key` is a secret identifier
	// if not, then it is a secret name, and we need to get the secret by
	// name before accessing the version.
	if _, err := uuid.Parse(key); err != nil {
		var secrets []*smsV1.Secret
		secrets, err = c.apiClient.ListSecrets(ctx, &adapter.ListSecretsRequest{
			ParentID:  c.productInstanceID,
			NameExact: key,
		})
		if err != nil {
			return nil, fmt.Errorf("list secrets by name '%s': %w", key, err)
		}
		if len(secrets) == 0 {
			return nil, fmt.Errorf("secret with name '%s' not found", key)
		}

		key = secrets[0].GetId()
	}

	return c.apiClient.AccessSecretVersion(ctx, key, version)
}

func (c *Client) PushSecret(context.Context, *corev1.Secret, esv1beta1.PushSecretData) error {
	return fmt.Errorf("push secret is not supported")
}

func (c *Client) DeleteSecret(context.Context, esv1beta1.PushSecretRemoteRef) error {
	return fmt.Errorf("delete secret is not supported")
}

func (c *Client) SecretExists(context.Context, esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("secret exists is not supported")
}

// Validate validates the client.
func (c *Client) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

// Close closes the client.
func (c *Client) Close(_ context.Context) error { return nil }
