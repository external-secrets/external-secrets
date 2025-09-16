/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type DexUsernamePasswordSpec struct {
	// DexURL configures the Dex server URL
	// +kubebuilder:validation:Required
	DexURL string `json:"dexUrl"`
	// ClientID is the OAuth2 client ID
	// +kubebuilder:validation:Required
	ClientID string `json:"clientId"`
	// ClientSecretRef is a reference to the secret containing the OAuth2 client secret
	// +kubebuilder:validation:Optional
	ClientSecretRef *esmeta.SecretKeySelector `json:"clientSecretRef,omitempty"`
	// Scopes is the list of OAuth2 scopes to request (defaults to ["openid"])
	// +kubebuilder:validation:Optional
	Scopes []string `json:"scopes,omitempty"`
	// UsernameRef is a reference to the secret containing the username
	// +kubebuilder:validation:Required
	UsernameRef esmeta.SecretKeySelector `json:"usernameRef"`
	// PasswordRef is a reference to the secret containing the password
	// +kubebuilder:validation:Required
	PasswordRef esmeta.SecretKeySelector `json:"passwordRef"`
}

// DexUsernamePassword generates Dex access tokens using username/password authentication
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type DexUsernamePassword struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DexUsernamePasswordSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// DexUsernamePasswordList contains a list of DexUsernamePassword resources.
type DexUsernamePasswordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexUsernamePassword `json:"items"`
}

type DexTokenExchangeSpec struct {
	// DexURL configures the Dex server URL
	// +kubebuilder:validation:Required
	DexURL string `json:"dexUrl"`
	// ClientID is the OAuth2 client ID
	// +kubebuilder:validation:Required
	ClientID string `json:"clientId"`
	// ClientSecretRef is a reference to the secret containing the OAuth2 client secret
	// +kubebuilder:validation:Optional
	ClientSecretRef *esmeta.SecretKeySelector `json:"clientSecretRef,omitempty"`
	// ConnectorID is the connector ID for token exchange (defaults to "kubernetes")
	// +kubebuilder:validation:Optional
	ConnectorID string `json:"connectorId,omitempty"`
	// Scopes is the list of OAuth2 scopes to request (defaults to ["openid"])
	// +kubebuilder:validation:Optional
	Scopes []string `json:"scopes,omitempty"`
	// ServiceAccountRef is the service account to use for token exchange
	// +kubebuilder:validation:Required
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef"`
}

// DexTokenExchange generates Dex access tokens using service account token exchange
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type DexTokenExchange struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DexTokenExchangeSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// DexTokenExchangeList contains a list of DexTokenExchange resources.
type DexTokenExchangeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexTokenExchange `json:"items"`
}
