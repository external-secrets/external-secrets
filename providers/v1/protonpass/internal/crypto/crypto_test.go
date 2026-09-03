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

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encrypt(t *testing.T, plaintext, key []byte, aad string) []byte {
	t.Helper()
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)
	return gcm.Seal(nonce, nonce, plaintext, []byte(aad))
}

func TestDecryptRoundTripPerAAD(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plaintext := []byte("super secret value")
	blob := encrypt(t, plaintext, key, shareKeyAAD)
	got, err := OpenShareKey(blob, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)

	blob = encrypt(t, plaintext, key, itemKeyAAD)
	got, err = OpenItemKey(blob, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)

	blob = encrypt(t, plaintext, key, itemContentAAD)
	got, err = OpenContent(blob, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestDecryptWrongKey(t *testing.T) {
	key := make([]byte, 32)
	wrong := make([]byte, 32)
	_, _ = rand.Read(key)
	_, _ = rand.Read(wrong)

	blob := encrypt(t, []byte("value"), key, shareKeyAAD)
	_, err := OpenShareKey(blob, wrong)
	assert.Error(t, err)
}

func TestDecryptWrongAAD(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	blob := encrypt(t, []byte("value"), key, shareKeyAAD)
	_, err := OpenItemKey(blob, key)
	assert.Error(t, err)
}

func TestDecryptTruncatedBlob(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	blob := encrypt(t, []byte("value"), key, shareKeyAAD)
	_, err := Decrypt(blob[:10], key, []byte(shareKeyAAD))
	assert.Error(t, err)
}

func TestDecryptInvalidKeySize(t *testing.T) {
	_, err := Decrypt([]byte("someblobdata"), make([]byte, 16), []byte(shareKeyAAD))
	assert.Error(t, err)
}

func TestDecryptRejectsShortKey(t *testing.T) {
	_, err := Decrypt([]byte("someblobdata"), make([]byte, 16), []byte(shareKeyAAD))
	assert.Error(t, err)
}

func TestDecryptRejectsShortCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	_, err := Decrypt([]byte("toolittle"), key, []byte(shareKeyAAD))
	assert.Error(t, err)
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	blob := encrypt(t, []byte("value"), key, shareKeyAAD)
	blob[len(blob)-1] ^= 0xFF
	_, err := Decrypt(blob, key, []byte(shareKeyAAD))
	assert.Error(t, err)
}
