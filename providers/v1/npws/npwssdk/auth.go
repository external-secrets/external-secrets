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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// AuthenticationManager handles login/logout via API key.
type AuthenticationManager struct {
	authClient    *HTTPClient
	serviceClient *HTTPClient
	encryption    *EncryptionManager
	userKeys      *UserKeyManager
	clientInfo    ClientInformation
}

// NewAuthenticationManager creates a new AuthenticationManager.
func NewAuthenticationManager(authClient, serviceClient *HTTPClient, encryption *EncryptionManager, userKeys *UserKeyManager) *AuthenticationManager {
	return &AuthenticationManager{
		authClient:    authClient,
		serviceClient: serviceClient,
		encryption:    encryption,
		userKeys:      userKeys,
		clientInfo: ClientInformation{
			ClientType:       "PsrApiGo",
			ClientVersion:    "1.0.0",
			ClientInstanceID: uuid.New().String(),
		},
	}
}

// LoginWithAPIKey authenticates using an API key.
// API key format: "{JWT}:{Base64EncodedPrivateKey}".
func (am *AuthenticationManager) LoginWithAPIKey(ctx context.Context, apiKey string) error {
	jwt, privateKey, err := parseAPIKey(apiKey)
	if err != nil {
		return fmt.Errorf("LoginWithAPIKey: %w", err)
	}

	// Extract database and username from JWT claims
	database, username := extractJWTClaims(jwt)

	// Step 1: Send JWT to server
	credential := struct {
		Credential AuthenticationAPIKeyCredential `json:"credential"`
	}{
		Credential: AuthenticationAPIKeyCredential{
			AuthType:               14,
			Type:                   "AuthenticationApiKeyCredential",
			JSONWebToken:           jwt,
			RequiredFieldsFromUser: RequiredFieldsFromUser{AuthenticationFields: map[string]interface{}{}},
			Database:               nil,
			Username:               nil,
			SessionID:              nil,
			OperationMode:          0,
			ClientInformation:      am.clientInfo,
		},
	}

	var result AuthenticationResult
	if err := am.authClient.Post(ctx, "AuthenticateApiKey", credential, &result); err != nil {
		return fmt.Errorf("LoginWithAPIKey: %w", err)
	}

	// Step 2: Handle signature challenge if present
	var decryptedUserKey []byte
	if !result.IsAuthCompleted() && result.Challenge != "" {
		// Set encryption version from server response
		if result.EncryptionVersion > 0 {
			am.encryption.SetEncryptionVersion(EncryptionVersion(result.EncryptionVersion))
		}

		// Determine the signing key:
		// If server provides a UserKey, decrypt it with the API key's private key.
		// Otherwise, sign directly with the API key's private key.
		signingKey := privateKey
		if result.UserKey != "" {
			encryptedUserKey, err := base64.StdEncoding.DecodeString(result.UserKey)
			if err != nil {
				return fmt.Errorf("LoginWithAPIKey: decoding user key: %w", err)
			}
			decryptedUserKey, err = am.encryption.Decrypt(privateKey, encryptedUserKey)
			if err != nil {
				return fmt.Errorf("LoginWithAPIKey: decrypting user key: %w", err)
			}
			signingKey = decryptedUserKey
		}

		// Decode the challenge bytes
		challengeBytes, err := base64.StdEncoding.DecodeString(result.Challenge)
		if err != nil {
			return fmt.Errorf("LoginWithAPIKey: decoding challenge: %w", err)
		}

		// Sign the challenge
		signature, err := am.encryption.SignData(signingKey, challengeBytes)
		if err != nil {
			return fmt.Errorf("LoginWithAPIKey: signing challenge: %w", err)
		}

		// Decode the server's challenge signature to echo back
		challengeSig, err := base64.StdEncoding.DecodeString(result.ChallengeSignature)
		if err != nil {
			return fmt.Errorf("LoginWithAPIKey: decoding challenge signature: %w", err)
		}

		sigCredential := struct {
			Credential AuthenticationUserKeySignatureCredential `json:"credential"`
		}{
			Credential: AuthenticationUserKeySignatureCredential{
				Type:                      "AuthenticationUserKeySignatureCredential",
				UserKeyChallengeSignature: signature,
				Challenge:                 challengeBytes,
				ChallengeSignature:        challengeSig,
				Database:                  database,
				Username:                  username,
				SessionID:                 result.SessionID,
				OperationMode:             0,
				ClientInformation:         am.clientInfo,
			},
		}

		if err := am.authClient.Post(ctx, "AuthenticateUserKeySignatureCredential", sigCredential, &result); err != nil {
			return fmt.Errorf("LoginWithAPIKey: submitting signature: %w", err)
		}
	}

	if !result.IsAuthCompleted() {
		return fmt.Errorf("LoginWithAPIKey: authentication not completed")
	}

	// Step 3: Set session
	// Database may not be in the step 2 response — fall back to JWT claim
	db := result.Database
	if db == "" {
		db = database
	}
	token2 := &AuthenticationToken2{
		Database:   db,
		SessionID:  result.SessionID,
		SessionKey: result.SessionKey,
	}

	am.authClient.SetAuth(token2, jwt)
	am.serviceClient.SetAuth(token2, jwt)

	// Store user keys — the decrypted UserKey from Step 1 is the user's private key
	var keys []UserKey
	if len(result.UserKeys) > 0 {
		keys = result.UserKeys
	} else if decryptedUserKey != nil {
		keys = []UserKey{{
			ID:         result.UserID,
			PrivateKey: decryptedUserKey,
		}}
	}

	// Decrypt EncryptedRoleRightKey — each role's private key is encrypted with the user's private key
	if len(result.EncryptedRoleRightKey) > 0 && decryptedUserKey != nil {
		for roleID, rawValue := range result.EncryptedRoleRightKey {
			if roleID == "$type" {
				continue // Skip JSON type discriminator
			}
			// rawValue is a JSON string containing base64-encoded bytes
			var b64str string
			if err := json.Unmarshal(rawValue, &b64str); err != nil {
				continue
			}
			encryptedRoleKey, err := base64.StdEncoding.DecodeString(b64str)
			if err != nil {
				continue
			}
			decryptedRoleKey, err := am.encryption.Decrypt(decryptedUserKey, encryptedRoleKey)
			if err != nil {
				continue
			}
			keys = append(keys, UserKey{
				ID:         roleID,
				PrivateKey: decryptedRoleKey,
			})
		}
	}

	am.userKeys.SetUserKeys(keys)
	am.userKeys.SetCurrentUserID(result.UserID)

	return nil
}

// Logout ends the current session.
func (am *AuthenticationManager) Logout(ctx context.Context) error {
	err := am.serviceClient.Get(ctx, "CloseSession", nil, nil)

	// Always clean up regardless of error
	am.authClient.ClearAuth()
	am.serviceClient.ClearAuth()
	am.userKeys.SetUserKeys(nil)

	return err
}

// parseAPIKey splits an API key into JWT and private key components.
func parseAPIKey(apiKey string) (jwt string, privateKey []byte, err error) {
	parts := strings.SplitN(apiKey, ":", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("malformed API key: expected format 'JWT:Base64PrivateKey'")
	}

	jwt = parts[0]
	privateKey, err = base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, fmt.Errorf("decoding private key: %w", err)
	}

	return jwt, privateKey, nil
}

// extractJWTClaims extracts database name and username from the JWT payload (without validation).
func extractJWTClaims(jwt string) (database, username string) {
	parts := strings.Split(jwt, ".")
	if len(parts) < 2 {
		return "", ""
	}

	// JWT payload is base64url encoded
	payload := parts[1]
	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	// Convert base64url to standard base64
	payload = strings.ReplaceAll(payload, "-", "+")
	payload = strings.ReplaceAll(payload, "_", "/")

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", ""
	}

	var claims struct {
		DatabaseName string `json:"dbn"`
		UserName     string `json:"usrn"`
	}
	if err := json.Unmarshal(data, &claims); err != nil {
		return "", ""
	}

	return claims.DatabaseName, claims.UserName
}
