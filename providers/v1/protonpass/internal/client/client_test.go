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
	"time"

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

func TestSessionKeyDoesNotLeakRawPAT(t *testing.T) {
	pat1 := "pst_" + strings.Repeat("a", 64) + "::" + base64.RawURLEncoding.EncodeToString(fixedKey())
	pat2 := "pst_" + strings.Repeat("z", 64) + "::" + base64.RawURLEncoding.EncodeToString(fixedKey())

	c1, err := NewClient(pat1)
	require.NoError(t, err)
	c2, err := NewClient(pat2)
	require.NoError(t, err)

	k1 := c1.sessionKey()
	k2 := c2.sessionKey()

	assert.NotEqual(t, pat1, k1.Name, "cache key must not be the raw PAT")
	assert.Len(t, k1.Name, 64, "cache key should be a hex-encoded sha256 digest")
	assert.NotEqual(t, k1, k2, "different PATs must fingerprint to different cache keys")

	c1Again, err := NewClient(pat1)
	require.NoError(t, err)
	assert.Equal(t, k1, c1Again.sessionKey(), "same PAT must fingerprint to same key")
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
					"AccessExpirationTime": 9999999999,
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
					"AccessExpirationTime": 9999999999,
				},
			})
		case r.URL.Path == "/pass/v1/share":
			_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{
				{"ShareID": "share1", "VaultID": "vault1", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": nil},
				{"ShareID": "groupshare", "VaultID": "vault2", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": "grp-1"},
				{"ShareID": "notvault", "VaultID": "vault3", "TargetType": 2, "ContentKeyRotation": 1, "GroupID": nil},
			}})
		case r.URL.Path == "/pass/v1/share/share1/key":
			_ = json.NewEncoder(w).Encode(map[string]any{"ShareKeys": map[string]any{"Keys": []map[string]any{
				{"KeyRotation": 1, "Key": encryptAES(t, shareKey, patKey, "sharekey")},
			}}})
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

func TestGetItemByIDSkipsDecryptingNonMatches(t *testing.T) {
	patKey := fixedKey()
	shareKey := make([]byte, 32)
	copy(shareKey, patKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal-access-token/session"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code":    codeSuccess,
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": 9999999999},
			})
		case r.URL.Path == "/pass/v1/share":
			_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{
				{"ShareID": "share1", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": nil},
			}})
		case r.URL.Path == "/pass/v1/share/share1/key":
			_ = json.NewEncoder(w).Encode(map[string]any{"ShareKeys": map[string]any{"Keys": []map[string]any{
				{"KeyRotation": 1, "Key": encryptAES(t, shareKey, patKey, "sharekey")},
			}}})
		case strings.HasPrefix(r.URL.Path, "/pass/v1/share/share1/item"):
			_ = json.NewEncoder(w).Encode(map[string]any{"Items": map[string]any{
				"RevisionsData": []map[string]any{
					{
						"ItemID": "corrupt", "State": 1, "KeyRotation": 1, "ItemKey": "",
						"Content": base64.StdEncoding.EncodeToString([]byte("not valid gcm ciphertext at all")),
					},
					{
						"ItemID": "wanted", "State": 1, "KeyRotation": 1, "ItemKey": "",
						"Content": encryptAES(t, buildLoginItemProto("Target", "dave", "goodpass"), shareKey, "itemcontent"),
					},
				},
				"LastToken": "",
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("j", 64) + "::" + base64.RawURLEncoding.EncodeToString(patKey)
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	got, err := c.GetItem(context.Background(), "id:wanted")
	require.NoError(t, err, "must not attempt to decrypt the corrupt non-matching item")
	assert.Equal(t, []byte("dave"), got["username"])
	assert.Equal(t, []byte("goodpass"), got["password"])
}

func TestGetItemByTitleSkipsCorruptItems(t *testing.T) {
	patKey := fixedKey()
	shareKey := make([]byte, 32)
	copy(shareKey, patKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal-access-token/session"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code":    codeSuccess,
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": 9999999999},
			})
		case r.URL.Path == "/pass/v1/share":
			_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{
				{"ShareID": "share1", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": nil},
			}})
		case r.URL.Path == "/pass/v1/share/share1/key":
			_ = json.NewEncoder(w).Encode(map[string]any{"ShareKeys": map[string]any{"Keys": []map[string]any{
				{"KeyRotation": 1, "Key": encryptAES(t, shareKey, patKey, "sharekey")},
			}}})
		case strings.HasPrefix(r.URL.Path, "/pass/v1/share/share1/item"):
			_ = json.NewEncoder(w).Encode(map[string]any{"Items": map[string]any{
				"RevisionsData": []map[string]any{
					{
						"ItemID": "corrupt", "State": 1, "KeyRotation": 1, "ItemKey": "",
						"Content": base64.StdEncoding.EncodeToString([]byte("not valid gcm ciphertext at all")),
					},
					{
						"ItemID": "wanted", "State": 1, "KeyRotation": 1, "ItemKey": "",
						"Content": encryptAES(t, buildLoginItemProto("Target", "eve", "goodpass"), shareKey, "itemcontent"),
					},
				},
				"LastToken": "",
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("p", 64) + "::" + base64.RawURLEncoding.EncodeToString(patKey)
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	got, err := c.GetItem(context.Background(), "Target")
	require.NoError(t, err, "corrupt item must not abort the title scan")
	assert.Equal(t, []byte("eve"), got["username"])
	assert.Equal(t, []byte("goodpass"), got["password"])
}

func TestGetItemByIDStillFailsOnCorruptTargetItem(t *testing.T) {
	patKey := fixedKey()
	shareKey := make([]byte, 32)
	copy(shareKey, patKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal-access-token/session"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code":    codeSuccess,
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": 9999999999},
			})
		case r.URL.Path == "/pass/v1/share":
			_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{
				{"ShareID": "share1", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": nil},
			}})
		case r.URL.Path == "/pass/v1/share/share1/key":
			_ = json.NewEncoder(w).Encode(map[string]any{"ShareKeys": map[string]any{"Keys": []map[string]any{
				{"KeyRotation": 1, "Key": encryptAES(t, shareKey, patKey, "sharekey")},
			}}})
		case strings.HasPrefix(r.URL.Path, "/pass/v1/share/share1/item"):
			_ = json.NewEncoder(w).Encode(map[string]any{"Items": map[string]any{
				"RevisionsData": []map[string]any{
					{
						"ItemID": "wanted", "State": 1, "KeyRotation": 1, "ItemKey": "",
						"Content": base64.StdEncoding.EncodeToString([]byte("not valid gcm ciphertext at all")),
					},
				},
				"LastToken": "",
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("q", 64) + "::" + base64.RawURLEncoding.EncodeToString(patKey)
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	_, err = c.GetItem(context.Background(), "id:wanted")
	require.Error(t, err, "ID lookup must surface decrypt errors on the requested item")
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
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": 9999999999},
			})
		case r.URL.Path == "/pass/v1/share":
			_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{
				{"ShareID": "share1", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": nil},
			}})
		case r.URL.Path == "/pass/v1/share/share1/key":
			// Two rotation-sealed share keys; the item below is on rotation 2.
			_ = json.NewEncoder(w).Encode(map[string]any{"ShareKeys": map[string]any{"Keys": []map[string]any{
				{"KeyRotation": 1, "Key": encryptAES(t, shareKey1, patKey, "sharekey")},
				{"KeyRotation": 2, "Key": encryptAES(t, shareKey2, patKey, "sharekey")},
			}}})
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
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": 9999999999},
			})
		case r.URL.Path == "/pass/v1/share":
			_ = json.NewEncoder(w).Encode(map[string]any{"Shares": []map[string]any{
				{"ShareID": "share1", "TargetType": 1, "ContentKeyRotation": 1, "GroupID": nil},
			}})
		case r.URL.Path == "/pass/v1/share/share1/key":
			_ = json.NewEncoder(w).Encode(map[string]any{"ShareKeys": map[string]any{"Keys": []map[string]any{
				{"KeyRotation": 1, "Key": encryptAES(t, shareKey, patKey, "sharekey")},
			}}})
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
				"Session": map[string]any{"SessionUID": "u", "AccessToken": fmt.Sprintf("acc-%d", sessions.Load()), "AccessExpirationTime": 9999999999},
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
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": 9999999999},
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

func TestRetryOn429RespectsContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/personal-access-token/session") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code":    codeSuccess,
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": 9999999999},
			})
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("k", 64) + "::" + base64.RawURLEncoding.EncodeToString(fixedKey())
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = c.ListShares(ctx)
	elapsed := time.Since(start)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "must surface ctx cancellation, not swallow it in Sleep")
	assert.Less(t, elapsed, 500*time.Millisecond, "must return promptly after ctx cancel, not wait full backoff")
}

func TestListItemsStopsOnStuckLastToken(t *testing.T) {
	patKey := fixedKey()
	shareKey := make([]byte, 32)
	copy(shareKey, patKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal-access-token/session"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code":    codeSuccess,
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": 9999999999},
			})
		case strings.HasPrefix(r.URL.Path, "/pass/v1/share/share1/item"):
			// Always return the same non-empty LastToken to simulate a stuck server.
			_ = json.NewEncoder(w).Encode(map[string]any{"Items": map[string]any{
				"RevisionsData": []map[string]any{},
				"LastToken":     "stuck",
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("m", 64) + "::" + base64.RawURLEncoding.EncodeToString(patKey)
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	_, err = c.ListItems(context.Background(), "share1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pagination did not advance")
}

func TestListItemsEscapesLastTokenInURL(t *testing.T) {
	patKey := fixedKey()
	var seenSince []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal-access-token/session"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Code":    codeSuccess,
				"Session": map[string]any{"SessionUID": "u", "AccessToken": "a", "AccessExpirationTime": 9999999999},
			})
		case strings.HasPrefix(r.URL.Path, "/pass/v1/share/share1/item"):
			seenSince = append(seenSince, r.URL.Query().Get("Since"))
			if len(seenSince) == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{"Items": map[string]any{
					"RevisionsData": []map[string]any{},
					"LastToken":     "a+b&c=d%25",
				}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"Items": map[string]any{
				"RevisionsData": []map[string]any{},
				"LastToken":     "",
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	pat := "pst_" + strings.Repeat("n", 64) + "::" + base64.RawURLEncoding.EncodeToString(patKey)
	c, err := NewClient(pat, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err)

	_, err = c.ListItems(context.Background(), "share1")
	require.NoError(t, err)
	require.Len(t, seenSince, 2)
	assert.Equal(t, "", seenSince[0])
	assert.Equal(t, "a+b&c=d%25", seenSince[1], "server should decode Since to the original token, meaning it was properly url-encoded")
}
