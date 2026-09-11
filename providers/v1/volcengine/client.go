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

package volcengine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/tidwall/gjson"
	"github.com/volcengine/volcengine-go-sdk/service/kms"
	"github.com/volcengine/volcengine-go-sdk/volcengine/volcengineerr"
	corev1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	notImplemented            = "not implemented"
	errUninitializedKMSClient = "kms client is not initialized"
)

var _ esapi.SecretsClient = &Client{}

// Client is a client for the Volcengine provider.
type Client struct {
	kms kms.KMSAPI
}

// NewClient creates a new Volcengine client.
func NewClient(kms kms.KMSAPI) *Client {
	return &Client{
		kms: kms,
	}
}

// GetSecret retrieves a secret value from Volcengine Secrets Manager.
func (c *Client) GetSecret(ctx context.Context, ref esapi.ExternalSecretDataRemoteRef) ([]byte, error) {
	return c.getSecretValue(ctx, ref)
}

// SecretExists checks if a secret exists in Volcengine Secrets Manager.
func (c *Client) SecretExists(ctx context.Context, remoteRef esapi.PushSecretRemoteRef) (bool, error) {
	secretName := remoteRef.GetRemoteKey()
	if secretName == "" {
		return false, errors.New("secret name is empty")
	}
	if c.kms == nil {
		return false, errors.New(errUninitializedKMSClient)
	}

	_, err := c.kms.DescribeSecretWithContext(ctx, &kms.DescribeSecretInput{
		SecretName: &secretName,
	})
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Validate checks if the provider is configured correctly.
func (c *Client) Validate() (esapi.ValidationResult, error) {
	if c.kms != nil {
		return esapi.ValidationResultReady, nil
	}
	return esapi.ValidationResultError, errors.New(errUninitializedKMSClient)
}

// GetSecretMap retrieves a secret value and unmarshals it as a map.
func (c *Client) GetSecretMap(ctx context.Context, ref esapi.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	value, err := c.getSecretValue(ctx, ref)
	if err != nil {
		return nil, err
	}

	var rawSecretMap map[string]json.RawMessage
	if err := json.Unmarshal(value, &rawSecretMap); err != nil {
		// Do not wrap the original error as json.Unmarshal errors may contain
		// sensitive secret data in the error message
		return nil, errors.New("failed to unmarshal secret: invalid JSON format")
	}

	secretMap := make(map[string][]byte, len(rawSecretMap))
	for key, value := range rawSecretMap {
		secretMap[key] = rawMessageBytes(value)
	}
	return secretMap, nil
}

// GetAllSecrets retrieves all secrets matching the given criteria.
func (c *Client) GetAllSecrets(_ context.Context, _ esapi.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(notImplemented)
}

// PushSecret creates or updates a secret in Volcengine Secrets Manager.
func (c *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esapi.PushSecretData) error {
	return errors.New(notImplemented)
}

// DeleteSecret deletes a secret from Volcengine Secrets Manager.
func (c *Client) DeleteSecret(_ context.Context, _ esapi.PushSecretRemoteRef) error {
	return errors.New(notImplemented)
}

// Close is a no-op for the Volcengine client.
func (c *Client) Close(_ context.Context) error {
	return nil
}

func (c *Client) getSecretValue(ctx context.Context, ref esapi.ExternalSecretDataRemoteRef) ([]byte, error) {
	if c.kms == nil {
		return nil, errors.New(errUninitializedKMSClient)
	}

	output, err := c.kms.GetSecretValueWithContext(ctx, &kms.GetSecretValueInput{
		SecretName: &ref.Key,
		VersionID:  resolveVersion(ref),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, esapi.NoSecretErr
		}
		return nil, err
	}

	if output.SecretValue == nil {
		return nil, fmt.Errorf("secret %s has no value", ref.Key)
	}

	secret := *output.SecretValue

	if ref.Property == "" {
		return []byte(secret), nil
	}

	return extractProperty(secret, ref.Property)
}

func extractProperty(secret, property string) ([]byte, error) {
	if !gjson.Valid(secret) {
		return nil, errors.New("failed to unmarshal secret: invalid JSON format")
	}

	val := gjson.Get(secret, property)
	if !val.Exists() {
		return nil, fmt.Errorf("property %q not found in secret", property)
	}
	return []byte(val.String()), nil
}

func rawMessageBytes(value json.RawMessage) []byte {
	trimmedValue := bytes.TrimSpace(value)
	if len(trimmedValue) > 0 && trimmedValue[0] == '"' {
		var stringValue string
		if err := json.Unmarshal(value, &stringValue); err == nil {
			return []byte(stringValue)
		}
	}
	return []byte(value)
}

func isNotFoundError(err error) bool {
	var requestFailure volcengineerr.RequestFailure
	return errors.As(err, &requestFailure) && requestFailure.StatusCode() == http.StatusNotFound
}

func resolveVersion(ref esapi.ExternalSecretDataRemoteRef) *string {
	if ref.Version != "" {
		return &ref.Version
	}
	return nil
}
