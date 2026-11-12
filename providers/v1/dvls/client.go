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

// vaultError reports a failure to resolve or reach a vault, as opposed to a missing entry.
// isNotFoundError rejects it outright, so a 404 from a vault endpoint cannot become
// NoSecretErr and let deletionPolicy: Delete prune a live Secret.
type vaultError struct {
	vaultRef string
	err      error
}

func (e *vaultError) Error() string {
	if e.vaultRef == "" {
		return fmt.Sprintf("failed to list DVLS vaults: %v", e.err)
	}
	return fmt.Sprintf("vault %q was not found or is unreachable: %v", e.vaultRef, e.err)
}

func (e *vaultError) Unwrap() error { return e.err }

var _ esv1.SecretsClient = &Client{}

// Client implements the SecretsClient interface for DVLS.
// The nameCache maps vault-scoped entry name/path keys to resolved UUIDs, avoiding
// repeated GetEntries calls during a single reconciliation. vaultIndex does the same
// for vault names. Neither is persisted: each reconciliation creates a new Client via
// NewClient, so stale entries (e.g. deleted or renamed) are naturally discarded.
// folderCache records the folder paths a push has already found or created, for the
// same lifetime.
type Client struct {
	cred        credentialClient
	folder      folderClient
	vaults      vaultGetter
	vaultID     string
	pinnedVault bool
	mu          sync.RWMutex
	nameCache   map[string]string
	folderCache map[string]struct{}
	indexMu     sync.Mutex
	vaultIndex  map[string]string
}

type credentialClient interface {
	GetByID(ctx context.Context, vaultID, entryID string) (dvls.Entry, error)
	GetEntries(ctx context.Context, vaultID string, opts dvls.GetEntriesOptions) ([]dvls.Entry, error)
	New(ctx context.Context, entry dvls.Entry) (string, error)
	Update(ctx context.Context, entry dvls.Entry) (dvls.Entry, error)
	DeleteByID(ctx context.Context, vaultID, entryID string) error
}

// folderClient covers the folder entries a push has to create, since DVLS
// keeps folders as entries of their own rather than as a path string on the
// entry that lives in them.
type folderClient interface {
	GetEntries(ctx context.Context, vaultID string, opts dvls.GetEntriesOptions) ([]dvls.Entry, error)
	New(ctx context.Context, entry dvls.Entry) (string, error)
}

type vaultGetter interface {
	GetByName(ctx context.Context, name string) (dvls.Vault, error)
	Get(ctx context.Context, id string) (dvls.Vault, error)
	List(ctx context.Context) ([]dvls.Vault, error)
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

func (r *realCredentialClient) New(ctx context.Context, entry dvls.Entry) (string, error) {
	return r.cred.NewWithContext(ctx, entry)
}

func (r *realCredentialClient) Update(ctx context.Context, entry dvls.Entry) (dvls.Entry, error) {
	return r.cred.UpdateWithContext(ctx, entry)
}

func (r *realCredentialClient) DeleteByID(ctx context.Context, vaultID, entryID string) error {
	return r.cred.DeleteByIdWithContext(ctx, vaultID, entryID)
}

type realFolderClient struct {
	folder *dvls.EntryFolderService
}

func (r *realFolderClient) GetEntries(ctx context.Context, vaultID string, opts dvls.GetEntriesOptions) ([]dvls.Entry, error) {
	return r.folder.GetEntriesWithContext(ctx, vaultID, opts)
}

func (r *realFolderClient) New(ctx context.Context, entry dvls.Entry) (string, error) {
	return r.folder.NewWithContext(ctx, entry)
}

// NewClient creates a new DVLS secrets client. When pinnedVault is true the store
// configured a vault and vaultID holds it; otherwise the vault comes from the key.
func NewClient(cred credentialClient, folder folderClient, vaults vaultGetter, vaultID string, pinnedVault bool) *Client {
	return &Client{
		cred:        cred,
		folder:      folder,
		vaults:      vaults,
		vaultID:     vaultID,
		pinnedVault: pinnedVault,
		nameCache:   make(map[string]string),
		folderCache: make(map[string]struct{}),
	}
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
		if vaultErr := c.confirmVault(ctx, vaultID); vaultErr != nil {
			return nil, vaultErr
		}
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
		if vaultErr := c.confirmVault(ctx, vaultID); vaultErr != nil {
			return nil, vaultErr
		}
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

// PushSecret writes the value to the entry the remote key addresses, creating
// that entry when it does not exist yet.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if secret == nil {
		return errors.New("secret is required for DVLS push")
	}

	value, err := extractPushValue(secret, data)
	if err != nil {
		return err
	}

	// resolveRef reports a missing entry only once the vault is resolved, so
	// vaultID is set on that path and the entry can be created in it.
	vaultID, entryID, err := c.resolveRef(ctx, data.GetRemoteKey())
	switch {
	case isNotFoundError(err):
		return c.createEntry(ctx, vaultID, c.entryRef(data.GetRemoteKey()), value)
	case err != nil:
		return err
	}

	existingEntry, err := c.cred.GetByID(ctx, vaultID, entryID)
	if isVaultNotFoundError(err) {
		return fmt.Errorf(errVaultNotFound, vaultID, err)
	}
	if isNotFoundError(err) {
		if vaultErr := c.confirmVault(ctx, vaultID); vaultErr != nil {
			return vaultErr
		}
		// The reference resolved to an id the vault no longer has, so the name
		// cache is stale. Drop it and create the entry the key asks for.
		ref := c.entryRef(data.GetRemoteKey())
		c.mu.Lock()
		delete(c.nameCache, entryCacheKey(vaultID, ref))
		c.mu.Unlock()
		return c.createEntry(ctx, vaultID, ref, value)
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

// createEntry creates the credential entry a push addressed but that the vault
// does not hold yet. entryRef is the part of the remote key that names the
// entry within its vault: the entry's name and, optionally, the folder path it
// belongs under, because DVLS addresses an entry by name within a path rather
// than by the whole string.
//
// The entry is created as a Credential/AccessCode: that is the subtype whose
// single field GetSecret reads back as "password", so a value pushed here is
// readable by an ExternalSecret that sets no property. Any folder the path
// names is created first, so that the entry lands in a tree that exists.
//
// An entry referenced by UUID is never created: the UUID names one specific
// entry, which no longer exists, so there is nothing to create it as. That
// covers the "<vault-uuid>/<entry-uuid>" keys of a store without a vault too.
func (c *Client) createEntry(ctx context.Context, vaultID, entryRef string, value []byte) error {
	if vaultID == "" {
		return fmt.Errorf("cannot create entry %q: no vault to create it in", entryRef)
	}
	if isUUID(entryRef) {
		return fmt.Errorf("entry %s not found in vault %s: entry must exist before pushing secrets", entryRef, vaultID)
	}

	name, path := parseEntryRef(entryRef)
	if name == "" {
		return errors.New("entry name cannot be empty")
	}
	// A UUID anywhere in the reference is an id, not a name to create: a legacy
	// "<vault-uuid>/<entry-uuid>" key on a store that sets a vault would
	// otherwise become a folder named after the vault with the entry inside it,
	// and the push would report success while the real entry stopped updating.
	for _, segment := range append(strings.Split(path, `\`), name) {
		if isUUID(strings.TrimSpace(segment)) {
			return fmt.Errorf("cannot create entry %q: %q is a UUID, which names an existing vault or entry "+
				"rather than a folder or entry to create; on a store that sets a vault, "+
				"reference an existing entry by its UUID alone", entryRef, segment)
		}
	}

	if err := c.ensureFolderPath(ctx, vaultID, path); err != nil {
		return err
	}

	entry := dvls.Entry{
		VaultId: vaultID,
		Name:    name,
		Path:    path,
		Type:    dvls.EntryCredentialType,
		SubType: dvls.EntryCredentialSubTypeAccessCode,
	}
	// The subtype must be set first: SetCredentialSecret builds the data struct
	// that matches it.
	if err := entry.SetCredentialSecret(string(value)); err != nil {
		return err
	}

	id, err := c.cred.New(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to create entry %q: %w", entryRef, err)
	}

	c.mu.Lock()
	c.nameCache[entryCacheKey(vaultID, entryRef)] = id
	c.mu.Unlock()
	return nil
}

// ensureFolderPath creates every folder the path names that the vault does not
// hold yet. DVLS keeps a folder as an entry of its own and an entry's path is
// only a reference to it, so creating an entry under a path whose folders are
// missing would place it in a tree that is not there. The path is walked from
// the root because each level has to name the parent it sits under.
func (c *Client) ensureFolderPath(ctx context.Context, vaultID, path string) error {
	if path == "" {
		return nil
	}
	if c.folder == nil {
		return fmt.Errorf("cannot create folder path %q: the client has no folder access", path)
	}

	var parent string
	for segment := range strings.SplitSeq(path, `\`) {
		if segment == "" {
			return fmt.Errorf("invalid folder path %q: a path segment cannot be empty", path)
		}

		current := segment
		if parent != "" {
			current = parent + `\` + segment
		}
		if err := c.ensureFolder(ctx, vaultID, segment, parent, current); err != nil {
			return err
		}
		parent = current
	}
	return nil
}

// ensureFolder creates the single folder named name directly under parent,
// unless it is already there. fullPath is where that folder ends up, used for
// the cache key and for error messages.
func (c *Client) ensureFolder(ctx context.Context, vaultID, name, parent, fullPath string) error {
	cacheKey := vaultID + "/" + fullPath
	c.mu.RLock()
	_, known := c.folderCache[cacheKey]
	c.mu.RUnlock()
	if known {
		return nil
	}

	opts := dvls.GetEntriesOptions{Name: &name}
	if parent != "" {
		opts.Path = &parent
	}

	entries, err := c.folder.GetEntries(ctx, vaultID, opts)
	if err != nil {
		// A missing folder yields an empty list, so a not-found here is about the vault.
		if isNotFoundError(err) || isVaultNotFoundError(err) {
			return &vaultError{vaultRef: vaultID, err: err}
		}
		return fmt.Errorf("failed to look up folder %q: %w", fullPath, err)
	}

	// Neither filter narrows the result to exactly this folder: the path filter
	// also matches everything nested below it, so the match has to be checked.
	if !hasFolder(entries, name, fullPath) {
		folder := dvls.Entry{
			VaultId: vaultID,
			Name:    name,
			Path:    parent,
			Type:    dvls.EntryFolderType,
			SubType: dvls.EntryFolderSubTypeFolder,
			Data:    &dvls.EntryFolderData{},
		}
		if _, err := c.folder.New(ctx, folder); err != nil {
			return fmt.Errorf("failed to create folder %q: %w", fullPath, err)
		}
	}

	c.mu.Lock()
	c.folderCache[cacheKey] = struct{}{}
	c.mu.Unlock()
	return nil
}

// hasFolder reports whether these entries contain the folder itself rather than
// something merely near it. The server is asymmetric about Path: creating a
// folder expects Path to name the parent it goes under, but reading one back
// returns the folder's own full path, so a folder "db" created under "prod"
// comes back with Path "prod\\db". Matching on the parent would therefore never
// hit, and every push would try to create folders that are already there.
func hasFolder(entries []dvls.Entry, name, fullPath string) bool {
	for _, e := range entries {
		if e.Name == name && e.Path == fullPath {
			return true
		}
	}
	return false
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
			// Only idempotent once the vault is known to be reachable.
			if vaultErr := c.confirmVault(ctx, vaultID); vaultErr != nil {
				return vaultErr
			}
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
		if vaultErr := c.confirmVault(ctx, vaultID); vaultErr != nil {
			return false, vaultErr
		}
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
	// Without a pinned vault, keys name their own vault and need a vault client to resolve it.
	if !c.pinnedVault && c.vaults == nil {
		return esv1.ValidationResultError, errors.New("DVLS vault client is not initialized")
	}
	return esv1.ValidationResultReady, nil
}

// Close is a no-op for the DVLS client.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// resolveRef resolves a key to a vault ID and entry ID.
// When the store pins a vault, the whole key is an entry reference inside it.
// Otherwise the first separator splits a vault reference from the entry reference:
//
//	"my-vault/db-creds"             → vault "my-vault", entry "db-creds"
//	"my-vault/folder/db-creds"      → vault "my-vault", path "folder", entry "db-creds"
//	"<vault-uuid>/<entry-uuid>"     → both used directly
func (c *Client) resolveRef(ctx context.Context, key string) (vaultID, entryID string, err error) {
	if c.pinnedVault {
		entryID, err = c.resolveEntryRef(ctx, c.vaultID, key)
		return c.vaultID, entryID, err
	}

	vaultRef, entryRef, err := splitVaultRef(key)
	if err != nil {
		return "", "", err
	}
	vaultID, err = c.resolveVault(ctx, vaultRef)
	if err != nil {
		return "", "", err
	}
	entryID, err = c.resolveEntryRef(ctx, vaultID, entryRef)
	return vaultID, entryID, err
}

// entryRef returns the part of a key that names the entry within its vault:
// the whole key when the store pins a vault, and what follows the vault
// reference otherwise. It is only called once resolveRef has accepted the key,
// so splitting it again cannot fail. The reference is trimmed as resolving it
// was: a key that resolves as " <uuid>" must also be created as one, or the
// UUID check is bypassed and each reconcile creates another entry.
func (c *Client) entryRef(key string) string {
	if c.pinnedVault {
		return strings.TrimSpace(key)
	}
	_, ref, _ := splitVaultRef(key)
	return ref
}

// confirmVault reports whether a 404 from the entry endpoint really means the entry is gone.
// The SDK addresses entries as /vault/{vaultID}/entry/{entryID}, so a 404 is ambiguous: an
// unreachable vault looks identical to a deleted entry. Returning nil means the vault is
// reachable and the caller may treat the entry as absent.
func (c *Client) confirmVault(ctx context.Context, vaultID string) error {
	if c.vaults == nil {
		return nil
	}
	if _, err := c.vaults.Get(ctx, vaultID); err != nil {
		return &vaultError{vaultRef: vaultID, err: err}
	}
	return nil
}

// resolveVault resolves a vault reference (name or UUID) to a vault UUID, caching names.
func (c *Client) resolveVault(ctx context.Context, vaultRef string) (string, error) {
	if isUUID(vaultRef) {
		return vaultRef, nil
	}
	if c.vaults == nil {
		return "", fmt.Errorf("cannot resolve vault %q by name: no vault client configured", vaultRef)
	}

	index, err := c.loadVaultIndex(ctx)
	if err != nil {
		return "", err
	}

	id, ok := index[vaultRef]
	if !ok {
		return "", &vaultError{vaultRef: vaultRef, err: dvls.ErrVaultNotFound}
	}
	return id, nil
}

// loadVaultIndex lists every visible vault once and indexes it by name. GetByName would
// enumerate all vaults per lookup, so a single index keeps a multi-vault key set to one call.
// The lock is held across List so concurrent callers share one enumeration.
func (c *Client) loadVaultIndex(ctx context.Context) (map[string]string, error) {
	c.indexMu.Lock()
	defer c.indexMu.Unlock()

	if c.vaultIndex != nil {
		return c.vaultIndex, nil
	}

	vaults, err := c.vaults.List(ctx)
	if err != nil {
		return nil, &vaultError{err: err}
	}

	index := make(map[string]string, len(vaults))
	for _, v := range vaults {
		// DVLS enforces unique vault names; fail loudly rather than pick one arbitrarily.
		if _, dup := index[v.Name]; dup {
			return nil, fmt.Errorf("found multiple vaults named %q; use the vault UUID to select one", v.Name)
		}
		if !isUUID(v.Id) {
			return nil, fmt.Errorf("vault %q has an invalid UUID: %q", v.Name, v.Id)
		}
		index[v.Name] = v.Id
	}

	c.vaultIndex = index
	return index, nil
}

// resolveEntryRef resolves an entry reference to a UUID within the given vault.
// The key can be:
//   - A UUID: used directly.
//   - A name: looked up via GetEntries.
//   - A path/name: "folder/subfolder/entry-name" — path is used to filter.
func (c *Client) resolveEntryRef(ctx context.Context, vaultID, key string) (entryID string, err error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("entry reference cannot be empty")
	}

	// UUID passes through directly.
	if isUUID(key) {
		return key, nil
	}

	// Return cached result if available. Scoped by vault: one client can serve several.
	cacheKey := entryCacheKey(vaultID, key)
	c.mu.RLock()
	id, ok := c.nameCache[cacheKey]
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

	entries, err := c.cred.GetEntries(ctx, vaultID, opts)
	if err != nil {
		// A missing entry yields an empty list, so a not-found here is about the vault.
		if isNotFoundError(err) || isVaultNotFoundError(err) {
			return "", &vaultError{vaultRef: vaultID, err: err}
		}
		return "", fmt.Errorf("failed to resolve entry %q: %w", key, err)
	}

	switch len(entries) {
	case 0:
		return "", fmt.Errorf("entry %q not found in vault: %w", key, dvls.ErrEntryNotFound)
	case 1:
		c.mu.Lock()
		c.nameCache[cacheKey] = entries[0].Id
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

// entryCacheKey is the nameCache key for an entry reference within a vault. A
// push that creates an entry records it under the same key that resolving the
// reference looks up, so a later push in the same reconcile finds it.
func entryCacheKey(vaultID, key string) string {
	return vaultID + "/" + strings.TrimSpace(key)
}

// splitVaultRef splits "<vault-ref>/<entry-ref>" on the first separator. Both forward
// slashes and backslashes are accepted, matching parseEntryRef.
// Empty segments are rejected, and a UUID entry segment must be the whole entry reference,
// so keys that failed validation before vault names were supported keep failing.
func splitVaultRef(key string) (vaultRef, entryRef string, err error) {
	key = strings.TrimSpace(key)
	idx := strings.IndexAny(key, `/\`)
	if idx < 0 {
		return "", "", fmt.Errorf("invalid key format: expected '<vault>/<entry>', got %q", key)
	}

	vaultRef = strings.TrimSpace(key[:idx])
	entryRef = strings.TrimSpace(key[idx+1:])

	if vaultRef == "" {
		return "", "", errors.New("vault reference cannot be empty")
	}
	if entryRef == "" {
		return "", "", errors.New("entry reference cannot be empty")
	}

	segments := strings.Split(strings.ReplaceAll(entryRef, "/", `\`), `\`)
	for _, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			return "", "", fmt.Errorf("invalid key format: empty path segment in %q", key)
		}
	}
	if len(segments) > 1 && isUUID(strings.TrimSpace(segments[0])) {
		return "", "", fmt.Errorf("invalid key format: entry UUID %q must be the whole entry reference", segments[0])
	}

	return vaultRef, entryRef, nil
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
	if !isUUID(vault.Id) {
		return "", fmt.Errorf("vault %q resolved to an invalid UUID: %q", vaultRef, vault.Id)
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
	if path, name, found := strings.CutLast(normalized, `\`); found {
		return name, path
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

	// A vault-stage failure is never a missing secret, even when the cause is a 404.
	if _, ok := errors.AsType[*vaultError](err); ok {
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
