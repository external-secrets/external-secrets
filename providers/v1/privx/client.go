/*
Copyright © 2026 SSH Communications

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

// Package privx implements the ESO SecretsClient for SSH PrivX
package privx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/SSHcom/privx-sdk-go/v2/api/filters"
	"github.com/SSHcom/privx-sdk-go/v2/api/rolestore"
	"github.com/SSHcom/privx-sdk-go/v2/api/vault"
	privxapi "github.com/SSHcom/privx-sdk-go/v2/restapi"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var (
	// ErrNoName is returned when no name is provided for the secret.
	ErrNoName = errors.New("No name provided for secret")

	// ErrUnsupportedDecodingStrategy is returned when the decoding strategy is not supported.
	ErrUnsupportedDecodingStrategy = errors.New("unsupported decoding strategy")

	// ErrSecretDataMissing is returned when secret data is missing.
	ErrSecretDataMissing = errors.New("secret data missing")

	// ErrPropertyNotFound is returned when the requested property is not found in the secret.
	ErrPropertyNotFound = errors.New("property not found in secret")
)

// Check during compile that we implement the interface.
var _ esv1.SecretsClient = (*SecretsClient)(nil)

// SecretsClient provides access to PrivX secrets.
type SecretsClient struct {
	conn      privxapi.Connector
	vault     *vault.Vault // PrivX Vault instance
	store     esv1.GenericStore
	kube      kclient.Client
	namespace string

	// PrivX needs roles when creating a new secret.
	defaultReadRoles  []string
	defaultWriteRoles []string
}

// GetSecret returns a single secret from the provider.
func (c *SecretsClient) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	secret, err := c.vault.GetSecret(ref.Key)
	if err != nil {
		return nil, err
	}
	if secret.Data == nil {
		return nil, fmt.Errorf("%w: %s", ErrSecretDataMissing, ref.Key)
	}

	// If no property requested, return whole JSON object
	if ref.Property == "" {
		return json.Marshal(*secret.Data)
	}

	v, ok := (*secret.Data)[ref.Property]
	if !ok || v == nil {
		return nil, fmt.Errorf("%w: %s/%s", ErrPropertyNotFound, ref.Key, ref.Property)
	}

	// Convert the selected value to []byte
	b, err := anyToBytes(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// packRoles forms RoleHandles from a list of role ID
//
// The PrivX API will ignore the name field.
// See https://privx.docs.ssh.com/v42/api/vault/create-a-secret
func packRoles(roleIDs []string) []rolestore.RoleHandle {
	result := []rolestore.RoleHandle{}
	for _, id := range roleIDs {
		result = append(result, rolestore.RoleHandle{ID: id, Name: ""})
	}
	return result
}

// PushSecret will write a single secret into PrivX.
//
// Access for the new secret in PrivX is defined by variables default*Roles set for the store.
func (c *SecretsClient) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	remoteKey := data.GetRemoteKey()
	name := remoteKey
	if name == "" {
		name = secret.Name
	}
	if name == "" {
		return ErrNoName
	}

	secretKey := data.GetSecretKey()
	secretValue := secret.Data[secretKey]
	m := &map[string]interface{}{secretKey: secretValue}

	request := vault.SecretRequest{
		Name:       name,
		ReadRoles:  packRoles(c.defaultReadRoles),
		WriteRoles: packRoles(c.defaultWriteRoles),
		Data:       m,
	}
	_, err := c.vault.CreateSecret(&request)

	if err != nil {
		logger := log.FromContext(ctx)
		logger.Error(
			err,
			"privx error",
			"errorType", fmt.Sprintf("%T", err),
			"remoteKey", name,
			"readRoles", c.defaultReadRoles,
			"writeRoles", c.defaultWriteRoles,
		)
	}
	return err
}

// DeleteSecret will delete the secret from PrivX.
func (c *SecretsClient) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := c.vault.DeleteSecret(ref.GetRemoteKey())
	if err == nil {
		return nil
	}
	if isNotFound(err) {
		return nil
	}
	return err
}

// SecretExists checks if a secret is already present in PrivX at the given location.
func (c *SecretsClient) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	remoteRef := esv1.ExternalSecretDataRemoteRef{Key: ref.GetRemoteKey()}
	_, err := c.GetSecret(context.TODO(), remoteRef)
	if err == nil {
		return true, nil
	}

	if isNotFound(err) {
		return false, nil
	}

	// Other error than just "not found"
	return false, err
}

// Validate checks if the client is configured correctly
// and is able to retrieve secrets from the provider.
// If the validation result is unknown it will be ignored.
func (c *SecretsClient) Validate() (esv1.ValidationResult, error) {
	_, err := c.GetSecret(context.TODO(), esv1.ExternalSecretDataRemoteRef{Key: "2F0vZqCe0Z3XU5"})

	if isNotFound(err) {
		// We requested a non-existing secret and this is the proper response from PrivX -- all ok.
		return esv1.ValidationResultReady, nil
	}

	return esv1.ValidationResultError, err
}

// GetSecretMap returns multiple key/value pairs from a PrivX secret.
//
// If ref.Property is empty, all top-level keys are returned.
// If ref.Property refers to a nested JSON object, its fields are returned.
// Otherwise, a single key/value pair is returned containing the selected property.
func (c *SecretsClient) GetSecretMap(
	ctx context.Context,
	ref esv1.ExternalSecretDataRemoteRef,
) (map[string][]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	secret, err := c.vault.GetSecret(ref.Key)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil {
		return nil, ErrSecretDataMissing
	}

	data := *secret.Data

	// 1) No property specified: return all top-level keys
	if ref.Property == "" {
		out := make(map[string][]byte, len(data))

		for k, v := range data {
			b, err := anyToBytes(v)
			if err != nil {
				return nil, err
			}
			out[k] = b
		}
		return out, nil
	}

	// 2) Property specified: extract it
	v, ok := data[ref.Property]
	if !ok || v == nil {
		return nil, ErrPropertyNotFound
	}

	// If property is a nested object, return its fields
	if nested, ok := v.(map[string]interface{}); ok {
		out := make(map[string][]byte, len(nested))
		for k, nv := range nested {
			b, err := anyToBytes(nv)
			if err != nil {
				return nil, err
			}
			out[k] = b
		}
		return out, nil
	}

	// Otherwise return a single key/value pair
	b, err := anyToBytes(v)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		ref.Property: b,
	}, nil
}

// GetAllSecrets returns multiple secrets and their JSON values from PrivX.
//
// The returned map key is the secret name and the value is the full JSON document
// for that secret (the whole secret.Data marshaled as JSON). This avoids key
// collisions between secrets that may contain identical JSON keys internally.
func (c *SecretsClient) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	results := make(map[string][]byte)

	if ref.Path != nil {
		return results, fmt.Errorf("parameter %q: %w", "ref.Path", ErrNotImplemented)
	}
	if ref.Tags != nil {
		return results, fmt.Errorf("parameter %q: %w", "ref.Tags", ErrNotImplemented)
	}
	if ref.ConversionStrategy != esv1.ExternalSecretConversionDefault {
		return results, fmt.Errorf("parameter %q: %w", "ref.ConversionStrategy", ErrNotImplemented)
	}

	searchString := ""
	if ref.Name != nil {
		// Missing search parameter is considered an empty string, which matches all
		searchString = ref.Name.RegExp
	}

	nameRegexp, err := regexp.Compile(searchString)
	if err != nil {
		return results, fmt.Errorf("invalid regex %q: %w", searchString, err)
	}

	// Loop through all secrets 100 at a time
	const limit = 100
	for offset := 0; ; offset += limit {
		secrets, err := c.vault.GetSecrets(filters.Limit(limit), filters.Offset(offset))
		if err != nil {
			return results, err
		}

		if secrets.Count == 0 {
			break
		}

		for _, secret := range secrets.Items {
			if !nameRegexp.MatchString(secret.Name) {
				continue
			}

			secretDetails, err := c.vault.GetSecret(secret.Name)
			if err != nil {
				return results, err
			}

			if secretDetails.Data == nil {
				return results, ErrSecretDataMissing
			}

			// Marshal the full JSON object (top-level map) as the secret value
			b, err := json.Marshal(*secretDetails.Data)
			if err != nil {
				return results, err
			}

			results[secret.Name] = b
		}

		if secrets.Count < limit {
			break
		}
	}

	return results, nil
}

// Close closes the client and releases all resources.
func (c *SecretsClient) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Nothing to close or release.
	return nil
}

// Helper functions

// isNotFound return whether the error is a 404 - Not Found.
func isNotFound(err error) bool {
	// PrivX loses the HTTP code so we need to test the error message
	return strings.Contains(strings.ToLower(err.Error()), "secret not found")
}

// anyToBytes converts a JSON-unmarshaled value (interface{}) to []byte.
func anyToBytes(v any) ([]byte, error) {
	switch t := v.(type) {
	case []byte:
		// Already bytes
		return t, nil

	case string:
		// Common case for JSON: strings become string (not []byte).
		return []byte(t), nil

	case bool:
		return []byte(strconv.FormatBool(t)), nil

	case float64:
		// JSON numbers become float64 when unmarshaling into interface{}
		return []byte(strconv.FormatFloat(t, 'f', -1, 64)), nil

	case json.Number:
		return []byte(t.String()), nil

	default:
		// For objects/arrays (map/slice) and other types: return JSON encoding
		return json.Marshal(t)
	}
}
