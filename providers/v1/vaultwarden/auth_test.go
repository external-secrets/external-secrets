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
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // OAEP-SHA1 matches Bitwarden wire format
	"encoding/base64"
	"testing"
)

func TestCachedTokenInvalidateZeroes(t *testing.T) {
	ct := &cachedToken{
		accessToken: "bearer-xxx",
		symEncKey:   make32(1),
		symMacKey:   make32(33),
		orgID:       "af061424-2700-425a-8800-80e988194e8e",
		orgEncKey:   make32(65),
		orgMacKey:   make32(97),
		rsaPriv:     &rsa.PrivateKey{},
	}

	ct.invalidate()

	zero := make([]byte, 32)
	if ct.accessToken != "" {
		t.Errorf("accessToken not cleared: %q", ct.accessToken)
	}
	if ct.symEncKey != nil && !bytes.Equal(ct.symEncKey, zero) {
		t.Errorf("symEncKey not zeroed: %v", ct.symEncKey)
	}
	if ct.symMacKey != nil && !bytes.Equal(ct.symMacKey, zero) {
		t.Errorf("symMacKey not zeroed: %v", ct.symMacKey)
	}
	if ct.orgID != "" {
		t.Errorf("orgID not cleared: %q", ct.orgID)
	}
	if ct.orgEncKey != nil && !bytes.Equal(ct.orgEncKey, zero) {
		t.Errorf("orgEncKey not zeroed: %v", ct.orgEncKey)
	}
	if ct.orgMacKey != nil && !bytes.Equal(ct.orgMacKey, zero) {
		t.Errorf("orgMacKey not zeroed: %v", ct.orgMacKey)
	}
	if ct.rsaPriv != nil {
		t.Errorf("rsaPriv not cleared")
	}
}

func TestUnlockOrgKey(t *testing.T) {
	// Generate a fake user RSA keypair.
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil { t.Fatalf("genkey: %v", err) }

	// Fabricate a 64-byte org symkey, encrypt with user public key.
	orgSymkey := make([]byte, 64)
	if _, err := rand.Read(orgSymkey); err != nil { t.Fatalf("rand: %v", err) }
	ct, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, &priv.PublicKey, orgSymkey, nil) //nolint:gosec // OAEP-SHA1 matches Bitwarden wire format
	if err != nil { t.Fatalf("encrypt: %v", err) }
	orgEncKeyStr := "4." + base64.StdEncoding.EncodeToString(ct)

	// Call the helper that, given an RSA private key and the "4." blob,
	// returns (orgEncKey, orgMacKey).
	enc, mac, err := unlockOrgKey(orgEncKeyStr, priv)
	if err != nil { t.Fatalf("unlockOrgKey: %v", err) }
	if !bytes.Equal(enc, orgSymkey[:32]) {
		t.Fatalf("enc mismatch")
	}
	if !bytes.Equal(mac, orgSymkey[32:]) {
		t.Fatalf("mac mismatch")
	}
}

func make32(start byte) []byte {
	out := make([]byte, 32)
	for i := range out {
		out[i] = start + byte(i)
	}
	return out
}

func TestResolveOrgByName(t *testing.T) {
	profile := &vaultwardenProfile{
		Organizations: []orgEntry{
			{ID: "uuid-a", Name: "Tiberius-Grail", Key: "4.aaa"},
			{ID: "uuid-b", Name: "Tiberius", Key: "4.bbb"},
			{ID: "uuid-c", Name: "Tiberius", Key: "4.ccc"},
		},
	}

	id, key, err := resolveOrgByName(profile, "Tiberius-Grail")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "uuid-a" {
		t.Fatalf("got id %q want uuid-a", id)
	}
	if key != "4.aaa" {
		t.Fatalf("got key %q want 4.aaa", key)
	}

	_, _, err = resolveOrgByName(profile, "Tiberius")
	if err == nil {
		t.Fatalf("expected error on duplicate name, got nil")
	}

	_, _, err = resolveOrgByName(profile, "Nonexistent")
	if err == nil {
		t.Fatalf("expected error on missing name, got nil")
	}
}
