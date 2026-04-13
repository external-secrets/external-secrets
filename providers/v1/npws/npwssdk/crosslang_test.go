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
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

// testVectors mirrors the C# TestVectors class.
type testVectors struct {
	Plaintext    string `json:"Plaintext"`
	SymmetricKey string `json:"SymmetricKey"`
	Password     string `json:"Password"`

	RsaPublicKey  string `json:"RsaPublicKey"`
	RsaPrivateKey string `json:"RsaPrivateKey"`
	EccPublicKey  string `json:"EccPublicKey"`
	EccPrivateKey string `json:"EccPrivateKey"`

	V1EncryptWithPublicKey string `json:"V1_EncryptWithPublicKey"`
	V1EncryptWithKey       string `json:"V1_EncryptWithKey"`
	V1EncryptWithPassword  string `json:"V1_EncryptWithPassword"`

	V2EncryptWithPublicKey string `json:"V2_EncryptWithPublicKey"`
	V2EncryptWithKey       string `json:"V2_EncryptWithKey"`

	StaticEncryptWithPublicKeyRSA string `json:"Static_EncryptWithPublicKey_RSA"`
	StaticEncryptWithPublicKeyECC string `json:"Static_EncryptWithPublicKey_ECC"`
	Chain4AesGcm                  string `json:"Chain4_AesGcm"`
	Chain3Pbkdf2Sha256AesGcm      string `json:"Chain3_Pbkdf2Sha256AesGcm"`

	Chain1Pbkdf2Sha1AesCbc             string `json:"Chain1_Pbkdf2Sha1AesCbc"`
	Chain5Pbkdf2Sha256HighAesGcmPadded string `json:"Chain5_Pbkdf2Sha256HighAesGcmPadded"`
	Chain6AesGcmPadded                 string `json:"Chain6_AesGcmPadded"`
	V2EncryptWithPassword              string `json:"V2_EncryptWithPassword"`

	SignData     string `json:"SignData"`
	RsaSignature string `json:"RsaSignature"`
	EccSignature string `json:"EccSignature"`
}

func loadTestVectors(t *testing.T) *testVectors {
	t.Helper()
	data, err := os.ReadFile("testdata/test_vectors.json")
	if err != nil {
		t.Skipf("test vectors not found: %v (run the C# TestVectorGenerator first)", err)
	}
	var v testVectors
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("parsing test vectors: %v", err)
	}
	return &v
}

func b64(t *testing.T, s string) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	return data
}

// --- Cross-Language Decryption Tests ---

func TestCrossLang_V1_DecryptWithPublicKey(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	privKey := b64(t, v.RsaPrivateKey)
	ciphertext := b64(t, v.V1EncryptWithPublicKey)

	decrypted, err := DecryptWithChain(privKey, ciphertext)
	if err != nil {
		t.Fatalf("V1 DecryptWithPublicKey: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("V1 DecryptWithPublicKey: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_V1_DecryptWithKey(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	key := b64(t, v.SymmetricKey)
	ciphertext := b64(t, v.V1EncryptWithKey)

	decrypted, err := DecryptWithChain(key, ciphertext)
	if err != nil {
		t.Fatalf("V1 DecryptWithKey: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("V1 DecryptWithKey: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_V1_DecryptWithPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow PBKDF2 test in short mode")
	}
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	password := b64(t, v.Password)
	ciphertext := b64(t, v.V1EncryptWithPassword)

	decrypted, err := DecryptWithChain(password, ciphertext)
	if err != nil {
		t.Fatalf("V1 DecryptWithPassword: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("V1 DecryptWithPassword: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_V2_DecryptWithPublicKey(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	privKey := b64(t, v.EccPrivateKey)
	ciphertext := b64(t, v.V2EncryptWithPublicKey)

	decrypted, err := DecryptWithChain(privKey, ciphertext)
	if err != nil {
		t.Fatalf("V2 DecryptWithPublicKey: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("V2 DecryptWithPublicKey: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_V2_DecryptWithKey(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	key := b64(t, v.SymmetricKey)
	ciphertext := b64(t, v.V2EncryptWithKey)

	decrypted, err := DecryptWithChain(key, ciphertext)
	if err != nil {
		t.Fatalf("V2 DecryptWithKey: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("V2 DecryptWithKey: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_Static_RSA(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	privKey := b64(t, v.RsaPrivateKey)
	ciphertext := b64(t, v.StaticEncryptWithPublicKeyRSA)

	decrypted, err := DecryptWithChain(privKey, ciphertext)
	if err != nil {
		t.Fatalf("Static RSA decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Static RSA: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_Static_ECC(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	privKey := b64(t, v.EccPrivateKey)
	ciphertext := b64(t, v.StaticEncryptWithPublicKeyECC)

	decrypted, err := DecryptWithChain(privKey, ciphertext)
	if err != nil {
		t.Fatalf("Static ECC decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Static ECC: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_Chain4_AesGcm(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	key := b64(t, v.SymmetricKey)
	ciphertext := b64(t, v.Chain4AesGcm)

	decrypted, err := DecryptWithChain(key, ciphertext)
	if err != nil {
		t.Fatalf("Chain4 decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Chain4: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_Chain3_Pbkdf2Sha256AesGcm(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	password := b64(t, v.Password)
	ciphertext := b64(t, v.Chain3Pbkdf2Sha256AesGcm)

	decrypted, err := DecryptWithChain(password, ciphertext)
	if err != nil {
		t.Fatalf("Chain3 decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Chain3: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_Chain1_Pbkdf2Sha1AesCbc(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	password := b64(t, v.Password)
	ciphertext := b64(t, v.Chain1Pbkdf2Sha1AesCbc)

	decrypted, err := DecryptWithChain(password, ciphertext)
	if err != nil {
		t.Fatalf("Chain1 decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Chain1: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_Chain5_Pbkdf2Sha256HighAesGcmPadded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow PBKDF2 test in short mode")
	}
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	password := b64(t, v.Password)
	ciphertext := b64(t, v.Chain5Pbkdf2Sha256HighAesGcmPadded)

	decrypted, err := DecryptWithChain(password, ciphertext)
	if err != nil {
		t.Fatalf("Chain5 decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Chain5: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_Chain6_AesGcmPadded(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	key := b64(t, v.SymmetricKey)
	ciphertext := b64(t, v.Chain6AesGcmPadded)

	decrypted, err := DecryptWithChain(key, ciphertext)
	if err != nil {
		t.Fatalf("Chain6 decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Chain6: got %q, want %q", decrypted, plaintext)
	}
}

func TestCrossLang_V2_DecryptWithPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow PBKDF2 test in short mode")
	}
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	password := b64(t, v.Password)
	ciphertext := b64(t, v.V2EncryptWithPassword)

	decrypted, err := DecryptWithChain(password, ciphertext)
	if err != nil {
		t.Fatalf("V2 DecryptWithPassword: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("V2 password: got %q, want %q", decrypted, plaintext)
	}
}

// --- Go Encrypt → C# Decrypt Compatibility Tests ---
// These verify that Go-encrypted output can be decrypted with C# keys,
// and that the output format (MessagePack array encoding, chain prefixes) matches C#.

func TestCrossLang_GoEncrypt_V2_PublicKey_Roundtrip(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	pubKey := b64(t, v.EccPublicKey)
	privKey := b64(t, v.EccPrivateKey)

	// Go encrypts with C#'s ECC public key
	encrypted, err := EncryptWithChain(ChainEcdhHkdfSha256AesGcm, pubKey, plaintext)
	if err != nil {
		t.Fatalf("Go encrypt: %v", err)
	}

	// Verify chain prefix
	if encrypted[0] != byte(ChainEcdhHkdfSha256AesGcm) {
		t.Fatalf("Wrong chain prefix: got %d, want %d", encrypted[0], ChainEcdhHkdfSha256AesGcm)
	}

	// Verify format: MessagePack array (first byte after prefix should be 0x94 = fixarray(4))
	if encrypted[1] != 0x94 {
		t.Fatalf("MessagePack format mismatch: got 0x%02x, want 0x94 (fixarray 4). Likely using map encoding instead of array.", encrypted[1])
	}

	// Decrypt with C#'s ECC private key
	decrypted, err := DecryptWithChain(privKey, encrypted)
	if err != nil {
		t.Fatalf("Decrypt Go-encrypted data with C# key: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Roundtrip mismatch: got %q, want %q", decrypted, plaintext)
	}
	t.Logf("Go→C# ECC roundtrip OK (encrypted size: %d bytes)", len(encrypted))
}

func TestCrossLang_GoEncrypt_V1_PublicKey_Roundtrip(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	pubKey := b64(t, v.RsaPublicKey)
	privKey := b64(t, v.RsaPrivateKey)

	// Go encrypts with C#'s RSA public key
	encrypted, err := EncryptWithChain(ChainRsaPbkdf2Sha1AesCbc, pubKey, plaintext)
	if err != nil {
		t.Fatalf("Go encrypt: %v", err)
	}

	// Verify chain prefix
	if encrypted[0] != byte(ChainRsaPbkdf2Sha1AesCbc) {
		t.Fatalf("Wrong chain prefix: got %d, want %d", encrypted[0], ChainRsaPbkdf2Sha1AesCbc)
	}

	// Decrypt with C#'s RSA private key
	decrypted, err := DecryptWithChain(privKey, encrypted)
	if err != nil {
		t.Fatalf("Decrypt Go-encrypted data with C# key: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Roundtrip mismatch: got %q, want %q", decrypted, plaintext)
	}
	t.Logf("Go→C# RSA roundtrip OK (encrypted size: %d bytes)", len(encrypted))
}

func TestCrossLang_GoEncrypt_SymmetricKey_Roundtrip(t *testing.T) {
	v := loadTestVectors(t)
	plaintext := b64(t, v.Plaintext)
	key := b64(t, v.SymmetricKey)

	// Test all symmetric chains
	chains := []struct {
		name  string
		chain MtoEncryptionChain
	}{
		{"Chain4_AesGcm", ChainAesGcm},
		{"Chain6_AesGcmPadded", ChainAesGcmPadded},
	}

	for _, tc := range chains {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := EncryptWithChain(tc.chain, key, plaintext)
			if err != nil {
				t.Fatalf("Go encrypt: %v", err)
			}
			if encrypted[0] != byte(tc.chain) {
				t.Fatalf("Wrong chain prefix: got %d, want %d", encrypted[0], tc.chain)
			}
			decrypted, err := DecryptWithChain(key, encrypted)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if !bytes.Equal(decrypted, plaintext) {
				t.Fatalf("Roundtrip mismatch: got %q, want %q", decrypted, plaintext)
			}
		})
	}
}

func TestCrossLang_GoEncrypt_ECC_MessagePackArrayFormat(t *testing.T) {
	// Verify that Chain 2 (ECDH) output uses MessagePack ARRAY format, not MAP format.
	// C# uses array format; map format would add ~35 bytes of field name overhead.
	v := loadTestVectors(t)
	pubKey := b64(t, v.EccPublicKey)
	plaintext := []byte("format check")

	encrypted, err := EncryptWithChain(ChainEcdhHkdfSha256AesGcm, pubKey, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// After chain prefix byte (0x02), the MessagePack data should start with 0x94 (fixarray of 4).
	// If it starts with 0x84 (fixmap of 4) or 0xDE (map16), we're using the wrong format.
	msgpackByte := encrypted[1]
	if msgpackByte == 0x84 || msgpackByte == 0xDE {
		t.Fatalf("Chain 2 uses MessagePack MAP format (0x%02x) — must use ARRAY format (0x94) to match C#", msgpackByte)
	}
	if msgpackByte != 0x94 {
		t.Fatalf("Unexpected MessagePack header: 0x%02x, want 0x94 (fixarray 4)", msgpackByte)
	}
	t.Logf("Chain 2 correctly uses MessagePack array format (0x94), size: %d bytes", len(encrypted))
}

// --- Cross-Language Signature Verification ---

func TestCrossLang_RSA_VerifySignature(t *testing.T) {
	v := loadTestVectors(t)
	pubKey := b64(t, v.RsaPublicKey)
	data := b64(t, v.SignData)
	signature := b64(t, v.RsaSignature)

	pub, err := ParseRSAPublicKeyFromXML(string(pubKey))
	if err != nil {
		t.Fatalf("parsing RSA public key: %v", err)
	}

	if err := RSAVerify(pub, data, signature); err != nil {
		t.Fatalf("RSA signature verification failed: %v", err)
	}
}

func TestCrossLang_ECC_VerifySignature(t *testing.T) {
	v := loadTestVectors(t)
	pubKey := b64(t, v.EccPublicKey)
	data := b64(t, v.SignData)
	signature := b64(t, v.EccSignature)

	ecdhPub, err := parseECCPublicKey(pubKey)
	if err != nil {
		t.Fatalf("parsing ECC public key: %v", err)
	}

	ecdsaPub, err := ECDHPublicToECDSA(ecdhPub)
	if err != nil {
		t.Fatalf("converting ECC key: %v", err)
	}

	if err := ECDSAVerify(ecdsaPub, data, signature); err != nil {
		t.Fatalf("ECC signature verification failed: %v", err)
	}
}
