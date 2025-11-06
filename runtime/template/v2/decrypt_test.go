/*
Copyright © 2025 ESO Maintainer Team

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

package template

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateRSAPrivateKeyPEM(t testing.TB) (string, *rsa.PrivateKey) {
	t.Helper()
	// Generate a new RSA private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate RSA key")
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	return string(privPEM), priv
}

func TestRsaDecrypt_NoneScheme(t *testing.T) {
	input := "plaintext"
	privateKey := "irrelevant"
	out, err := rsaDecrypt("None", "SHA256", input, privateKey)
	assert.NoError(t, err)
	assert.Equal(t, input, out)
}

func TestRsaDecrypt_InvalidPEM(t *testing.T) {
	_, err := rsaDecrypt("RSA-OAEP", "SHA256", "data", "not-a-valid-pem")
	assert.Error(t, err)
	assert.Equal(t, errDecodePEM, err.Error())
}

func TestRsaDecrypt_InvalidPrivateKey(t *testing.T) {
	// Use a valid PEM block but not a private key
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("invalid")})
	_, err := rsaDecrypt("RSA-OAEP", "SHA256", "data", string(pemBlock))
	assert.Error(t, err)
	assert.ErrorIs(t, err, errParsePK)
}

func TestRsaDecrypt_UnsupportedScheme(t *testing.T) {
	privateKey, _ := generateRSAPrivateKeyPEM(t)
	_, err := rsaDecrypt("Unsupported", "SHA256", "data", privateKey)
	assert.Error(t, err)
	assert.Equal(t, fmt.Errorf(errSchemeNotSupported, "Unsupported"), err)
}

func TestRsaDecrypt_RSAOAEP_Success(t *testing.T) {
	privateKeyPEM, priv := generateRSAPrivateKeyPEM(t)
	plaintext := []byte("secret-data")
	ciphertext, err := rsa.EncryptOAEP(getHash("SHA256"), rand.Reader, &priv.PublicKey, plaintext, nil)
	assert.NoError(t, err)
	out, err := rsaDecrypt("RSA-OAEP", "SHA256", string(ciphertext), privateKeyPEM)
	assert.NoError(t, err)
	assert.Equal(t, string(plaintext), out)
}

func TestRsaDecrypt_RSAOAEP_DecryptionError(t *testing.T) {
	privateKeyPEM, _ := generateRSAPrivateKeyPEM(t)
	// Pass random data as ciphertext
	_, err := rsaDecrypt("RSA-OAEP", "SHA256", "not-encrypted-data", privateKeyPEM)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errRSADecrypt)
}
