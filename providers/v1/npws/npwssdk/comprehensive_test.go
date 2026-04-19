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

// ============================================================
// Go Encrypt → Go Decrypt Round-Trip for ALL chains
// ============================================================

func TestChain0_Rsa_Pbkdf2Sha1_AesCbc_RoundTrip(t *testing.T) {
	priv, _ := GenerateRSAKeyPair()
	pubXML := ExportRSAPublicKeyToXML(&priv.PublicKey)
	privXML := ExportRSAPrivateKeyToXML(priv)
	plaintext := []byte("Chain 0 round-trip test data with special chars: äöü €")

	encrypted, err := EncryptWithChain(ChainRsaPbkdf2Sha1AesCbc, []byte(pubXML), plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if encrypted[0] != byte(ChainRsaPbkdf2Sha1AesCbc) {
		t.Fatalf("chain ID: got %d, want %d", encrypted[0], ChainRsaPbkdf2Sha1AesCbc)
	}

	decrypted, err := DecryptWithChain([]byte(privXML), encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestChain1_Pbkdf2Sha1_AesCbc_RoundTrip(t *testing.T) {
	password := []byte("mypassword")
	plaintext := []byte("Chain 1 password-based AES-CBC test")

	encrypted, err := EncryptWithChain(ChainPbkdf2Sha1AesCbc, password, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if encrypted[0] != byte(ChainPbkdf2Sha1AesCbc) {
		t.Fatalf("chain ID: got %d, want %d", encrypted[0], ChainPbkdf2Sha1AesCbc)
	}

	decrypted, err := DecryptWithChain(password, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestChain2_Ecdh_HkdfSha256_AesGcm_RoundTrip(t *testing.T) {
	priv, _ := GenerateECDHKeyPair()
	pubBytes := priv.PublicKey().Bytes()
	privBytes := priv.Bytes()
	plaintext := []byte("Chain 2 ECDH+HKDF+AES-GCM test")

	encrypted, err := EncryptWithChain(ChainEcdhHkdfSha256AesGcm, pubBytes, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if encrypted[0] != byte(ChainEcdhHkdfSha256AesGcm) {
		t.Fatalf("chain ID: got %d, want %d", encrypted[0], ChainEcdhHkdfSha256AesGcm)
	}

	decrypted, err := DecryptWithChain(privBytes, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestChain3_Pbkdf2Sha256_AesGcm_RoundTrip(t *testing.T) {
	password := []byte("chain3password")
	plaintext := []byte("Chain 3 PBKDF2-SHA256+AES-GCM")

	encrypted, err := EncryptWithChain(ChainPbkdf2Sha256AesGcm, password, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptWithChain(password, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestChain4_AesGcm_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i * 3)
	}
	plaintext := []byte("Chain 4 plain AES-GCM")

	encrypted, err := EncryptWithChain(ChainAesGcm, key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptWithChain(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestChain5_Pbkdf2Sha256High_AesGcmPadded_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow PBKDF2 610005 iteration test")
	}
	password := []byte("highiterpassword")
	plaintext := []byte("Chain 5 high-iteration password test")

	encrypted, err := EncryptWithChain(ChainPbkdf2Sha256_610005_AesGcmPadded, password, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptWithChain(password, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestChain6_AesGcmPadded_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 10)
	}
	plaintext := []byte("Chain 6 AES-GCM with padding")

	encrypted, err := EncryptWithChain(ChainAesGcmPadded, key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptWithChain(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("mismatch: got %q, want %q", decrypted, plaintext)
	}
}

// ============================================================
// Edge cases: empty data, large data
// ============================================================

func TestChain6_EmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("")

	encrypted, err := EncryptWithChain(ChainAesGcmPadded, key, plaintext)
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}

	decrypted, err := DecryptWithChain(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestChain4_LargeData(t *testing.T) {
	key := make([]byte, 32)
	plaintext := make([]byte, 10000)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	encrypted, err := EncryptWithChain(ChainAesGcm, key, plaintext)
	if err != nil {
		t.Fatalf("encrypt large: %v", err)
	}

	decrypted, err := DecryptWithChain(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt large: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("large data mismatch")
	}
}

// ============================================================
// V1 Collection: ALL methods
// ============================================================

func TestV1_EncryptWithPublicKey_RoundTrip(t *testing.T) {
	v1 := &EncryptionCollectionV1{}
	pub, priv, _ := v1.GenerateKeyPair()
	plaintext := []byte("V1 EncryptWithPublicKey test")

	encrypted, err := v1.EncryptWithPublicKey(pub, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	decrypted, err := v1.Decrypt(priv, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("mismatch")
	}
}

func TestV1_EncryptWithKey_RoundTrip(t *testing.T) {
	v1 := &EncryptionCollectionV1{}
	key, _ := v1.GenerateSymmetricKey()
	plaintext := []byte("V1 EncryptWithKey test")

	encrypted, err := v1.EncryptWithKey(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	decrypted, err := v1.Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("mismatch")
	}
}

func TestV1_EncryptWithPassword_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow PBKDF2 test")
	}
	v1 := &EncryptionCollectionV1{}
	password := []byte("v1testpassword")
	plaintext := []byte("V1 EncryptWithPassword test")

	encrypted, err := v1.EncryptWithPassword(password, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	decrypted, err := v1.Decrypt(password, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("mismatch")
	}
}

func TestV1_SignData_VerifyData_RoundTrip(t *testing.T) {
	v1 := &EncryptionCollectionV1{}
	pub, priv, _ := v1.GenerateKeyPair()
	data := []byte("V1 sign+verify test data")

	sig, err := v1.SignData(priv, data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := v1.VerifyData(pub, data, sig); err != nil {
		t.Fatalf("verify: %v", err)
	}
	// Tampered data
	if err := v1.VerifyData(pub, []byte("tampered"), sig); err == nil {
		t.Fatal("expected verify to fail with tampered data")
	}
}

func TestV1_GenerateDataKey_IsAsymmetric(t *testing.T) {
	v1 := &EncryptionCollectionV1{}
	if !v1.AreDataKeysAsymmetric() {
		t.Fatal("V1 data keys should be asymmetric")
	}

	encKey, decKey, err := v1.GenerateDataKey()
	if err != nil {
		t.Fatalf("GenerateDataKey: %v", err)
	}
	// encKey (public) and decKey (private) should be different
	if bytes.Equal(encKey, decKey) {
		t.Fatal("V1 data keys should be different (asymmetric)")
	}
}

// ============================================================
// V2 Collection: ALL methods
// ============================================================

func TestV2_EncryptWithPublicKey_RoundTrip(t *testing.T) {
	v2 := &EncryptionCollectionV2{}
	pub, priv, _ := v2.GenerateKeyPair()
	plaintext := []byte("V2 EncryptWithPublicKey test")

	encrypted, err := v2.EncryptWithPublicKey(pub, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	ecdhPriv, _ := ecdh.P521().NewPrivateKey(priv)
	decrypted, err := v2.Decrypt(ecdhPriv.Bytes(), encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("mismatch")
	}
}

func TestV2_EncryptWithKey_RoundTrip(t *testing.T) {
	v2 := &EncryptionCollectionV2{}
	key, _ := v2.GenerateSymmetricKey()
	plaintext := []byte("V2 EncryptWithKey test")

	encrypted, err := v2.EncryptWithKey(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	decrypted, err := v2.Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("mismatch")
	}
}

func TestV2_EncryptWithPassword_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow PBKDF2 test")
	}
	v2 := &EncryptionCollectionV2{}
	password := []byte("v2testpassword")
	plaintext := []byte("V2 EncryptWithPassword test")

	encrypted, err := v2.EncryptWithPassword(password, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	decrypted, err := v2.Decrypt(password, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("mismatch")
	}
}

func TestV2_SignData_VerifyData_ECC_RoundTrip(t *testing.T) {
	v2 := &EncryptionCollectionV2{}
	pub, priv, _ := v2.GenerateKeyPair()
	data := []byte("V2 ECC sign+verify test")

	sig, err := v2.SignData(priv, data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := v2.VerifyData(pub, data, sig); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := v2.VerifyData(pub, []byte("tampered"), sig); err == nil {
		t.Fatal("expected verify to fail")
	}
}

func TestV2_SignData_VerifyData_RSA_Legacy(t *testing.T) {
	// V2 should also handle RSA keys (for legacy data)
	v2 := &EncryptionCollectionV2{}
	v1 := &EncryptionCollectionV1{}
	pub, priv, _ := v1.GenerateKeyPair() // RSA keys
	data := []byte("V2 with RSA legacy key sign+verify")

	sig, err := v2.SignData(priv, data)
	if err != nil {
		t.Fatalf("sign RSA via V2: %v", err)
	}
	if err := v2.VerifyData(pub, data, sig); err != nil {
		t.Fatalf("verify RSA via V2: %v", err)
	}
}

func TestV2_GenerateDataKey_IsSymmetric(t *testing.T) {
	v2 := &EncryptionCollectionV2{}
	if v2.AreDataKeysAsymmetric() {
		t.Fatal("V2 data keys should be symmetric")
	}

	encKey, decKey, err := v2.GenerateDataKey()
	if err != nil {
		t.Fatalf("GenerateDataKey: %v", err)
	}
	if !bytes.Equal(encKey, decKey) {
		t.Fatal("V2 data keys should be identical (symmetric)")
	}
	if len(encKey) != 32 {
		t.Fatalf("V2 data key should be 32 bytes, got %d", len(encKey))
	}
}

// ============================================================
// EncryptionManager: full dispatch tests
// ============================================================

func TestEncryptionManager_V1_FullCycle(t *testing.T) {
	em := NewEncryptionManager(EncryptionV1)

	// Key generation
	pub, priv, err := em.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	// Encrypt with public key
	plaintext := []byte("EncryptionManager V1 full cycle")
	encrypted, err := em.EncryptWithPublicKey(pub, plaintext)
	if err != nil {
		t.Fatalf("EncryptWithPublicKey: %v", err)
	}
	decrypted, err := em.Decrypt(priv, encrypted)
	if err != nil {
		t.Fatalf("Decrypt (public key): %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("public key decrypt mismatch")
	}

	// Encrypt with symmetric key
	symKey, _ := em.GenerateSymmetricKey()
	encrypted2, err := em.EncryptWithKey(symKey, plaintext)
	if err != nil {
		t.Fatalf("EncryptWithKey: %v", err)
	}
	decrypted2, err := em.Decrypt(symKey, encrypted2)
	if err != nil {
		t.Fatalf("Decrypt (symmetric): %v", err)
	}
	if !bytes.Equal(plaintext, decrypted2) {
		t.Fatal("symmetric decrypt mismatch")
	}

	// Sign and verify
	data := []byte("data for signing")
	sig, err := em.SignData(priv, data)
	if err != nil {
		t.Fatalf("SignData: %v", err)
	}
	if err := em.VerifyData(pub, data, sig); err != nil {
		t.Fatalf("VerifyData: %v", err)
	}
}

func TestEncryptionManager_V2_FullCycle(t *testing.T) {
	em := NewEncryptionManager(EncryptionV2)

	pub, priv, err := em.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	plaintext := []byte("EncryptionManager V2 full cycle")
	encrypted, err := em.EncryptWithPublicKey(pub, plaintext)
	if err != nil {
		t.Fatalf("EncryptWithPublicKey: %v", err)
	}

	// V2 needs raw ECDH private key for decryption
	ecdhPriv, _ := ecdh.P521().NewPrivateKey(priv)
	decrypted, err := em.Decrypt(ecdhPriv.Bytes(), encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("mismatch")
	}

	// Symmetric
	symKey, _ := em.GenerateSymmetricKey()
	encrypted2, err := em.EncryptWithKey(symKey, plaintext)
	if err != nil {
		t.Fatalf("EncryptWithKey: %v", err)
	}
	decrypted2, err := em.Decrypt(symKey, encrypted2)
	if err != nil {
		t.Fatalf("Decrypt symmetric: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted2) {
		t.Fatal("symmetric mismatch")
	}

	// Sign and verify
	sig, err := em.SignData(priv, plaintext)
	if err != nil {
		t.Fatalf("SignData: %v", err)
	}
	if err := em.VerifyData(pub, plaintext, sig); err != nil {
		t.Fatalf("VerifyData: %v", err)
	}
}

func TestEncryptionManager_SwitchVersion(t *testing.T) {
	em := NewEncryptionManager(EncryptionV1)
	if em.GetEncryptionVersion() != EncryptionV1 {
		t.Fatal("expected V1")
	}

	em.SetEncryptionVersion(EncryptionV2)
	if em.GetEncryptionVersion() != EncryptionV2 {
		t.Fatal("expected V2 after switch")
	}

	if em.AreDataKeysAsymmetric() {
		t.Fatal("V2 should have symmetric data keys")
	}
}

func TestEncryptionManager_EncryptContainerItem_V2(t *testing.T) {
	// V2 uses symmetric data keys, so EncryptContainerItem works directly
	em := NewEncryptionManager(EncryptionV2)

	item := &PsrContainerItem{
		ContainerItemType: ContainerItemPassword,
	}
	plaintext := []byte("container item secret value")

	result, err := em.EncryptContainerItem(item, plaintext, nil)
	if err != nil {
		t.Fatalf("EncryptContainerItem: %v", err)
	}

	if result.NewDecryptionKey == nil {
		t.Fatal("expected new decryption key for new item")
	}
	if item.Value == "" {
		t.Fatal("item.Value should be set")
	}
	if len(item.PublicKey) == 0 {
		t.Fatal("item.PublicKey should be set")
	}
	if len(result.NewDecryptionKey) != 32 {
		t.Fatalf("V2 data key should be 32 bytes, got %d", len(result.NewDecryptionKey))
	}

	// Decrypt with the symmetric key
	decrypted, err := em.Decrypt(result.NewDecryptionKey, result.EncryptedValue)
	if err != nil {
		t.Fatalf("Decrypt container item: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("container item mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptionManager_EncryptContainerItem_V1(t *testing.T) {
	// V1 uses asymmetric data keys: RSA public key encrypts, RSA private key decrypts
	em := NewEncryptionManager(EncryptionV1)

	item := &PsrContainerItem{
		ContainerItemType: ContainerItemPassword,
	}
	plaintext := []byte("V1 container item test")

	result, err := em.EncryptContainerItem(item, plaintext, nil)
	if err != nil {
		t.Fatalf("EncryptContainerItem: %v", err)
	}

	if result.NewDecryptionKey == nil {
		t.Fatal("expected new decryption key (RSA private key)")
	}
	if item.Value == "" {
		t.Fatal("item.Value should be set")
	}
	if len(item.PublicKey) == 0 {
		t.Fatal("item.PublicKey should be set")
	}

	// Decrypt with RSA private key
	decrypted, err := em.Decrypt(result.NewDecryptionKey, result.EncryptedValue)
	if err != nil {
		t.Fatalf("Decrypt V1 container item: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("mismatch: got %q, want %q", decrypted, plaintext)
	}
}

// ============================================================
// Wrong key / tampered data tests
// ============================================================

func TestChain0_WrongKey(t *testing.T) {
	priv1, _ := GenerateRSAKeyPair()
	priv2, _ := GenerateRSAKeyPair()
	pub1XML := ExportRSAPublicKeyToXML(&priv1.PublicKey)
	priv2XML := ExportRSAPrivateKeyToXML(priv2)

	encrypted, _ := EncryptWithChain(ChainRsaPbkdf2Sha1AesCbc, []byte(pub1XML), []byte("secret"))
	_, err := DecryptWithChain([]byte(priv2XML), encrypted)
	if err == nil {
		t.Fatal("expected error with wrong RSA key")
	}
}

func TestChain2_WrongKey(t *testing.T) {
	priv1, _ := GenerateECDHKeyPair()
	priv2, _ := GenerateECDHKeyPair()

	encrypted, _ := EncryptWithChain(ChainEcdhHkdfSha256AesGcm, priv1.PublicKey().Bytes(), []byte("secret"))
	_, err := DecryptWithChain(priv2.Bytes(), encrypted)
	if err == nil {
		t.Fatal("expected error with wrong ECC key")
	}
}

func TestChain6_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 1

	encrypted, _ := EncryptWithChain(ChainAesGcmPadded, key1, []byte("secret"))
	_, err := DecryptWithChain(key2, encrypted)
	if err == nil {
		t.Fatal("expected error with wrong symmetric key")
	}
}
