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

// Package client implements the Proton Pass wire protocol: PAT parsing, session
// minting/caching and the read-only share/item endpoints with decryption.
package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/protonpass/internal/codec"
	"github.com/external-secrets/external-secrets/providers/v1/protonpass/internal/crypto"
	"github.com/external-secrets/external-secrets/runtime/cache"
)

const (
	defaultBaseURL = "https://pass-api.proton.me"
	appVersion     = "cli-pass@1.0.0"
	pageSize       = "50"
)

const (
	codeSuccess       = 1000
	codeTooManyLogins = 2028
)

// Interface is the read-only Proton Pass client surface used by the provider.
type Interface interface {
	// GetItem returns the projected key/value map for the item identified by
	// title or "id:<ItemID>".
	GetItem(ctx context.Context, key string) (map[string][]byte, error)
	// Validate mints a session to confirm the PAT is valid.
	Validate(ctx context.Context) error
}

// Share is a Proton Pass vault share.
type Share struct {
	ShareID            string  `json:"ShareID"`
	VaultID            string  `json:"VaultID"`
	TargetType         int     `json:"TargetType"`
	ShareRoleID        string  `json:"ShareRoleID"`
	Content            string  `json:"Content"`
	ContentKeyRotation int     `json:"ContentKeyRotation"`
	GroupID            *string `json:"GroupID"`
	Owner              string  `json:"Owner"`
	Permission         int     `json:"Permission"`
}

// ShareKey is a rotation-sealed share key.
type ShareKey struct {
	KeyRotation int    `json:"KeyRotation"`
	Key         string `json:"Key"`
}

// Item is a single item revision.
type Item struct {
	ItemID      string `json:"ItemID"`
	Revision    int    `json:"Revision"`
	KeyRotation int    `json:"KeyRotation"`
	Content     string `json:"Content"`
	ItemKey     string `json:"ItemKey"`
	State       int    `json:"State"`
	FolderID    string `json:"FolderID"`
}

// Client is a Proton Pass HTTP client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	pat        string
	patKey     []byte
}

type session struct {
	accessToken string
	uid         string
	expiresAt   time.Time
}

type sessionResponse struct {
	Session *sessionPayload `json:"Session"`
	Code    int             `json:"Code"`
}

type sessionPayload struct {
	SessionUID           string    `json:"SessionUID"`
	AccessToken          string    `json:"AccessToken"`
	AccessExpirationTime time.Time `json:"AccessExpirationTime"`
}

// sessionCache caches minted sessions per PAT to avoid the per-account
// "too many recent logins" rate limit (code 2028).
var sessionCache = cache.Must(1024, func(_ *session) {})

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the default Proton Pass API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = strings.TrimSuffix(url, "/") }
}

// WithHTTPClient injects an http.Client (used in tests).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient constructs a Client from a raw PAT string.
func NewClient(pat string, opts ...Option) (*Client, error) {
	_, patKey, err := ParsePAT(pat)
	if err != nil {
		return nil, err
	}
	if len(patKey) != 32 {
		return nil, fmt.Errorf("protonpass: PAT key must be 32 bytes, got %d", len(patKey))
	}

	c := &Client{
		httpClient: &http.Client{},
		baseURL:    defaultBaseURL,
		pat:        pat,
		patKey:     patKey,
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// ParsePAT splits a PAT into its API token and 32-byte AES key halves.
func ParsePAT(pat string) (string, []byte, error) {
	parts := strings.SplitN(pat, "::", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("protonpass: invalid PAT format")
	}
	key, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, fmt.Errorf("protonpass: invalid PAT key: %w", err)
	}
	return parts[0], key, nil
}

func (c *Client) sessionKey() cache.Key {
	return cache.Key{Name: c.pat}
}

func (c *Client) getSession(ctx context.Context) (*session, error) {
	if s, ok := sessionCache.Get("", c.sessionKey()); ok {
		if time.Now().Before(s.expiresAt) {
			return s, nil
		}
	}
	s, err := c.mintSession(ctx)
	if err != nil {
		return nil, err
	}
	sessionCache.Add("", c.sessionKey(), s)
	return s, nil
}

// No public Remove on runtime/cache; the API evicts on version mismatch,
// so Get with a different version evicts.
func (c *Client) invalidateSession() {
	sessionCache.Get("__invalidate__", c.sessionKey())
}

func (c *Client) mintSession(ctx context.Context) (*session, error) {
	body, err := json.Marshal(map[string]string{"Token": c.pat})
	if err != nil {
		return nil, fmt.Errorf("protonpass: failed to marshal session request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/account/v4/personal-access-token/session", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("protonpass: failed to build session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-pm-appversion", appVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("protonpass: session request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("protonpass: failed to read session response: %w", err)
	}

	var sr sessionResponse
	if err := json.Unmarshal(respBody, &sr); err != nil {
		return nil, fmt.Errorf("protonpass: failed to decode session response: %w", err)
	}
	if sr.Code == codeTooManyLogins {
		return nil, fmt.Errorf("protonpass: too many recent logins (code 2028)")
	}
	if sr.Code != codeSuccess {
		return nil, fmt.Errorf("protonpass: session mint failed with code %d", sr.Code)
	}
	if sr.Session == nil {
		return nil, fmt.Errorf("protonpass: session mint returned no session")
	}
	return &session{
		accessToken: sr.Session.AccessToken,
		uid:         sr.Session.SessionUID,
		expiresAt:   sr.Session.AccessExpirationTime,
	}, nil
}

func (c *Client) request(ctx context.Context, method, path string, body []byte, out any) error {
	backoff := time.Second
	for range 5 {
		s, err := c.getSession(ctx)
		if err != nil {
			return err
		}
		code, respBody, err := c.raw(ctx, method, path, body, s)
		if err != nil {
			return err
		}
		switch {
		case code == http.StatusUnauthorized:
			c.invalidateSession()
			continue
		case code == http.StatusTooManyRequests:
			time.Sleep(backoff)
			backoff *= 2
			continue
		case code >= 200 && code < 300:
			if out != nil {
				if err := json.Unmarshal(respBody, out); err != nil {
					return fmt.Errorf("protonpass: failed to decode response: %w", err)
				}
			}
			return nil
		default:
			return fmt.Errorf("protonpass: unexpected status %d: %s", code, strings.TrimSpace(string(respBody)))
		}
	}
	return errors.New("protonpass: request failed after retries")
}

func (c *Client) raw(ctx context.Context, method, path string, body []byte, s *session) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return 0, nil, fmt.Errorf("protonpass: failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("x-pm-uid", s.uid)
	req.Header.Set("x-pm-appversion", appVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("protonpass: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("protonpass: failed to read response: %w", err)
	}
	return resp.StatusCode, respBody, nil
}

// ListShares returns vault shares, skipping group-shared vaults.
func (c *Client) ListShares(ctx context.Context) ([]Share, error) {
	var resp struct {
		Shares []Share `json:"Shares"`
	}
	if err := c.request(ctx, http.MethodGet, "/pass/v1/share", nil, &resp); err != nil {
		return nil, err
	}
	var out []Share
	for _, s := range resp.Shares {
		if s.TargetType == 1 && s.GroupID == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

// GetShareKeys returns the rotation keys for a share.
func (c *Client) GetShareKeys(ctx context.Context, shareID string) ([]ShareKey, error) {
	var resp struct {
		Keys []ShareKey `json:"Keys"`
	}
	if err := c.request(ctx, http.MethodGet, "/pass/v1/share/"+shareID+"/key", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Keys, nil
}

// ListItems returns active items for a share, paginating until LastToken is empty.
func (c *Client) ListItems(ctx context.Context, shareID string) ([]Item, error) {
	var all []Item
	since := ""
	for {
		path := fmt.Sprintf("/pass/v1/share/%s/item?PageSize=%s", shareID, pageSize)
		if since != "" {
			path += "&Since=" + since
		}
		var resp struct {
			Items struct {
				RevisionsData []Item `json:"RevisionsData"`
				LastToken     string `json:"LastToken"`
			} `json:"Items"`
		}
		if err := c.request(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Items.RevisionsData...)
		since = resp.Items.LastToken
		if since == "" {
			break
		}
	}
	var active []Item
	for _, it := range all {
		if it.State == 1 {
			active = append(active, it)
		}
	}
	return active, nil
}

// GetItem resolves an item by title or "id:<ItemID>" and returns its projected
// key/value map.
func (c *Client) GetItem(ctx context.Context, key string) (map[string][]byte, error) {
	isID := strings.HasPrefix(key, "id:")
	id := ""
	if isID {
		id = strings.TrimPrefix(key, "id:")
	}

	shares, err := c.ListShares(ctx)
	if err != nil {
		return nil, err
	}

	// Title is encrypted inside each item's content, so title-based lookup
	// must decrypt every candidate item.
	var byTitle []map[string][]byte
	// Memoize share-key decryption across items on the same share/rotation.
	keyCache := make(map[string]map[int][]byte)
	for _, sh := range shares {
		items, err := c.ListItems(ctx, sh.ShareID)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			rot := it.KeyRotation
			if keyCache[sh.ShareID] == nil {
				keyCache[sh.ShareID] = make(map[int][]byte)
			}
			shareKey, ok := keyCache[sh.ShareID][rot]
			if !ok {
				shareKey, err = c.shareKeyForRotation(ctx, sh.ShareID, rot)
				if err != nil {
					return nil, err
				}
				keyCache[sh.ShareID][rot] = shareKey
			}
			proj, err := c.projectItem(it, shareKey)
			if err != nil {
				return nil, err
			}
			if isID {
				if it.ItemID == id {
					return proj, nil
				}
			} else {
				byTitle = append(byTitle, proj)
			}
		}
	}
	if isID {
		return nil, esv1.NoSecretErr
	}

	var matches []map[string][]byte
	for _, p := range byTitle {
		if string(p["title"]) == key {
			matches = append(matches, p)
		}
	}
	if len(matches) == 0 {
		return nil, esv1.NoSecretErr
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("protonpass: ambiguous item title %q: %d matches", key, len(matches))
	}
	return matches[0], nil
}

// Validate mints a session to confirm the PAT is valid.
func (c *Client) Validate(ctx context.Context) error {
	_, err := c.getSession(ctx)
	return err
}

// shareKeyForRotation returns the decrypted share key for the given rotation.
// Each item can sit on a different rotation, so the matching rotation must come
// from the item's KeyRotation, not the share's ContentKeyRotation.
func (c *Client) shareKeyForRotation(ctx context.Context, shareID string, rotation int) ([]byte, error) {
	keys, err := c.GetShareKeys(ctx, shareID)
	if err != nil {
		return nil, err
	}
	for i := range keys {
		if keys[i].KeyRotation == rotation {
			blob, err := base64.StdEncoding.DecodeString(keys[i].Key)
			if err != nil {
				return nil, fmt.Errorf("protonpass: failed to decode share key: %w", err)
			}
			return crypto.OpenShareKey(blob, c.patKey)
		}
	}
	return nil, fmt.Errorf("protonpass: no share key for rotation %d", rotation)
}

func (c *Client) projectItem(it Item, shareKey []byte) (map[string][]byte, error) {
	var itemKey []byte
	if it.ItemKey != "" {
		blob, err := base64.StdEncoding.DecodeString(it.ItemKey)
		if err != nil {
			return nil, fmt.Errorf("protonpass: failed to decode item key: %w", err)
		}
		itemKey, err = crypto.OpenItemKey(blob, shareKey)
		if err != nil {
			return nil, err
		}
	} else {
		itemKey = shareKey
	}
	contentBlob, err := base64.StdEncoding.DecodeString(it.Content)
	if err != nil {
		return nil, fmt.Errorf("protonpass: failed to decode item content: %w", err)
	}
	content, err := crypto.OpenContent(contentBlob, itemKey)
	if err != nil {
		return nil, err
	}
	return codec.Project(content)
}
