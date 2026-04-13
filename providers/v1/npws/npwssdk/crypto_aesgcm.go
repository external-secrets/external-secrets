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

const (
	aesGcmIVSize  = 12 // 96-bit nonce
	aesGcmTagSize = 16 // 128-bit authentication tag
)

// AesGcmEncrypt encrypts plaintext using AES-256-GCM.
// Returns: IV (12 bytes) || ciphertext || tag (16 bytes).
func AesGcmEncrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-GCM: key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM: %w", err)
	}

	iv := make([]byte, aesGcmIVSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("AES-GCM: generating IV: %w", err)
	}

	// Seal appends ciphertext+tag to iv
	ciphertext := gcm.Seal(iv, iv, plaintext, nil)
	return ciphertext, nil
}

// AesGcmDecrypt decrypts AES-256-GCM ciphertext.
// Input format: IV (12 bytes) || ciphertext || tag (16 bytes).
func AesGcmDecrypt(key, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-GCM: key must be 32 bytes, got %d", len(key))
	}

	if len(ciphertext) < aesGcmIVSize+aesGcmTagSize {
		return nil, fmt.Errorf("AES-GCM: ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM: %w", err)
	}

	iv := ciphertext[:aesGcmIVSize]
	encrypted := ciphertext[aesGcmIVSize:]

	plaintext, err := gcm.Open(nil, iv, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM: decryption failed: %w", err)
	}

	return plaintext, nil
}
