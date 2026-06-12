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

package v1

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// SAPCredentialStoreProvider configures the SAP Credential Store ESO provider.
type SAPCredentialStoreProvider struct {
	// ServiceURL is the base URL of the SAP Credential Store REST API endpoint,
	// as provided in the BTP service binding (the "url" field).
	// Example: https://credstore.mesh.cf.sap.hana.ondemand.com/api/v1/credentials
	// +kubebuilder:validation:Required
	ServiceURL string `json:"serviceURL"`

	// Namespace is the credential namespace within the SAP Credential Store
	// instance. This is a BTP Credential Store namespace, not a Kubernetes namespace.
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// Auth contains the authentication credentials for SAP Credential Store.
	// Exactly one of oauth2 or mtls must be specified.
	// +kubebuilder:validation:Required
	Auth SAPCSAuth `json:"auth"`

	// Encryption configures JWE payload encryption for SAP Credential Store.
	// Required when the service binding was created with encryption.payload=enabled
	// (the default for most BTP environments). The client_private_key and
	// server_public_key values are taken directly from the binding JSON.
	// +optional
	Encryption *SAPCSEncryption `json:"encryption,omitempty"`
}

// SAPCSEncryption holds the JWE keys from the BTP service binding encryption block.
// When set, all request bodies are encrypted with the server public key and all
// response bodies are decrypted with the client private key using RSA-OAEP-256 + AES-256-GCM.
type SAPCSEncryption struct {
	// ClientPrivateKey references a Kubernetes Secret key containing the base64-encoded
	// PKCS8 DER private key used to decrypt JWE responses from SAP Credential Store.
	// Corresponds to the "encryption.client_private_key" field in the BTP service binding.
	// +kubebuilder:validation:Required
	ClientPrivateKey esmeta.SecretKeySelector `json:"clientPrivateKey"`

	// ServerPublicKey references a Kubernetes Secret key containing the base64-encoded
	// SPKI DER public key used to encrypt JWE request bodies sent to SAP Credential Store.
	// Corresponds to the "encryption.server_public_key" field in the BTP service binding.
	// +kubebuilder:validation:Required
	ServerPublicKey esmeta.SecretKeySelector `json:"serverPublicKey"`
}

// SAPCSAuth configures authentication for the SAP Credential Store provider.
// Exactly one of OAuth2 or MTLS must be set.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type SAPCSAuth struct {
	// OAuth2 configures OAuth2 client credentials grant authentication
	// using client ID, client secret, and a token URL.
	// Suitable for standard BTP service bindings.
	// +optional
	OAuth2 *SAPCSOAuth2Auth `json:"oauth2,omitempty"`

	// MTLS configures mutual TLS authentication using a client certificate
	// and private key.
	// +optional
	MTLS *SAPCSMTLSAuth `json:"mtls,omitempty"`
}

// SAPCSOAuth2Auth holds OAuth2 client credentials for the SAP Credential Store provider.
type SAPCSOAuth2Auth struct {
	// TokenURL is the OAuth2 token endpoint URL.
	// Example: https://<subaccount>.authentication.<region>.hana.ondemand.com/oauth/token
	// +kubebuilder:validation:Required
	TokenURL string `json:"tokenURL"`

	// ClientID references a Kubernetes Secret key containing the OAuth2 client ID.
	// +kubebuilder:validation:Required
	ClientID esmeta.SecretKeySelector `json:"clientId"`

	// ClientSecret references a Kubernetes Secret key containing the OAuth2 client secret.
	// +kubebuilder:validation:Required
	ClientSecret esmeta.SecretKeySelector `json:"clientSecret"`
}

// SAPCSMTLSAuth holds mTLS certificate credentials for the SAP Credential Store provider.
type SAPCSMTLSAuth struct {
	// Certificate references a Kubernetes Secret key containing the
	// PEM-encoded client certificate.
	// +kubebuilder:validation:Required
	Certificate esmeta.SecretKeySelector `json:"certificate"`

	// PrivateKey references a Kubernetes Secret key containing the
	// PEM-encoded private key that corresponds to Certificate.
	// +kubebuilder:validation:Required
	PrivateKey esmeta.SecretKeySelector `json:"privateKey"`
}
