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
	"testing"
)

func key32() []byte {
	k := make([]byte, keyLength)
	for i := range k {
		k[i] = byte(i + 1)
	}
	return k
}

func TestAEADRoundTripPerTag(t *testing.T) {
	key := key32()
	plaintext := []byte("the password is correct horse battery staple")
	for _, tag := range []encryptionTag{tagShareKey, tagItemKey, tagItemContent, tagVaultContent} {
		blob, err := aeadEncrypt(plaintext, key, tag)
		if err != nil {
			t.Fatalf("encrypt(%q): %v", tag, err)
		}
		// Wire layout: nonce(12) || ciphertext || gcm tag(16).
		if want := nonceLength + len(plaintext) + 16; len(blob) != want {
			t.Errorf("tag %q: blob len = %d, want %d", tag, len(blob), want)
		}
		got, err := aeadDecrypt(blob, key, tag)
		if err != nil {
			t.Fatalf("decrypt(%q): %v", tag, err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Errorf("tag %q: round-trip mismatch: %q", tag, got)
		}
	}
}

func TestAEADWrongKeyFails(t *testing.T) {
	blob, err := aeadEncrypt([]byte("secret"), key32(), tagItemContent)
	if err != nil {
		t.Fatal(err)
	}
	wrong := key32()
	wrong[0] ^= 0xff
	if _, err := aeadDecrypt(blob, wrong, tagItemContent); err == nil {
		t.Fatal("decrypt with wrong key should fail")
	}
}

func TestAEADWrongTagFails(t *testing.T) {
	blob, err := aeadEncrypt([]byte("secret"), key32(), tagItemKey)
	if err != nil {
		t.Fatal(err)
	}
	// Same key, different AAD tag must fail authentication.
	if _, err := aeadDecrypt(blob, key32(), tagItemContent); err == nil {
		t.Fatal("decrypt with wrong tag should fail")
	}
}

func TestAEADTamperedCiphertextFails(t *testing.T) {
	blob, err := aeadEncrypt([]byte("secret"), key32(), tagItemContent)
	if err != nil {
		t.Fatal(err)
	}
	blob[len(blob)-1] ^= 0xff // flip a byte of the GCM tag
	if _, err := aeadDecrypt(blob, key32(), tagItemContent); err == nil {
		t.Fatal("decrypt of tampered ciphertext should fail")
	}
}

func TestAEADShortBlob(t *testing.T) {
	if _, err := aeadDecrypt([]byte("tooshort"), key32(), tagItemContent); err == nil {
		t.Fatal("decrypt of sub-nonce blob should fail")
	}
}

func TestNewGCMRejectsBadKeyLength(t *testing.T) {
	for _, n := range []int{0, 16, 31, 33, 64} {
		if _, err := newGCM(make([]byte, n)); err == nil {
			t.Errorf("newGCM accepted %d-byte key", n)
		}
		if _, err := aeadEncrypt([]byte("x"), make([]byte, n), tagItemKey); err == nil {
			t.Errorf("aeadEncrypt accepted %d-byte key", n)
		}
	}
}

func TestNonceUniqueAndPrepended(t *testing.T) {
	key := key32()
	pt := []byte("same plaintext every time")
	seen := make(map[string]struct{})
	const n = 1024
	for i := range n {
		blob, err := aeadEncrypt(pt, key, tagItemContent)
		if err != nil {
			t.Fatal(err)
		}
		nonce := string(blob[:nonceLength])
		if _, dup := seen[nonce]; dup {
			t.Fatalf("nonce reused after %d encryptions", i)
		}
		seen[nonce] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique nonces, got %d", n, len(seen))
	}
}

func TestNewItemKeyLengthAndRandomness(t *testing.T) {
	a, err := newItemKey()
	if err != nil || len(a) != keyLength {
		t.Fatalf("newItemKey: len=%d err=%v", len(a), err)
	}
	b, _ := newItemKey()
	if bytes.Equal(a, b) {
		t.Fatal("two item keys were identical")
	}
}
