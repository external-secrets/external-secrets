// /*
// Copyright © The ESO Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package npwssdk

import (
	"bytes"
	"crypto/ecdh"
	"testing"
)

// --- AES-GCM Round-Trip ---

func TestAesGcmRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("Hello, NPWS!")

	encrypted, err := AesGcmEncrypt(key, plaintext)
	if err != nil {
		t.Fatalf("AesGcmEncrypt: %v", err)
	}

	decrypted, err := AesGcmDecrypt(key, encrypted)
	if err != nil {
		t.Fatalf("AesGcmDecrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestAesGcmWrongKey(t *testing.T) {
	key := make([]byte, 32)
	wrongKey := make([]byte, 32)
	wrongKey[0] = 1

	encrypted, _ := AesGcmEncrypt(key, []byte("secret"))
	_, err := AesGcmDecrypt(wrongKey, encrypted)
	if err == nil {
		t.Fatal("expected error with wrong key")
	}
}

// --- AES-CBC Round-Trip ---

func TestAesCbcRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("Hello, AES-CBC!")

	encrypted, err := AesCbcEncrypt(key, plaintext)
	if err != nil {
		t.Fatalf("AesCbcEncrypt: %v", err)
	}

	decrypted, err := AesCbcDecrypt(key, encrypted)
	if err != nil {
		t.Fatalf("AesCbcDecrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

// --- RSA Round-Trip ---

func TestRsaRoundTrip(t *testing.T) {
	priv, err := GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair: %v", err)
	}

	plaintext := []byte("RSA test data")

	encrypted, err := RSAEncrypt(&priv.PublicKey, plaintext)
	if err != nil {
		t.Fatalf("RSAEncrypt: %v", err)
	}

	decrypted, err := RSADecrypt(priv, encrypted)
	if err != nil {
		t.Fatalf("RSADecrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestRsaXmlKeyRoundTrip(t *testing.T) {
	priv, err := GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair: %v", err)
	}

	pubXML := ExportRSAPublicKeyToXML(&priv.PublicKey)
	privXML := ExportRSAPrivateKeyToXML(priv)

	pub2, err := ParseRSAPublicKeyFromXML(pubXML)
	if err != nil {
		t.Fatalf("ParseRSAPublicKeyFromXML: %v", err)
	}

	priv2, err := ParseRSAPrivateKeyFromXML(privXML)
	if err != nil {
		t.Fatalf("ParseRSAPrivateKeyFromXML: %v", err)
	}

	// Encrypt with parsed public key, decrypt with parsed private key
	plaintext := []byte("XML key round-trip")
	encrypted, err := RSAEncrypt(pub2, plaintext)
	if err != nil {
		t.Fatalf("RSAEncrypt with parsed key: %v", err)
	}
	decrypted, err := RSADecrypt(priv2, encrypted)
	if err != nil {
		t.Fatalf("RSADecrypt with parsed key: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("round-trip failed")
	}
}

func TestRsaSignVerify(t *testing.T) {
	priv, _ := GenerateRSAKeyPair()
	data := []byte("data to sign")

	sig, err := RSASign(priv, data)
	if err != nil {
		t.Fatalf("RSASign: %v", err)
	}

	if err := RSAVerify(&priv.PublicKey, data, sig); err != nil {
		t.Fatalf("RSAVerify: %v", err)
	}

	// Tampered data should fail
	if err := RSAVerify(&priv.PublicKey, []byte("tampered"), sig); err == nil {
		t.Fatal("expected verification to fail with tampered data")
	}
}

// --- ECC Round-Trip ---

func TestEcdhSharedSecret(t *testing.T) {
	priv1, _ := GenerateECDHKeyPair()
	priv2, _ := GenerateECDHKeyPair()

	secret1, err := ECDHSharedSecret(priv1, priv2.PublicKey())
	if err != nil {
		t.Fatalf("ECDHSharedSecret 1→2: %v", err)
	}

	secret2, err := ECDHSharedSecret(priv2, priv1.PublicKey())
	if err != nil {
		t.Fatalf("ECDHSharedSecret 2→1: %v", err)
	}

	if !bytes.Equal(secret1, secret2) {
		t.Fatal("shared secrets don't match")
	}
}

func TestEcdsaSignVerify(t *testing.T) {
	priv, _ := GenerateECDHKeyPair()
	ecdsaPriv, _ := ECDHPrivateToECDSA(priv)
	ecdsaPub, _ := ECDHPublicToECDSA(priv.PublicKey())

	data := []byte("ECDSA test data")

	sig, err := ECDSASign(ecdsaPriv, data)
	if err != nil {
		t.Fatalf("ECDSASign: %v", err)
	}

	if len(sig) != eccP521SignatureSize {
		t.Fatalf("signature length: got %d, want %d", len(sig), eccP521SignatureSize)
	}

	if err := ECDSAVerify(ecdsaPub, data, sig); err != nil {
		t.Fatalf("ECDSAVerify: %v", err)
	}
}

// --- Padding Round-Trip ---

func TestMtoPadUnpad(t *testing.T) {
	tests := []int{0, 1, 40, 127, 128, 200, 256}
	for _, size := range tests {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}

		padded, err := MtoPad(data, 128)
		if err != nil {
			t.Fatalf("MtoPad(%d): %v", size, err)
		}

		if len(padded)%128 != 0 {
			t.Fatalf("MtoPad(%d): not aligned: %d", size, len(padded))
		}

		unpadded, err := MtoUnpad(padded)
		if err != nil {
			t.Fatalf("MtoUnpad(%d): %v", size, err)
		}

		if !bytes.Equal(data, unpadded) {
			t.Fatalf("pad round-trip(%d): data mismatch", size)
		}
	}
}

// --- Encryption Chain Round-Trips ---

func TestChainRsaRoundTrip(t *testing.T) {
	priv, _ := GenerateRSAKeyPair()
	pubXML := ExportRSAPublicKeyToXML(&priv.PublicKey)
	privXML := ExportRSAPrivateKeyToXML(priv)

	plaintext := []byte("chain0 RSA test")

	encrypted, err := EncryptWithChain(ChainRsaPbkdf2Sha1AesCbc, []byte(pubXML), plaintext)
	if err != nil {
		t.Fatalf("chain0 encrypt: %v", err)
	}

	decrypted, err := DecryptWithChain([]byte(privXML), encrypted)
	if err != nil {
		t.Fatalf("chain0 decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("chain0 round-trip failed")
	}
}

func TestChainEcdhRoundTrip(t *testing.T) {
	priv, _ := GenerateECDHKeyPair()
	pubBytes := priv.PublicKey().Bytes()
	privBytes := priv.Bytes()

	plaintext := []byte("chain2 ECDH test")

	encrypted, err := EncryptWithChain(ChainEcdhHkdfSha256AesGcm, pubBytes, plaintext)
	if err != nil {
		t.Fatalf("chain2 encrypt: %v", err)
	}

	decrypted, err := DecryptWithChain(privBytes, encrypted)
	if err != nil {
		t.Fatalf("chain2 decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("chain2 round-trip failed")
	}
}

func TestChainAesGcmPaddedRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("chain6 AES-GCM padded test")

	encrypted, err := EncryptWithChain(ChainAesGcmPadded, key, plaintext)
	if err != nil {
		t.Fatalf("chain6 encrypt: %v", err)
	}

	decrypted, err := DecryptWithChain(key, encrypted)
	if err != nil {
		t.Fatalf("chain6 decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("chain6 round-trip failed")
	}
}

func TestChainPbkdf2Sha256HighRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow PBKDF2 test in short mode")
	}

	password := []byte("mysecretpassword")
	plaintext := []byte("chain5 password test")

	encrypted, err := EncryptWithChain(ChainPbkdf2Sha256_610005_AesGcmPadded, password, plaintext)
	if err != nil {
		t.Fatalf("chain5 encrypt: %v", err)
	}

	decrypted, err := DecryptWithChain(password, encrypted)
	if err != nil {
		t.Fatalf("chain5 decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("chain5 round-trip failed")
	}
}

// --- Key Type Detection ---

func TestDetectKeyType(t *testing.T) {
	// Symmetric key
	sym := make([]byte, 32)
	if DetectKeyType(sym) != KeyTypeSymmetric {
		t.Fatal("expected symmetric key type")
	}

	// RSA key starts with "PF" (base64 XML starts with <RSAKeyValue)
	rsaKey := []byte{80, 70, 1, 2, 3}
	if DetectKeyType(rsaKey) != KeyTypeRSA {
		t.Fatal("expected RSA key type")
	}

	// ECC public key
	eccPub := []byte{146, 196, 1, 2}
	if DetectKeyType(eccPub) != KeyTypeECCPublic {
		t.Fatal("expected ECC public key type")
	}

	// ECC private key
	eccPriv := []byte{147, 196, 1, 2}
	if DetectKeyType(eccPriv) != KeyTypeECCPrivate {
		t.Fatal("expected ECC private key type")
	}
}

// --- Encryption Collection Round-Trips ---

func TestEncryptionCollectionV1RoundTrip(t *testing.T) {
	v1 := &EncryptionCollectionV1{}

	pub, priv, err := v1.GenerateKeyPair()
	if err != nil {
		t.Fatalf("V1 GenerateKeyPair: %v", err)
	}

	plaintext := []byte("V1 collection test")

	encrypted, err := v1.EncryptWithPublicKey(pub, plaintext)
	if err != nil {
		t.Fatalf("V1 EncryptWithPublicKey: %v", err)
	}

	decrypted, err := v1.Decrypt(priv, encrypted)
	if err != nil {
		t.Fatalf("V1 Decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("V1 round-trip failed")
	}
}

func TestEncryptionCollectionV2RoundTrip(t *testing.T) {
	v2 := &EncryptionCollectionV2{}

	pub, priv, err := v2.GenerateKeyPair()
	if err != nil {
		t.Fatalf("V2 GenerateKeyPair: %v", err)
	}

	plaintext := []byte("V2 collection test")

	encrypted, err := v2.EncryptWithPublicKey(pub, plaintext)
	if err != nil {
		t.Fatalf("V2 EncryptWithPublicKey: %v", err)
	}

	// V2 ECDH: need raw ECDH private key bytes for decryption
	_ = priv
	ecdhPriv, _ := ecdh.P521().NewPrivateKey(priv)
	decrypted, err := v2.Decrypt(ecdhPriv.Bytes(), encrypted)
	if err != nil {
		t.Fatalf("V2 Decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("V2 round-trip failed")
	}
}

func TestEncryptionCollectionV1SymmetricRoundTrip(t *testing.T) {
	v1 := &EncryptionCollectionV1{}

	key, err := v1.GenerateSymmetricKey()
	if err != nil {
		t.Fatalf("V1 GenerateSymmetricKey: %v", err)
	}

	plaintext := []byte("V1 symmetric test")

	encrypted, err := v1.EncryptWithKey(key, plaintext)
	if err != nil {
		t.Fatalf("V1 EncryptWithKey: %v", err)
	}

	decrypted, err := v1.Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("V1 Decrypt symmetric: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("V1 symmetric round-trip failed")
	}
}
