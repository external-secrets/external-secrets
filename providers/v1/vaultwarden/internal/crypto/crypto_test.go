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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	encKey := make([]byte, 32)
	macKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 1)
	}
	for i := range macKey {
		macKey[i] = byte(i + 33)
	}

	plaintext := "hello, vaultwarden!"
	enc, err := EncryptString(plaintext, encKey, macKey)
	require.NoError(t, err)
	assert.Contains(t, enc, "2.")

	got, err := DecryptString(enc, encKey, macKey)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestMACValidation(t *testing.T) {
	encKey := make([]byte, 32)
	macKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 1)
	}
	for i := range macKey {
		macKey[i] = byte(i + 33)
	}

	enc, err := EncryptString("test", encKey, macKey)
	require.NoError(t, err)

	// Tamper with the MAC by using the wrong macKey
	wrongMacKey := make([]byte, 32)
	es, err := ParseEncString(enc)
	require.NoError(t, err)
	_, err = Decrypt(es, encKey, wrongMacKey)
	assert.ErrorContains(t, err, "MAC validation failed")
}

func TestDeriveKeyPBKDF2(t *testing.T) {
	key, err := DeriveKey("parolaribame4", "test@vaultwarden.local", KdfTypePBKDF2, 600000, 0, 0)
	require.NoError(t, err)
	assert.Len(t, key, 32)
}

func TestStretchKey(t *testing.T) {
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i + 1)
	}
	encKey, macKey, err := StretchKey(masterKey)
	require.NoError(t, err)
	assert.Len(t, encKey, 32)
	assert.Len(t, macKey, 32)
	assert.NotEqual(t, encKey, macKey)
}
