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

package client

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func encryptAES(t *testing.T, plaintext, key []byte, aad string) string {
	t.Helper()
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, plaintext, []byte(aad)))
}

// buildLoginItemProto assembles an item_v1 Item matching the real proto:
// Item.metadata=1 (name=1), Item.content=2 (Content.login=3),
// ItemLogin.item_username=6, password=2.
func buildLoginItemProto(title, username, password string) []byte {
	var meta []byte
	meta = protowire.AppendTag(meta, 1, protowire.BytesType)
	meta = protowire.AppendString(meta, title)

	var login []byte
	if username != "" {
		login = protowire.AppendTag(login, 6, protowire.BytesType)
		login = protowire.AppendString(login, username)
	}
	if password != "" {
		login = protowire.AppendTag(login, 2, protowire.BytesType)
		login = protowire.AppendString(login, password)
	}

	var content []byte
	content = protowire.AppendTag(content, 3, protowire.BytesType) // Content.login
	content = protowire.AppendBytes(content, login)

	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType) // Item.metadata
	b = protowire.AppendBytes(b, meta)
	b = protowire.AppendTag(b, 2, protowire.BytesType) // Item.content
	b = protowire.AppendBytes(b, content)
	return b
}

func fixedKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return k
}

func TestPATParse(t *testing.T) {
	token, key, err := ParsePAT("pst_" + strings.Repeat("a", 64) + "::" + base64.RawURLEncoding.EncodeToString(fixedKey()))
	require.NoError(t, err)
	assert.Equal(t, "pst_"+strings.Repeat("a", 64), token)
	assert.Equal(t, fixedKey(), key)

	_, _, err = ParsePAT("notapat")
	assert.Error(t, err)

	_, _, err = ParsePAT("pst_" + strings.Repeat("a", 64) + "::" + "!!!notbase64url!!!")
	assert.Error(t, err)
}

func TestSessionMintAndCache(t *testing.T) {
	var mintCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/account/v4/personal-access-token/session":
			mintCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code": codeSuccess,
				"Session": map[string]any{
					"SessionUID": "uid-123", "AccessToken": "acc-123",
					"AccessExpirationTime": "2999-01-01T00:00:00Z",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("b", 64) + "::" + base64.RawURLEncoding.EncodeToString(fixedKey())
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	require.NoError(t, c.Validate(context.Background()))
	require.NoError(t, c.Validate(context.Background()))
	assert.Equal(t, int32(1), mintCount.Load(), "session should be cached")
}

func TestGetItemByTitleAndID(t *testing.T) {
	patKey := fixedKey()
	shareKey := make([]byte, 32)
	copy(shareKey, patKey)
	for i := range shareKey {
		shareKey[i] ^= 0xFF
	}
	itemKey := make([]byte, 32)
	copy(itemKey, patKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal-access-token/session"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code": codeSuccess,
				"Session": map[string]any{
					"SessionUID": "uid-1", "AccessToken": "acc-1",
					"AccessExpirationTime": "2999-01-01T00:00:00Z",
				},
			})
		case r.URL.Path == "/pass/v1/share":
			_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{
				{"ShareID": "share1", "VaultID": "vault1", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": nil},
				{"ShareID": "groupshare", "VaultID": "vault2", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": "grp-1"},
				{"ShareID": "notvault", "VaultID": "vault3", "TargetType": 2, "ContentKeyRotation": 1, "GroupID": nil},
			}})
		case r.URL.Path == "/pass/v1/share/share1/key":
			_ = json.NewEncoder(w).Encode(map[string]any{"Keys": []map[string]any{
				{"KeyRotation": 1, "Key": encryptAES(t, shareKey, patKey, "sharekey")},
			}})
		case strings.HasPrefix(r.URL.Path, "/pass/v1/share/share1/item"):
			_ = json.NewEncoder(w).Encode(map[string]any{"Items": map[string]any{
				"RevisionsData": []map[string]any{
					{
						"ItemID": "item1", "Revision": 1, "KeyRotation": 1, "State": 1,
						"ItemKey": encryptAES(t, itemKey, shareKey, "itemkey"),
						"Content": encryptAES(t, buildLoginItemProto("My Login", "alice", "p@ss"), itemKey, "itemcontent"),
					},
					{
						"ItemID": "item2", "Revision": 1, "KeyRotation": 1, "State": 1,
						"ItemKey": "",
						"Content": encryptAES(t, buildLoginItemProto("Second Item", "bob", "secret2"), shareKey, "itemcontent"),
					},
					{
						"ItemID": "item3", "Revision": 1, "KeyRotation": 1, "State": 2,
						"ItemKey": "",
						"Content": encryptAES(t, buildLoginItemProto("Trashed", "trash", "x"), shareKey, "itemcontent"),
					},
				},
				"LastToken": "",
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("c", 64) + "::" + base64.RawURLEncoding.EncodeToString(patKey)
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	// By title.
	got, err := c.GetItem(context.Background(), "My Login")
	require.NoError(t, err)
	assert.Equal(t, []byte("My Login"), got["title"])
	assert.Equal(t, []byte("alice"), got["username"])
	assert.Equal(t, []byte("p@ss"), got["password"])

	// By id (uses the share key directly for items without an item key).
	got, err = c.GetItem(context.Background(), "id:item2")
	require.NoError(t, err)
	assert.Equal(t, []byte("bob"), got["username"])

	// Trashed item is not resolvable.
	_, err = c.GetItem(context.Background(), "Trashed")
	require.Error(t, err)
	assert.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestGetItemUsesCorrectShareKeyRotation(t *testing.T) {
	patKey := fixedKey()
	shareKey1 := make([]byte, 32)
	shareKey2 := make([]byte, 32)
	copy(shareKey1, patKey)
	copy(shareKey2, patKey)
	for i := range shareKey1 {
		shareKey1[i] ^= 0x0F
	}
	for i := range shareKey2 {
		shareKey2[i] ^= 0xF0
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal-access-token/session"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code":    codeSuccess,
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": "2999-01-01T00:00:00Z"},
			})
		case r.URL.Path == "/pass/v1/share":
			_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{
				{"ShareID": "share1", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": nil},
			}})
		case r.URL.Path == "/pass/v1/share/share1/key":
			// Two rotation-sealed share keys; the item below is on rotation 2.
			_ = json.NewEncoder(w).Encode(map[string]any{"Keys": []map[string]any{
				{"KeyRotation": 1, "Key": encryptAES(t, shareKey1, patKey, "sharekey")},
				{"KeyRotation": 2, "Key": encryptAES(t, shareKey2, patKey, "sharekey")},
			}})
		case strings.HasPrefix(r.URL.Path, "/pass/v1/share/share1/item"):
			_ = json.NewEncoder(w).Encode(map[string]any{"Items": map[string]any{
				"RevisionsData": []map[string]any{
					{
						"ItemID": "rot2", "State": 1, "KeyRotation": 2, "ItemKey": "",
						"Content": encryptAES(t, buildLoginItemProto("Rotated", "carol", "rot2pass"), shareKey2, "itemcontent"),
					},
				},
				"LastToken": "",
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("g", 64) + "::" + base64.RawURLEncoding.EncodeToString(patKey)
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	got, err := c.GetItem(context.Background(), "Rotated")
	require.NoError(t, err)
	assert.Equal(t, []byte("Rotated"), got["title"])
	assert.Equal(t, []byte("carol"), got["username"])
	assert.Equal(t, []byte("rot2pass"), got["password"])
}

func TestGetItemMissingAndAmbiguous(t *testing.T) {
	patKey := fixedKey()
	shareKey := make([]byte, 32)
	copy(shareKey, patKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal-access-token/session"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code":    codeSuccess,
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": "2999-01-01T00:00:00Z"},
			})
		case r.URL.Path == "/pass/v1/share":
			_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{
				{"ShareID": "share1", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": nil},
			}})
		case r.URL.Path == "/pass/v1/share/share1/key":
			_ = json.NewEncoder(w).Encode(map[string]any{"Keys": []map[string]any{
				{"KeyRotation": 1, "Key": encryptAES(t, shareKey, patKey, "sharekey")},
			}})
		case strings.HasPrefix(r.URL.Path, "/pass/v1/share/share1/item"):
			_ = json.NewEncoder(w).Encode(map[string]any{"Items": map[string]any{
				"RevisionsData": []map[string]any{
					{"ItemID": "a1", "State": 1, "KeyRotation": 1, "ItemKey": "", "Content": encryptAES(t, buildLoginItemProto("Dup", "x", "1"), shareKey, "itemcontent")},
					{"ItemID": "a2", "State": 1, "KeyRotation": 1, "ItemKey": "", "Content": encryptAES(t, buildLoginItemProto("Dup", "y", "2"), shareKey, "itemcontent")},
				},
				"LastToken": "",
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("d", 64) + "::" + base64.RawURLEncoding.EncodeToString(patKey)
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	// Missing title.
	_, err = c.GetItem(context.Background(), "Nope")
	require.Error(t, err)
	assert.ErrorIs(t, err, esv1.NoSecretErr)

	// Ambiguous title must be a hard error, never a silent pick.
	_, err = c.GetItem(context.Background(), "Dup")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
}

func TestSessionReMintOn401(t *testing.T) {
	patKey := fixedKey()
	var sessions atomic.Int32
	var authed atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/personal-access-token/session") {
			sessions.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code":    codeSuccess,
				"Session": map[string]any{"SessionUID": "u", "AccessToken": fmt.Sprintf("acc-%d", sessions.Load()), "AccessExpirationTime": "2999-01-01T00:00:00Z"},
			})
			return
		}
		// First authenticated call is rejected with 401 (expired token), forcing a re-mint.
		if authed.Add(1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{}})
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("e", 64) + "::" + base64.RawURLEncoding.EncodeToString(patKey)
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	shares, err := c.ListShares(context.Background())
	require.NoError(t, err)
	assert.Empty(t, shares)
	assert.Equal(t, int32(2), sessions.Load(), "session should be re-minted after 401")
}

func TestParsePATErrors(t *testing.T) {
	tests := []struct {
		name string
		pat  string
	}{
		{name: "empty", pat: ""},
		{name: "no separator", pat: "notapat"},
		{name: "invalid base64url key", pat: "pst_" + strings.Repeat("a", 64) + "::" + "!!!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParsePAT(tt.pat)
			assert.Error(t, err)
		})
	}
}

func TestSessionMintRejectsNonSuccessCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"Code": 1001})
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("h", 64) + "::" + base64.RawURLEncoding.EncodeToString(fixedKey())
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	err = c.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "1001")
}

func TestSessionMintHandlesRateLimitCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"Code": codeTooManyLogins})
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("i", 64) + "::" + base64.RawURLEncoding.EncodeToString(fixedKey())
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	err = c.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many recent logins")
}

func TestRetryOn429(t *testing.T) {
	patKey := fixedKey()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/personal-access-token/session") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code":    codeSuccess,
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": "2999-01-01T00:00:00Z"},
			})
			return
		}
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{}})
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("f", 64) + "::" + base64.RawURLEncoding.EncodeToString(patKey)
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	shares, err := c.ListShares(context.Background())
	require.NoError(t, err)
	assert.Empty(t, shares)
	assert.Equal(t, int32(3), calls.Load(), "should retry through 429s")
}
