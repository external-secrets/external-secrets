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

type OIDCSpec struct {
	// TokenURL configures the OAuth2 token endpoint URL (e.g., https://dex.example.com/token, https://keycloak.example.com/realms/master/protocol/openid-connect/token)
		// +kubebuilder:validation:XValidation:rule="isURL(self)",message="tokenUrl must be a valid URL"
	// +kubebuilder:validation:Required
	TokenURL string `json:"tokenUrl"`
	// ClientID is the OAuth2 client ID
	// +kubebuilder:validation:Required
	ClientID string `json:"clientId"`
	// ClientSecretRef is a reference to the secret containing the OAuth2 client secret
	// +kubebuilder:validation:Optional
	ClientSecretRef *esmeta.SecretKeySelector `json:"clientSecretRef,omitempty"`
	// Scopes is the list of OAuth2 scopes to request (defaults to ["openid"])
    // +kubebuilder:default={"openid"}
	// +kubebuilder:validation:Optional
	Scopes []string `json:"scopes,omitempty"`
	// Grant specifies the OAuth2 grant type and its parameters
	// Exactly one grant type must be specified
	// +kubebuilder:validation:Required
	Grant GrantSpec `json:"grant"`
	// AdditionalParameters allows specifying provider-specific form parameters (e.g., Dex's connector_id).
	// +kubebuilder:validation:Optional
	AdditionalParameters map[string]string `json:"additionalParameters,omitempty"`
	// AdditionalHeaders allows specifying additional HTTP headers for all requests.
	// This can be used for provider-specific authentication or metadata requirements.
	// +kubebuilder:validation:Optional
	AdditionalHeaders map[string]string `json:"additionalHeaders,omitempty"`
}

// GrantSpec is a union type for different OAuth2 grant types
// Exactly one of the grant types must be specified.
// +kubebuilder:validation:XValidation:rule="[has(self.password), has(self.tokenExchange)].filter(x, x).size() == 1",message="exactly one grant type must be specified"
type GrantSpec struct {
	// Password grant (Resource Owner Password Credentials)
	// +kubebuilder:validation:Optional
	Password *PasswordGrantSpec `json:"password,omitempty"`
	// TokenExchange grant (RFC 8693)
	// +kubebuilder:validation:Optional
	TokenExchange *TokenExchangeGrantSpec `json:"tokenExchange,omitempty"`
}

// PasswordGrantSpec defines parameters for Resource Owner Password Credentials grant.
type PasswordGrantSpec struct {
	// UsernameRef is a reference to the secret containing the username
	// +kubebuilder:validation:Required
	UsernameRef esmeta.SecretKeySelector `json:"usernameRef"`
	// PasswordRef is a reference to the secret containing the password
	// +kubebuilder:validation:Required
	PasswordRef esmeta.SecretKeySelector `json:"passwordRef"`
}

// TokenExchangeGrantSpec defines parameters for OAuth2 Token Exchange (RFC 8693).
// +kubebuilder:validation:XValidation:rule="has(self.subjectTokenRef) != has(self.serviceAccountRef)",message="exactly one of subjectTokenRef or serviceAccountRef must be specified"
type TokenExchangeGrantSpec struct {
	// SubjectTokenRef is a reference to the secret containing the subject token to exchange
	// Mutually exclusive with ServiceAccountRef
	// +kubebuilder:validation:Optional
	SubjectTokenRef *esmeta.SecretKeySelector `json:"subjectTokenRef,omitempty"`
	// ServiceAccountRef is the service account to use for token exchange
	// Mutually exclusive with SubjectTokenRef
	// This will be used to obtain the 'subject_token' parameter value
	// +kubebuilder:validation:Optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
	// SubjectTokenType specifies the type of the subject token (defaults to ID token for service accounts)
	// +kubebuilder:validation:Enum=urn:ietf:params:oauth:token-type:access_token;urn:ietf:params:oauth:token-type:id_token;urn:ietf:params:oauth:token-type:jwt
	// +kubebuilder:default:="urn:ietf:params:oauth:token-type:id_token"
	// +kubebuilder:validation:Optional
	SubjectTokenType string `json:"subjectTokenType,omitempty"`
	// RequestedTokenType specifies the type of token to request (defaults to access token)
	// +kubebuilder:validation:Optional
	RequestedTokenType string `json:"requestedTokenType,omitempty"`
	// ActorTokenRef is a reference to an optional actor token for delegation scenarios
	// +kubebuilder:validation:Optional
	ActorTokenRef *esmeta.SecretKeySelector `json:"actorTokenRef,omitempty"`
	// ActorTokenType specifies the type of the actor token
	// +kubebuilder:validation:Optional
	ActorTokenType string `json:"actorTokenType,omitempty"`
	// Audience specifies the logical name of the target service or resource
	// +kubebuilder:validation:Optional
	Audience string `json:"audience,omitempty"`
	// Resource specifies the physical or logical URI of the target service or resource
	// +kubebuilder:validation:Optional
	Resource string `json:"resource,omitempty"`
	// AdditionalParameters allows specifying provider-specific form parameters for the token exchange request (e.g., Dex's connector_id).
	// +kubebuilder:validation:Optional
	AdditionalParameters map[string]string `json:"additionalParameters,omitempty"`
	// AdditionalHeaders allows specifying additional HTTP headers for the token exchange request.
	// This can be used for provider-specific authentication or metadata requirements.
	// +kubebuilder:validation:Optional
	AdditionalHeaders map[string]string `json:"additionalHeaders,omitempty"`
}

// OIDC generates OAuth2/OIDC access tokens using various grant types
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
// +kubebuilder:validation:XValidation:rule="[has(self.spec.grant.password), has(self.spec.grant.tokenExchange)].filter(x, x).size() == 1",message="exactly one grant type must be specified"
type OIDC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OIDCSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// OIDCList contains a list of OIDC resources.
type OIDCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OIDC `json:"items"`
}
