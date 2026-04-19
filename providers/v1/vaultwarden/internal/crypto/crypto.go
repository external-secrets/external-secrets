// Package crypto implements Bitwarden-compatible cryptographic operations
// (key derivation, AES-256-CBC encryption, HMAC-SHA256 verification).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
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
		return EncString{}, errors.New("invalid EncString: missing type prefix")
	}
	if typeStr != "2" {
		return EncString{}, fmt.Errorf("unsupported EncString type %q (only type 2 supported)", typeStr)
	}
	parts := strings.Split(rest, "|")
	if len(parts) != 3 {
		return EncString{}, fmt.Errorf("invalid EncString: expected 3 parts, got %d", len(parts))
	}
	iv, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return EncString{}, fmt.Errorf("invalid EncString IV: %w", err)
	}
	ct, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return EncString{}, fmt.Errorf("invalid EncString CT: %w", err)
	}
	mac, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return EncString{}, fmt.Errorf("invalid EncString MAC: %w", err)
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
		return nil, errors.New("EncString MAC validation failed")
	}

	// Decrypt AES-256-CBC
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}
	if len(es.IV) != aes.BlockSize {
		return nil, fmt.Errorf("invalid IV length %d (expected %d)", len(es.IV), aes.BlockSize)
	}
	if len(es.CT)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of AES block size")
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
		return nil, errors.New("pkcs7Unpad: empty input")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > aes.BlockSize {
		return nil, fmt.Errorf("pkcs7Unpad: invalid padding %d", padding)
	}
	if padding > len(data) {
		return nil, errors.New("pkcs7Unpad: padding larger than data")
	}
	for _, b := range data[len(data)-padding:] {
		if b != byte(padding) {
			return nil, errors.New("pkcs7Unpad: inconsistent padding bytes")
		}
	}
	return data[:len(data)-padding], nil
}
