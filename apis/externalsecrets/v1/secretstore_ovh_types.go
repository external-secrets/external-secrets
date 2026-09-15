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

// OvhProvider holds the configuration to synchronize secrets with OVHcloud's Secret Manager.
type OvhProvider struct {
	// specifies the OKMS server endpoint.
	// +required
	Server string `json:"server"`
	// specifies the OKMS ID.
	// +required
	OkmsID string `json:"okmsid"`
	// Enables or disables check-and-set (CAS) (default: false).
	// +optional
	CasRequired *bool `json:"casRequired,omitempty"`
	// Setup a timeout in seconds when requests to the KMS are made (default: 30).
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=30
	OkmsTimeout *uint32 `json:"okmsTimeout,omitempty"`
	// Authentication method (mtls, token or oauth2).
	// +required
	Auth OvhAuth `json:"auth"`
}

// OvhAuth tells the controller how to authenticate to OVHcloud's Secret Manager, using mTLS, a token or OAuth2.
// Exactly one method must be set: the markers below make the API server say so, rather than leaving
// it to the provider to discover at reconcile time.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type OvhAuth struct {
	// +optional
	ClientMTLS *OvhClientMTLS `json:"mtls,omitempty"`
	// +optional
	ClientToken *OvhClientToken `json:"token,omitempty"`
	// +optional
	ClientOAuth2 *OvhClientOAuth2 `json:"oauth2,omitempty"`
}

// OvhClientMTLS defines the configuration required to authenticate to OVHcloud's Secret Manager using mTLS.
type OvhClientMTLS struct {
	// +required
	ClientCertificate esmeta.SecretKeySelector `json:"certSecretRef"`
	// +required
	ClientKey esmeta.SecretKeySelector `json:"keySecretRef"`
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`
}

// OvhClientToken defines the configuration required to authenticate to OVHcloud's Secret Manager using a token.
type OvhClientToken struct {
	// +required
	ClientTokenSecret esmeta.SecretKeySelector `json:"tokenSecretRef"`
}

// OvhClientOAuth2 defines the configuration required to authenticate to OVHcloud's Secret Manager
// using an OVHcloud service account.
//
// A service account is the identity OVHcloud intends for machines: it yields an OAuth2 client id
// and client secret, and no browser step is involved in creating one. The access token it is
// exchanged for is short lived, and the client refreshes it on its own.
type OvhClientOAuth2 struct {
	// +required
	ClientID esmeta.SecretKeySelector `json:"clientIDSecretRef"`
	// +required
	ClientSecret esmeta.SecretKeySelector `json:"clientSecretSecretRef"`
	// TokenURL is the OVHcloud OAuth2 token endpoint. It differs per region, and defaults to the
	// European one. The Canadian endpoint is https://ca.ovh.com/auth/oauth2/token and the US one
	// is https://us.ovhcloud.com/auth/oauth2/token.
	// +optional
	// +kubebuilder:default="https://www.ovh.com/auth/oauth2/token"
	TokenURL string `json:"tokenURL,omitempty"`
}
