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

// Package protonpass implements an External Secrets provider for Proton Pass.
//
// Authentication is via a Proton Pass Personal Access Token (PAT); the provider
// speaks directly to the Proton Pass HTTP API. Item, item-key and share-key
// material are protected with symmetric AES-256-GCM — no PGP is involved on the
// PAT path. This file holds only that symmetric primitive.
package protonpass

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// encryptionTag is the AES-GCM additional-authenticated-data (AAD) that Proton
// Pass binds to each ciphertext for domain separation. The exact byte strings
// are part of the wire protocol; a value encrypted under one tag will not
// decrypt under another. Only the tags used on the read/write paths are defined.
type encryptionTag string

const (
	tagShareKey     encryptionTag = "sharekey"
	tagItemKey      encryptionTag = "itemkey"
	tagItemContent  encryptionTag = "itemcontent"
	tagVaultContent encryptionTag = "vaultcontent"
)

const (
	// keyLength is the AES-256 key size used for every symmetric key in Proton Pass.
	keyLength = 32
	// nonceLength is the GCM nonce size; it is prepended to every ciphertext.
	nonceLength = 12
)

// errCiphertextTooShort is returned when a blob cannot contain a nonce.
var errCiphertextTooShort = errors.New("protonpass: ciphertext shorter than nonce")

// aeadDecrypt opens a Proton Pass blob of the form nonce(12) || ciphertext || tag(16)
// using AES-256-GCM, with tag's bytes as AAD.
func aeadDecrypt(blob, key []byte, tag encryptionTag) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(blob) < nonceLength {
		return nil, errCiphertextTooShort
	}
	nonce, ciphertext := blob[:nonceLength], blob[nonceLength:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(tag))
	if err != nil {
		return nil, fmt.Errorf("protonpass: decrypt (tag %q): %w", tag, err)
	}
	return plaintext, nil
}

// aeadEncrypt seals plaintext as nonce(12) || ciphertext || tag(16) with a fresh
// cryptographically-random nonce and tag's bytes as AAD. The nonce is never
// reused: it is drawn from crypto/rand on every call.
func aeadEncrypt(plaintext, key []byte, tag encryptionTag) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("protonpass: generate nonce: %w", err)
	}
	// Seal appends the ciphertext+tag to nonce, yielding nonce||ciphertext||tag.
	return gcm.Seal(nonce, nonce, plaintext, []byte(tag)), nil
}

// newGCM builds an AES-256-GCM AEAD, surfacing (never swallowing) key-length and
// construction errors.
func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != keyLength {
		return nil, fmt.Errorf("protonpass: key must be %d bytes, got %d", keyLength, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("protonpass: new aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("protonpass: new gcm: %w", err)
	}
	return gcm, nil
}

// newItemKey returns a fresh random 32-byte symmetric key for a newly created item.
func newItemKey() ([]byte, error) {
	key := make([]byte, keyLength)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("protonpass: generate item key: %w", err)
	}
	return key, nil
}
