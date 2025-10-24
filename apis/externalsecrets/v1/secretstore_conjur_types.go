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

package v1

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// ConjurProvider provides access to a Conjur provider.
type ConjurProvider struct {
	// URL is the endpoint of the Conjur instance.
	URL string `json:"url"`

	// CABundle is a PEM encoded CA bundle that will be used to validate the Conjur server certificate.
	// +optional
	CABundle string `json:"caBundle,omitempty"`

	// Used to provide custom certificate authority (CA) certificates
	// for a secret store. The CAProvider points to a Secret or ConfigMap resource
	// that contains a PEM-encoded certificate.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`

	// Defines authentication settings for connecting to Conjur.
	Auth ConjurAuth `json:"auth"`
}

// ConjurAuth is the way to provide authentication credentials to the ConjurProvider.
type ConjurAuth struct {
	// Authenticates with Conjur using an API key.
	// +optional
	APIKey *ConjurAPIKey `json:"apikey,omitempty"`

	// Jwt enables JWT authentication using Kubernetes service account tokens.
	// +optional
	Jwt *ConjurJWT `json:"jwt,omitempty"`
}

// ConjurAPIKey contains references to a Secret resource that holds
// the Conjur username and API key.
type ConjurAPIKey struct {
	// Account is the Conjur organization account name.
	Account string `json:"account"`

	// A reference to a specific 'key' containing the Conjur username
	// within a Secret resource. In some instances, `key` is a required field.
	UserRef *esmeta.SecretKeySelector `json:"userRef"`

	// A reference to a specific 'key' containing the Conjur API key
	// within a Secret resource. In some instances, `key` is a required field.
	APIKeyRef *esmeta.SecretKeySelector `json:"apiKeyRef"`
}

// ConjurJWT defines the JWT authentication configuration for Conjur provider.
type ConjurJWT struct {
	// Account is the Conjur organization account name.
	Account string `json:"account"`

	// The conjur authn jwt webservice id
	ServiceID string `json:"serviceID"`

	// Optional HostID for JWT authentication. This may be used depending
	// on how the Conjur JWT authenticator policy is configured.
	// +optional
	HostID string `json:"hostId"`

	// Optional SecretRef that refers to a key in a Secret resource containing JWT token to
	// authenticate with Conjur using the JWT authentication method.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`

	// Optional ServiceAccountRef specifies the Kubernetes service account for which to request
	// a token for with the `TokenRequest` API.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}
