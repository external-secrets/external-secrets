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
	"encoding/base64"
	"encoding/json"
	"strings"
)

// PsrSessionState represents the connection state of the API.
type PsrSessionState int

// SessionDisconnected and SessionConnected define API session states.
const (
	SessionDisconnected PsrSessionState = iota
	SessionConnected
)

// AuthenticationToken2 is the session token used for API requests (new auth flow).
type AuthenticationToken2 struct {
	Database   string `json:"Database"`
	SessionID  string `json:"SessionId"`
	SessionKey string `json:"SessionKey"`
}

// Serialize returns the token as Base64(JSON) for use in the token2 HTTP header.
func (t *AuthenticationToken2) Serialize() (string, error) {
	data, err := json.Marshal(t) //nolint:gosec // SessionKey is a session identifier, not a secret
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// UserKey represents a user's private key for decryption.
type UserKey struct {
	ID         string `json:"Id"`
	PrivateKey []byte `json:"PrivateKey"`
}

// ClientInformation contains metadata about the API client.
// Matches C# ClientInformation with all fields.
type ClientInformation struct {
	ClientComputerName *string `json:"ClientComputerName"`
	ClientUser         *string `json:"ClientUser"`
	ClientType         string  `json:"ClientType"`
	ClientVersion      string  `json:"ClientVersion"`
	ClientMacAddress   *string `json:"ClientMacAdress"`
	ClientInstanceID   string  `json:"ClientInstanceId"`
	OS                 *string `json:"OS"`
	Browser            *string `json:"Browser"`
}

// RequiredFieldsFromUser matches C# AuthCredential.RequiredFieldsFromUser.
type RequiredFieldsFromUser struct {
	AuthenticationFields map[string]interface{} `json:"AuthenticationFields"`
}

// AuthenticationAPIKeyCredential is sent to authenticate via API key.
// Matches C# AuthenticationApiKeyCredential + OptionallyAuthenticatedRequest base fields.
type AuthenticationAPIKeyCredential struct {
	// Derived class fields
	AuthType               int                    `json:"AuthType"`
	Type                   string                 `json:"$type"`
	JSONWebToken           string                 `json:"JsonWebToken"`
	RequiredFieldsFromUser RequiredFieldsFromUser `json:"RequiredFieldsFromUser"`
	// Base class fields (OptionallyAuthenticatedRequest)
	Database          *string           `json:"Database"`
	Username          *string           `json:"Username"`
	SessionID         *string           `json:"SessionId"`
	OperationMode     int               `json:"OperationMode"`
	ClientInformation ClientInformation `json:"ClientInformation"`
}

// AuthenticationResult is the server response after an authentication step.
// Step 1 (AuthenticateApiKey) returns a challenge; Step 2 returns the completed session.
type AuthenticationResult struct {
	Type                  string                     `json:"$type,omitempty"`
	IsCompleted           bool                       `json:"IsCompleted"`
	SessionID             string                     `json:"SessionId,omitempty"`
	SessionKey            string                     `json:"SessionKey,omitempty"`
	Database              string                     `json:"Database,omitempty"`
	UserID                string                     `json:"UserId,omitempty"`
	UserKeys              []UserKey                  `json:"UserKeys,omitempty"`
	EncryptedRoleRightKey map[string]json.RawMessage `json:"EncryptedRoleRightKey,omitempty"`

	// Challenge fields (Step 1 response)
	Challenge          string `json:"Challenge,omitempty"`
	ChallengeSignature string `json:"ChallengeSignature,omitempty"`
	UserKey            string `json:"UserKey,omitempty"`
	EncryptionVersion  int    `json:"EncryptionVersion,omitempty"`
	UserName           string `json:"UserName,omitempty"`
}

// IsAuthCompleted checks if the authentication is complete.
// The server indicates completion either via the IsCompleted field or
// by returning a type containing "Completed" in the $type discriminator.
func (r *AuthenticationResult) IsAuthCompleted() bool {
	if r.IsCompleted {
		return true
	}
	return strings.Contains(r.Type, "Completed")
}

// AuthenticationUserKeySignatureCredential is sent in response to a signature challenge.
// Matches C# AuthenticationUserKeySignatureCredential + OptionallyAuthenticatedRequest base.
type AuthenticationUserKeySignatureCredential struct {
	// Derived class fields
	Type                      string `json:"$type"`
	UserKeyChallengeSignature []byte `json:"UserKeyChallengeSignature"`
	Challenge                 []byte `json:"Challenge"`
	ChallengeSignature        []byte `json:"ChallengeSignature"`
	// Base class fields (OptionallyAuthenticatedRequest)
	Database          string            `json:"Database"`
	Username          string            `json:"Username"`
	SessionID         string            `json:"SessionId"`
	OperationMode     int               `json:"OperationMode"`
	ClientInformation ClientInformation `json:"ClientInformation"`
}
