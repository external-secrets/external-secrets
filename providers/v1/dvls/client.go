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

package dvls

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Devolutions/go-dvls"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const errFailedToGetEntry = "failed to get entry: %w"

var errNotImplemented = errors.New("not implemented")

var _ esv1.SecretsClient = &Client{}

// Client implements the SecretsClient interface for DVLS.
type Client struct {
	dvls credentialClient
}

type credentialClient interface {
	GetByID(ctx context.Context, vaultID, entryID string) (dvls.Entry, error)
	Update(ctx context.Context, entry dvls.Entry) (dvls.Entry, error)
	DeleteByID(ctx context.Context, vaultID, entryID string) error
}

type realCredentialClient struct {
	cred *dvls.EntryCredentialService
}

func (r *realCredentialClient) GetByID(ctx context.Context, vaultID, entryID string) (dvls.Entry, error) {
	return r.cred.GetByIdWithContext(ctx, vaultID, entryID)
}

func (r *realCredentialClient) Update(ctx context.Context, entry dvls.Entry) (dvls.Entry, error) {
	return r.cred.UpdateWithContext(ctx, entry)
}

func (r *realCredentialClient) DeleteByID(ctx context.Context, vaultID, entryID string) error {
	return r.cred.DeleteByIdWithContext(ctx, vaultID, entryID)
}

// NewClient creates a new DVLS secrets client.
func NewClient(dvlsClient credentialClient) *Client {
	return &Client{dvls: dvlsClient}
}

// GetSecret retrieves a secret from DVLS.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	vaultID, entryID, err := c.parseSecretRef(ref.Key)
	if err != nil {
		return nil, err
	}

	entry, err := c.dvls.GetByID(ctx, vaultID, entryID)
	if isNotFoundError(err) {
		return nil, esv1.NoSecretErr
	}
	if err != nil {
		return nil, fmt.Errorf(errFailedToGetEntry, err)
	}

	secretMap, err := c.entryToSecretMap(entry)
	if err != nil {
		return nil, err
	}

	// Default to "password" when no property specified (consistent with 1Password provider).
	property := ref.Property
	if property == "" {
		property = "password"
	}

	value, ok := secretMap[property]
	if !ok {
		return nil, fmt.Errorf("property %q not found in entry", property)
	}
	return value, nil
}

// GetSecretMap retrieves all fields from a DVLS entry.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	vaultID, entryID, err := c.parseSecretRef(ref.Key)
	if err != nil {
		return nil, err
	}

	entry, err := c.dvls.GetByID(ctx, vaultID, entryID)
	if isNotFoundError(err) {
		return nil, esv1.NoSecretErr
	}
	if err != nil {
		return nil, fmt.Errorf(errFailedToGetEntry, err)
	}

	return c.entryToSecretMap(entry)
}

// GetAllSecrets is not implemented for DVLS.
func (c *Client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errNotImplemented
}

// PushSecret updates an existing entry's password field.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if secret == nil {
		return errors.New("secret is required for DVLS push")
	}
	vaultID, entryID, err := c.parseSecretRef(data.GetRemoteKey())
	if err != nil {
		return err
	}

	value, err := extractPushValue(secret, data)
	if err != nil {
		return err
	}

	existingEntry, err := c.dvls.GetByID(ctx, vaultID, entryID)
	if isNotFoundError(err) {
		return fmt.Errorf("entry %s not found in vault %s: entry must exist before pushing secrets", entryID, vaultID)
	}
	if err != nil {
		return fmt.Errorf(errFailedToGetEntry, err)
	}

	// SetCredentialSecret only updates the password/secret field.
	if err := existingEntry.SetCredentialSecret(string(value)); err != nil {
		return err
	}

	_, err = c.dvls.Update(ctx, existingEntry)
	if err != nil {
		return fmt.Errorf("failed to update entry: %w", err)
	}
	return nil
}

// DeleteSecret deletes a secret from DVLS.
func (c *Client) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) error {
	vaultID, entryID, err := c.parseSecretRef(ref.GetRemoteKey())
	if err != nil {
		return err
	}
	return c.dvls.DeleteByID(ctx, vaultID, entryID)
}

// SecretExists checks if a secret exists in DVLS.
func (c *Client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	vaultID, entryID, err := c.parseSecretRef(ref.GetRemoteKey())
	if err != nil {
		return false, err
	}

	_, err = c.dvls.GetByID(ctx, vaultID, entryID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Validate checks if the client is properly configured.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	if c.dvls == nil {
		return esv1.ValidationResultError, errors.New("DVLS client is not initialized")
	}
	return esv1.ValidationResultReady, nil
}

// Close is a no-op for the DVLS client.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// parseSecretRef parses the secret reference key.
// Format: "<vault-id>/<entry-id>".
func (c *Client) parseSecretRef(key string) (vaultID, entryID string, err error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid key format: expected '<vault-id>/<entry-id>', got %q", key)
	}

	vaultID = strings.TrimSpace(parts[0])
	entryID = strings.TrimSpace(parts[1])

	if vaultID == "" {
		return "", "", errors.New("vault ID cannot be empty")
	}
	if entryID == "" {
		return "", "", errors.New("entry ID cannot be empty")
	}

	return vaultID, entryID, nil
}

// entryToSecretMap converts a DVLS entry to a map of secret values.
func (c *Client) entryToSecretMap(entry dvls.Entry) (map[string][]byte, error) {
	secretMap, err := entry.ToCredentialMap()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(secretMap))
	for k, v := range secretMap {
		result[k] = []byte(v)
	}

	return result, nil
}

func extractPushValue(secret *corev1.Secret, data esv1.PushSecretData) ([]byte, error) {
	if data.GetSecretKey() == "" {
		return nil, fmt.Errorf("secretKey is required for DVLS push")
	}

	if secret.Data == nil {
		return nil, fmt.Errorf("secret %q has no data", secret.Name)
	}

	value, ok := secret.Data[data.GetSecretKey()]
	if !ok {
		return nil, fmt.Errorf("key %q not found in secret %q", data.GetSecretKey(), secret.Name)
	}

	if len(value) == 0 {
		return nil, fmt.Errorf("key %q in secret %q is empty", data.GetSecretKey(), secret.Name)
	}

	return value, nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if dvls.IsNotFound(err) {
		return true
	}

	var reqErr dvls.RequestError
	return errors.As(err, &reqErr) && reqErr.StatusCode == http.StatusNotFound
}
