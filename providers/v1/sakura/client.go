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

// ----------------- Utilities -----------------

// Retrieve the value of a secret
func (c *Client) unveilSecret(ctx context.Context, name string, version string) ([]byte, error) {
	versionOpt := v1.OptNilInt{}
	if version != "" {
		versionInt, err := strconv.Atoi(version)
		if err != nil {
			return nil, fmt.Errorf("invalid version: %w", err)
		}

		versionOpt = v1.NewOptNilInt(versionInt)
	}

	res, err := c.api.Unveil(ctx, v1.Unveil{
		Name:    name,
		Version: versionOpt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to unveil secret: %w", err)
	}

	return []byte(res.GetValue()), nil
}

// Check if a secret with the given name exists
func (c *Client) secretExists(ctx context.Context, name string) (bool, error) {
	secrets, err := c.api.List(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list secrets: %w", err)
	}

	for _, secret := range secrets {
		if secret.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// ----------------- Interface implementation -----------------

func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return c.unveilSecret(ctx, ref.Key, ref.Version)
}

func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	value, err := esutils.ExtractSecretData(data, secret)
	if err != nil {
		return fmt.Errorf("failed to extract secret data: %w", err)
	}

	// Since Create and Update methods are not distinguished in SecretAPI, simply call Create here
	// 	ref: https://github.com/sacloud/secretmanager-api-go/blob/main/secrets.go#L65-L68
	if _, err = c.api.Create(ctx, v1.CreateSecret{
		Name:  data.GetRemoteKey(),
		Value: string(value),
	}); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	return nil
}

func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	if err := c.api.Delete(ctx, v1.DeleteSecret{
		Name: remoteRef.GetRemoteKey(),
	}); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}

func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	return c.secretExists(ctx, remoteRef.GetRemoteKey())
}

func (c *Client) Validate() (esv1.ValidationResult, error) {
	if _, err := c.api.List(context.Background()); err != nil {
		return esv1.ValidationResultError, fmt.Errorf("failed to validate client: %w", err)
	}

	return esv1.ValidationResultReady, nil
}

func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := c.unveilSecret(ctx, ref.Key, ref.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	kv := make(map[string]json.RawMessage)
	if err = json.Unmarshal(data, &kv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret %s as JSON: %w", ref.Key, err)
	}

	secretData := make(map[string][]byte)
	for k, v := range kv {
		var strVal string
		if err = json.Unmarshal(v, &strVal); err == nil {
			secretData[k] = []byte(strVal)
		} else {
			secretData[k] = v
		}
	}

	return secretData, nil
}

// Only Name filter is supported
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
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

		res, err := c.unveilSecret(ctx, s.Name, "")
		if err != nil {
			return nil, fmt.Errorf("failed to unveil secret %s: %w", s.Name, err)
		}

		secretMap[s.Name] = res
	}

	return secretMap, nil
}

func (c *Client) Close(ctx context.Context) error {
	return nil
}
