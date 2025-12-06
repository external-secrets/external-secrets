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

// Package barbican client implementation.
package barbican

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/secrets"

	corev1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errClientGeneric      = "barbican client: %w"
	errClientMissingField = "barbican client: missing field %w"
	errClientListAllSecrets = "barbican client: failed to list all secrets: %w"
	errClientExtractSecrets = "barbican client: failed to extract secrets: %w"
	errClientGetSecretPayload = "barbican client: failed to get secret payload: %w"
	errClientGetSecretPayloadProperty = "barbican client: failed to get secret payload property: %w"
	errClientJSONUnmarshal = "barbican client: failed to unmarshal json: %w"
)

var _ esapi.SecretsClient = &Client{}

// Client is a Barbican secrets client.
type Client struct {
	keyManager *gophercloud.ServiceClient
}

// GetAllSecrets retrieves all secrets matching the given name.
func (c *Client) GetAllSecrets(ctx context.Context, ref esapi.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name == nil || ref.Name.RegExp == "" {
		return nil, fmt.Errorf(errClientMissingField, errors.New("name and/or regexp"))
	}

	opts := secrets.ListOpts{
		Name: ref.Name.RegExp,
	}

	allPages, err := secrets.List(c.keyManager, opts).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf(errClientListAllSecrets, err)
	}

	allSecrets, err := secrets.ExtractSecrets(allPages)
	if err != nil {
		return nil, fmt.Errorf(errClientExtractSecrets, err)
	}

	if len(allSecrets) == 0 {
		return nil, fmt.Errorf(errClientGeneric, errors.New("no secrets found"))
	}

	var secretsMap = make(map[string][]byte)

	// return a secret map with all found secrets.
	for _, secret := range allSecrets {
		secretUUID := extractUUIDFromRef(secret.SecretRef)
		secretsMap[secretUUID], err = secrets.GetPayload(ctx, c.keyManager, secretUUID, nil).Extract()
		if err != nil {
			return nil, fmt.Errorf(errClientGetSecretPayload, fmt.Errorf("failed to get secret payload for secret %s: %w", secretUUID, err))
		}
	}
	return secretsMap, nil
}

// GetSecret retrieves a secret from Barbican.
func (c *Client) GetSecret(ctx context.Context, ref esapi.ExternalSecretDataRemoteRef) ([]byte, error) {
	payload, err := secrets.GetPayload(ctx, c.keyManager, ref.Key, nil).Extract()
	if err != nil {
		return nil, fmt.Errorf(errClientGetSecretPayload, err)
	}

	if ref.Property == "" {
		return payload, nil
	}

	propertyValue, err := getSecretPayloadProperty(payload, ref.Property)
	if err != nil {
		return nil, fmt.Errorf(errClientGetSecretPayloadProperty, fmt.Errorf("failed to get property %s from secret payload: %w", ref.Property, err))
	}

	return propertyValue, nil
}

// GetSecretMap retrieves a secret and parses it as a JSON object.
func (c *Client) GetSecretMap(ctx context.Context, ref esapi.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	payload, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf(errClientGeneric, err)
	}

	var rawJSON map[string]json.RawMessage
	if err := json.Unmarshal(payload, &rawJSON); err != nil {
		return nil, fmt.Errorf(errClientJSONUnmarshal, err)
	}

	secretMap := make(map[string][]byte, len(rawJSON))
	for k, v := range rawJSON {
		secretMap[k] = []byte(v)
	}

	return secretMap, nil
}

// PushSecret is not implemented right now for Barbican.
func (c *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esapi.PushSecretData) error {
	return fmt.Errorf("barbican provider does not support pushing secrets")
}

// SecretExists is not implemented right now for Barbican.
func (c *Client) SecretExists(_ context.Context, _ esapi.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("barbican provider does not pushing secrets with update policy IfNotExists")
}

// DeleteSecret is not implemented right now for Barbican.
func (c *Client) DeleteSecret(_ context.Context, _ esapi.PushSecretRemoteRef) error {
	return fmt.Errorf("barbican provider does not support deleting secrets (delete policy Delete)")
}

// Validate checks if the client is properly configured.
func (c *Client) Validate() (esapi.ValidationResult, error) {
	return esapi.ValidationResultUnknown, nil
}

// Close closes the client and any underlying connections.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// getSecretPayloadProperty extracts a property from a JSON payload.
func getSecretPayloadProperty(payload []byte, property string) ([]byte, error) {
	if property == "" {
		return payload, nil
	}

	var rawJSON map[string]json.RawMessage
	if err := json.Unmarshal(payload, &rawJSON); err != nil {
		return nil, fmt.Errorf(errClientJSONUnmarshal, err)
	}

	value, ok := rawJSON[property]
	if !ok {
		return nil, fmt.Errorf(errClientGeneric, fmt.Errorf("property %s not found in secret payload", property))
	}

	return value, nil
}

// extractUUIDFromRef extracts the UUID from a Barbican secret reference URL.
func extractUUIDFromRef(secretRef string) string {
	// Barbican secret refs are usually of the form: https://<endpoint>/v1/secrets/<uuid>
	// We'll just take the last part after the last '/'
	// If there's a trailing slash, the UUID part would be empty, so return empty string

	lastSlash := strings.LastIndex(secretRef, "/")
	if lastSlash > -1 {
		return secretRef[lastSlash+1:] // <- will not result in overflow even if it's the last `/`
	}

  return ""
}
