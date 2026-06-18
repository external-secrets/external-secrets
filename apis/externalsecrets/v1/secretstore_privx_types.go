/*
Copyright © 2026 ESO Maintainer Team

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

package v1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// PrivxProvider configures a store to sync secrets using PrivX backend.
type PrivxProvider struct {

	// Server is the connection address for the server, e.g: "https://privx.example.com:8080".
	Host string `json:"host"`

	// Auth configures how secret-manager authenticates with PrivX server.
	Auth *PrivXAuth `json:"auth,omitempty"`

	// DefaultReadRoles are used upon pushing new secrets to PrivX to set read access.
	// Only required when using PushSecret.
	// +optional
	DefaultReadRoles []string `json:"defaultReadRoles,omitempty"`

	// DefaultWriteRoles are used upon pushing new secrets to PrivX to set write access.
	// Only required when using PushSecret.
	// +optional
	DefaultWriteRoles []string `json:"defaultWriteRoles,omitempty"`
}

// PrivXAuth contains the information needed for authentication towards PrivX.
//
// Use only one of the authentication options.
type PrivXAuth struct {
	// OAuth is the OAuth2 authentication option
	OAuth *PrivXOAuth `json:"oauth,omitempty"`

	// JWTAuth configures JWT-based authentication using a signing key.
	JWTAuth *PrivxJWTAuth `json:"jwtAuth,omitempty"`
}

// PrivXOAuth contains the information needed for authentication with OAuth2.
type PrivXOAuth struct {
	ClientIDRef        esmeta.SecretKeySelector `json:"clientIDRef"`
	ClientSecretRef    esmeta.SecretKeySelector `json:"clientSecretRef"`
	ApiClientIDRef     esmeta.SecretKeySelector `json:"apiClientIDRef"`
	ApiClientSecretRef esmeta.SecretKeySelector `json:"apiClientSecretRef"`
}

// PrivxJWTAuth contains the information needed for JWT-based authentication.
type PrivxJWTAuth struct {
	// PublicKeyRef references a Kubernetes Secret containing the PEM-encoded private key
	// used for signing the JWT. The corresponding public key must be configured in PrivX
	// for signature verification.
	// Note: Despite the field name, this must contain the private (signing) key.
	PublicKeyRef esmeta.SecretKeySelector `json:"publicKeyRef"`

	// Iss is the issuer claim — must match the issuer configured in PrivX's External Token Authentication.
	Iss string `json:"iss"`

	// Sub must match a user name resolvable in the PrivX user directory.
	Sub string `json:"sub"`
}
