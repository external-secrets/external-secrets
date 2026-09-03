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

// Package crypto implements the AES-256-GCM primitives used to unwrap
// Proton Pass share, item and content keys.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
)

const (
	shareKeyAAD    = "sharekey"
	itemKeyAAD     = "itemkey"
	itemContentAAD = "itemcontent"

	keySize   = 32
	nonceSize = 12
)

// Decrypt opens an AES-256-GCM blob laid out as nonce(12)||ciphertext||tag(16).
func Decrypt(blob, key, aad []byte) ([]byte, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("protonpass: invalid key size %d, want %d", len(key), keySize)
	}
	if len(blob) < nonceSize {
		return nil, fmt.Errorf("protonpass: ciphertext too short: %d bytes", len(blob))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("protonpass: failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("protonpass: failed to create gcm: %w", err)
	}
	plaintext, err := gcm.Open(nil, blob[:nonceSize], blob[nonceSize:], aad)
	if err != nil {
		return nil, fmt.Errorf("protonpass: failed to decrypt: %w", err)
	}
	return plaintext, nil
}

// OpenShareKey unwraps a share key using the PAT key.
func OpenShareKey(blob, patKey []byte) ([]byte, error) {
	return Decrypt(blob, patKey, []byte(shareKeyAAD))
}

// OpenItemKey unwraps an item key using the share key.
func OpenItemKey(blob, shareKey []byte) ([]byte, error) {
	return Decrypt(blob, shareKey, []byte(itemKeyAAD))
}

// OpenContent decrypts the item content using the item key.
func OpenContent(blob, itemKey []byte) ([]byte, error) {
	return Decrypt(blob, itemKey, []byte(itemContentAAD))
}

// ErrInvalidKey is returned when a key is not exactly 32 bytes.
var ErrInvalidKey = errors.New("protonpass: invalid key size")
