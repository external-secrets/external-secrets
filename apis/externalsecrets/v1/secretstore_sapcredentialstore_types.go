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
	// as provided in the BTP service binding.
	// Example: https://<instance>.credstore.cfapps.<region>.hana.ondemand.com
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
