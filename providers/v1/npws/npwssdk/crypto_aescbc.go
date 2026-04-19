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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

const aesCbcIVSize = 16 // 128-bit IV

// AesCbcEncrypt encrypts plaintext using AES-256-CBC with ISO10126 padding.
// Returns: IV (16 bytes) || ciphertext.
func AesCbcEncrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-CBC: key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES-CBC: %w", err)
	}

	padded := iso10126Pad(plaintext, aes.BlockSize)

	iv := make([]byte, aesCbcIVSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("AES-CBC: generating IV: %w", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(padded))
	mode.CryptBlocks(ciphertext, padded)

	// Prepend IV
	result := make([]byte, aesCbcIVSize+len(ciphertext))
	copy(result, iv)
	copy(result[aesCbcIVSize:], ciphertext)
	return result, nil
}

// AesCbcDecryptWithIV decrypts AES-256-CBC ciphertext with a separate IV.
// This is used by the legacy RSA chain where IV is stored separately.
func AesCbcDecryptWithIV(key, iv, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-CBC: key must be 32 bytes, got %d", len(key))
	}
	if len(iv) != aesCbcIVSize {
		return nil, fmt.Errorf("AES-CBC: IV must be 16 bytes, got %d", len(iv))
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("AES-CBC: ciphertext not aligned to block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES-CBC: %w", err)
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	return iso10126Unpad(plaintext)
}

// AesCbcDecrypt decrypts AES-256-CBC ciphertext where IV is prepended.
// Input format: IV (16 bytes) || ciphertext.
func AesCbcDecrypt(key, data []byte) ([]byte, error) {
	if len(data) < aesCbcIVSize+aes.BlockSize {
		return nil, fmt.Errorf("AES-CBC: data too short")
	}
	iv := data[:aesCbcIVSize]
	ciphertext := data[aesCbcIVSize:]
	return AesCbcDecryptWithIV(key, iv, ciphertext)
}

// iso10126Pad applies ISO 10126 padding.
// Pads to blockSize boundary with random bytes; last byte is the padding length.
func iso10126Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	padding := make([]byte, padLen)
	// Fill with random bytes (except the last byte)
	if padLen > 1 {
		rand.Read(padding[:padLen-1])
	}
	padding[padLen-1] = byte(padLen) //nolint:gosec // padLen is guaranteed to be in range [1,blockSize]
	return append(data, padding...)
}

// iso10126Unpad removes ISO 10126 padding.
func iso10126Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("AES-CBC: empty data for unpadding")
	}
	padLen := int(data[len(data)-1])
	if padLen < 1 || padLen > aes.BlockSize || padLen > len(data) {
		return nil, fmt.Errorf("AES-CBC: invalid ISO10126 padding length %d", padLen)
	}
	return data[:len(data)-padLen], nil
}
