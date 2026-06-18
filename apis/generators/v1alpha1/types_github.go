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

// GithubAccessTokenSpec defines the desired state to generate a GitHub access token.
// +kubebuilder:validation:XValidation:rule="has(self.appID) != has(self.appIDRef)",message="exactly one of appID or appIDRef must be set"
// +kubebuilder:validation:XValidation:rule="has(self.installID) != has(self.installIDRef)",message="exactly one of installID or installIDRef must be set"
type GithubAccessTokenSpec struct {
	// URL configures the GitHub instance URL. Defaults to https://github.com/.
	URL string `json:"url,omitempty"`
	// AppID is the GitHub App ID. Mutually exclusive with AppIDRef.
	// +optional
	AppID string `json:"appID,omitempty"`
	// AppIDRef references a secret key containing the GitHub App ID. Mutually exclusive with AppID.
	// +optional
	AppIDRef *esmeta.SecretKeySelector `json:"appIDRef,omitempty"`
	// InstallID is the GitHub App installation ID. Mutually exclusive with InstallIDRef.
	// +optional
	InstallID string `json:"installID,omitempty"`
	// InstallIDRef references a secret key containing the GitHub App installation ID. Mutually exclusive with InstallID.
	// +optional
	InstallIDRef *esmeta.SecretKeySelector `json:"installIDRef,omitempty"`
	// List of repositories the token will have access to. If omitted, defaults to all repositories the GitHub App
	// is installed to.
	Repositories []string `json:"repositories,omitempty"`
	// Map of permissions the token will have. If omitted, defaults to all permissions the GitHub App has.
	Permissions map[string]string `json:"permissions,omitempty"`
	// Auth configures how ESO authenticates with a Github instance.
	Auth GithubAuth `json:"auth"`
}

// GithubAuth defines the authentication configuration for GitHub access.
type GithubAuth struct {
	PrivateKey GithubSecretRef `json:"privateKey"`
}

// GithubSecretRef references a secret containing GitHub credentials.
type GithubSecretRef struct {
	SecretRef esmeta.SecretKeySelector `json:"secretRef"`
}

// GithubAccessToken generates ghs_ accessToken
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type GithubAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GithubAccessTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// GithubAccessTokenList contains a list of GithubAccessToken resources.
type GithubAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubAccessToken `json:"items"`
}
