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

package protonpass

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	idPrefix        = "id:"
	defaultProperty = "password"
)

// client implements esv1.SecretsClient for Proton Pass.
type client struct {
	api    *apiClient
	vaults []string // optional allow-list of vault names; empty means all readable vaults
}

var _ esv1.SecretsClient = &client{}

// vaultCtx is a resolved, decryptable vault (share) the token can access.
type vaultCtx struct {
	share apiShare
	keys  map[uint8][]byte
	name  string
}

// scopedVaults lists the vaults this store may use, decrypts their keys and names,
// and applies the optional allow-list. Group-shared vaults are skipped: their
// share key is PGP-wrapped and a PAT cannot decrypt it (so we cannot even read
// their name).
func (c *client) scopedVaults(ctx context.Context) ([]vaultCtx, error) {
	shares, err := c.api.listShares(ctx)
	if err != nil {
		return nil, err
	}
	var out []vaultCtx
	for _, sh := range shares {
		if sh.isGroupShared() {
			continue
		}
		keys, err := c.api.openShareKeys(ctx, sh.ShareID)
		if err != nil {
			return nil, err
		}
		name, err := c.api.openVaultName(sh, keys)
		if err != nil {
			return nil, err
		}
		if len(c.vaults) > 0 && !slices.Contains(c.vaults, name) {
			continue
		}
		out = append(out, vaultCtx{share: sh, keys: keys, name: name})
	}
	return out, nil
}

// match is a resolved item plus the vault it lives in.
type match struct {
	vault vaultCtx
	rev   apiItemRevision
	item  *decodedItem
}

// resolveItem finds the single item addressed by key (an item title, or "id:<ItemID>")
// within the scoped vaults. Ambiguous titles are a hard error (never a silent pick).
func (c *client) resolveItem(ctx context.Context, key string) (match, error) {
	vaults, err := c.scopedVaults(ctx)
	if err != nil {
		return match{}, err
	}
	if id, ok := strings.CutPrefix(key, idPrefix); ok {
		return c.resolveByID(ctx, vaults, id)
	}
	var matches []match
	for _, v := range vaults {
		found, err := c.matchesByTitle(ctx, v, key)
		if err != nil {
			return match{}, err
		}
		matches = append(matches, found...)
	}
	switch len(matches) {
	case 0:
		return match{}, errNotFound
	case 1:
		return matches[0], nil
	default:
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, m.rev.ItemID)
		}
		return match{}, fmt.Errorf("protonpass: %q is ambiguous across %d items; address it with id:<ItemID> (candidates: %s)",
			key, len(matches), strings.Join(ids, ", "))
	}
}

// resolveByID returns the active item with the given ItemID across the scoped vaults.
func (c *client) resolveByID(ctx context.Context, vaults []vaultCtx, id string) (match, error) {
	for _, v := range vaults {
		revs, err := c.api.listItems(ctx, v.share.ShareID)
		if err != nil {
			return match{}, err
		}
		for _, rev := range revs {
			if rev.State != itemStateActive || rev.ItemID != id {
				continue
			}
			item, err := c.api.openItem(rev, v.keys)
			if err != nil {
				return match{}, err
			}
			return match{vault: v, rev: rev, item: item}, nil
		}
	}
	return match{}, errNotFound
}

// matchesByTitle returns every active item in a vault whose title equals title.
// Items that fail to decrypt (e.g. corrupt content or an unknown key rotation)
// are skipped, matching Proton's own batch-list behavior; the explicit id: path
// (resolveByID) instead surfaces such a failure, matching its single-item get.
func (c *client) matchesByTitle(ctx context.Context, v vaultCtx, title string) ([]match, error) {
	revs, err := c.api.listItems(ctx, v.share.ShareID)
	if err != nil {
		return nil, err
	}
	var out []match
	for _, rev := range revs {
		if rev.State != itemStateActive {
			continue
		}
		item, err := c.api.openItem(rev, v.keys)
		if err != nil {
			continue
		}
		if item.name == title {
			out = append(out, match{vault: v, rev: rev, item: item})
		}
	}
	return out, nil
}

// GetSecret returns one field of one item.
func (c *client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	m, err := c.resolveItem(ctx, ref.Key)
	if err != nil {
		return nil, asNoSecret(err)
	}
	prop := ref.Property
	if prop == "" {
		prop = defaultProperty
	}
	if prop == fieldKeyTOTP {
		code, err := generateTOTPCode(loginTOTPURI(m.item.plaintext))
		if err != nil {
			return nil, asNoSecret(err)
		}
		return []byte(code), nil
	}
	// Proton Pass fields are a flat label->value set, so a property is a direct
	// field-label lookup (a label may contain characters like '.' that a path
	// extractor would misread).
	fields, err := projectItem(m.item.plaintext)
	if err != nil {
		return nil, err
	}
	v, ok := fields[prop]
	if !ok {
		// The item exists but the requested property does not. This is a usage
		// error, not an absent secret: returning NoSecretErr here would let
		// deletionPolicy=Delete remove the target Secret over a typo'd property.
		return nil, fmt.Errorf("protonpass: item %q has no property %q", ref.Key, prop)
	}
	return v, nil
}

// GetSecretMap returns every field of one item as a key/value map.
func (c *client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	m, err := c.resolveItem(ctx, ref.Key)
	if err != nil {
		return nil, asNoSecret(err)
	}
	// Keys are raw field labels; the controller applies the user's
	// conversionStrategy (esutils.ConvertKeys) downstream.
	return projectItem(m.item.plaintext)
}

// GetAllSecrets returns the default value of every item whose title matches the
// find criteria. Tags are unsupported (Proton Pass items have none).
func (c *client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if len(ref.Tags) > 0 {
		return nil, errors.New("protonpass: tag-based find is not supported (Proton Pass items have no tags)")
	}
	var re *regexp.Regexp
	if ref.Name != nil && ref.Name.RegExp != "" {
		var err error
		if re, err = regexp.Compile(ref.Name.RegExp); err != nil {
			return nil, fmt.Errorf("protonpass: invalid find name regexp: %w", err)
		}
	}
	vaults, err := c.scopedVaults(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, v := range vaults {
		if ref.Path != nil && v.name != *ref.Path {
			continue
		}
		revs, err := c.api.listItems(ctx, v.share.ShareID)
		if err != nil {
			return nil, err
		}
		for _, rev := range revs {
			if rev.State != itemStateActive {
				continue
			}
			item, err := c.api.openItem(rev, v.keys)
			if err != nil {
				continue
			}
			title := item.name
			if re != nil && !re.MatchString(title) {
				continue
			}
			fields, err := projectItem(item.plaintext)
			if err != nil {
				continue // skip an item we cannot decode rather than failing the whole find
			}
			val, ok := fields[defaultProperty]
			if !ok {
				continue // no default value to surface (e.g. note/identity) — use extract for those
			}
			// Key on the raw title; the controller applies the user's
			// conversionStrategy (esutils.ConvertKeys) downstream.
			if _, dup := out[title]; dup {
				return nil, fmt.Errorf("protonpass: find produced colliding title %q; narrow the query or use rewrite", title)
			}
			out[title] = val
		}
	}
	return out, nil
}

// PushSecret stores a value as a hidden field on an item, creating a custom item
// if none with that title exists. The field label is the push Property (default
// "password"). Requires the store to resolve to exactly one writable vault.
func (c *client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	value, err := pushValue(secret, data)
	if err != nil {
		return err
	}
	label := data.GetProperty()
	if label == "" {
		label = defaultProperty
	}
	v, err := c.singleWritableVault(ctx)
	if err != nil {
		return err
	}
	existing, err := c.findInVault(ctx, v, data.GetRemoteKey())
	if err != nil && !errors.Is(err, errNotFound) {
		return err
	}
	if errors.Is(err, errNotFound) {
		content := marshalCustomItem(data.GetRemoteKey(), "", []writeField{{Label: label, Value: string(value), Hidden: true}})
		return c.api.createItem(ctx, v.share.ShareID, v.keys, content)
	}
	content, err := setExtraField(existing.item.plaintext, label, string(value))
	if err != nil {
		return err
	}
	return c.api.updateItem(ctx, v.share.ShareID, existing.rev, v.keys, content)
}

// DeleteSecret removes a field from an item, or trashes the whole item when no
// property is given.
func (c *client) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) error {
	v, err := c.singleWritableVault(ctx)
	if err != nil {
		return err
	}
	existing, err := c.findInVault(ctx, v, ref.GetRemoteKey())
	if errors.Is(err, errNotFound) {
		return nil // idempotent
	}
	if err != nil {
		return err
	}
	if ref.GetProperty() == "" {
		return c.api.trashItem(ctx, v.share.ShareID, existing.rev.ItemID, existing.rev.Revision)
	}
	content, removed, err := removeExtraField(existing.item.plaintext, ref.GetProperty())
	if err != nil {
		return err
	}
	if !removed {
		return nil // field already absent
	}
	return c.api.updateItem(ctx, v.share.ShareID, existing.rev, v.keys, content)
}

// SecretExists reports whether the addressed item (and property, if set) exists.
func (c *client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	m, err := c.resolveItem(ctx, ref.GetRemoteKey())
	if errors.Is(err, errNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if ref.GetProperty() == "" {
		return true, nil
	}
	fields, err := projectItem(m.item.plaintext)
	if err != nil {
		return false, err
	}
	_, ok := fields[ref.GetProperty()]
	return ok, nil
}

// Validate confirms the token can mint a session and list shares.
func (c *client) Validate() (esv1.ValidationResult, error) {
	if _, err := c.api.listShares(context.Background()); err != nil {
		return esv1.ValidationResultError, err
	}
	return esv1.ValidationResultReady, nil
}

// Close is a no-op; the client holds no long-lived resources.
func (c *client) Close(_ context.Context) error { return nil }

// --- write helpers ---

// singleWritableVault resolves the one vault writes target. PushSecret/DeleteSecret
// require an unambiguous, write-capable destination.
func (c *client) singleWritableVault(ctx context.Context) (vaultCtx, error) {
	vaults, err := c.scopedVaults(ctx)
	if err != nil {
		return vaultCtx{}, err
	}
	writable := vaults[:0]
	for _, v := range vaults {
		if v.share.ShareRoleID == shareRoleViewer {
			continue
		}
		writable = append(writable, v)
	}
	switch len(writable) {
	case 1:
		return writable[0], nil
	case 0:
		return vaultCtx{}, errors.New("protonpass: no writable vault (token has no editor/manager access); writes require an editor or manager PAT")
	default:
		return vaultCtx{}, errors.New("protonpass: multiple writable vaults in scope; set spec.provider.protonpass.vaults to a single vault for writes")
	}
}

const shareRoleViewer = "3" // 1=manager, 2=editor, 3=viewer

// findInVault locates the first active item with the given title in a single vault.
func (c *client) findInVault(ctx context.Context, v vaultCtx, title string) (match, error) {
	matches, err := c.matchesByTitle(ctx, v, title)
	if err != nil {
		return match{}, err
	}
	if len(matches) == 0 {
		return match{}, errNotFound
	}
	return matches[0], nil
}

// pushValue extracts the byte value to push from the source Secret.
func pushValue(secret *corev1.Secret, data esv1.PushSecretData) ([]byte, error) {
	key := data.GetSecretKey()
	if key == "" {
		return nil, errors.New("protonpass: pushing an entire secret is not supported; set spec.data[].match.secretKey")
	}
	v, ok := secret.Data[key]
	if !ok {
		return nil, fmt.Errorf("protonpass: secret key %q not found in source secret", key)
	}
	return v, nil
}

// --- value/key helpers ---

// asNoSecret maps the internal "absent" sentinels (missing item, or no TOTP
// configured) to the ESO contract error so the reconciler can apply deletionPolicy.
func asNoSecret(err error) error {
	if errors.Is(err, errNotFound) || errors.Is(err, errNoTOTP) {
		return esv1.NoSecretErr
	}
	return err
}
