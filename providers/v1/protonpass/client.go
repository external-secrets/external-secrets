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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const (
	defaultHost      = "https://pass-api.proton.me"
	httpTimeout      = 30 * time.Second
	itemPageSize     = 100
	contentFormatVer = 7 // ITEM_CONTENT_CONTENT_FORMAT_VERSION

	// appProduct is the platform-product segment of the x-pm-appversion header,
	// which Proton allowlists. "cli-pass" identifies the Proton Pass CLI, whose
	// Personal Access Token flow this provider speaks; it is the closest accepted
	// identity for a headless integration. Change it to the external-pass-<project>
	// identifier once Proton registers one for Pass, as it has for Drive
	// (e.g. external-drive-rclone).
	appProduct = "cli-pass"

	patPrefix    = "pst_"
	patSeparator = "::"
	patTokenLen  = 64 // characters after the prefix
)

// errNotFound signals an absent item/property; the SecretsClient layer maps it
// to esv1.NoSecretErr.
var errNotFound = errors.New("protonpass: not found")

// parsedPAT is a split Personal Access Token: an API token and a symmetric key.
type parsedPAT struct {
	token string // pst_<token> — sent to the API
	key   []byte // 32-byte AES key — used locally to unwrap share keys, never sent
}

// parsePAT validates and splits a "pst_<token>::<base64url-key>" string.
func parsePAT(s string) (parsedPAT, error) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, patSeparator, 2)
	if len(parts) != 2 {
		return parsedPAT{}, errors.New("protonpass: invalid PAT format, expected pst_<token>::<key>")
	}
	token, keyB64 := parts[0], parts[1]
	if !strings.HasPrefix(token, patPrefix) {
		return parsedPAT{}, fmt.Errorf("protonpass: PAT must start with %q", patPrefix)
	}
	if len(token)-len(patPrefix) != patTokenLen {
		return parsedPAT{}, fmt.Errorf("protonpass: PAT token must be %d characters after the prefix", patTokenLen)
	}
	key, err := base64.RawURLEncoding.DecodeString(keyB64)
	if err != nil {
		return parsedPAT{}, fmt.Errorf("protonpass: decode PAT key: %w", err)
	}
	if len(key) != keyLength {
		return parsedPAT{}, fmt.Errorf("protonpass: PAT key must be %d bytes, got %d", keyLength, len(key))
	}
	return parsedPAT{token: token, key: key}, nil
}

// session is an authenticated Proton API session minted from a PAT.
type session struct {
	uid    string
	access string
}

// apiClient is the low-level Proton Pass HTTP client. A session is minted lazily
// on first use and reused; on a 401 it is re-minted from the PAT (the durable
// credential), which is cheaper and simpler than refresh-token plumbing and
// correct because the session lives as long as the PAT.
type apiClient struct {
	http       *http.Client
	host       string
	appVersion string
	pat        parsedPAT

	mu   sync.Mutex
	sess *session
}

func newAPIClient(pat parsedPAT, host string) *apiClient {
	if host == "" {
		host = defaultHost
	}
	return &apiClient{
		http:       &http.Client{Timeout: httpTimeout},
		host:       host,
		appVersion: appVersionHeader(),
		pat:        pat,
	}
}

// appVersionHeader returns the x-pm-appversion value. The version is computed from
// the running binary's build info (Proton does not pin it); the platform-product
// segment is appProduct.
func appVersionHeader() string {
	version := "0.0.0"
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := strings.TrimPrefix(bi.Main.Version, "v"); v != "" && v != "(devel)" {
			version = v
		}
	}
	return appProduct + "@" + version
}

// --- transport ---

type apiError struct {
	Code int    `json:"Code"`
	Err  string `json:"Error"`
}

func (e *apiError) Error() string { return fmt.Sprintf("proton api code %d: %s", e.Code, e.Err) }

// do performs an authenticated request, (re)minting the session as needed and
// retrying once on 401 and a few times on 429. A nil body sends no payload; out,
// if non-nil, receives the decoded JSON response.
func (c *apiClient) do(ctx context.Context, method, path string, body, out any) error {
	const maxRateLimitRetries = 3
	for attempt := 0; ; attempt++ {
		sess, err := c.ensureSession(ctx)
		if err != nil {
			return err
		}
		status, raw, err := c.roundtrip(ctx, method, path, body, sess)
		if err != nil {
			return err
		}
		switch {
		case status >= 200 && status < 300:
			if out != nil {
				if err := json.Unmarshal(raw, out); err != nil {
					return fmt.Errorf("protonpass: decode response: %w", err)
				}
			}
			return nil
		case status == http.StatusUnauthorized && attempt == 0:
			c.invalidateSession()
			continue
		case status == http.StatusTooManyRequests && attempt < maxRateLimitRetries:
			if err := sleepCtx(ctx, backoff(attempt)); err != nil {
				return err
			}
			continue
		default:
			return apiErrorFrom(status, raw)
		}
	}
}

// roundtrip issues a single HTTP request with auth headers and returns the raw body.
func (c *apiClient) roundtrip(ctx context.Context, method, path string, body any, sess *session) (int, []byte, error) {
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("protonpass: encode request: %w", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	// G704: the host is a constant (defaultHost, no spec override); only
	// API-issued share/item IDs ever populate the path, never the host or scheme.
	req, err := http.NewRequestWithContext(ctx, method, c.host+path, reader) //nolint:gosec
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("x-pm-appversion", c.appVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if sess != nil {
		req.Header.Set("x-pm-uid", sess.uid)
		req.Header.Set("Authorization", "Bearer "+sess.access)
	}
	resp, err := c.http.Do(req) //nolint:gosec // G704: fixed host (defaultHost); only the path is templated from API-issued IDs.
	if err != nil {
		return 0, nil, fmt.Errorf("protonpass: %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("protonpass: read response body: %w", err)
	}
	return resp.StatusCode, raw, nil
}

func (c *apiClient) ensureSession(ctx context.Context) (*session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sess != nil {
		return c.sess, nil
	}
	sess, err := c.mintSession(ctx)
	if err != nil {
		return nil, err
	}
	c.sess = sess
	return sess, nil
}

func (c *apiClient) invalidateSession() {
	c.mu.Lock()
	c.sess = nil
	c.mu.Unlock()
}

// mintSession exchanges the PAT for a bearer session (no SRP, no unauth bootstrap).
func (c *apiClient) mintSession(ctx context.Context) (*session, error) {
	var resp struct {
		Session struct {
			SessionUID  string `json:"SessionUID"`
			AccessToken string `json:"AccessToken"`
		} `json:"Session"`
	}
	status, raw, err := c.roundtrip(ctx, http.MethodPost, "/account/v4/personal-access-token/session",
		map[string]string{"Token": c.pat.token}, nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("protonpass: mint session: %w", apiErrorFrom(status, raw))
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("protonpass: decode session: %w", err)
	}
	if resp.Session.SessionUID == "" || resp.Session.AccessToken == "" {
		return nil, errors.New("protonpass: session response missing credentials")
	}
	return &session{uid: resp.Session.SessionUID, access: resp.Session.AccessToken}, nil
}

// --- Proton Pass API types ---

type apiShare struct {
	ShareID            string  `json:"ShareID"`
	VaultID            string  `json:"VaultID"`
	TargetType         uint8   `json:"TargetType"`
	Owner              bool    `json:"Owner"`
	Permission         uint16  `json:"Permission"`
	ShareRoleID        string  `json:"ShareRoleID"`
	Content            string  `json:"Content"`
	ContentKeyRotation uint8   `json:"ContentKeyRotation"`
	GroupID            *string `json:"GroupID"`
}

// isGroupShared reports whether the vault is shared via a Proton group, in which
// case its share key is PGP-wrapped and a PAT cannot decrypt it.
func (s apiShare) isGroupShared() bool { return s.GroupID != nil && *s.GroupID != "" }

type apiShareKey struct {
	KeyRotation uint8  `json:"KeyRotation"`
	Key         string `json:"Key"`
}

type apiItemRevision struct {
	ItemID      string  `json:"ItemID"`
	Revision    uint64  `json:"Revision"`
	KeyRotation uint8   `json:"KeyRotation"`
	Content     string  `json:"Content"`
	ItemKey     *string `json:"ItemKey"`
	State       uint8   `json:"State"`
}

const itemStateActive = 1

// decryptField standard-base64-decodes a wire blob and AES-GCM-opens it under
// key with the given AAD tag.
func decryptField(b64 string, key []byte, tag encryptionTag) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("protonpass: decode %s: %w", tag, err)
	}
	return aeadDecrypt(raw, key, tag)
}

// unwrapItemKey returns the per-item key for a revision, decrypting it with the
// share key; items without a per-item key are sealed with the share key directly.
func unwrapItemKey(rev apiItemRevision, shareKey []byte) ([]byte, error) {
	if rev.ItemKey == nil || *rev.ItemKey == "" {
		return shareKey, nil
	}
	return decryptField(*rev.ItemKey, shareKey, tagItemKey)
}

// --- read API ---

func (c *apiClient) listShares(ctx context.Context) ([]apiShare, error) {
	var resp struct {
		Shares []apiShare `json:"Shares"`
	}
	if err := c.do(ctx, http.MethodGet, "/pass/v1/share", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Shares, nil
}

// openShareKeys fetches and symmetrically decrypts a share's keys with the PAT key.
func (c *apiClient) openShareKeys(ctx context.Context, shareID string) (map[uint8][]byte, error) {
	var resp struct {
		ShareKeys struct {
			Keys []apiShareKey `json:"Keys"`
		} `json:"ShareKeys"`
	}
	if err := c.do(ctx, http.MethodGet, "/pass/v1/share/"+shareID+"/key", nil, &resp); err != nil {
		return nil, err
	}
	out := make(map[uint8][]byte, len(resp.ShareKeys.Keys))
	for _, k := range resp.ShareKeys.Keys {
		sk, err := decryptField(k.Key, c.pat.key, tagShareKey)
		if err != nil {
			return nil, fmt.Errorf("protonpass: share key rotation %d: %w", k.KeyRotation, err)
		}
		out[k.KeyRotation] = sk
	}
	return out, nil
}

// latestRotation returns the highest key rotation present.
func latestRotation(keys map[uint8][]byte) (uint8, bool) {
	var maxRot uint8
	found := false
	for r := range keys {
		if !found || r > maxRot {
			maxRot, found = r, true
		}
	}
	return maxRot, found
}

// listItems returns all item revisions in a share, following pagination.
func (c *apiClient) listItems(ctx context.Context, shareID string) ([]apiItemRevision, error) {
	var all []apiItemRevision
	since := ""
	for {
		path := fmt.Sprintf("/pass/v1/share/%s/item?PageSize=%d", url.PathEscape(shareID), itemPageSize)
		if since != "" {
			// Since is an opaque server cursor; escape it so reserved characters
			// (+, &, =, …) don't corrupt the query and truncate the item scan.
			path += "&Since=" + url.QueryEscape(since)
		}
		var resp struct {
			Items struct {
				RevisionsData []apiItemRevision `json:"RevisionsData"`
				LastToken     *string           `json:"LastToken"`
			} `json:"Items"`
		}
		if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Items.RevisionsData...)
		if resp.Items.LastToken == nil || *resp.Items.LastToken == "" || len(resp.Items.RevisionsData) < itemPageSize {
			break
		}
		since = *resp.Items.LastToken
		if len(all) > itemPageSize*1000 { // safety cap against a pathological loop
			return nil, errors.New("protonpass: item pagination exceeded safety cap")
		}
	}
	return all, nil
}

// openItem decrypts an item revision's content into a typed Item using the share
// keys. Folder membership does not affect this: an item key is wrapped with the
// share key regardless of folder, and folder keys protect only folder metadata —
// so folder-resident items decrypt like any other.
func (c *apiClient) openItem(rev apiItemRevision, shareKeys map[uint8][]byte) (*decodedItem, error) {
	sk, ok := shareKeys[rev.KeyRotation]
	if !ok {
		return nil, fmt.Errorf("protonpass: no share key for rotation %d", rev.KeyRotation)
	}
	itemKey, err := unwrapItemKey(rev, sk)
	if err != nil {
		return nil, err
	}
	plain, err := decryptField(rev.Content, itemKey, tagItemContent)
	if err != nil {
		return nil, err
	}
	return &decodedItem{name: itemName(plain), plaintext: plain}, nil
}

// openVaultName decrypts a share's vault metadata to recover the vault name.
func (c *apiClient) openVaultName(share apiShare, shareKeys map[uint8][]byte) (string, error) {
	if share.Content == "" {
		return "", nil
	}
	sk, ok := shareKeys[share.ContentKeyRotation]
	if !ok {
		return "", fmt.Errorf("protonpass: no share key for vault content rotation %d", share.ContentKeyRotation)
	}
	plain, err := decryptField(share.Content, sk, tagVaultContent)
	if err != nil {
		return "", err
	}
	return vaultName(plain), nil
}

// --- write API ---

type createItemRequest struct {
	KeyRotation          uint8  `json:"KeyRotation"`
	ContentFormatVersion uint32 `json:"ContentFormatVersion"`
	Content              string `json:"Content"`
	ItemKey              string `json:"ItemKey"`
}

type updateItemRequest struct {
	KeyRotation          uint8  `json:"KeyRotation"`
	LastRevision         uint64 `json:"LastRevision"`
	ContentFormatVersion uint32 `json:"ContentFormatVersion"`
	Content              string `json:"Content"`
}

// createItem encrypts and creates a new item in a share.
func (c *apiClient) createItem(ctx context.Context, shareID string, shareKeys map[uint8][]byte, content []byte) error {
	rot, ok := latestRotation(shareKeys)
	if !ok {
		return errors.New("protonpass: share has no keys")
	}
	itemKey, err := newItemKey()
	if err != nil {
		return err
	}
	encContent, err := aeadEncrypt(content, itemKey, tagItemContent)
	if err != nil {
		return err
	}
	encItemKey, err := aeadEncrypt(itemKey, shareKeys[rot], tagItemKey)
	if err != nil {
		return err
	}
	req := createItemRequest{
		KeyRotation:          rot,
		ContentFormatVersion: contentFormatVer,
		Content:              base64.StdEncoding.EncodeToString(encContent),
		ItemKey:              base64.StdEncoding.EncodeToString(encItemKey),
	}
	return c.do(ctx, http.MethodPost, "/pass/v1/share/"+shareID+"/item", req, nil)
}

// updateItem re-encrypts content for an existing item using its current item key,
// guarded by the last seen revision (optimistic concurrency).
func (c *apiClient) updateItem(ctx context.Context, shareID string, rev apiItemRevision, shareKeys map[uint8][]byte, content []byte) error {
	sk, ok := shareKeys[rev.KeyRotation]
	if !ok {
		return fmt.Errorf("protonpass: no share key for rotation %d", rev.KeyRotation)
	}
	itemKey, err := unwrapItemKey(rev, sk)
	if err != nil {
		return err
	}
	encContent, err := aeadEncrypt(content, itemKey, tagItemContent)
	if err != nil {
		return err
	}
	req := updateItemRequest{
		KeyRotation:          rev.KeyRotation,
		LastRevision:         rev.Revision,
		ContentFormatVersion: contentFormatVer,
		Content:              base64.StdEncoding.EncodeToString(encContent),
	}
	return c.do(ctx, http.MethodPut, "/pass/v1/share/"+shareID+"/item/"+rev.ItemID, req, nil)
}

// trashItem moves an item to the trash.
func (c *apiClient) trashItem(ctx context.Context, shareID, itemID string, revision uint64) error {
	req := map[string]any{
		"Items": []map[string]any{{"ItemID": itemID, "Revision": revision}},
	}
	return c.do(ctx, http.MethodPost, "/pass/v1/share/"+shareID+"/item/trash", req, nil)
}

// --- helpers ---

func apiErrorFrom(status int, raw []byte) error {
	ae := &apiError{}
	if err := json.Unmarshal(raw, ae); err == nil && ae.Err != "" {
		return ae
	}
	return fmt.Errorf("protonpass: unexpected status %d: %s", status, strings.TrimSpace(string(raw)))
}

func backoff(attempt int) time.Duration {
	return time.Duration(1<<attempt) * 500 * time.Millisecond
}

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
