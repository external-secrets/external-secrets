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
	"encoding/base32"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// These tests hit the live Proton Pass API and are skipped unless the relevant
// env var points at a PAT file scoped to the non-sensitive "estest" demo vault:
//   PROTONPASS_TEST_PAT_FILE         any-role token (read tests)
//   PROTONPASS_TEST_EDITOR_PAT_FILE  editor/manager token (write round-trip)

func liveClientFromEnv(t *testing.T, env string) *client {
	t.Helper()
	path := os.Getenv(env)
	if path == "" {
		t.Skipf("set %s to a PAT file to run this live test", env)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read PAT file: %v", err)
	}
	pat, err := parsePAT(string(raw))
	if err != nil {
		t.Fatalf("parse PAT: %v", err)
	}
	return &client{api: newAPIClient(pat, defaultHost), vaults: []string{"estest"}}
}

func TestLiveRead(t *testing.T) {
	c := liveClientFromEnv(t, "PROTONPASS_TEST_PAT_FILE")
	ctx := context.Background()

	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "demo login", Property: "password"})
	if err != nil {
		t.Fatalf("GetSecret: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("GetSecret returned empty password")
	}

	m, err := c.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: "demo database"})
	if err != nil {
		t.Fatalf("GetSecretMap: %v", err)
	}
	if _, ok := m["Host"]; !ok {
		t.Errorf("GetSecretMap(demo database) missing Host; got keys %v", keysOf(m))
	}

	all, err := c.GetAllSecrets(ctx, esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "^demo "}})
	if err != nil {
		t.Fatalf("GetAllSecrets: %v", err)
	}
	if len(all) == 0 {
		t.Error("GetAllSecrets(^demo ) returned nothing")
	}
}

func TestLiveWriteRoundTrip(t *testing.T) {
	c := liveClientFromEnv(t, "PROTONPASS_TEST_EDITOR_PAT_FILE")
	ctx := context.Background()
	const title = "eso-it-roundtrip"

	secret := &corev1.Secret{Data: map[string][]byte{"k": []byte("eso-val-1")}}
	if err := c.PushSecret(ctx, secret, fakePush{secretKey: "k", remoteKey: title, property: "password"}); err != nil {
		t.Fatalf("PushSecret(create): %v", err)
	}
	t.Cleanup(func() { _ = c.DeleteSecret(ctx, fakeRef{remoteKey: title}) })

	ok, err := c.SecretExists(ctx, fakeRef{remoteKey: title})
	if err != nil || !ok {
		t.Fatalf("SecretExists after create = %v, %v", ok, err)
	}
	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: title, Property: "password"})
	if err != nil || string(got) != "eso-val-1" {
		t.Fatalf("GetSecret after create = %q, %v", got, err)
	}

	// update the same field
	secret.Data["k"] = []byte("eso-val-2")
	if err := c.PushSecret(ctx, secret, fakePush{secretKey: "k", remoteKey: title, property: "password"}); err != nil {
		t.Fatalf("PushSecret(update): %v", err)
	}
	got, err = c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: title, Property: "password"})
	if err != nil || string(got) != "eso-val-2" {
		t.Fatalf("GetSecret after update = %q, %v", got, err)
	}

	// delete and confirm gone
	if err := c.DeleteSecret(ctx, fakeRef{remoteKey: title}); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	ok, err = c.SecretExists(ctx, fakeRef{remoteKey: title})
	if err != nil {
		t.Fatalf("SecretExists after delete: %v", err)
	}
	if ok {
		t.Error("item still exists after delete")
	}
}

func TestLiveTOTP(t *testing.T) {
	c := liveClientFromEnv(t, "PROTONPASS_TEST_EDITOR_PAT_FILE")
	ctx := context.Background()
	const title = "eso-it-totp"
	secret := base32.StdEncoding.EncodeToString([]byte("12345678912345678912"))
	uri := "otpauth://totp/eso?secret=" + secret + "&algorithm=SHA1&digits=6&period=30"

	// Locate the estest vault and create a login item carrying the TOTP seed.
	vaults, err := c.scopedVaults(ctx)
	if err != nil {
		t.Fatalf("scopedVaults: %v", err)
	}
	var v vaultCtx
	for _, x := range vaults {
		if x.name == "estest" {
			v = x
		}
	}
	if v.name == "" {
		t.Skip("estest vault not accessible to this token")
	}
	login := putString(nil, fLoginPassword, "pw")
	login = putString(login, fLoginTOTPURI, uri)
	content := assembleItem(title, "", putMessage(nil, fContentLogin, login), nil)
	if err := c.api.createItem(ctx, v.share.ShareID, v.keys, content); err != nil {
		t.Fatalf("createItem: %v", err)
	}
	t.Cleanup(func() { _ = c.DeleteSecret(ctx, fakeRef{remoteKey: title}) })

	before := time.Now()
	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: title, Property: "totp"})
	if err != nil {
		t.Fatalf("GetSecret(totp): %v", err)
	}
	// The server-read code must equal the RFC code for the current 30s window;
	// accept the window straddling the call to avoid boundary flakiness.
	want1, _ := totpCodeAt(uri, before)
	want2, _ := totpCodeAt(uri, before.Add(-30*time.Second))
	want3, _ := totpCodeAt(uri, before.Add(30*time.Second))
	if string(got) != want1 && string(got) != want2 && string(got) != want3 {
		t.Errorf("totp code %q not in expected window set {%s,%s,%s}", got, want2, want1, want3)
	}
}

func keysOf(m map[string][]byte) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

type fakePush struct{ secretKey, remoteKey, property string }

func (f fakePush) GetMetadata() *apiextensionsv1.JSON { return nil }
func (f fakePush) GetSecretKey() string               { return f.secretKey }
func (f fakePush) GetRemoteKey() string               { return f.remoteKey }
func (f fakePush) GetProperty() string                { return f.property }

type fakeRef struct{ remoteKey, property string }

func (f fakeRef) GetRemoteKey() string { return f.remoteKey }
func (f fakeRef) GetProperty() string  { return f.property }
