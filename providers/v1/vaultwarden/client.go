//go:build vaultwarden || all_providers

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

package vaultwarden

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/vaultwarden/internal/crypto"
)

// vaultwardenCipher represents a vault item returned from /api/ciphers.
type vaultwardenCipher struct {
	ID          string        `json:"id"`
	Type        int           `json:"type"`        // 1=Login, 2=SecureNote
	Name        string        `json:"name"`        // EncString
	Notes       string        `json:"notes"`       // EncString (may be empty)
	Login       *cipherLogin  `json:"login"`
	Fields      []cipherField `json:"fields"`
	DeletedDate interface{}   `json:"deletedDate"`
}

type cipherLogin struct {
	Password string `json:"password"` // EncString
	Username string `json:"username"` // EncString
}

type cipherField struct {
	Name  string `json:"name"`  // EncString
	Value string `json:"value"` // EncString
	Type  int    `json:"type"`  // 0=text
}

type ciphersListResponse struct {
	Data   []vaultwardenCipher `json:"data"`
	Object string              `json:"object"`
}

// cipherCreateBody is the JSON body for POST /api/ciphers and PUT /api/ciphers/{id}.
type cipherCreateBody struct {
	Type           int            `json:"type"`
	Name           string         `json:"name"`
	Notes          string         `json:"notes"`
	SecureNote     *secureNoteObj `json:"secureNote,omitempty"`
	FolderID       interface{}    `json:"folderId"`
	OrganizationID interface{}    `json:"organizationId"`
}

type secureNoteObj struct {
	Type int `json:"type"` // 0
}

// cachedToken holds a short-lived access token and symmetric key material
// so we don't re-authenticate on every API call within the same reconciliation.
type cachedToken struct {
	accessToken string
	symEncKey   []byte
	symMacKey   []byte
	expiresAt   time.Time
}

// Client implements esv1.SecretsClient for Vaultwarden.
type Client struct {
	httpClient *http.Client
	provider   *esv1.VaultwardenProvider
	crClient   client.Client
	namespace  string
	store      esv1.GenericStore

	mu    sync.Mutex
	cache *cachedToken
}

var _ esv1.SecretsClient = &Client{}

// bearerToken returns the Authorization header value for a given access token.
const bearerPrefix = "Bearer "

// Close is a no-op; the HTTP client is reusable and has no per-session state.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// Validate checks whether the client can authenticate and list ciphers.
// ClusterSecretStore cannot be validated (returns Unknown).
func (c *Client) Validate() (esv1.ValidationResult, error) {
	if c.store.GetObjectKind().GroupVersionKind().Kind == esv1.ClusterSecretStoreKind {
		return esv1.ValidationResultUnknown, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Use getSymKey so we benefit from the token cache.
	accessToken, _, _, err := c.getSymKey(ctx)
	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf("vaultwarden: validate: %w", err)
	}
	if _, err := c.listCiphersWithToken(ctx, accessToken); err != nil {
		return esv1.ValidationResultError, fmt.Errorf("vaultwarden: validate list: %w", err)
	}
	return esv1.ValidationResultReady, nil
}

// GetSecret returns a single secret value by cipher name (ref.Key).
// If ref.Property is set it looks up a named custom field; otherwise it returns
// the decrypted notes (SecureNote) or password (Login).
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	accessToken, symEncKey, symMacKey, err := c.getSymKey(ctx)
	if err != nil {
		return nil, err
	}
	ciphers, err := c.listCiphersWithToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	cipher, err := findCipherByName(ciphers, ref.Key, symEncKey, symMacKey)
	if err != nil {
		return nil, err
	}
	if ref.Property != "" {
		return getCipherProperty(cipher, ref.Property, symEncKey, symMacKey)
	}
	return getCipherValue(cipher, ref.Key, symEncKey, symMacKey)
}

// getCipherProperty looks up a named property from a cipher's custom fields,
// falling back to the cipher's Notes JSON for secrets written by this provider.
func getCipherProperty(cipher *vaultwardenCipher, property string, symEncKey, symMacKey []byte) ([]byte, error) {
	for _, f := range cipher.Fields {
		name, err := crypto.DecryptString(f.Name, symEncKey, symMacKey)
		if err != nil {
			continue
		}
		if name == property {
			val, err := crypto.DecryptString(f.Value, symEncKey, symMacKey)
			if err != nil {
				return nil, fmt.Errorf("vaultwarden: decrypting field value: %w", err)
			}
			return []byte(val), nil
		}
	}
	if v := getCipherPropertyFromNotes(cipher, property, symEncKey, symMacKey); v != nil {
		return v, nil
	}
	return nil, fmt.Errorf("vaultwarden: field %q not found in secret %q", property, cipher.Name)
}

// getCipherPropertyFromNotes attempts to extract a property from Notes JSON.
func getCipherPropertyFromNotes(cipher *vaultwardenCipher, property string, symEncKey, symMacKey []byte) []byte {
	if cipher.Notes == "" {
		return nil
	}
	notes, err := crypto.DecryptString(cipher.Notes, symEncKey, symMacKey)
	if err != nil {
		return nil
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal([]byte(notes), &obj) != nil {
		return nil
	}
	if raw, ok := obj[property]; ok {
		return jsonRawToBytes(raw)
	}
	return nil
}

// getCipherValue returns the primary value of a cipher:
// Notes for SecureNote (type 2), password for Login (type 1).
func getCipherValue(cipher *vaultwardenCipher, key string, symEncKey, symMacKey []byte) ([]byte, error) {
	if cipher.Type == 2 {
		val, err := crypto.DecryptString(cipher.Notes, symEncKey, symMacKey)
		if err != nil {
			return nil, fmt.Errorf("vaultwarden: decrypting notes: %w", err)
		}
		return []byte(val), nil
	}
	if cipher.Login != nil {
		val, err := crypto.DecryptString(cipher.Login.Password, symEncKey, symMacKey)
		if err != nil {
			return nil, fmt.Errorf("vaultwarden: decrypting password: %w", err)
		}
		return []byte(val), nil
	}
	return nil, fmt.Errorf("vaultwarden: secret %q has no usable value", key)
}

// GetSecretMap returns a map of keys to values by parsing the cipher's decrypted notes as JSON.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	accessToken, symEncKey, symMacKey, err := c.getSymKey(ctx)
	if err != nil {
		return nil, err
	}

	ciphers, err := c.listCiphersWithToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	cipher, err := findCipherByName(ciphers, ref.Key, symEncKey, symMacKey)
	if err != nil {
		return nil, err
	}

	if cipher.Notes == "" {
		return nil, fmt.Errorf("vaultwarden: secret %q has empty notes; cannot produce map", ref.Key)
	}

	notes, err := crypto.DecryptString(cipher.Notes, symEncKey, symMacKey)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: decrypting notes for map: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(notes), &raw); err != nil {
		return nil, fmt.Errorf("vaultwarden: notes for %q are not valid JSON: %w", ref.Key, err)
	}

	out := make(map[string][]byte, len(raw))
	for k, v := range raw {
		out[k] = jsonRawToBytes(v)
	}
	return out, nil
}

// GetAllSecrets returns all ciphers (optionally filtered by name regexp) as a name→value map.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	accessToken, symEncKey, symMacKey, err := c.getSymKey(ctx)
	if err != nil {
		return nil, err
	}
	ciphers, err := c.listCiphersWithToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	nameRe, err := compileNameRegexp(ref)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte)
	for i := range ciphers {
		name, val, ok := decryptCipherEntry(&ciphers[i], nameRe, symEncKey, symMacKey)
		if ok {
			out[name] = val
		}
	}
	return out, nil
}

// compileNameRegexp returns a compiled regexp from the find ref, or nil if none is set.
func compileNameRegexp(ref esv1.ExternalSecretFind) (*regexp.Regexp, error) {
	if ref.Name == nil || ref.Name.RegExp == "" {
		return nil, nil
	}
	re, err := regexp.Compile(ref.Name.RegExp)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: invalid name regexp %q: %w", ref.Name.RegExp, err)
	}
	return re, nil
}

// decryptCipherEntry decrypts a cipher's name and value, applying the optional name filter.
// Returns (name, value, true) on success or ("", nil, false) if the entry should be skipped.
func decryptCipherEntry(cipher *vaultwardenCipher, nameRe *regexp.Regexp, symEncKey, symMacKey []byte) (string, []byte, bool) {
	if cipher.DeletedDate != nil {
		return "", nil, false
	}
	name, err := crypto.DecryptString(cipher.Name, symEncKey, symMacKey)
	if err != nil || (nameRe != nil && !nameRe.MatchString(name)) {
		return "", nil, false
	}
	val, err := decryptCipherPrimaryValue(cipher, symEncKey, symMacKey)
	if err != nil {
		return "", nil, false
	}
	return name, []byte(val), true
}

// decryptCipherPrimaryValue returns the primary plaintext value of a cipher.
func decryptCipherPrimaryValue(cipher *vaultwardenCipher, symEncKey, symMacKey []byte) (string, error) {
	if cipher.Type == 2 {
		return crypto.DecryptString(cipher.Notes, symEncKey, symMacKey)
	}
	if cipher.Login != nil {
		return crypto.DecryptString(cipher.Login.Password, symEncKey, symMacKey)
	}
	return "", nil
}

// PushSecret writes a secret value to Vaultwarden as a SecureNote cipher.
// It creates the cipher if it doesn't exist, or updates it if it does.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	accessToken, symEncKey, symMacKey, err := c.getSymKey(ctx)
	if err != nil {
		return err
	}
	value, err := buildPushValue(secret, data)
	if err != nil {
		return err
	}
	cipherName := data.GetRemoteKey()
	bodyBytes, err := c.buildCipherBody(cipherName, string(value), symEncKey, symMacKey)
	if err != nil {
		return err
	}
	ciphers, err := c.listCiphersWithToken(ctx, accessToken)
	if err != nil {
		return err
	}
	existing := findCipherByNameNoErr(ciphers, cipherName, symEncKey, symMacKey)
	return c.upsertCipher(ctx, accessToken, existing, bodyBytes)
}

// buildPushValue extracts the value to push from the Kubernetes secret.
// If a specific key is requested, that key's raw bytes are returned.
// Otherwise all data keys are JSON-marshalled as a string map.
func buildPushValue(secret *corev1.Secret, data esv1.PushSecretData) ([]byte, error) {
	if key := data.GetSecretKey(); key != "" {
		val, ok := secret.Data[key]
		if !ok {
			return nil, fmt.Errorf("vaultwarden: key %q not found in secret %s/%s", key, secret.Namespace, secret.Name)
		}
		return val, nil
	}
	// Convert []byte values to strings so json.Marshal does not base64-encode them,
	// preserving round-trip fidelity with GetSecretMap.
	strData := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		strData[k] = string(v)
	}
	b, err := json.Marshal(strData)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: marshalling secret data: %w", err)
	}
	return b, nil
}

// buildCipherBody encrypts the cipher name and notes and serialises the body.
func (c *Client) buildCipherBody(name, notes string, symEncKey, symMacKey []byte) ([]byte, error) {
	encName, err := crypto.EncryptString(name, symEncKey, symMacKey)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: encrypting cipher name: %w", err)
	}
	encNotes, err := crypto.EncryptString(notes, symEncKey, symMacKey)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: encrypting cipher notes: %w", err)
	}
	b, err := json.Marshal(cipherCreateBody{
		Type:       2,
		Name:       encName,
		Notes:      encNotes,
		SecureNote: &secureNoteObj{Type: 0},
	})
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: marshalling cipher body: %w", err)
	}
	return b, nil
}

// upsertCipher sends a PUT (update) or POST (create) request depending on whether existing is set.
func (c *Client) upsertCipher(ctx context.Context, accessToken string, existing *vaultwardenCipher, bodyBytes []byte) error {
	base := strings.TrimRight(c.provider.URL, "/")
	var method, url string
	if existing != nil {
		method = http.MethodPut
		url = fmt.Sprintf("%s/api/ciphers/%s", base, existing.ID)
	} else {
		method = http.MethodPost
		url = base + "/api/ciphers"
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("vaultwarden: building cipher request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerPrefix+accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vaultwarden: cipher request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vaultwarden: cipher request returned HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

// DeleteSecret deletes the cipher matching the remote ref's key. Idempotent — returns nil if not found.
func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	accessToken, symEncKey, symMacKey, err := c.getSymKey(ctx)
	if err != nil {
		return err
	}

	ciphers, err := c.listCiphersWithToken(ctx, accessToken)
	if err != nil {
		return err
	}

	cipher := findCipherByNameNoErr(ciphers, remoteRef.GetRemoteKey(), symEncKey, symMacKey)
	if cipher == nil {
		// Not found — idempotent delete.
		return nil
	}

	base := strings.TrimRight(c.provider.URL, "/")
	deleteURL := fmt.Sprintf("%s/api/ciphers/%s", base, cipher.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return fmt.Errorf("vaultwarden: building delete request: %w", err)
	}
	req.Header.Set("Authorization", bearerPrefix+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vaultwarden: deleting cipher: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vaultwarden: delete cipher returned HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

// SecretExists returns true if a cipher with the given remote key name exists in the vault.
func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	accessToken, symEncKey, symMacKey, err := c.getSymKey(ctx)
	if err != nil {
		return false, err
	}

	ciphers, err := c.listCiphersWithToken(ctx, accessToken)
	if err != nil {
		return false, err
	}

	cipher := findCipherByNameNoErr(ciphers, remoteRef.GetRemoteKey(), symEncKey, symMacKey)
	return cipher != nil, nil
}

// listCiphersWithToken retrieves all ciphers using the provided bearer token.
func (c *Client) listCiphersWithToken(ctx context.Context, accessToken string) ([]vaultwardenCipher, error) {
	ciphersURL := strings.TrimRight(c.provider.URL, "/") + "/api/ciphers"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ciphersURL, nil)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: building ciphers request: %w", err)
	}
	req.Header.Set("Authorization", bearerPrefix+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: ciphers request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vaultwarden: ciphers endpoint returned HTTP %d", resp.StatusCode)
	}

	var list ciphersListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("vaultwarden: decoding ciphers response: %w", err)
	}
	return list.Data, nil
}

// findCipherByName returns the first cipher whose decrypted name equals target.
// Returns an error if not found.
func findCipherByName(ciphers []vaultwardenCipher, target string, symEncKey, symMacKey []byte) (*vaultwardenCipher, error) {
	c := findCipherByNameNoErr(ciphers, target, symEncKey, symMacKey)
	if c == nil {
		return nil, fmt.Errorf("vaultwarden: secret %q not found", target)
	}
	return c, nil
}

// findCipherByNameNoErr returns the first cipher whose decrypted name equals target, or nil.
func findCipherByNameNoErr(ciphers []vaultwardenCipher, target string, symEncKey, symMacKey []byte) *vaultwardenCipher {
	for i := range ciphers {
		if ciphers[i].DeletedDate != nil {
			continue
		}
		name, err := crypto.DecryptString(ciphers[i].Name, symEncKey, symMacKey)
		if err != nil {
			continue
		}
		if name == target {
			return &ciphers[i]
		}
	}
	return nil
}

// jsonRawToBytes converts a json.RawMessage to []byte.
// For JSON strings the surrounding quotes are stripped; other types are returned as-is.
func jsonRawToBytes(raw json.RawMessage) []byte {
	if len(raw) > 1 && raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return []byte(s)
		}
	}
	return []byte(raw)
}
