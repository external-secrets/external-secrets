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

package dvls

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Devolutions/go-dvls"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errFailedToGetEntry = "failed to get entry: %w"
	errVaultNotFound    = "vault %q was not found or has been deleted: %w"
)

var errNotImplemented = errors.New("not implemented")

var _ esv1.SecretsClient = &Client{}

// Client implements the SecretsClient interface for DVLS.
// The nameCache maps entry name/path keys to resolved UUIDs, avoiding
// repeated GetEntries calls during a single reconciliation. The cache is
// not persisted: each reconciliation creates a new Client via NewClient,
// so stale entries (e.g. deleted or renamed) are naturally discarded.
type Client struct {
	cred      credentialClient
	vaultID   string
	mu        sync.RWMutex
	nameCache map[string]string
}

type credentialClient interface {
	GetByID(ctx context.Context, vaultID, entryID string) (dvls.Entry, error)
	GetEntries(ctx context.Context, vaultID string, opts dvls.GetEntriesOptions) ([]dvls.Entry, error)
	Update(ctx context.Context, entry dvls.Entry) (dvls.Entry, error)
	DeleteByID(ctx context.Context, vaultID, entryID string) error
}

type vaultGetter interface {
	GetByName(ctx context.Context, name string) (dvls.Vault, error)
}

type realCredentialClient struct {
	cred *dvls.EntryCredentialService
}

func (r *realCredentialClient) GetByID(ctx context.Context, vaultID, entryID string) (dvls.Entry, error) {
	return r.cred.GetByIdWithContext(ctx, vaultID, entryID)
}

func (r *realCredentialClient) GetEntries(ctx context.Context, vaultID string, opts dvls.GetEntriesOptions) ([]dvls.Entry, error) {
	return r.cred.GetEntriesWithContext(ctx, vaultID, opts)
}

func (r *realCredentialClient) Update(ctx context.Context, entry dvls.Entry) (dvls.Entry, error) {
	return r.cred.UpdateWithContext(ctx, entry)
}

func (r *realCredentialClient) DeleteByID(ctx context.Context, vaultID, entryID string) error {
	return r.cred.DeleteByIdWithContext(ctx, vaultID, entryID)
}

// NewClient creates a new DVLS secrets client.
func NewClient(cred credentialClient, vaultID string) *Client {
	return &Client{cred: cred, vaultID: vaultID, nameCache: make(map[string]string)}
}

// GetSecret retrieves a secret from DVLS.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	vaultID, entryID, err := c.resolveRef(ctx, ref.Key)
	if isNotFoundError(err) {
		return nil, esv1.NoSecretErr
	}
	if err != nil {
		return nil, err
	}

	entry, err := c.cred.GetByID(ctx, vaultID, entryID)
	if isVaultNotFoundError(err) {
		return nil, fmt.Errorf(errVaultNotFound, vaultID, err)
	}
	if isNotFoundError(err) {
		return nil, esv1.NoSecretErr
	}
	if err != nil {
		return nil, fmt.Errorf(errFailedToGetEntry, err)
	}

	secretMap, err := entryToSecretMap(entry)
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
	vaultID, entryID, err := c.resolveRef(ctx, ref.Key)
	if isNotFoundError(err) {
		return nil, esv1.NoSecretErr
	}
	if err != nil {
		return nil, err
	}

	entry, err := c.cred.GetByID(ctx, vaultID, entryID)
	if isVaultNotFoundError(err) {
		return nil, fmt.Errorf(errVaultNotFound, vaultID, err)
	}
	if isNotFoundError(err) {
		return nil, esv1.NoSecretErr
	}
	if err != nil {
		return nil, fmt.Errorf(errFailedToGetEntry, err)
	}

	return entryToSecretMap(entry)
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
	vaultID, entryID, err := c.resolveRef(ctx, data.GetRemoteKey())
	if isVaultNotFoundError(err) {
		return fmt.Errorf(errVaultNotFound, c.vaultID, err)
	}
	if isNotFoundError(err) {
		return fmt.Errorf("entry %q not found: entry must exist before pushing secrets", data.GetRemoteKey())
	}
	if err != nil {
		return err
	}

	value, err := extractPushValue(secret, data)
	if err != nil {
		return err
	}

	existingEntry, err := c.cred.GetByID(ctx, vaultID, entryID)
	if isVaultNotFoundError(err) {
		return fmt.Errorf(errVaultNotFound, vaultID, err)
	}
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

	_, err = c.cred.Update(ctx, existingEntry)
	if err != nil {
		return fmt.Errorf("failed to update entry: %w", err)
	}
	return nil
}

// DeleteSecret deletes a secret from DVLS.
func (c *Client) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) error {
	vaultID, entryID, err := c.resolveRef(ctx, ref.GetRemoteKey())
	if isNotFoundError(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := c.cred.DeleteByID(ctx, vaultID, entryID); err != nil {
		if isNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("failed to delete entry %q from vault %q: %w", entryID, vaultID, err)
	}
	return nil
}

// SecretExists checks if a secret exists in DVLS.
func (c *Client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	vaultID, entryID, err := c.resolveRef(ctx, ref.GetRemoteKey())
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	_, err = c.cred.GetByID(ctx, vaultID, entryID)
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
	if c.cred == nil {
		return esv1.ValidationResultError, errors.New("DVLS client is not initialized")
	}
	return esv1.ValidationResultReady, nil
}

// Close is a no-op for the DVLS client.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// resolveRef resolves a key to a vault ID and entry ID.
// When c.vaultID is set, the key is treated as an entry reference.
// When c.vaultID is empty, the key is parsed as the legacy "<vault-uuid>/<entry-uuid>" format.
func (c *Client) resolveRef(ctx context.Context, key string) (vaultID, entryID string, err error) {
	if c.vaultID == "" {
		return parseLegacyRef(key)
	}
	entryID, err = c.resolveEntryRef(ctx, key)
	return c.vaultID, entryID, err
}

// resolveEntryRef resolves an entry reference to a UUID.
// The key can be:
//   - A UUID: used directly.
//   - A name: looked up via GetEntries.
//   - A path/name: "folder/subfolder/entry-name" — path is used to filter.
func (c *Client) resolveEntryRef(ctx context.Context, key string) (entryID string, err error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("entry reference cannot be empty")
	}

	// UUID passes through directly.
	if isUUID(key) {
		return key, nil
	}

	// Return cached result if available.
	c.mu.RLock()
	id, ok := c.nameCache[key]
	c.mu.RUnlock()
	if ok {
		return id, nil
	}

	// Split into optional path + entry name.
	entryName, entryPath := parseEntryRef(key)
	if entryName == "" {
		return "", errors.New("entry name cannot be empty")
	}

	opts := dvls.GetEntriesOptions{Name: &entryName}
	if entryPath != "" {
		opts.Path = &entryPath
	}

	entries, err := c.cred.GetEntries(ctx, c.vaultID, opts)
	if isVaultNotFoundError(err) {
		return "", fmt.Errorf(errVaultNotFound, c.vaultID, err)
	}
	if err != nil {
		return "", fmt.Errorf("failed to resolve entry %q: %w", key, err)
	}

	switch len(entries) {
	case 0:
		return "", fmt.Errorf("entry %q not found in vault: %w", key, dvls.ErrEntryNotFound)
	case 1:
		c.mu.Lock()
		c.nameCache[key] = entries[0].Id
		c.mu.Unlock()
		return entries[0].Id, nil
	default:
		details := make([]string, len(entries))
		for i, e := range entries {
			details[i] = fmt.Sprintf("  %s (path=%q, type=%s)", e.Id, e.Path, e.Type)
		}
		return "", fmt.Errorf("found %d credential entries named %q; use the entry UUID to select one:\n%s", len(entries), entryName, strings.Join(details, "\n"))
	}
}

// parseLegacyRef parses the legacy secret reference format "<vault-uuid>/<entry-uuid>".
// This preserves backward compatibility for users who don't set the vault field.
func parseLegacyRef(key string) (vaultID, entryID string, err error) {
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
	if !isUUID(vaultID) {
		return "", "", fmt.Errorf("invalid vault UUID: %q", vaultID)
	}
	if !isUUID(entryID) {
		return "", "", fmt.Errorf("invalid entry UUID: %q", entryID)
	}

	return vaultID, entryID, nil
}

// resolveVaultRef resolves a vault reference (name or UUID) to a vault UUID.
func resolveVaultRef(ctx context.Context, vaultRef string, vc vaultGetter) (string, error) {
	if isUUID(vaultRef) {
		return vaultRef, nil
	}
	vault, err := vc.GetByName(ctx, vaultRef)
	if err != nil {
		return "", fmt.Errorf("failed to resolve vault %q: %w", vaultRef, err)
	}
	return vault.Id, nil
}

// parseEntryRef splits an entry reference into name and optional path.
// Both forward slashes and backslashes are accepted as path separators.
// The last separator splits the path from the entry name.
// Paths are normalized to backslashes to match the DVLS path format.
// e.g. "folder/subfolder/my-entry" → name="my-entry", path="folder\subfolder".
// e.g. "folder\subfolder\my-entry" → name="my-entry", path="folder\subfolder".
func parseEntryRef(ref string) (name, path string) {
	// Normalize forward slashes to backslashes.
	normalized := strings.ReplaceAll(ref, "/", `\`)
	if idx := strings.LastIndex(normalized, `\`); idx >= 0 {
		return normalized[idx+1:], normalized[:idx]
	}
	return ref, ""
}

// isUUID returns true if the string is a valid UUID.
func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// entryToSecretMap converts a DVLS entry to a map of secret values.
func entryToSecretMap(entry dvls.Entry) (map[string][]byte, error) {
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

	if errors.Is(err, dvls.ErrEntryNotFound) {
		return true
	}

	return false
}

func isVaultNotFoundError(err error) bool {
	return err != nil && errors.Is(err, dvls.ErrVaultNotFound)
}
