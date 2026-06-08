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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// ArtifactoryAccessTokenSpec defines configuration for generating JFrog Artifactory access tokens.
type ArtifactoryAccessTokenSpec struct {
	// URL is the JFrog Platform base URL (e.g. https://acme.jfrog.io).
	// +kubebuilder:validation:Required
	URL string `json:"url"`
	// Registry is the Docker registry hostname used in dockerconfigjson templates.
	// Defaults to the hostname from URL.
	// +kubebuilder:validation:Optional
	Registry string `json:"registry,omitempty"`
	// Auth configures how ESO obtains or creates the Artifactory token.
	Auth ArtifactoryAccessTokenAuth `json:"auth"`
}

// ArtifactoryAccessTokenAuth defines the authentication method for token generation.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type ArtifactoryAccessTokenAuth struct {
	// OIDC exchanges a Kubernetes service account token for an Artifactory access token.
	// +optional
	OIDC *ArtifactoryOIDCAuth `json:"oidc,omitempty"`
	// ReferenceToken creates a scoped token using a bootstrap identity or access token.
	// +optional
	ReferenceToken *ArtifactoryReferenceTokenAuth `json:"referenceToken,omitempty"`
}

// ArtifactoryOIDCAuth defines OIDC token exchange against JFrog Platform.
type ArtifactoryOIDCAuth struct {
	// ProviderName is the OIDC provider name configured in JFrog Platform.
	// +kubebuilder:validation:Required
	ProviderName string `json:"providerName"`
	// ServiceAccountRef references the Kubernetes service account used for OIDC exchange.
	// +kubebuilder:validation:Required
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef"`
	// ProviderType is the OIDC provider type. Defaults to GenericOidc for Kubernetes.
	// +kubebuilder:validation:Optional
	ProviderType string `json:"providerType,omitempty"`
	// ApplicationKey is required when using application-scoped OIDC configurations.
	// +kubebuilder:validation:Optional
	ApplicationKey string `json:"applicationKey,omitempty"`
	// ProjectKey scopes the token to a JFrog project.
	// +kubebuilder:validation:Optional
	ProjectKey string `json:"projectKey,omitempty"`
	// IdentityMappingName selects a specific identity mapping.
	// +kubebuilder:validation:Optional
	IdentityMappingName string `json:"identityMappingName,omitempty"`
	// IncludeReferenceToken requests a shortened reference token in the response.
	// +kubebuilder:validation:Optional
	IncludeReferenceToken bool `json:"includeReferenceToken,omitempty"`
}

// ArtifactoryReferenceTokenAuth creates a scoped token from bootstrap credentials.
type ArtifactoryReferenceTokenAuth struct {
	// Token references a Secret containing a bootstrap identity or access token.
	// +kubebuilder:validation:Required
	Token esmeta.SecretKeySelector `json:"token"`
	// Scope defines the permissions granted to the generated token.
	// +kubebuilder:validation:Required
	Scope string `json:"scope"`
	// Description is an optional description for the generated token.
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`
	// ExpiresIn is the token lifetime in seconds.
	// +kubebuilder:validation:Optional
	ExpiresIn int64 `json:"expiresIn,omitempty"`
	// IncludeReferenceToken requests a shortened reference token in the response.
	// +kubebuilder:validation:Optional
	IncludeReferenceToken bool `json:"includeReferenceToken,omitempty"`
	// Refreshable marks the token as refreshable.
	// +kubebuilder:validation:Optional
	Refreshable bool `json:"refreshable,omitempty"`
	// ProjectKey scopes the token to a JFrog project.
	// +kubebuilder:validation:Optional
	ProjectKey string `json:"projectKey,omitempty"`
}

// ArtifactoryAccessToken generates short-lived JFrog Artifactory access tokens.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type ArtifactoryAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ArtifactoryAccessTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ArtifactoryAccessTokenList contains a list of ArtifactoryAccessToken resources.
type ArtifactoryAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArtifactoryAccessToken `json:"items"`
}
