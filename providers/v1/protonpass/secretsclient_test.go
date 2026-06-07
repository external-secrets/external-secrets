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
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// fakeAPI is a stateful in-memory Proton Pass API for unit tests. It mirrors the
// real wire crypto: share key wrapped with the PAT key; item keys wrapped with the
// share key; content sealed with the item key.
type fakeAPI struct {
	patKey   []byte
	shareKey []byte
	shareID  string
	roleID   string
	items    []apiItemRevision
	nextID   int
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	f := &fakeAPI{
		patKey:   bytes.Repeat([]byte{0x11}, keyLength),
		shareKey: bytes.Repeat([]byte{0x22}, keyLength),
		shareID:  "share-1",
		roleID:   "2", // editor: writable
	}
	return f
}

func encB64(t *testing.T, data, key []byte, tag encryptionTag) string {
	t.Helper()
	b, err := aeadEncrypt(data, key, tag)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// addLogin seeds an encrypted login item and returns its ItemID.
func (f *fakeAPI) addLogin(t *testing.T, title, password, totpURI string) string {
	t.Helper()
	login := putString(nil, fLoginPassword, password)
	login = putString(login, fLoginTOTPURI, totpURI)
	content := assembleItem(title, "", putMessage(nil, fContentLogin, login), nil)
	itemKey := bytes.Repeat([]byte{0x33}, keyLength)
	encKey := encB64(t, itemKey, f.shareKey, tagItemKey)
	id := "item-" + strconv.Itoa(f.nextID)
	f.nextID++
	f.items = append(f.items, apiItemRevision{
		ItemID:      id,
		Revision:    1,
		KeyRotation: 1,
		Content:     encB64(t, content, itemKey, tagItemContent),
		ItemKey:     &encKey,
		State:       itemStateActive,
	})
	return id
}

func (f *fakeAPI) findItem(id string) *apiItemRevision {
	for i := range f.items {
		if f.items[i].ItemID == id {
			return &f.items[i]
		}
	}
	return nil
}

func (f *fakeAPI) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("/account/v4/personal-access-token/session", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{"Code": 1000, "Session": map[string]any{"SessionUID": "uid", "AccessToken": "acc"}})
	})
	mux.HandleFunc("/pass/v1/share", func(w http.ResponseWriter, _ *http.Request) {
		name := vaultContent(t, "estest", f.shareKey)
		write(w, map[string]any{"Shares": []map[string]any{{
			"ShareID": f.shareID, "VaultID": "v1", "TargetType": 1, "Owner": true,
			"Permission": 126, "ShareRoleID": f.roleID, "Content": name, "ContentKeyRotation": 1,
		}}})
	})
	mux.HandleFunc("/pass/v1/share/"+"share-1"+"/key", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{"ShareKeys": map[string]any{"Keys": []map[string]any{
			{"KeyRotation": 1, "Key": encB64(t, f.shareKey, f.patKey, tagShareKey)},
		}}})
	})
	mux.HandleFunc("/pass/v1/share/share-1/item/trash", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Items []struct {
				ItemID string `json:"ItemID"`
			} `json:"Items"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		for _, it := range req.Items {
			if rev := f.findItem(it.ItemID); rev != nil {
				rev.State = 2 // trashed
			}
		}
		write(w, map[string]any{"Code": 1000})
	})
	mux.HandleFunc("/pass/v1/share/share-1/item", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			active := make([]apiItemRevision, 0, len(f.items))
			for _, it := range f.items {
				if it.State == itemStateActive {
					active = append(active, it)
				}
			}
			write(w, map[string]any{"Items": map[string]any{"RevisionsData": active, "LastToken": nil}})
		case http.MethodPost:
			var req createItemRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			id := "item-" + strconv.Itoa(f.nextID)
			f.nextID++
			rev := apiItemRevision{ItemID: id, Revision: 1, KeyRotation: req.KeyRotation, Content: req.Content, ItemKey: &req.ItemKey, State: itemStateActive}
			f.items = append(f.items, rev)
			write(w, map[string]any{"Item": rev})
		}
	})
	// update: /pass/v1/share/share-1/item/{id}
	mux.HandleFunc("/pass/v1/share/share-1/item/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.NotFound(w, r)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/pass/v1/share/share-1/item/")
		rev := f.findItem(id)
		if rev == nil {
			http.NotFound(w, r)
			return
		}
		var req updateItemRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		rev.Content = req.Content
		rev.Revision = req.LastRevision + 1
		write(w, map[string]any{"Item": *rev})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func vaultContent(t *testing.T, name string, shareKey []byte) string {
	t.Helper()
	return encB64(t, putString(nil, fVaultName, name), shareKey, tagVaultContent)
}

func newTestClient(t *testing.T, f *fakeAPI) *client {
	t.Helper()
	srv := f.server(t)
	return &client{
		api:    newAPIClient(parsedPAT{token: "pst_test", key: f.patKey}, srv.URL),
		vaults: nil,
	}
}

func TestGetSecretDefaultAndProperty(t *testing.T) {
	f := newFakeAPI(t)
	f.addLogin(t, "db", "p@ss", "")
	c := newTestClient(t, f)
	ctx := context.Background()

	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "db"})
	if err != nil || string(got) != "p@ss" {
		t.Fatalf("default property = %q, %v", got, err)
	}
	got, err = c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "db", Property: "password"})
	if err != nil || string(got) != "p@ss" {
		t.Fatalf("explicit password = %q, %v", got, err)
	}
}

func TestGetSecretNotFound(t *testing.T) {
	f := newFakeAPI(t)
	c := newTestClient(t, f)
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "nope"})
	if !errors.Is(err, esv1.NoSecretErr) {
		t.Fatalf("want NoSecretErr, got %v", err)
	}
}

func TestGetSecretMissingPropertyIsNotNoSecret(t *testing.T) {
	// An existing item missing the requested property is a usage error, not an
	// absent secret — mapping it to NoSecretErr would let deletionPolicy=Delete
	// drop the target Secret over a typo.
	f := newFakeAPI(t)
	f.addLogin(t, "db", "p@ss", "")
	c := newTestClient(t, f)
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "db", Property: "nonexistent"})
	if err == nil {
		t.Fatal("expected an error for a missing property")
	}
	if errors.Is(err, esv1.NoSecretErr) {
		t.Errorf("missing property must not map to NoSecretErr, got %v", err)
	}
}

func TestGetSecretAmbiguousTitleIsHardError(t *testing.T) {
	f := newFakeAPI(t)
	f.addLogin(t, "dup", "a", "")
	f.addLogin(t, "dup", "b", "")
	c := newTestClient(t, f)
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "dup"})
	if err == nil || errors.Is(err, esv1.NoSecretErr) {
		t.Fatalf("ambiguous title must be a hard error, got %v", err)
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error should mention ambiguity: %v", err)
	}
}

func TestGetAllSecretsRegexpAndTagsUnsupported(t *testing.T) {
	f := newFakeAPI(t)
	f.addLogin(t, "demo-1", "x", "")
	f.addLogin(t, "demo-2", "y", "")
	f.addLogin(t, "other", "z", "")
	c := newTestClient(t, f)
	ctx := context.Background()

	all, err := c.GetAllSecrets(ctx, esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "^demo-"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("want 2 matches, got %d (%v)", len(all), all)
	}

	if _, err := c.GetAllSecrets(ctx, esv1.ExternalSecretFind{Tags: map[string]string{"a": "b"}}); err == nil {
		t.Error("tags must be unsupported")
	}
}

func TestPushUpdateDeleteRoundTrip(t *testing.T) {
	f := newFakeAPI(t)
	c := newTestClient(t, f)
	ctx := context.Background()
	secret := &corev1.Secret{Data: map[string][]byte{"k": []byte("v1")}}

	if err := c.PushSecret(ctx, secret, fakePush{secretKey: "k", remoteKey: "app", property: "token"}); err != nil {
		t.Fatalf("push create: %v", err)
	}
	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "app", Property: "token"})
	if err != nil || string(got) != "v1" {
		t.Fatalf("read after create = %q, %v", got, err)
	}

	secret.Data["k"] = []byte("v2")
	if err := c.PushSecret(ctx, secret, fakePush{secretKey: "k", remoteKey: "app", property: "token"}); err != nil {
		t.Fatalf("push update: %v", err)
	}
	got, _ = c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "app", Property: "token"})
	if string(got) != "v2" {
		t.Fatalf("read after update = %q", got)
	}

	ok, err := c.SecretExists(ctx, fakeRef{remoteKey: "app"})
	if err != nil || !ok {
		t.Fatalf("exists = %v, %v", ok, err)
	}
	if err := c.DeleteSecret(ctx, fakeRef{remoteKey: "app"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	ok, _ = c.SecretExists(ctx, fakeRef{remoteKey: "app"})
	if ok {
		t.Error("item should be gone after delete")
	}
}

func TestPushEmptyValueRoundTrips(t *testing.T) {
	// Pushing an empty value must round-trip: the field exists (SecretExists true)
	// and reads back empty, rather than vanishing and forcing a re-push each cycle.
	f := newFakeAPI(t)
	c := newTestClient(t, f)
	ctx := context.Background()
	secret := &corev1.Secret{Data: map[string][]byte{"k": {}}}

	if err := c.PushSecret(ctx, secret, fakePush{secretKey: "k", remoteKey: "app", property: "token"}); err != nil {
		t.Fatalf("push empty: %v", err)
	}
	ok, err := c.SecretExists(ctx, fakeRef{remoteKey: "app", property: "token"})
	if err != nil || !ok {
		t.Fatalf("SecretExists(empty pushed field) = %v, %v; want true, nil", ok, err)
	}
	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "app", Property: "token"})
	if err != nil {
		t.Fatalf("GetSecret: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %q, want empty", got)
	}
}

func TestPushRequiresWritableVault(t *testing.T) {
	f := newFakeAPI(t)
	f.roleID = "3" // viewer
	c := newTestClient(t, f)
	err := c.PushSecret(context.Background(), &corev1.Secret{Data: map[string][]byte{"k": []byte("v")}},
		fakePush{secretKey: "k", remoteKey: "app", property: "token"})
	if err == nil || !strings.Contains(err.Error(), "writable") {
		t.Fatalf("viewer token must refuse writes, got %v", err)
	}
}
