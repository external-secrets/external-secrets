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

// Package beyondtrustsecrets provides a client for BeyondTrust Secrets Manager.
package beyondtrustsecrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/httpclient"
	btsutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/util"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

// ErrMsgNotImplemented is the error message for unimplemented methods.
const ErrMsgNotImplemented = "not implemented: %s"

// Client implements the SecretsClient interface for BeyondTrust Secrets.
type Client struct {
	beyondtrustSecretsClient btsutil.Client
	store                    *esv1.BeyondtrustSecretsProvider
}

// Validate checks if the client is configured correctly
// and is able to retrieve secrets from the BeyondTrust Secrets provider.
// If the validation result is unknown it will be ignored.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	timeout := 15 * time.Second
	clientURL := c.beyondtrustSecretsClient.BaseURL().String()
	if err := esutils.NetworkValidate(clientURL, timeout); err != nil {
		return esv1.ValidationResultError, err
	}

	// --TODO: validate auth?

	return esv1.ValidationResultReady, nil
}

// GetSecret returns a single secret from the BeyondTrust Secrets provider
//
//	if GetSecret returns an error with type NoSecretError
//	then the secret entry will be deleted depending on the deletionPolicy.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	folderPath := c.store.FolderPath

	secret, err := c.beyondtrustSecretsClient.GetSecret(ctx, ref.Key, &folderPath)
	if err != nil {
		// Wrap 404s as NoSecretError to allow ESO deletionPolicy handling
		var apiErr *httpclient.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return nil, esv1.NoSecretError{}
		}
		return nil, fmt.Errorf("failed to get secret %w", err)
	}

	// Extract value from map
	if secret.Secret == nil {
		return nil, fmt.Errorf("secret value is nil")
	}

	// If there's a property key in the remote reference, use it
	if ref.Property != "" {
		value, ok := secret.Secret[ref.Property]
		if !ok {
			return nil, fmt.Errorf("property %s not found in secret", ref.Property)
		}

		// Handle different value types to preserve binary and object data
		switch val := value.(type) {
		case string:
			return []byte(val), nil
		case []byte:
			return val, nil
		default:
			// non-string: marshal to JSON to preserve structure
			b, err := json.Marshal(val)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal secret value for property %q: %w", ref.Property, err)
			}
			return b, nil
		}
	}

	// If no property specified, return the entire secret as JSON
	secretBytes, err := json.Marshal(secret.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret: %w", err)
	}

	return secretBytes, nil
}

// GetAllSecrets retrieves all secrets from BeyondTrust Secrets that match the given criteria.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	// Determine folder path: use ref.Path as folder scope if provided, otherwise use store default
	folderPath := c.store.FolderPath
	if ref.Path != nil {
		folderPath = strings.TrimSuffix(*ref.Path, "/")
	}

	result := map[string][]byte{}

	// List all secrets in the folder
	secretsList, err := c.beyondtrustSecretsClient.GetSecrets(ctx, &folderPath)
	if err != nil {
		// Treat 404 from listing API as NoSecretError (folder not found or empty)
		var apiErr *httpclient.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return nil, esv1.NoSecretError{}
		}
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	// If no regexp provided, include everything. If regexp provided, filter names.
	var nameRe *regexp.Regexp
	if ref.Name != nil && ref.Name.RegExp != "" {
		nameRe, err = regexp.Compile(ref.Name.RegExp)
		if err != nil {
			return nil, fmt.Errorf("invalid name regexp %q: %w", ref.Name.RegExp, err)
		}
	}

	for _, item := range secretsList {
		// item.Path may be a full path; split to derive folder/name as the API expects
		dir, itemName := path.Split(item.Path)
		dir = strings.TrimSuffix(dir, "/")
		itemFolderPath := folderPath
		if dir != "" {
			itemFolderPath = dir
		}

		if nameRe != nil {
			if !nameRe.MatchString(itemName) {
				continue
			}
		}

		// Fetch the full secret for this matched item
		fullSecret, err := c.beyondtrustSecretsClient.GetSecret(ctx, itemName, &itemFolderPath)
		if err != nil {
			// In name-regex listing, skip missing items instead of failing the entire operation
			var apiErr *httpclient.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
				continue
			}
			return nil, fmt.Errorf("failed to get secret at path %q: %w", path.Join(itemFolderPath, itemName), err)
		}
		if fullSecret == nil || fullSecret.Secret == nil {
			// Skip empty/missing entries in list mode
			continue
		}
		// Merge keys from this secret into result.
		for k, v := range fullSecret.Secret {
			switch val := v.(type) {
			case string:
				result[k] = []byte(val)
			case []byte:
				result[k] = val
			default:
				b, err := json.Marshal(val)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal secret value for key %q from %s: %w", k, itemName, err)
				}
				result[k] = b
			}
		}
	}

	// If no secrets matched the criteria, return NoSecretError
	if len(result) == 0 {
		return nil, esv1.NoSecretError{}
	}

	return result, nil
}

// GetSecretMap returns multiple k/v pairs from the BeyondTrust Secrets provider as separate keys.
// Each key-value pair in the secret becomes a separate entry in the returned map.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	folderPath := c.store.FolderPath

	secret, err := c.beyondtrustSecretsClient.GetSecret(ctx, ref.Key, &folderPath)
	if err != nil {
		// Wrap 404s as NoSecretError to allow ESO deletionPolicy handling
		var apiErr *httpclient.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return nil, esv1.NoSecretError{}
		}
		return nil, fmt.Errorf("failed to get secret %w", err)
	}

	if secret == nil || secret.Secret == nil {
		return nil, fmt.Errorf("secret value is nil")
	}

	// Convert all k/v pairs to []byte, preserving structure for non-string values
	result := make(map[string][]byte)
	for k, v := range secret.Secret {
		switch val := v.(type) {
		case string:
			result[k] = []byte(val)
		case []byte:
			result[k] = val
		default:
			// non-string: marshal to JSON to preserve structure
			b, err := json.Marshal(val)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal secret value for key %q: %w", k, err)
			}
			result[k] = b
		}
	}

	return result, nil
}

/////////////////////////
// NOT YET IMPLEMENTED //
/////////////////////////

// PushSecret will write a single secret into the BeyondTrust Secrets provider.
func (c *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return fmt.Errorf(ErrMsgNotImplemented, "PushSecret")
}

// DeleteSecret will delete the secret from the BeyondTrust Secrets provider.
func (c *Client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return fmt.Errorf(ErrMsgNotImplemented, "DeleteSecret")
}

// SecretExists checks if a secret is already present in the BeyondTrust Secrets provider at the given location.
func (c *Client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf(ErrMsgNotImplemented, "SecretExists")
}

// Close implements cleanup operations for the BeyondTrust Secrets client.
func (c *Client) Close(_ context.Context) error {
	return nil
}
