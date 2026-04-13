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
	"crypto/rand"
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

// MtoEncryptionChain identifies the encryption chain used.
type MtoEncryptionChain byte

// ChainRsaPbkdf2Sha1AesCbc and related constants identify encryption chains.
const (
	ChainRsaPbkdf2Sha1AesCbc              MtoEncryptionChain = 0
	ChainPbkdf2Sha1AesCbc                 MtoEncryptionChain = 1
	ChainEcdhHkdfSha256AesGcm             MtoEncryptionChain = 2
	ChainPbkdf2Sha256AesGcm               MtoEncryptionChain = 3
	ChainAesGcm                           MtoEncryptionChain = 4
	ChainPbkdf2Sha256_610005_AesGcmPadded MtoEncryptionChain = 5 //nolint:revive // matches C# constant name with iteration count
	ChainAesGcmPadded                     MtoEncryptionChain = 6
)

// separator used in legacy RSA/password chains to delimit fields.
var chainSeparator = []byte{1, 5, 0, 5}

const (
	pbkdf2Sha1Iterations   = 1000
	pbkdf2Sha256Iterations = 1000
	pbkdf2HighIterations   = 610005
	pbkdf2DefaultSaltSize  = 16
	pbkdf2DefaultKeySize   = 32
)

// eccEncryptedPayload is MessagePack-serialized for the ECDH chain.
// Uses array encoding to match C# MtoEncryptionLibV2 format (positional, not named keys).
type eccEncryptedPayload struct {
	_msgpack       struct{} `msgpack:",as_array"` //nolint:unused // required by msgpack for array encoding
	EccPublicKey   []byte
	Salt           []byte
	AesIV          []byte
	EncryptedValue []byte
}

// --- Chain 0: RSA + PBKDF2-SHA1 + AES-CBC ---

// chainRsaEncrypt: generate random, RSA-encrypt it, derive AES key with PBKDF2, AES-CBC encrypt.
func chainRsaEncrypt(publicKeyXML, plaintext []byte) ([]byte, error) {
	pub, err := ParseRSAPublicKeyFromXML(string(publicKeyXML))
	if err != nil {
		return nil, fmt.Errorf("chain0 encrypt: %w", err)
	}

	// Generate 32 random bytes
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return nil, err
	}

	// Generate salt for PBKDF2
	salt := make([]byte, pbkdf2DefaultSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// Derive AES key from random bytes via PBKDF2-SHA1
	aesKey := Pbkdf2SHA1(random, salt, pbkdf2Sha1Iterations, pbkdf2DefaultKeySize)

	// RSA-encrypt the random bytes
	encryptedRandom, err := RSAEncrypt(pub, random)
	if err != nil {
		return nil, fmt.Errorf("chain0 encrypt: RSA: %w", err)
	}

	// AES-CBC encrypt
	encrypted, err := AesCbcEncrypt(aesKey, plaintext)
	if err != nil {
		return nil, fmt.Errorf("chain0 encrypt: AES: %w", err)
	}

	iv := encrypted[:aesCbcIVSize]
	ciphertext := encrypted[aesCbcIVSize:]

	// Format: chain_id | separator | salt | separator | rsaCipher | separator | iv | separator | ciphertext
	var buf bytes.Buffer
	buf.WriteByte(byte(ChainRsaPbkdf2Sha1AesCbc))
	buf.Write(chainSeparator)
	buf.Write(salt)
	buf.Write(chainSeparator)
	buf.Write(encryptedRandom)
	buf.Write(chainSeparator)
	buf.Write(iv)
	buf.Write(chainSeparator)
	buf.Write(ciphertext)
	return buf.Bytes(), nil
}

// chainRsaDecrypt: split fields, RSA-decrypt random, derive AES key, AES-CBC decrypt.
func chainRsaDecrypt(privateKeyXML, data []byte) ([]byte, error) {
	priv, err := ParseRSAPrivateKeyFromXML(string(privateKeyXML))
	if err != nil {
		return nil, fmt.Errorf("chain0 decrypt: %w", err)
	}

	// Skip chain ID byte, then split by separator.
	// Format: sep + salt + sep + rsaCipher + sep + iv + sep + ciphertext
	// First element after split is empty (before first separator).
	parts := splitBySeparator(data[1:], chainSeparator)
	// Filter empty leading part
	var filtered [][]byte
	for _, p := range parts {
		if len(p) > 0 || len(filtered) > 0 {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) < 4 {
		return nil, fmt.Errorf("chain0 decrypt: expected 4 parts, got %d", len(filtered))
	}

	salt := filtered[0]
	encryptedRandom := filtered[1]
	iv := filtered[2]
	ciphertext := filtered[3]

	// RSA-decrypt to get random bytes
	random, err := RSADecrypt(priv, encryptedRandom)
	if err != nil {
		return nil, fmt.Errorf("chain0 decrypt: RSA: %w", err)
	}

	// Derive AES key
	aesKey := Pbkdf2SHA1(random, salt, pbkdf2Sha1Iterations, pbkdf2DefaultKeySize)

	// AES-CBC decrypt
	return AesCbcDecryptWithIV(aesKey, iv, ciphertext)
}

// --- Chain 1: PBKDF2-SHA1 + AES-CBC (password-based) ---

func chainPbkdf2Sha1AesCbcEncrypt(password, plaintext []byte) ([]byte, error) {
	salt := make([]byte, pbkdf2DefaultSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	aesKey := Pbkdf2SHA1(password, salt, pbkdf2Sha1Iterations, pbkdf2DefaultKeySize)
	encrypted, err := AesCbcEncrypt(aesKey, plaintext)
	if err != nil {
		return nil, err
	}

	iv := encrypted[:aesCbcIVSize]
	ciphertext := encrypted[aesCbcIVSize:]

	var buf bytes.Buffer
	buf.WriteByte(byte(ChainPbkdf2Sha1AesCbc))
	buf.Write(chainSeparator)
	buf.Write(salt)
	buf.Write(chainSeparator)
	buf.Write(iv)
	buf.Write(chainSeparator)
	buf.Write(ciphertext)
	return buf.Bytes(), nil
}

func chainPbkdf2Sha1AesCbcDecrypt(password, data []byte) ([]byte, error) {
	parts := splitBySeparator(data[1:], chainSeparator)
	var filtered [][]byte
	for _, p := range parts {
		if len(p) > 0 || len(filtered) > 0 {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) < 3 {
		return nil, fmt.Errorf("chain1 decrypt: expected 3 parts, got %d", len(filtered))
	}

	salt := filtered[0]
	iv := filtered[1]
	ciphertext := filtered[2]

	aesKey := Pbkdf2SHA1(password, salt, pbkdf2Sha1Iterations, pbkdf2DefaultKeySize)
	return AesCbcDecryptWithIV(aesKey, iv, ciphertext)
}

// --- Chain 2: ECDH + HKDF-SHA256 + AES-GCM ---

func chainEcdhEncrypt(recipientPubKeyBytes, plaintext []byte) ([]byte, error) {
	var recipientPub *ecdh.PublicKey
	var err error

	// Try MessagePack format first (C# serialized: [X, Y]), then raw ECDH format
	if IsECCKey(recipientPubKeyBytes) {
		recipientPub, err = parseECCPublicKey(recipientPubKeyBytes)
	} else {
		recipientPub, err = ecdh.P521().NewPublicKey(recipientPubKeyBytes)
	}
	if err != nil {
		return nil, fmt.Errorf("chain2 encrypt: parsing public key: %w", err)
	}

	// Generate ephemeral key pair
	ephPriv, err := GenerateECDHKeyPair()
	if err != nil {
		return nil, fmt.Errorf("chain2 encrypt: generating ephemeral key: %w", err)
	}

	// ECDH shared secret
	shared, err := ECDHSharedSecret(ephPriv, recipientPub)
	if err != nil {
		return nil, fmt.Errorf("chain2 encrypt: %w", err)
	}

	// HKDF-SHA256 to derive AES key
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	aesKey, err := HkdfSHA256(shared, salt, 32)
	if err != nil {
		return nil, fmt.Errorf("chain2 encrypt: %w", err)
	}

	// AES-GCM encrypt
	encrypted, err := AesGcmEncrypt(aesKey, plaintext)
	if err != nil {
		return nil, fmt.Errorf("chain2 encrypt: %w", err)
	}

	// Serialize ephemeral public key to C# MessagePack [X, Y] format
	ephPubSerialized, err := serializeECCPublicKey(ephPriv.PublicKey())
	if err != nil {
		return nil, fmt.Errorf("chain2 encrypt: serializing ephemeral key: %w", err)
	}

	// Serialize with MessagePack
	payload := eccEncryptedPayload{
		EccPublicKey:   ephPubSerialized,
		Salt:           salt,
		AesIV:          encrypted[:aesGcmIVSize],
		EncryptedValue: encrypted[aesGcmIVSize:],
	}

	result, err := msgpack.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("chain2 encrypt: msgpack: %w", err)
	}

	// Prepend chain ID
	return append([]byte{byte(ChainEcdhHkdfSha256AesGcm)}, result...), nil
}

func chainEcdhDecrypt(privateKeyBytes, data []byte) ([]byte, error) {
	var priv *ecdh.PrivateKey
	var err error

	// Try MessagePack format first (C# serialized: [D, X, Y]), then raw ECDH format
	if IsECCKey(privateKeyBytes) {
		priv, err = parseECCPrivateKey(privateKeyBytes)
	} else {
		priv, err = ecdh.P521().NewPrivateKey(privateKeyBytes)
	}
	if err != nil {
		return nil, fmt.Errorf("chain2 decrypt: parsing private key: %w", err)
	}

	var payload eccEncryptedPayload
	if err := msgpack.Unmarshal(data[1:], &payload); err != nil {
		return nil, fmt.Errorf("chain2 decrypt: msgpack: %w", err)
	}

	var ephPub *ecdh.PublicKey
	if IsECCKey(payload.EccPublicKey) {
		ephPub, err = parseECCPublicKey(payload.EccPublicKey)
	} else {
		ephPub, err = ecdh.P521().NewPublicKey(payload.EccPublicKey)
	}
	if err != nil {
		return nil, fmt.Errorf("chain2 decrypt: parsing ephemeral public key: %w", err)
	}

	shared, err := ECDHSharedSecret(priv, ephPub)
	if err != nil {
		return nil, fmt.Errorf("chain2 decrypt: %w", err)
	}

	aesKey, err := HkdfSHA256(shared, payload.Salt, 32)
	if err != nil {
		return nil, fmt.Errorf("chain2 decrypt: %w", err)
	}

	// Reconstruct IV || ciphertext+tag
	ciphertext := append([]byte{}, payload.AesIV...)
	ciphertext = append(ciphertext, payload.EncryptedValue...)
	return AesGcmDecrypt(aesKey, ciphertext)
}

// --- Chain 5: PBKDF2-SHA256 (610005 iterations) + AES-GCM + Padding ---

func chainPbkdf2Sha256HighAesGcmPaddedEncrypt(password, plaintext []byte) ([]byte, error) {
	salt := make([]byte, pbkdf2DefaultSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	aesKey := Pbkdf2SHA256(password, salt, pbkdf2HighIterations, pbkdf2DefaultKeySize)

	padded, err := MtoPad(plaintext, defaultPaddingBlockSize)
	if err != nil {
		return nil, err
	}

	encrypted, err := AesGcmEncrypt(aesKey, padded)
	if err != nil {
		return nil, err
	}

	payload := aesGcmWithSaltPayload{
		Salt:           salt,
		AesIV:          encrypted[:aesGcmIVSize],
		EncryptedValue: encrypted[aesGcmIVSize:],
	}
	packed, err := msgpack.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return append([]byte{byte(ChainPbkdf2Sha256_610005_AesGcmPadded)}, packed...), nil
}

func chainPbkdf2Sha256HighAesGcmPaddedDecrypt(password, data []byte) ([]byte, error) {
	var payload aesGcmWithSaltPayload
	if err := msgpack.Unmarshal(data[1:], &payload); err != nil {
		return nil, fmt.Errorf("chain5 decrypt: unmarshal: %w", err)
	}

	aesKey := Pbkdf2SHA256(password, payload.Salt, pbkdf2HighIterations, pbkdf2DefaultKeySize)
	combined := append([]byte{}, payload.AesIV...)
	combined = append(combined, payload.EncryptedValue...)

	padded, err := AesGcmDecrypt(aesKey, combined)
	if err != nil {
		return nil, err
	}

	return MtoUnpad(padded)
}

// aesGcmPayload is the MessagePack format used by C# for AES-GCM encrypted data.
// Uses array encoding to match C# SymmetricAesRightKey: [AesIv, EncryptedValue].
type aesGcmPayload struct {
	_msgpack       struct{} `msgpack:",as_array"` //nolint:unused // required by msgpack for array encoding
	AesIV          []byte
	EncryptedValue []byte
}

// aesGcmWithSaltPayload is the MessagePack format for chains that include a salt.
// Uses array encoding to match C# format: [Salt, AesIv, EncryptedValue].
type aesGcmWithSaltPayload struct {
	_msgpack       struct{} `msgpack:",as_array"` //nolint:unused // required by msgpack for array encoding
	Salt           []byte
	AesIV          []byte
	EncryptedValue []byte
}

// marshalAesGcmPayload wraps IV and ciphertext+tag in MessagePack format.
func marshalAesGcmPayload(encrypted []byte) ([]byte, error) {
	payload := aesGcmPayload{
		AesIV:          encrypted[:aesGcmIVSize],
		EncryptedValue: encrypted[aesGcmIVSize:],
	}
	return msgpack.Marshal(payload)
}

// unmarshalAesGcmPayload extracts IV and ciphertext from MessagePack format.
func unmarshalAesGcmPayload(data []byte) ([]byte, error) {
	var payload aesGcmPayload
	if err := msgpack.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal AES-GCM payload: %w", err)
	}
	// Reconstruct IV || ciphertext+tag
	return append(payload.AesIV, payload.EncryptedValue...), nil
}

// --- Chain 6: AES-GCM + Padding ---

func chainAesGcmPaddedEncrypt(key, plaintext []byte) ([]byte, error) {
	padded, err := MtoPad(plaintext, defaultPaddingBlockSize)
	if err != nil {
		return nil, err
	}

	encrypted, err := AesGcmEncrypt(key, padded)
	if err != nil {
		return nil, err
	}

	packed, err := marshalAesGcmPayload(encrypted)
	if err != nil {
		return nil, err
	}

	return append([]byte{byte(ChainAesGcmPadded)}, packed...), nil
}

func chainAesGcmPaddedDecrypt(key, data []byte) ([]byte, error) {
	combined, err := unmarshalAesGcmPayload(data[1:])
	if err != nil {
		return nil, fmt.Errorf("chain6 decrypt: %w", err)
	}

	padded, err := AesGcmDecrypt(key, combined)
	if err != nil {
		return nil, err
	}

	return MtoUnpad(padded)
}

// --- Chain 4: Plain AES-GCM (no padding) ---

func chainAesGcmEncrypt(key, plaintext []byte) ([]byte, error) {
	encrypted, err := AesGcmEncrypt(key, plaintext)
	if err != nil {
		return nil, err
	}

	packed, err := marshalAesGcmPayload(encrypted)
	if err != nil {
		return nil, err
	}

	return append([]byte{byte(ChainAesGcm)}, packed...), nil
}

func chainAesGcmDecrypt(key, data []byte) ([]byte, error) {
	combined, err := unmarshalAesGcmPayload(data[1:])
	if err != nil {
		return nil, fmt.Errorf("chain4 decrypt: %w", err)
	}
	return AesGcmDecrypt(key, combined)
}

// --- Chain 3: PBKDF2-SHA256 (1000 iterations) + AES-GCM ---

func chainPbkdf2Sha256AesGcmEncrypt(password, plaintext []byte) ([]byte, error) {
	salt := make([]byte, pbkdf2DefaultSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	aesKey := Pbkdf2SHA256(password, salt, pbkdf2Sha256Iterations, pbkdf2DefaultKeySize)

	encrypted, err := AesGcmEncrypt(aesKey, plaintext)
	if err != nil {
		return nil, err
	}

	payload := aesGcmWithSaltPayload{
		Salt:           salt,
		AesIV:          encrypted[:aesGcmIVSize],
		EncryptedValue: encrypted[aesGcmIVSize:],
	}
	packed, err := msgpack.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return append([]byte{byte(ChainPbkdf2Sha256AesGcm)}, packed...), nil
}

func chainPbkdf2Sha256AesGcmDecrypt(password, data []byte) ([]byte, error) {
	var payload aesGcmWithSaltPayload
	if err := msgpack.Unmarshal(data[1:], &payload); err != nil {
		return nil, fmt.Errorf("chain3 decrypt: unmarshal: %w", err)
	}

	aesKey := Pbkdf2SHA256(password, payload.Salt, pbkdf2Sha256Iterations, pbkdf2DefaultKeySize)
	combined := append([]byte{}, payload.AesIV...)
	combined = append(combined, payload.EncryptedValue...)
	return AesGcmDecrypt(aesKey, combined)
}

// EncryptWithChain encrypts using the specified chain.
func EncryptWithChain(chain MtoEncryptionChain, key, plaintext []byte) ([]byte, error) {
	switch chain {
	case ChainRsaPbkdf2Sha1AesCbc:
		return chainRsaEncrypt(key, plaintext)
	case ChainPbkdf2Sha1AesCbc:
		return chainPbkdf2Sha1AesCbcEncrypt(key, plaintext)
	case ChainEcdhHkdfSha256AesGcm:
		return chainEcdhEncrypt(key, plaintext)
	case ChainPbkdf2Sha256AesGcm:
		return chainPbkdf2Sha256AesGcmEncrypt(key, plaintext)
	case ChainAesGcm:
		return chainAesGcmEncrypt(key, plaintext)
	case ChainPbkdf2Sha256_610005_AesGcmPadded:
		return chainPbkdf2Sha256HighAesGcmPaddedEncrypt(key, plaintext)
	case ChainAesGcmPadded:
		return chainAesGcmPaddedEncrypt(key, plaintext)
	default:
		return nil, fmt.Errorf("unknown encryption chain: %d", chain)
	}
}

// DecryptWithChain decrypts data by detecting the chain from the first byte.
func DecryptWithChain(key, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("decrypt: empty data")
	}

	chain := MtoEncryptionChain(data[0])
	switch chain {
	case ChainRsaPbkdf2Sha1AesCbc:
		return chainRsaDecrypt(key, data)
	case ChainPbkdf2Sha1AesCbc:
		return chainPbkdf2Sha1AesCbcDecrypt(key, data)
	case ChainEcdhHkdfSha256AesGcm:
		return chainEcdhDecrypt(key, data)
	case ChainPbkdf2Sha256AesGcm:
		return chainPbkdf2Sha256AesGcmDecrypt(key, data)
	case ChainAesGcm:
		return chainAesGcmDecrypt(key, data)
	case ChainPbkdf2Sha256_610005_AesGcmPadded:
		return chainPbkdf2Sha256HighAesGcmPaddedDecrypt(key, data)
	case ChainAesGcmPadded:
		return chainAesGcmPaddedDecrypt(key, data)
	default:
		return nil, fmt.Errorf("unknown encryption chain: %d", chain)
	}
}

// splitBySeparator splits data by a multi-byte separator pattern.
func splitBySeparator(data, sep []byte) [][]byte {
	var parts [][]byte
	for {
		idx := bytes.Index(data, sep)
		if idx < 0 {
			parts = append(parts, data)
			break
		}
		parts = append(parts, data[:idx])
		data = data[idx+len(sep):]
	}
	return parts
}
