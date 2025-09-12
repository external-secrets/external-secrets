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

package barbican

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/secrets"

	corev1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errClientGeneric      = "barbican client: %w"
	errClientMissingField = "barbican client: missing field %s"
)

var _ esapi.SecretsClient = &Client{}

type Client struct {
	keyManager *gophercloud.ServiceClient
}

func (c *Client) GetAllSecrets(ctx context.Context, ref esapi.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name == nil || ref.Name.RegExp == "" {
		return nil, fmt.Errorf(errClientMissingField, errors.New("name and/or regexp"))
	}

	opts := secrets.ListOpts{
		Name: ref.Name.RegExp,
	}

	allPages, err := secrets.List(c.keyManager, opts).AllPages(context.TODO())
	if err != nil {
		return nil, fmt.Errorf(errClientGeneric, errors.New("failed to list all secrets"))
	}

	allSecrets, err := secrets.ExtractSecrets(allPages)
	if err != nil {
		return nil, fmt.Errorf(errClientGeneric, errors.New("failed to extract secrets from all pages"))
	}

	if len(allSecrets) == 0 {
		return nil, fmt.Errorf(errClientGeneric, errors.New("no secrets found"))
	}

	var secretsMap = make(map[string][]byte)

	// return a secret map with all found secrets
	for _, secret := range allSecrets {
		secretUUID := extractUUIDFromRef(secret.SecretRef)
		secretsMap[secretUUID], err = secrets.GetPayload(context.TODO(), c.keyManager, secretUUID, nil).Extract()
		if err != nil {
			return nil, fmt.Errorf(errClientGeneric, errors.New("failed to get secret payload for secret "+secretUUID))
		}
	}
	return secretsMap, nil
}

func (c *Client) GetSecret(ctx context.Context, ref esapi.ExternalSecretDataRemoteRef) ([]byte, error) {
	// secret, err := secrets.Get(context.TODO(), c.keyManager, *&ref.Key).Extract()
	payload, err := secrets.GetPayload(context.TODO(), c.keyManager, *&ref.Key, nil).Extract()
	if err != nil {
		return nil, fmt.Errorf(errClientGeneric, errors.New("failed to get secret payload for secret "+*&ref.Key))
	}

	if ref.Property == "" {
		return payload, nil
	}

	propertyValue, err := getSecretPayloadProperty(payload, ref.Property)
	if err != nil {
		return nil, fmt.Errorf(errClientGeneric, errors.New("failed to get property "+ref.Property+" from secret payload for secret "+*&ref.Key))
	}

	return propertyValue, nil
}

func (c *Client) GetSecretMap(ctx context.Context, ref esapi.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	payload, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf(errClientGeneric, err)
	}

	var rawJson map[string]json.RawMessage
	if err := json.Unmarshal(payload, &rawJson); err != nil {
		return nil, fmt.Errorf(errClientGeneric, errors.New("failed to unmarshal secret payload into JSON"))
	}

	secretMap := make(map[string][]byte, len(rawJson))
	for k, v := range rawJson {
		secretMap[k] = []byte(v)
	}

	return secretMap, nil
}

func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esapi.PushSecretData) error {
	return fmt.Errorf("barbican provider does not support pushing secrets")
}

func (c *Client) SecretExists(ctx context.Context, ref esapi.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("barbican provider does not support checking if a secret exists")
}

func (c *Client) DeleteSecret(ctx context.Context, ref esapi.PushSecretRemoteRef) error {
	return fmt.Errorf("barbican provider does not support deleting secrets")
}

func (c *Client) Validate() (esapi.ValidationResult, error) {
	return esapi.ValidationResultReady, nil
}

func (c *Client) Close(ctx context.Context) error {
	return nil
}

func getSecretPayloadProperty(payload []byte, property string) ([]byte, error) {
	if property == "" {
		return payload, nil
	}

	var rawJson map[string]json.RawMessage
	if err := json.Unmarshal(payload, &rawJson); err != nil {
		return nil, fmt.Errorf(errClientGeneric, errors.New("failed to unmarshal secret payload property into JSON"))
	}

	value, ok := rawJson[property]
	if !ok {
		return nil, fmt.Errorf(errClientGeneric, errors.New("property "+property+" not found in secret payload"))
	}

	return value, nil
}

func extractUUIDFromRef(secretRef string) string {
	// Barbican secret refs are usually of the form: https://<endpoint>/v1/secrets/<uuid>
	// We'll just take the last part after the last '/'
	// If there's a trailing slash, the UUID part would be empty, so return empty string
	if secretRef == "" {
		return ""
	}

	// Check for trailing slash - if present, it's an invalid format
	if secretRef[len(secretRef)-1] == '/' {
		return ""
	}

	lastSlash := -1
	for i := len(secretRef) - 1; i >= 0; i-- {
		if secretRef[i] == '/' {
			lastSlash = i
			break
		}
	}
	if lastSlash != -1 && lastSlash+1 < len(secretRef) {
		return secretRef[lastSlash+1:]
	}
	return secretRef
}
