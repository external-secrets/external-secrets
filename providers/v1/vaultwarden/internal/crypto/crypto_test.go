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

package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	encKey := make([]byte, 32)
	macKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 1)
	}
	for i := range macKey {
		macKey[i] = byte(i + 33)
	}

	plaintext := "hello, vaultwarden!"
	enc, err := EncryptString(plaintext, encKey, macKey)
	require.NoError(t, err)
	assert.Contains(t, enc, "2.")

	got, err := DecryptString(enc, encKey, macKey)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestMACValidation(t *testing.T) {
	encKey := make([]byte, 32)
	macKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 1)
	}
	for i := range macKey {
		macKey[i] = byte(i + 33)
	}

	enc, err := EncryptString("test", encKey, macKey)
	require.NoError(t, err)

	// Tamper with the MAC by using the wrong macKey
	wrongMacKey := make([]byte, 32)
	es, err := ParseEncString(enc)
	require.NoError(t, err)
	_, err = Decrypt(es, encKey, wrongMacKey)
	assert.ErrorContains(t, err, "vaultwarden: invalid ciphertext")
}

func TestDeriveKeyPBKDF2(t *testing.T) {
	key, err := DeriveKey("parolaribame4", "test@vaultwarden.local", KdfTypePBKDF2, 600000, 0, 0)
	require.NoError(t, err)
	assert.Len(t, key, 32)
}

func TestStretchKey(t *testing.T) {
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i + 1)
	}
	encKey, macKey, err := StretchKey(masterKey)
	require.NoError(t, err)
	assert.Len(t, encKey, 32)
	assert.Len(t, macKey, 32)
	assert.NotEqual(t, encKey, macKey)
}

func TestRSAOAEPRoundtrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	plaintext := make([]byte, 64)
	if _, err := rand.Read(plaintext); err != nil {
		t.Fatalf("rand: %v", err)
	}

	// Bitwarden uses OAEP-SHA1.
	ct, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, &priv.PublicKey, plaintext, nil)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	encString := "4." + base64.StdEncoding.EncodeToString(ct)

	got, err := DecryptRSAOAEP(encString, priv)
	if err != nil {
		t.Fatalf("DecryptRSAOAEP: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round-trip mismatch")
	}
}

func TestRSAPrivateKeyFromPKCS8DER(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	parsed, err := RSAPrivateKeyFromPKCS8DER(der)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.N.Cmp(priv.N) != 0 {
		t.Fatalf("modulus mismatch")
	}
}

const genericCiphertextErr = "vaultwarden: invalid ciphertext"

func TestMACTamperRejection(t *testing.T) {
	encKey := make([]byte, 32)
	macKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 1)
	}
	for i := range macKey {
		macKey[i] = byte(i + 33)
	}

	enc, err := EncryptString("hello-from-vault", encKey, macKey)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Flip the last MAC byte: format "2.iv|ct|mac".
	parts := strings.Split(enc[2:], "|")
	macBytes, _ := base64.StdEncoding.DecodeString(parts[2])
	macBytes[len(macBytes)-1] ^= 0x01
	parts[2] = base64.StdEncoding.EncodeToString(macBytes)
	tampered := "2." + strings.Join(parts, "|")

	_, err = DecryptString(tampered, encKey, macKey)
	if err == nil {
		t.Fatalf("expected error on tampered MAC")
	}
	if err.Error() != genericCiphertextErr {
		t.Fatalf("expected generic error %q, got %q", genericCiphertextErr, err.Error())
	}
}

func TestPaddingOracleSafety(t *testing.T) {
	encKey := make([]byte, 32)
	macKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 1)
	}
	for i := range macKey {
		macKey[i] = byte(i + 33)
	}

	// Build a ciphertext with VALID MAC but invalid PKCS#7 padding:
	// AES-CBC encrypt 32 bytes whose final byte 0xFF makes PKCS#7 unpadding fail.
	iv := make([]byte, 16)
	for i := range iv {
		iv[i] = byte(i)
	}
	plaintext := make([]byte, 32)
	for i := range plaintext {
		plaintext[i] = 0xAA
	}
	plaintext[31] = 0xFF // invalid PKCS#7
	block, _ := aes.NewCipher(encKey)
	ct := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, plaintext)

	mac := hmac.New(sha256.New, macKey)
	mac.Write(iv)
	mac.Write(ct)
	macSum := mac.Sum(nil)

	enc := "2." + base64.StdEncoding.EncodeToString(iv) + "|" +
		base64.StdEncoding.EncodeToString(ct) + "|" +
		base64.StdEncoding.EncodeToString(macSum)

	_, err := DecryptString(enc, encKey, macKey)
	if err == nil {
		t.Fatalf("expected error on bad padding")
	}
	if err.Error() != genericCiphertextErr {
		t.Fatalf("padding error must match MAC error string;\n  got:  %q\n  want: %q", err.Error(), genericCiphertextErr)
	}
}
