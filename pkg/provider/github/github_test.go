/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://wwg.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package github

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetInstallationToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err, "Failed to generate private key")
	appID := "123456"

	tkn, err := GetInstallationToken(key, appID)
	assert.NoError(t, err, "Should not error when generating token")

	// Validate the token string is not empty
	assert.NotEmpty(t, tkn, "Token string should not be empty")

	// Parse and validate the token
	token, err := jwt.Parse(tkn, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		assert.Equal(t, jwt.SigningMethodRS256, token.Method, "Token signing method mismatch")

		return &key.PublicKey, nil
	})

	assert.NoError(t, err, "Token should be valid")
	assert.NotNil(t, token, "Parsed token should not be nil")

	// Validate claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		assert.Equal(t, appID, claims["iss"], "Issuer (iss) claim should match the app ID")
		assert.WithinDuration(t, time.Now().Add(-time.Second*10), time.Unix(int64(claims["iat"].(float64)), 0), time.Second, "IssuedAt (iat) should be valid")
		assert.WithinDuration(t, time.Now().Add(time.Second*300), time.Unix(int64(claims["exp"].(float64)), 0), time.Second, "ExpiresAt (exp) should be valid")
	} else {
		t.Error("Failed to parse claims or token is invalid")
	}
}
