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

package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	clientv3 "go.etcd.io/etcd/client/v3"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/find"
)

const (
	errNotImplemented    = "not implemented"
	errSecretNotFound    = "secret not found: %s"
	errPropertyNotFound  = "property %s not found in secret %s"
	errFailedToGetSecret = "failed to get secret: %w"
	errFailedToPutSecret = "failed to put secret: %w"
	errFailedToDelSecret = "failed to delete secret: %w"
	errUnmarshalSecret   = "failed to unmarshal secret data: %w"
	errMarshalSecret     = "failed to marshal secret data: %w"
	managedByKey         = "managed-by"
	managedByValue       = "external-secrets"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Client{}

// secretData represents the structure of a secret stored in etcd.
type secretData struct {
	Data     map[string]string `json:"data"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// GetSecret retrieves a secret from etcd.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	key := c.buildKey(ref.Key)

	resp, err := c.kv.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf(errFailedToGetSecret, err)
	}

	if len(resp.Kvs) == 0 {
		return nil, esv1.NoSecretError{}
	}

	value := resp.Kvs[0].Value

	// If no property is specified, return the raw value
	if ref.Property == "" {
		// Try to parse as JSON and return data field if it exists
		var sd secretData
		if err := json.Unmarshal(value, &sd); err == nil && sd.Data != nil {
			// Return the data as JSON
			return esutils.JSONMarshal(sd.Data)
		}
		// Return raw value if not in our format
		return value, nil
	}

	// Extract property from the secret
	return c.extractProperty(value, ref.Key, ref.Property)
}

func (c *Client) extractProperty(value []byte, key, property string) ([]byte, error) {
	// First, try to parse as our secretData format
	var sd secretData
	if err := json.Unmarshal(value, &sd); err == nil && sd.Data != nil {
		if val, ok := sd.Data[property]; ok {
			return []byte(val), nil
		}
		// Try gjson for nested properties
		dataBytes, _ := esutils.JSONMarshal(sd.Data)
		result := gjson.GetBytes(dataBytes, property)
		if result.Exists() {
			return []byte(result.String()), nil
		}
		return nil, fmt.Errorf(errPropertyNotFound, property, key)
	}

	// Try gjson on raw value
	result := gjson.GetBytes(value, property)
	if result.Exists() {
		return []byte(result.String()), nil
	}

	return nil, fmt.Errorf(errPropertyNotFound, property, key)
}

// GetSecretMap retrieves a secret and returns it as a map.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	key := c.buildKey(ref.Key)

	resp, err := c.kv.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf(errFailedToGetSecret, err)
	}

	if len(resp.Kvs) == 0 {
		return nil, esv1.NoSecretError{}
	}

	value := resp.Kvs[0].Value

	// Parse as secretData
	var sd secretData
	if err := json.Unmarshal(value, &sd); err != nil {
		// Try parsing as raw JSON map
		var rawMap map[string]any
		if err := json.Unmarshal(value, &rawMap); err != nil {
			// Return the value as a single key
			return map[string][]byte{ref.Key: value}, nil
		}
		return convertToByteMap(rawMap)
	}

	if sd.Data == nil {
		return nil, fmt.Errorf(errSecretNotFound, ref.Key)
	}

	// If property is specified, filter the data
	if ref.Property != "" {
		if val, ok := sd.Data[ref.Property]; ok {
			// Try to unmarshal as JSON
			var nested map[string]any
			if err := json.Unmarshal([]byte(val), &nested); err == nil {
				return convertToByteMap(nested)
			}
			return map[string][]byte{ref.Property: []byte(val)}, nil
		}
		return nil, fmt.Errorf(errPropertyNotFound, ref.Property, ref.Key)
	}

	result := make(map[string][]byte, len(sd.Data))
	for k, v := range sd.Data {
		result[k] = []byte(v)
	}

	return result, nil
}

func convertToByteMap(m map[string]any) (map[string][]byte, error) {
	result := make(map[string][]byte, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = []byte(val)
		default:
			b, err := esutils.JSONMarshal(v)
			if err != nil {
				return nil, err
			}
			result[k] = b
		}
	}
	return result, nil
}

// GetAllSecrets retrieves all secrets matching the given find criteria.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name == nil && ref.Tags == nil {
		return nil, errors.New("either name or tags must be specified")
	}

	// Get all keys with the prefix
	resp, err := c.kv.Get(ctx, c.prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf(errFailedToGetSecret, err)
	}

	result := make(map[string][]byte)

	var matcher *find.Matcher
	if ref.Name != nil {
		matcher, err = find.New(*ref.Name)
		if err != nil {
			return nil, err
		}
	}

	for _, kv := range resp.Kvs {
		// Extract the key name without prefix
		keyName := strings.TrimPrefix(string(kv.Key), c.prefix)

		// Match by name if specified
		if matcher != nil && !matcher.MatchName(keyName) {
			continue
		}

		// Match by tags if specified
		if ref.Tags != nil {
			var sd secretData
			if err := json.Unmarshal(kv.Value, &sd); err != nil {
				continue
			}
			if !matchTags(sd.Metadata, ref.Tags) {
				continue
			}
		}

		// Get the data
		var sd secretData
		if err := json.Unmarshal(kv.Value, &sd); err == nil && sd.Data != nil {
			dataBytes, err := esutils.JSONMarshal(sd.Data)
			if err != nil {
				continue
			}
			result[keyName] = dataBytes
		} else {
			result[keyName] = kv.Value
		}
	}

	return esutils.ConvertKeys(ref.ConversionStrategy, result)
}

func matchTags(metadata, tags map[string]string) bool {
	if metadata == nil {
		return false
	}
	for k, v := range tags {
		if metadata[k] != v {
			return false
		}
	}
	return true
}

// PushSecret writes a secret to etcd.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	key := c.buildKey(data.GetRemoteKey())

	// Get existing secret if any
	resp, err := c.kv.Get(ctx, key)
	if err != nil {
		return fmt.Errorf(errFailedToGetSecret, err)
	}

	var existing secretData
	if len(resp.Kvs) > 0 {
		if err := json.Unmarshal(resp.Kvs[0].Value, &existing); err != nil {
			// If we can't unmarshal, treat as not managed by us
			existing = secretData{
				Data:     make(map[string]string),
				Metadata: make(map[string]string),
			}
		}

		// Check if managed by external-secrets
		if existing.Metadata != nil && existing.Metadata[managedByKey] != "" && existing.Metadata[managedByKey] != managedByValue {
			return errors.New("secret not managed by external-secrets")
		}
	} else {
		existing = secretData{
			Data:     make(map[string]string),
			Metadata: make(map[string]string),
		}
	}

	// Set managed-by metadata
	if existing.Metadata == nil {
		existing.Metadata = make(map[string]string)
	}
	existing.Metadata[managedByKey] = managedByValue

	// Determine what to push
	secretKey := data.GetSecretKey()
	property := data.GetProperty()

	if secretKey == "" {
		// Push the entire secret
		if existing.Data == nil {
			existing.Data = make(map[string]string)
		}
		for k, v := range secret.Data {
			existing.Data[k] = string(v)
		}
	} else {
		// Push a specific key
		value := secret.Data[secretKey]
		targetKey := secretKey
		if property != "" {
			targetKey = property
		}
		if existing.Data == nil {
			existing.Data = make(map[string]string)
		}
		existing.Data[targetKey] = string(value)
	}

	// Marshal and store
	valueBytes, err := json.Marshal(existing)
	if err != nil {
		return fmt.Errorf(errMarshalSecret, err)
	}

	_, err = c.kv.Put(ctx, key, string(valueBytes))
	if err != nil {
		return fmt.Errorf(errFailedToPutSecret, err)
	}

	return nil
}

// DeleteSecret removes a secret from etcd.
func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	key := c.buildKey(remoteRef.GetRemoteKey())

	property := remoteRef.GetProperty()

	// If no property specified, delete the entire key
	if property == "" {
		_, err := c.kv.Delete(ctx, key)
		if err != nil {
			return fmt.Errorf(errFailedToDelSecret, err)
		}
		return nil
	}

	// Otherwise, remove just the property
	resp, err := c.kv.Get(ctx, key)
	if err != nil {
		return fmt.Errorf(errFailedToGetSecret, err)
	}

	if len(resp.Kvs) == 0 {
		// Already doesn't exist
		return nil
	}

	var sd secretData
	if err := json.Unmarshal(resp.Kvs[0].Value, &sd); err != nil {
		// If we can't unmarshal, just delete the whole thing
		_, err := c.kv.Delete(ctx, key)
		if err != nil {
			return fmt.Errorf(errFailedToDelSecret, err)
		}
		return nil
	}

	// Remove the property
	delete(sd.Data, property)

	// If no data left, delete the entire key
	if len(sd.Data) == 0 {
		_, err := c.kv.Delete(ctx, key)
		if err != nil {
			return fmt.Errorf(errFailedToDelSecret, err)
		}
		return nil
	}

	// Otherwise, update with the remaining data
	valueBytes, err := json.Marshal(sd)
	if err != nil {
		return fmt.Errorf(errMarshalSecret, err)
	}

	_, err = c.kv.Put(ctx, key, string(valueBytes))
	if err != nil {
		return fmt.Errorf(errFailedToPutSecret, err)
	}

	return nil
}

// SecretExists checks if a secret exists in etcd.
func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	key := c.buildKey(remoteRef.GetRemoteKey())

	resp, err := c.kv.Get(ctx, key)
	if err != nil {
		return false, fmt.Errorf(errFailedToGetSecret, err)
	}

	if len(resp.Kvs) == 0 {
		return false, nil
	}

	property := remoteRef.GetProperty()
	if property == "" {
		return true, nil
	}

	// Check if property exists
	var sd secretData
	if err := json.Unmarshal(resp.Kvs[0].Value, &sd); err != nil {
		return false, nil
	}

	_, exists := sd.Data[property]
	return exists, nil
}

// Validate checks if the etcd client is configured correctly.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	// Try to get a key to verify connection
	_, err := c.kv.Get(ctx, c.prefix, clientv3.WithLimit(1))
	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	return esv1.ValidationResultReady, nil
}

// Close cleans up resources held by the client.
func (c *Client) Close(_ context.Context) error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// buildKey constructs the full etcd key path.
func (c *Client) buildKey(key string) string {
	// Ensure prefix ends with /
	prefix := c.prefix
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	// Remove leading / from key if present
	key = strings.TrimPrefix(key, "/")
	return prefix + key
}
