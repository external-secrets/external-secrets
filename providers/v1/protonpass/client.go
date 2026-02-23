/*
Copyright © 2026 ESO Maintainer Team

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
	"strings"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	defaultField = "password"
)

// errPushSecretNotSupported is returned when attempting to push secrets to Proton Pass.
var errPushSecretNotSupported = errors.New("push secret is not supported for Proton Pass provider")

// errDeleteSecretNotSupported is returned when attempting to delete secrets from Proton Pass.
var errDeleteSecretNotSupported = errors.New("delete secret is not supported for Proton Pass provider")

// errSecretExistsNotSupported is returned when attempting to test for secret existence in Proton Pass.
var errSecretExistsNotSupported = errors.New("secret exists is not supported for Proton Pass provider")

// GetSecret retrieves a single secret from Proton Pass.
// Key format: "itemName" or "itemName/fieldName"
// Property overrides field name if specified.
func (p *provider) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	itemName, fieldName := parseKey(ref.Key)

	// Property overrides field name
	if ref.Property != "" {
		fieldName = ref.Property
	}

	// Default to password field
	if fieldName == "" {
		fieldName = defaultField
	}

	itemID, err := p.cli.ResolveItemID(ctx, itemName)
	if err != nil {
		return nil, err
	}

	item, err := p.cli.GetItem(ctx, itemID)
	if err != nil {
		return nil, err
	}

	resolved := resolveFieldName(fieldName)
	for k, v := range item.Fields() {
		if strings.EqualFold(k, resolved) {
			return []byte(v), nil
		}
	}

	return nil, fmt.Errorf("%w: %s", errFieldNotFound, fieldName)
}

// GetSecretMap retrieves all fields from a Proton Pass item.
func (p *provider) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	itemName, _ := parseKey(ref.Key)

	itemID, err := p.cli.ResolveItemID(ctx, itemName)
	if err != nil {
		return nil, err
	}

	item, err := p.cli.GetItem(ctx, itemID)
	if err != nil {
		return nil, err
	}

	fields := item.Fields()
	secretData := make(map[string][]byte, len(fields))
	for k, v := range fields {
		secretData[k] = []byte(v)
	}

	return secretData, nil
}

// GetAllSecrets retrieves all secrets from the configured vault.
func (p *provider) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	items, err := p.cli.ListItems(ctx)
	if err != nil {
		return nil, err
	}

	secretData := make(map[string][]byte)

	for _, item := range items {
		// Check if item matches the find criteria
		matches, err := matchesFind(item, ref)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}

		// Get the full item details
		details, err := p.cli.GetItem(ctx, item.ID)
		if err != nil {
			// Skip items we can't access
			continue
		}

		// Use password as the default value (for login items)
		if login := details.Content.Content.Login; login != nil && login.Password != "" {
			secretData[item.Content.Title] = []byte(login.Password)
		}
	}

	return secretData, nil
}

// Close cleans up the provider resources.
// Sessions are kept on disk so that subsequent reconciliations for the
// same store can reuse the existing session without logging in again
// (which would fail with TOTP since codes are single-use per window).
// When session caching is enabled, the in-memory LRU cache manages
// logout and directory cleanup on eviction.
func (p *provider) Close(_ context.Context) error {
	return nil
}

// Validate validates the provider configuration.
func (p *provider) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

// PushSecret is not supported for Proton Pass.
func (p *provider) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errPushSecretNotSupported
}

// DeleteSecret is not supported for Proton Pass.
func (p *provider) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errDeleteSecretNotSupported
}

// SecretExists is not supported for Proton Pass.
func (p *provider) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errSecretExistsNotSupported
}

// fieldAliases maps alternative field names to their canonical names.
var fieldAliases = map[string]string{
	"notes": "note",
	"totp":  "totpSecret",
}

// resolveFieldName maps alias field names to their canonical names.
func resolveFieldName(name string) string {
	if canonical, ok := fieldAliases[strings.ToLower(name)]; ok {
		return canonical
	}
	return name
}

// parseKey parses a key in the format "itemName" or "itemName/fieldName".
func parseKey(key string) (itemName, fieldName string) {
	parts := strings.SplitN(key, "/", 2)
	itemName = parts[0]
	if len(parts) > 1 {
		fieldName = parts[1]
	}
	return itemName, fieldName
}

// matchesFind checks if an item matches the find criteria.
func matchesFind(item item, ref esv1.ExternalSecretFind) (bool, error) {
	// If no filter is specified, match all
	if ref.Name == nil && len(ref.Tags) == 0 && ref.Path == nil {
		return true, nil
	}

	itemName := item.Content.Title

	// Match by name pattern
	if ref.Name != nil && ref.Name.RegExp != "" {
		matched, err := regexp.MatchString(ref.Name.RegExp, itemName)
		if err != nil {
			return false, fmt.Errorf("invalid regexp %q: %w", ref.Name.RegExp, err)
		}
		if !matched {
			return false, nil
		}
	}

	// Path matching (not directly applicable to Proton Pass)
	if ref.Path != nil {
		// Treat path as an item name prefix
		if !strings.HasPrefix(itemName, *ref.Path) {
			return false, nil
		}
	}

	return true, nil
}
