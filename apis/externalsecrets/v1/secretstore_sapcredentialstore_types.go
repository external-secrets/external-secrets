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

// SAPCredentialStoreProvider configures a store to sync secrets using the SAP Credential Store.
type SAPCredentialStoreProvider struct {
	// ServiceURL is the base URL of the SAP Credential Store API.
	// Required when serviceBindingSecretRef is not set; ignored when a binding is used
	// because the URL is derived from the binding JSON.
	// +optional
	ServiceURL string `json:"serviceURL,omitempty"`

	// Namespace is the SAP Credential Store namespace for credential operations.
	// This is a Credential Store concept, not a Kubernetes namespace.
	Namespace string `json:"namespace"`

	// Auth configures inline authentication to the SAP Credential Store.
	// Not required when serviceBindingSecretRef is set, because the binding
	// contains all necessary credentials.
	// +optional
	Auth *SAPCSAuth `json:"auth,omitempty"`

	// Encryption configures JWE payload encryption for the SAP Credential Store API.
	// Required when the service instance has payload encryption enabled.
	// +optional
	Encryption *SAPCSEncryption `json:"encryption,omitempty"`

	// ServiceBindingSecretRef references a Kubernetes Secret containing a BTP service binding
	// for the SAP Credential Store. When set, credentials and the service URL are
	// derived from the binding JSON, and the Auth field becomes optional.
	// Currently only the "mtls" binding authentication type is supported.
	// +optional
	ServiceBindingSecretRef *SAPCSServiceBindingRef `json:"serviceBindingSecretRef,omitempty"`
}

// SAPCSAuth configures inline authentication to the SAP Credential Store.
// Exactly one authentication method must be specified.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type SAPCSAuth struct {
	// MTLS configures mutual TLS certificate authentication.
	// +optional
	MTLS *SAPCSMTLSAuth `json:"mtls,omitempty"`
}

// SAPCSMTLSAuth configures mutual TLS certificate authentication.
type SAPCSMTLSAuth struct {
	// Certificate is a reference to the client certificate in PEM format.
	Certificate esmeta.SecretKeySelector `json:"certificate"`

	// PrivateKey is a reference to the client private key in PEM format.
	PrivateKey esmeta.SecretKeySelector `json:"privateKey"`
}

// SAPCSEncryption configures JWE payload encryption keys for the SAP Credential Store API.
type SAPCSEncryption struct {
	// ClientPrivateKey is a reference to the RSA private key (PKCS8 DER, base64-encoded)
	// used to decrypt responses from the Credential Store.
	ClientPrivateKey esmeta.SecretKeySelector `json:"clientPrivateKey"`

	// ServerPublicKey is a reference to the RSA public key (SPKI DER, base64-encoded)
	// used to encrypt requests to the Credential Store.
	ServerPublicKey esmeta.SecretKeySelector `json:"serverPublicKey"`
}

// SAPCSServiceBindingRef references a Kubernetes Secret containing a BTP service binding.
// The binding JSON structure depends on the authentication type configured for the service
// instance (e.g. "mtls", "oauth:mtls", "oauth:key"). The provider detects the type from
// the "parameters.authentication.type" field in the binding JSON.
type SAPCSServiceBindingRef struct {
	// Name is the name of the Kubernetes Secret containing the service binding.
	Name string `json:"name"`

	// Namespace of the Kubernetes Secret. Defaults to the namespace of the SecretStore.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// CredentialsKey is the key in the Secret's Data map that holds the binding JSON.
	// +optional
	// +kubebuilder:default=credentials
	CredentialsKey string `json:"credentialsKey,omitempty"`
}
