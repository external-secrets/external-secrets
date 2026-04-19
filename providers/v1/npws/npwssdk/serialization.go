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
	"crypto/ecdh"
	"crypto/elliptic"
	"fmt"
	"math/big"

	"github.com/vmihailenco/msgpack/v5"
)

// KeyType identifies the type of an encryption key by its byte prefix.
type KeyType int

// KeyTypeUnknown and related constants identify encryption key types.
const (
	KeyTypeUnknown    KeyType = 0
	KeyTypeRSA        KeyType = 1
	KeyTypeECCPublic  KeyType = 2
	KeyTypeECCPrivate KeyType = 3
	KeyTypeSymmetric  KeyType = 4
)

// DetectKeyType identifies the key type by inspecting byte prefixes.
// RSA keys (XML format): start with bytes [80, 70] (Base64 "PF" = "<RSAKeyValue")
// ECC public keys: start with bytes [146, 196] (MessagePack array header)
// ECC private keys: start with bytes [147, 196] (MessagePack array header)
// Symmetric keys: exactly 32 bytes.
func DetectKeyType(key []byte) KeyType {
	if len(key) == 32 {
		return KeyTypeSymmetric
	}
	if len(key) >= 2 {
		if key[0] == 80 && key[1] == 70 {
			return KeyTypeRSA
		}
		if key[0] == 146 && key[1] == 196 {
			return KeyTypeECCPublic
		}
		if key[0] == 147 && key[1] == 196 {
			return KeyTypeECCPrivate
		}
	}
	return KeyTypeUnknown
}

// IsRSAKey returns true if the key appears to be an RSA key (XML format).
func IsRSAKey(key []byte) bool {
	return DetectKeyType(key) == KeyTypeRSA
}

// IsECCKey returns true if the key appears to be an ECC key.
func IsECCKey(key []byte) bool {
	kt := DetectKeyType(key)
	return kt == KeyTypeECCPublic || kt == KeyTypeECCPrivate
}

// IsSymmetricKey returns true if the key appears to be a 32-byte symmetric key.
func IsSymmetricKey(key []byte) bool {
	return DetectKeyType(key) == KeyTypeSymmetric
}

// parseECCPublicKey deserializes a C# MessagePack-serialized ECC public key.
// Format: MessagePack array [X([]byte), Y([]byte)] → uncompressed point 0x04||X||Y.
func parseECCPublicKey(data []byte) (*ecdh.PublicKey, error) {
	var coords [][]byte
	if err := msgpack.Unmarshal(data, &coords); err != nil {
		return nil, fmt.Errorf("ECC public key: msgpack unmarshal: %w", err)
	}
	if len(coords) != 2 {
		return nil, fmt.Errorf("ECC public key: expected 2 coords, got %d", len(coords))
	}

	x := new(big.Int).SetBytes(coords[0])
	y := new(big.Int).SetBytes(coords[1])

	// Marshal to uncompressed point format
	point := elliptic.Marshal(elliptic.P521(), x, y)
	return ecdh.P521().NewPublicKey(point)
}

// serializeECCPublicKey serializes an ECC public key to C# MessagePack format [X, Y].
func serializeECCPublicKey(pub *ecdh.PublicKey) ([]byte, error) {
	rawBytes := pub.Bytes()
	// Raw format is 0x04 || X || Y (uncompressed point)
	if len(rawBytes) != 133 || rawBytes[0] != 0x04 {
		return nil, fmt.Errorf("ECC public key: unexpected format (len=%d)", len(rawBytes))
	}
	x := rawBytes[1:67]
	y := rawBytes[67:133]
	return msgpack.Marshal([][]byte{x, y})
}

// parseECCPrivateKey deserializes a C# MessagePack-serialized ECC private key.
// Format: MessagePack array [D([]byte), X([]byte), Y([]byte)].
func parseECCPrivateKey(data []byte) (*ecdh.PrivateKey, error) {
	var parts [][]byte
	if err := msgpack.Unmarshal(data, &parts); err != nil {
		return nil, fmt.Errorf("ECC private key: msgpack unmarshal: %w", err)
	}
	if len(parts) != 3 {
		return nil, fmt.Errorf("ECC private key: expected 3 parts, got %d", len(parts))
	}

	d := parts[0]
	// P-521 scalar must be exactly 66 bytes
	if len(d) < 66 {
		padded := make([]byte, 66)
		copy(padded[66-len(d):], d)
		d = padded
	}

	return ecdh.P521().NewPrivateKey(d)
}

// EncryptWithPublicKeyAuto detects the key type and uses the appropriate encryption chain.
func EncryptWithPublicKeyAuto(publicKey, plaintext []byte) ([]byte, error) {
	switch DetectKeyType(publicKey) { //nolint:exhaustive // default handles unsupported key types
	case KeyTypeRSA:
		return EncryptWithChain(ChainRsaPbkdf2Sha1AesCbc, publicKey, plaintext)
	case KeyTypeECCPublic:
		return EncryptWithChain(ChainEcdhHkdfSha256AesGcm, publicKey, plaintext)
	default:
		return nil, fmt.Errorf("EncryptWithPublicKey: unsupported key type")
	}
}

// DecryptWithPrivateKeyAuto detects the key type and chain, then decrypts.
func DecryptWithPrivateKeyAuto(privateKey, ciphertext []byte) ([]byte, error) {
	return DecryptWithChain(privateKey, ciphertext)
}
