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

package sapcredentialstore

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const (
	credTypePassword = "password"
	credTypeKey      = "key"
)

// Client implements esv1.SecretsClient for the SAP Credential Store.
type Client struct {
	api       *APIClient
	namespace string
}

var _ esv1.SecretsClient = &Client{}

// credTypeFromProperty maps an ExternalSecret ref.Property value to a SAP CS credential type.
// Empty or "password" -> password, "key" -> key. Unknown values return an error.
func credTypeFromProperty(property string) (string, error) {
	switch strings.ToLower(property) {
	case "", credTypePassword:
		return credTypePassword, nil
	case credTypeKey:
		return credTypeKey, nil
	default:
		return "", fmt.Errorf(
			"unsupported credential type %q, supported types are: password, key",
			property,
		)
	}
}

// GetSecret returns a single secret from the SAP Credential Store.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	credType, err := credTypeFromProperty(ref.Property)
	if err != nil {
		return nil, err
	}

	cred, err := c.api.GetCredential(ctx, c.namespace, credType, ref.Key)
	if err != nil {
		if _, ok := errors.AsType[*NotFoundError](err); ok {
			return nil, esv1.NoSecretErr
		}
		return nil, fmt.Errorf("fetching credential %s/%s: %w", credType, ref.Key, err)
	}

	// Key-type values are base64-encoded in SAP CS.
	if credType == credTypeKey {
		decoded, err := base64.StdEncoding.DecodeString(cred.Value)
		if err != nil {
			return nil, fmt.Errorf("decoding key value: %w", err)
		}
		return decoded, nil
	}

	return []byte(cred.Value), nil
}

// PushSecret writes a secret to the SAP Credential Store.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	value, err := esutils.ExtractSecretData(data, secret)
	if err != nil {
		return fmt.Errorf("extracting secret data: %w", err)
	}

	credType, err := credTypeFromProperty(data.GetProperty())
	if err != nil {
		return err
	}
	name := data.GetRemoteKey()

	// Key-type values must be base64-encoded per SAP CS API spec.
	val := string(value)
	if credType == credTypeKey {
		val = base64.StdEncoding.EncodeToString(value)
	}

	body := &CredentialBody{
		Name:  name,
		Value: val,
	}

	return c.api.PutCredential(ctx, c.namespace, credType, body)
}

// DeleteSecret deletes a secret from the SAP Credential Store.
func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	credType, err := credTypeFromProperty(remoteRef.GetProperty())
	if err != nil {
		return err
	}
	name := remoteRef.GetRemoteKey()

	return c.api.DeleteCredential(ctx, c.namespace, credType, name)
}

// SecretExists checks whether a secret exists in the SAP Credential Store.
func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	credType, err := credTypeFromProperty(remoteRef.GetProperty())
	if err != nil {
		return false, err
	}
	name := remoteRef.GetRemoteKey()

	return c.api.CredentialExists(ctx, c.namespace, credType, name)
}

// Validate checks client connectivity.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultUnknown, nil
}

// GetSecretMap returns multiple k/v pairs from a single credential.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	credType, err := credTypeFromProperty(ref.Property)
	if err != nil {
		return nil, err
	}

	cred, err := c.api.GetCredential(ctx, c.namespace, credType, ref.Key)
	if err != nil {
		if _, ok := errors.AsType[*NotFoundError](err); ok {
			return nil, esv1.NoSecretErr
		}
		return nil, fmt.Errorf("fetching credential %s/%s: %w", credType, ref.Key, err)
	}

	result := map[string][]byte{
		"name":  []byte(cred.Name),
		"value": []byte(cred.Value),
	}
	if cred.Username != "" {
		result["username"] = []byte(cred.Username)
	}
	if cred.Key != "" {
		result["key"] = []byte(cred.Key)
	}

	// Decode base64-encoded key values.
	if credType == credTypeKey {
		decoded, derr := base64.StdEncoding.DecodeString(cred.Value)
		if derr == nil {
			result["value"] = decoded
		}
	}

	return result, nil
}

// GetAllSecrets returns all secrets matching the find criteria.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	result := make(map[string][]byte)

	var nameRegex *regexp.Regexp
	if ref.Name != nil && ref.Name.RegExp != "" {
		var err error
		nameRegex, err = regexp.Compile(ref.Name.RegExp)
		if err != nil {
			return nil, fmt.Errorf("compiling name regex: %w", err)
		}
	}

	credTypes := []string{credTypePassword, credTypeKey}
	var listErrors []error
	for _, ct := range credTypes {
		metas, err := c.api.ListCredentials(ctx, c.namespace, ct)
		if err != nil {
			log.V(1).Info("skipping credential type on list error", "type", ct, "error", err)
			listErrors = append(listErrors, fmt.Errorf("listing %s credentials: %w", ct, err))
			continue
		}
		for _, meta := range metas {
			if nameRegex != nil && !nameRegex.MatchString(meta.Name) {
				continue
			}

			cred, err := c.api.GetCredential(ctx, c.namespace, ct, meta.Name)
			if err != nil {
				log.V(1).Info("skipping credential on get error", "type", ct, "name", meta.Name, "error", err)
				continue
			}

			mapKey := fmt.Sprintf("%s/%s", ct, meta.Name)
			value := cred.Value
			if ct == credTypeKey {
				decoded, derr := base64.StdEncoding.DecodeString(value)
				if derr == nil {
					result[mapKey] = decoded
					continue
				}
			}
			result[mapKey] = []byte(value)
		}
	}

	// If every credential type failed to list, return an error rather than
	// silently returning an empty map (which the controller would interpret as
	// "no credentials exist").
	if len(listErrors) == len(credTypes) {
		return nil, fmt.Errorf("failed to list any credential types: %w", errors.Join(listErrors...))
	}

	return result, nil
}

// Close is a no-op for this provider.
func (c *Client) Close(_ context.Context) error {
	return nil
}
