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
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// EncryptionVersion represents the encryption collection version.
// Must match C# MtoEncryptionVersion enum: Version1=0, Version2=1.
type EncryptionVersion int

// EncryptionV1 and EncryptionV2 define the supported encryption versions.
const (
	EncryptionV1 EncryptionVersion = 0
	EncryptionV2 EncryptionVersion = 1
)

// EncryptionCollection is the interface for version-specific encryption operations.
type EncryptionCollection interface {
	Version() EncryptionVersion
	AreDataKeysAsymmetric() bool
	GenerateKeyPair() (publicKey, privateKey []byte, err error)
	GenerateSymmetricKey() ([]byte, error)
	GenerateDataKey() (encryptionKey, decryptionKey []byte, err error)
	EncryptWithPublicKey(publicKey, plaintext []byte) ([]byte, error)
	EncryptWithPassword(password, plaintext []byte) ([]byte, error)
	EncryptWithKey(key, plaintext []byte) ([]byte, error)
	Decrypt(key, ciphertext []byte) ([]byte, error)
	SignData(privateKey, data []byte) ([]byte, error)
	VerifyData(publicKey, data, signature []byte) error
}

// EncryptionManager dispatches encryption operations to the active version.
type EncryptionManager struct {
	collection EncryptionCollection
}

// NewEncryptionManager creates a new EncryptionManager with the given version.
func NewEncryptionManager(version EncryptionVersion) *EncryptionManager {
	em := &EncryptionManager{}
	em.SetEncryptionVersion(version)
	return em
}

// SetEncryptionVersion switches the active encryption collection.
func (em *EncryptionManager) SetEncryptionVersion(version EncryptionVersion) {
	switch version {
	case EncryptionV2:
		em.collection = &EncryptionCollectionV2{}
	case EncryptionV1:
		em.collection = &EncryptionCollectionV1{}
	default:
		em.collection = &EncryptionCollectionV1{}
	}
}

// GetEncryptionVersion returns the current encryption version.
func (em *EncryptionManager) GetEncryptionVersion() EncryptionVersion {
	return em.collection.Version()
}

// GenerateKeyPair generates a new asymmetric key pair.
func (em *EncryptionManager) GenerateKeyPair() (publicKey, privateKey []byte, err error) {
	return em.collection.GenerateKeyPair()
}

// GenerateSymmetricKey generates a new 32-byte AES key.
func (em *EncryptionManager) GenerateSymmetricKey() ([]byte, error) {
	return em.collection.GenerateSymmetricKey()
}

// GenerateDataKey generates a new data key (asymmetric in V1, symmetric in V2).
func (em *EncryptionManager) GenerateDataKey() (encryptionKey, decryptionKey []byte, err error) {
	return em.collection.GenerateDataKey()
}

// AreDataKeysAsymmetric returns true if the current version uses asymmetric data keys.
func (em *EncryptionManager) AreDataKeysAsymmetric() bool {
	return em.collection.AreDataKeysAsymmetric()
}

// EncryptWithPublicKey encrypts data with a public key (RSA or ECC).
func (em *EncryptionManager) EncryptWithPublicKey(publicKey, plaintext []byte) ([]byte, error) {
	return em.collection.EncryptWithPublicKey(publicKey, plaintext)
}

// EncryptWithPassword encrypts data with a password.
func (em *EncryptionManager) EncryptWithPassword(password, plaintext []byte) ([]byte, error) {
	return em.collection.EncryptWithPassword(password, plaintext)
}

// EncryptWithKey encrypts data with a symmetric key.
func (em *EncryptionManager) EncryptWithKey(key, plaintext []byte) ([]byte, error) {
	return em.collection.EncryptWithKey(key, plaintext)
}

// Decrypt decrypts data, auto-detecting the chain from the ciphertext prefix.
func (em *EncryptionManager) Decrypt(key, ciphertext []byte) ([]byte, error) {
	return em.collection.Decrypt(key, ciphertext)
}

// SignData signs data with a private key (RSA SHA-512 or ECDSA SHA-256).
func (em *EncryptionManager) SignData(privateKey, data []byte) ([]byte, error) {
	return em.collection.SignData(privateKey, data)
}

// VerifyData verifies a signature with a public key.
func (em *EncryptionManager) VerifyData(publicKey, data, signature []byte) error {
	return em.collection.VerifyData(publicKey, data, signature)
}

// EncryptionResult holds the result of encrypting a container item.
type EncryptionResult struct {
	PublicKey        []byte
	EncryptedValue   []byte
	NewDecryptionKey []byte // Non-nil if a new key was created
}

// EncryptContainerItem encrypts a container item's value.
// Matches C# MtoEncryptionCollectionExtension.EncryptData exactly:
//   - currentPublicKey is the item's existing PublicKey (nil for new items)
//   - getCurrentSymmetricKey returns an existing symmetric key (for V2 reuse)
func (em *EncryptionManager) EncryptContainerItem(item *PsrContainerItem, plainValue []byte, getCurrentSymmetricKey func() ([]byte, error)) (*EncryptionResult, error) {
	result := &EncryptionResult{}
	currentPublicKey := item.PublicKey

	if em.AreDataKeysAsymmetric() || currentPublicKey != nil {
		// Asymmetric path: use public key to encrypt
		if currentPublicKey == nil {
			// New item: generate keypair
			pub, priv, err := em.GenerateKeyPair()
			if err != nil {
				return nil, fmt.Errorf("EncryptContainerItem: generating keypair: %w", err)
			}
			result.PublicKey = pub
			result.NewDecryptionKey = priv
		} else {
			// Existing item: reuse public key
			result.PublicKey = currentPublicKey
		}
		encrypted, err := em.EncryptWithPublicKey(result.PublicKey, plainValue)
		if err != nil {
			return nil, fmt.Errorf("EncryptContainerItem: encrypting: %w", err)
		}
		result.EncryptedValue = encrypted
	} else {
		// Symmetric path (V2, new items without public key)
		var symmetricKey []byte
		if getCurrentSymmetricKey != nil {
			existingKey, err := getCurrentSymmetricKey()
			if err == nil && existingKey != nil {
				symmetricKey = existingKey
			}
		}
		if symmetricKey == nil {
			var err error
			symmetricKey, err = em.GenerateSymmetricKey()
			if err != nil {
				return nil, fmt.Errorf("EncryptContainerItem: generating key: %w", err)
			}
			result.NewDecryptionKey = symmetricKey
		}
		encrypted, err := em.EncryptWithKey(symmetricKey, plainValue)
		if err != nil {
			return nil, fmt.Errorf("EncryptContainerItem: encrypting: %w", err)
		}
		result.EncryptedValue = encrypted
		result.PublicKey = nil
	}

	// Update item in-place
	item.PublicKey = result.PublicKey
	item.Value = base64.StdEncoding.EncodeToString(result.EncryptedValue)

	return result, nil
}

// generateRandomBytes returns n cryptographically random bytes.
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
