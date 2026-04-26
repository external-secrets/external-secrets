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

package sakura

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sacloud/secretmanager-api-go"
	v1 "github.com/sacloud/secretmanager-api-go/apis/v1"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/find"
)

// Client implements the esv1.SecretsClient interface for Sakura Cloud Secret Manager.
type Client struct {
	api secretmanager.SecretAPI
}

// Check if the Client satisfies the esv1.SecretsClient interface.
var _ esv1.SecretsClient = &Client{}

// NewClient creates a new Client with the given SecretAPI.
func NewClient(api secretmanager.SecretAPI) *Client {
	return &Client{
		api: api,
	}
}

// ----------------- Utilities -----------------

// unveilSecret retrieves the secret value.
func (c *Client) unveilSecret(ctx context.Context, key, version, property string) ([]byte, error) {
	versionOpt := v1.OptNilInt{}
	if version != "" {
		versionInt, err := strconv.Atoi(version)
		if err != nil {
			return nil, fmt.Errorf("invalid version: %w", err)
		}

		versionOpt = v1.NewOptNilInt(versionInt)
	}

	res, err := c.api.Unveil(ctx, v1.Unveil{
		Name:    key,
		Version: versionOpt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to unveil secret with key %q: %w", key, err)
	}

	data := []byte(res.GetValue())
	if property == "" {
		return data, nil
	}

	kv := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &kv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret with key %q as JSON: %w", key, err)
	}

	value, ok := kv[property]
	if !ok {
		return nil, fmt.Errorf("property %q not found in secret %q", property, key)
	}

	var strVal string
	if err := json.Unmarshal(value, &strVal); err == nil {
		return []byte(strVal), nil
	}

	return value, nil
}

// secretKeyExists checks if a secret with the given key exists.
func (c *Client) secretKeyExists(ctx context.Context, key string) (bool, error) {
	secrets, err := c.api.List(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list secrets: %w", err)
	}

	for _, s := range secrets {
		if s.Name == key {
			return true, nil
		}
	}

	return false, nil
}

// ----------------- Interface implementation -----------------

// GetSecret returns a single secret from the provider.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	data, err := c.unveilSecret(ctx, ref.Key, ref.Version, ref.Property)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// PushSecret will write a single secret into the provider.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	value, err := esutils.ExtractSecretData(data, secret)
	if err != nil {
		return fmt.Errorf("failed to extract secret data: %w", err)
	}

	key := data.GetRemoteKey()
	property := data.GetProperty()

	// If property is specified, try to get existing secret value and merge with new value
	if property != "" {
		kv := make(map[string]json.RawMessage)

		// Since unveilSecret returns an error if the secret does not exist, we need to check existence first
		exists, err := c.secretKeyExists(ctx, key)
		if err != nil {
			return err
		}

		if exists {
			existingData, err := c.unveilSecret(ctx, key, "", "")
			if err != nil {
				return err
			}

			if err := json.Unmarshal(existingData, &kv); err != nil {
				return fmt.Errorf("failed to unmarshal existing secret as JSON: %w", err)
			}
		}

		if !json.Valid(value) {
			value, err = json.Marshal(string(value))
			if err != nil {
				return fmt.Errorf("failed to marshal value as JSON string: %w", err)
			}
		}

		kv[property] = value

		value, err = json.Marshal(kv)
		if err != nil {
			return fmt.Errorf("failed to marshal merged secret as JSON: %w", err)
		}
	}

	// Since Create and Update methods are not distinguished in SecretAPI, simply call Create here
	// 	ref: https://github.com/sacloud/secretmanager-api-go/blob/main/secrets.go#L65-L68
	if _, err := c.api.Create(ctx, v1.CreateSecret{
		Name:  key,
		Value: string(value),
	}); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	return nil
}

// DeleteSecret will delete the secret from a provider.
func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	key := remoteRef.GetRemoteKey()

	exists, err := c.secretKeyExists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		// If the secret does not exist, nothing to delete
		return nil
	}

	property := remoteRef.GetProperty()
	if property == "" {
		if err := c.api.Delete(ctx, v1.DeleteSecret{Name: key}); err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}
		return nil
	}

	existingData, err := c.unveilSecret(ctx, key, "", "")
	if err != nil {
		return err
	}

	kv := make(map[string]json.RawMessage)
	if err := json.Unmarshal(existingData, &kv); err != nil {
		return fmt.Errorf("failed to unmarshal existing secret as JSON: %w", err)
	}

	if _, ok := kv[property]; !ok {
		// If the property does not exist, nothing to delete
		return nil
	}
	delete(kv, property)

	if len(kv) == 0 {
		if err := c.api.Delete(ctx, v1.DeleteSecret{Name: key}); err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}
		return nil
	}

	value, err := json.Marshal(kv)
	if err != nil {
		return fmt.Errorf("failed to marshal merged secret as JSON: %w", err)
	}

	// Since Create and Update methods are not distinguished in SecretAPI, simply call Create here
	// 	ref: https://github.com/sacloud/secretmanager-api-go/blob/main/secrets.go#L65-L68
	if _, err := c.api.Create(ctx, v1.CreateSecret{
		Name:  key,
		Value: string(value),
	}); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	return nil
}

// SecretExists checks if a secret is already present in the provider at the given location.
func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	key := remoteRef.GetRemoteKey()
	property := remoteRef.GetProperty()

	exists, err := c.secretKeyExists(ctx, key)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	if property == "" {
		return true, nil
	}

	data, err := c.unveilSecret(ctx, key, "", "")
	if err != nil {
		return false, err
	}

	kv := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &kv); err != nil {
		return false, fmt.Errorf("failed to unmarshal secret as JSON: %w", err)
	}

	_, ok := kv[property]
	return ok, nil
}

// Validate checks if the client is configured correctly and is able to retrieve secrets from the provider.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	if _, err := c.api.List(context.Background()); err != nil {
		return esv1.ValidationResultError, fmt.Errorf("failed to validate client: %w", err)
	}

	return esv1.ValidationResultReady, nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := c.unveilSecret(ctx, ref.Key, ref.Version, ref.Property)
	if err != nil {
		return nil, err
	}

	// Unmarshal the secret value as JSON
	kv := make(map[string]json.RawMessage)
	if err = json.Unmarshal(data, &kv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret %s as JSON: %w", ref.Key, err)
	}

	secretData := make(map[string][]byte)
	for k, v := range kv {
		// Try to unmarshal each value as a string
		// 	If it fails, return the raw value
		var strVal string
		if err = json.Unmarshal(v, &strVal); err == nil {
			secretData[k] = []byte(strVal)
		} else {
			secretData[k] = v
		}
	}

	return secretData, nil
}

// GetAllSecrets returns multiple k/v pairs from the provider
//
//	Only Name filter is supported
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	// Fail fast for unsupported filters
	if ref.Path != nil {
		return nil, fmt.Errorf("path filter is not supported by the Sakura provider")
	}
	if len(ref.Tags) > 0 {
		return nil, fmt.Errorf("tag filter is not supported by the Sakura provider")
	}

	secrets, err := c.api.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	// Create regexp matcher for Name filter
	var matcher *find.Matcher
	if ref.Name != nil {
		m, err := find.New(*ref.Name)
		if err != nil {
			return nil, err
		}

		matcher = m
	}

	secretMap := make(map[string][]byte)
	for _, s := range secrets {
		// Skip unmatched secrets for Name filter
		if matcher != nil && !matcher.MatchName(s.Name) {
			continue
		}

		res, err := c.unveilSecret(ctx, s.Name, "", "")
		if err != nil {
			return nil, err
		}

		secretMap[s.Name] = res
	}

	return secretMap, nil
}

// Close closes the client.
func (c *Client) Close(_ context.Context) error {
	return nil
}
