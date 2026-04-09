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
		for _, f := range cipher.Fields {
			fieldName, err := crypto.DecryptString(f.Name, symEncKey, symMacKey)
			if err != nil {
				continue
			}
			if fieldName == ref.Property {
				val, err := crypto.DecryptString(f.Value, symEncKey, symMacKey)
				if err != nil {
					return nil, fmt.Errorf("vaultwarden: decrypting field value: %w", err)
				}
				return []byte(val), nil
			}
		}
		return nil, fmt.Errorf("vaultwarden: field %q not found in secret %q", ref.Property, ref.Key)
	}

	// SecureNote: return notes.
	if cipher.Type == 2 {
		val, err := crypto.DecryptString(cipher.Notes, symEncKey, symMacKey)
		if err != nil {
			return nil, fmt.Errorf("vaultwarden: decrypting notes: %w", err)
		}
		return []byte(val), nil
	}

	// Login: return password.
	if cipher.Login != nil {
		val, err := crypto.DecryptString(cipher.Login.Password, symEncKey, symMacKey)
		if err != nil {
			return nil, fmt.Errorf("vaultwarden: decrypting password: %w", err)
		}
		return []byte(val), nil
	}

	return nil, fmt.Errorf("vaultwarden: secret %q has no usable value", ref.Key)
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

	var nameRe *regexp.Regexp
	if ref.Name != nil && ref.Name.RegExp != "" {
		nameRe, err = regexp.Compile(ref.Name.RegExp)
		if err != nil {
			return nil, fmt.Errorf("vaultwarden: invalid name regexp %q: %w", ref.Name.RegExp, err)
		}
	}

	out := make(map[string][]byte)
	for _, cipher := range ciphers {
		if cipher.DeletedDate != nil {
			continue
		}
		name, err := crypto.DecryptString(cipher.Name, symEncKey, symMacKey)
		if err != nil {
			continue
		}
		if nameRe != nil && !nameRe.MatchString(name) {
			continue
		}

		var val string
		if cipher.Type == 2 {
			val, err = crypto.DecryptString(cipher.Notes, symEncKey, symMacKey)
		} else if cipher.Login != nil {
			val, err = crypto.DecryptString(cipher.Login.Password, symEncKey, symMacKey)
		}
		if err != nil {
			continue
		}
		out[name] = []byte(val)
	}
	return out, nil
}

// PushSecret writes a secret value to Vaultwarden as a SecureNote cipher.
// It creates the cipher if it doesn't exist, or updates it if it does.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	accessToken, symEncKey, symMacKey, err := c.getSymKey(ctx)
	if err != nil {
		return err
	}

	// Determine the value to push.
	var value []byte
	if key := data.GetSecretKey(); key != "" {
		val, ok := secret.Data[key]
		if !ok {
			return fmt.Errorf("vaultwarden: key %q not found in secret %s/%s", key, secret.Namespace, secret.Name)
		}
		value = val
	} else {
		value, err = json.Marshal(secret.Data)
		if err != nil {
			return fmt.Errorf("vaultwarden: marshalling secret data: %w", err)
		}
	}

	cipherName := data.GetRemoteKey()

	// Encrypt name and notes.
	encName, err := crypto.EncryptString(cipherName, symEncKey, symMacKey)
	if err != nil {
		return fmt.Errorf("vaultwarden: encrypting cipher name: %w", err)
	}
	encNotes, err := crypto.EncryptString(string(value), symEncKey, symMacKey)
	if err != nil {
		return fmt.Errorf("vaultwarden: encrypting cipher notes: %w", err)
	}

	body := cipherCreateBody{
		Type:           2,
		Name:           encName,
		Notes:          encNotes,
		SecureNote:     &secureNoteObj{Type: 0},
		FolderID:       nil,
		OrganizationID: nil,
	}

	// Check if a cipher with this name already exists.
	ciphers, err := c.listCiphersWithToken(ctx, accessToken)
	if err != nil {
		return err
	}

	existing := findCipherByNameNoErr(ciphers, cipherName, symEncKey, symMacKey)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("vaultwarden: marshalling cipher body: %w", err)
	}

	base := strings.TrimRight(c.provider.URL, "/")

	if existing != nil {
		// Update existing cipher.
		putURL := fmt.Sprintf("%s/api/ciphers/%s", base, existing.ID)
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, putURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("vaultwarden: building update request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("vaultwarden: updating cipher: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("vaultwarden: update cipher returned HTTP %d: %s", resp.StatusCode, body)
		}
		return nil
	}

	// Create new cipher.
	postURL := base + "/api/ciphers"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("vaultwarden: building create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vaultwarden: creating cipher: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vaultwarden: create cipher returned HTTP %d: %s", resp.StatusCode, body)
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
	req.Header.Set("Authorization", "Bearer "+accessToken)

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

// listCiphers fetches a fresh token and then retrieves all ciphers.
func (c *Client) listCiphers(ctx context.Context) ([]vaultwardenCipher, error) {
	tokenResp, err := c.fetchToken(ctx)
	if err != nil {
		return nil, err
	}
	return c.listCiphersWithToken(ctx, tokenResp.AccessToken)
}

// listCiphersWithToken retrieves all ciphers using the provided bearer token.
func (c *Client) listCiphersWithToken(ctx context.Context, accessToken string) ([]vaultwardenCipher, error) {
	ciphersURL := strings.TrimRight(c.provider.URL, "/") + "/api/ciphers"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ciphersURL, nil)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: building ciphers request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

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
