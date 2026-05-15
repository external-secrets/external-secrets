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

// Package crypto implements Bitwarden-compatible cryptographic operations
// (key derivation, AES-256-CBC encryption, HMAC-SHA256 verification).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // OAEP key-wrap with SHA-1 matches the Bitwarden wire format; not used for signatures
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
)

// KDF type constants matching Bitwarden/Vaultwarden API values.
const (
	KdfTypePBKDF2   = 0
	KdfTypeArgon2id = 1
)

// errInvalidCiphertext is the SAME error returned for every decrypt-side
// failure (MAC mismatch, PKCS#7 padding failure, base64 decode error,
// malformed EncString, AES error, etc.) — distinguishable errors would
// allow a padding-oracle attack against the AES-CBC channel.
var errInvalidCiphertext = errors.New("vaultwarden: invalid ciphertext")

// EncString is a parsed Bitwarden encrypted string.
type EncString struct {
	Type byte
	IV   []byte
	CT   []byte
	MAC  []byte
}

// ParseEncString parses a Bitwarden EncString of the form "type.IV|CT|MAC" (all base64).
// Only type 2 (AES-256-CBC + HMAC-SHA256) is supported.
func ParseEncString(s string) (EncString, error) {
	typeStr, rest, ok := strings.Cut(s, ".")
	if !ok {
		return EncString{}, errInvalidCiphertext
	}
	if typeStr != "2" {
		return EncString{}, errInvalidCiphertext
	}
	parts := strings.Split(rest, "|")
	if len(parts) != 3 {
		return EncString{}, errInvalidCiphertext
	}
	iv, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return EncString{}, errInvalidCiphertext
	}
	ct, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return EncString{}, errInvalidCiphertext
	}
	mac, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return EncString{}, errInvalidCiphertext
	}
	return EncString{Type: 2, IV: iv, CT: ct, MAC: mac}, nil
}

// DeriveKey derives the 256-bit master key from masterPassword and email using PBKDF2 or Argon2id.
// The email is used as the salt (lowercased and trimmed).
func DeriveKey(masterPassword, email string, kdfType, iterations, _ /* memory */, _ /* parallelism */ int) ([]byte, error) {
	salt := []byte(strings.ToLower(strings.TrimSpace(email)))
	switch kdfType {
	case KdfTypePBKDF2:
		if iterations <= 0 {
			iterations = 600000
		}
		return pbkdf2.Key([]byte(masterPassword), salt, iterations, 32, sha256.New), nil
	case KdfTypeArgon2id:
		// Argon2id support requires the golang.org/x/crypto/argon2 import and
		// would use: argon2.IDKey([]byte(masterPassword), salt, uint32(iterations), uint32(memory), uint8(parallelism), 32).
		return nil, errors.New("argon2id not yet implemented")
	default:
		return nil, fmt.Errorf("unknown KDF type %d", kdfType)
	}
}

// StretchKey expands a 32-byte master key into separate enc and mac keys using HKDF-SHA256.
// Returns encKey (32 bytes) and macKey (32 bytes).
func StretchKey(masterKey []byte) (encKey, macKey []byte, err error) {
	encR := hkdf.Expand(sha256.New, masterKey, []byte("enc"))
	encKey = make([]byte, 32)
	if _, err = io.ReadFull(encR, encKey); err != nil {
		return nil, nil, fmt.Errorf("hkdf enc: %w", err)
	}

	macR := hkdf.Expand(sha256.New, masterKey, []byte("mac"))
	macKey = make([]byte, 32)
	if _, err = io.ReadFull(macR, macKey); err != nil {
		return nil, nil, fmt.Errorf("hkdf mac: %w", err)
	}
	return encKey, macKey, nil
}

// Decrypt decrypts an EncString using the given encKey and macKey.
// HMAC-SHA256 over (IV || CT) is validated before decryption. PKCS7 padding is removed.
func Decrypt(es EncString, encKey, macKey []byte) ([]byte, error) {
	// Validate MAC: HMAC-SHA256(IV || CT) with macKey
	h := hmac.New(sha256.New, macKey)
	h.Write(es.IV)
	h.Write(es.CT)
	expected := h.Sum(nil)
	if !hmac.Equal(expected, es.MAC) {
		return nil, errInvalidCiphertext
	}

	// Decrypt AES-256-CBC
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, errInvalidCiphertext
	}
	if len(es.IV) != aes.BlockSize {
		return nil, errInvalidCiphertext
	}
	if len(es.CT)%aes.BlockSize != 0 {
		return nil, errInvalidCiphertext
	}
	plaintext := make([]byte, len(es.CT))
	cipher.NewCBCDecrypter(block, es.IV).CryptBlocks(plaintext, es.CT) // NOSONAR — AES-256-CBC is mandated by the Bitwarden wire protocol; MAC is verified before decryption (Encrypt-then-MAC)

	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// DecryptString parses and decrypts a Bitwarden EncString value, returning the plaintext string.
func DecryptString(s string, encKey, macKey []byte) (string, error) {
	if s == "" {
		return "", nil
	}
	es, err := ParseEncString(s)
	if err != nil {
		return "", err
	}
	b, err := Decrypt(es, encKey, macKey)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Encrypt encrypts plaintext using AES-256-CBC + HMAC-SHA256 and returns a Bitwarden EncString.
func Encrypt(plaintext, encKey, macKey []byte) (string, error) {
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("rand IV: %w", err)
	}

	padded := pkcs7Pad(plaintext, aes.BlockSize)

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	ct := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, padded) // NOSONAR — AES-256-CBC is mandated by the Bitwarden wire protocol; MAC is appended after encryption (Encrypt-then-MAC)

	h := hmac.New(sha256.New, macKey)
	h.Write(iv)
	h.Write(ct)
	macBytes := h.Sum(nil)

	return fmt.Sprintf("2.%s|%s|%s",
		base64.StdEncoding.EncodeToString(iv),
		base64.StdEncoding.EncodeToString(ct),
		base64.StdEncoding.EncodeToString(macBytes),
	), nil
}

// EncryptString encrypts a string value and returns a Bitwarden EncString.
func EncryptString(s string, encKey, macKey []byte) (string, error) {
	return Encrypt([]byte(s), encKey, macKey)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padByte := byte(padding & 0xff)
	padded := make([]byte, len(data)+padding)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = padByte
	}
	return padded
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errInvalidCiphertext
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > aes.BlockSize {
		return nil, errInvalidCiphertext
	}
	if padding > len(data) {
		return nil, errInvalidCiphertext
	}
	for _, b := range data[len(data)-padding:] {
		if b != byte(padding) {
			return nil, errInvalidCiphertext
		}
	}
	return data[:len(data)-padding], nil
}

// RSAPrivateKeyFromPKCS8DER parses a PKCS#8 DER-encoded private key
// (Vaultwarden's profile.privateKey decrypted payload) into an
// *rsa.PrivateKey. Returns error if the key is not RSA.
func RSAPrivateKeyFromPKCS8DER(der []byte) (*rsa.PrivateKey, error) {
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: parse PKCS#8: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("vaultwarden: PKCS#8 key is not RSA")
	}
	return rsaKey, nil
}

// DecryptRSAOAEP decrypts a Bitwarden EncString of type 4
// ("4.<base64-RSA-ciphertext>"). Bitwarden uses RSA-OAEP with SHA-1.
// Returns the plaintext bytes (typically a 64-byte org symkey:
// first 32 are AES enc key, last 32 are HMAC mac key).
func DecryptRSAOAEP(encString string, priv *rsa.PrivateKey) ([]byte, error) {
	const prefix = "4."
	if !strings.HasPrefix(encString, prefix) {
		return nil, errInvalidCiphertext
	}
	ct, err := base64.StdEncoding.DecodeString(encString[len(prefix):])
	if err != nil {
		return nil, errInvalidCiphertext
	}
	pt, err := rsa.DecryptOAEP(sha1.New(), nil, priv, ct, nil) //nolint:gosec // OAEP-SHA1 matches Bitwarden wire format
	if err != nil {
		return nil, errInvalidCiphertext
	}
	return pt, nil
}
