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
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

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
	Owner              bool    `json:"Owner"`
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
	// apiToken is the "pst_..." half of the PAT; the session endpoint
	// rejects the full "pat::key" string with code 2001.
	apiToken string
	patKey   []byte
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
	SessionUID           string `json:"SessionUID"`
	AccessToken          string `json:"AccessToken"`
	AccessExpirationTime int64  `json:"AccessExpirationTime"` // Unix epoch seconds, not RFC3339.
}

// sessionCache caches minted sessions per PAT to avoid the per-account
// "too many recent logins" rate limit (code 2028).
var sessionCache = cache.Must(1024, func(_ *session) {})

// mintFlight collapses concurrent mints for the same PAT so a cache-miss
// stampede after startup or eviction produces exactly one session request.
var mintFlight singleflight.Group

// mintCooldown backs off per PAT after a 2028 so requeue storms don't keep
// re-arming the rate limit. Entries are opportunistically pruned on access to
// bound memory growth as new PATs are seen over time.
var (
	mintCooldownMu sync.Mutex
	mintCooldown   = map[string]time.Time{}
)

const (
	rateLimitCooldown = 5 * time.Minute
	maxItemPages      = 1000
)

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
	apiToken, patKey, err := ParsePAT(pat)
	if err != nil {
		return nil, err
	}
	if len(patKey) != 32 {
		return nil, fmt.Errorf("protonpass: PAT key must be 32 bytes, got %d", len(patKey))
	}

	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
		pat:        pat,
		apiToken:   apiToken,
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

// sessionKey fingerprints the PAT so the raw credential is never used as
// the LRU key.
func (c *Client) sessionKey() cache.Key {
	sum := sha256.Sum256([]byte(c.pat))
	return cache.Key{Name: hex.EncodeToString(sum[:])}
}

func (c *Client) getSession(ctx context.Context) (*session, error) {
	key := c.sessionKey()
	if s, ok := sessionCache.Get("", key); ok && time.Now().Before(s.expiresAt) {
		return s, nil
	}
	v, err, _ := mintFlight.Do(key.Name, func() (any, error) {
		if s, ok := sessionCache.Get("", key); ok && time.Now().Before(s.expiresAt) {
			return s, nil
		}
		s, err := c.mintSession(ctx)
		if err != nil {
			return nil, err
		}
		sessionCache.Add("", key, s)
		return s, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*session), nil
}

// No public Remove on runtime/cache; the API evicts on version mismatch,
// so Get with a different version evicts.
func (c *Client) invalidateSession() {
	sessionCache.Get("__invalidate__", c.sessionKey())
}

// checkAndPruneCooldown returns an error if the given PAT is still cooling
// down after a rate-limit response. It also prunes expired entries so the map
// does not grow unboundedly as new PATs are seen over the controller lifetime.
func checkAndPruneCooldown(key string) error {
	mintCooldownMu.Lock()
	defer mintCooldownMu.Unlock()
	now := time.Now()
	for k, until := range mintCooldown {
		if !now.Before(until) {
			delete(mintCooldown, k)
		}
	}
	if until, ok := mintCooldown[key]; ok {
		return fmt.Errorf("protonpass: rate-limited, next mint attempt at %s", until.Format(time.RFC3339))
	}
	return nil
}

func setCooldown(key string, until time.Time) {
	mintCooldownMu.Lock()
	mintCooldown[key] = until
	mintCooldownMu.Unlock()
}

func (c *Client) mintSession(ctx context.Context) (*session, error) {
	ckey := c.sessionKey().Name
	if err := checkAndPruneCooldown(ckey); err != nil {
		return nil, err
	}

	body, err := json.Marshal(map[string]string{"Token": c.apiToken})
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
		wait := max(rateLimitCooldown, retryAfter(resp.Header))
		setCooldown(ckey, time.Now().Add(wait))
		return nil, fmt.Errorf("protonpass: too many recent logins (code 2028), backing off %s", wait)
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
		expiresAt:   time.Unix(sr.Session.AccessExpirationTime, 0),
	}, nil
}

func (c *Client) request(ctx context.Context, method, path string, body []byte, out any) error {
	backoff := time.Second
	for range 5 {
		s, err := c.getSession(ctx)
		if err != nil {
			return err
		}
		code, header, respBody, err := c.raw(ctx, method, path, body, s)
		if err != nil {
			return err
		}
		switch {
		case code == http.StatusUnauthorized:
			c.invalidateSession()
			continue
		case code == http.StatusTooManyRequests:
			wait := backoff
			if ra := retryAfter(header); ra > 0 {
				wait = ra
			}
			if err := sleepCtx(ctx, wait); err != nil {
				return err
			}
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

// sleepCtx blocks for d or until ctx is done, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// retryAfter parses the Retry-After header. RFC 7231 allows either a delta in
// seconds or an HTTP-date; we only honor the delta form since Proton uses it.
func retryAfter(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return 0
}

func (c *Client) raw(ctx context.Context, method, path string, body []byte, s *session) (int, http.Header, []byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("protonpass: failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("x-pm-uid", s.uid)
	req.Header.Set("x-pm-appversion", appVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("protonpass: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("protonpass: failed to read response: %w", err)
	}
	return resp.StatusCode, resp.Header, respBody, nil
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
	// Wire nests Keys inside a ShareKeys envelope; a bare top-level Keys
	// decode returns empty and item decrypt fails with "no share key for rotation N".
	var resp struct {
		ShareKeys struct {
			Keys []ShareKey `json:"Keys"`
		} `json:"ShareKeys"`
	}
	if err := c.request(ctx, http.MethodGet, "/pass/v1/share/"+shareID+"/key", nil, &resp); err != nil {
		return nil, err
	}
	return resp.ShareKeys.Keys, nil
}

// ListItems returns active items for a share, paginating until LastToken is empty.
func (c *Client) ListItems(ctx context.Context, shareID string) ([]Item, error) {
	var all []Item
	since := ""
	for page := range maxItemPages {
		q := url.Values{"PageSize": []string{pageSize}}
		if since != "" {
			q.Set("Since", since)
		}
		path := fmt.Sprintf("/pass/v1/share/%s/item?%s", shareID, q.Encode())
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
		if resp.Items.LastToken == "" {
			break
		}
		// A repeating LastToken means the server is not advancing; bail rather
		// than loop forever accumulating the same page.
		if resp.Items.LastToken == since {
			return nil, fmt.Errorf("protonpass: pagination did not advance for share %s", shareID)
		}
		since = resp.Items.LastToken
		if page == maxItemPages-1 {
			return nil, fmt.Errorf("protonpass: pagination exceeded %d pages for share %s", maxItemPages, shareID)
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
			// ItemID is cleartext; skip decrypt for non-matches on ID lookup.
			if isID && it.ItemID != id {
				continue
			}
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
				// For ID lookup the ItemID matched exactly, so surface the
				// error. For title scans one corrupt/unprojectable item must
				// not hide a valid match elsewhere in the vault.
				if isID {
					return nil, err
				}
				continue
			}
			if isID {
				return proj, nil
			}
			byTitle = append(byTitle, proj)
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
