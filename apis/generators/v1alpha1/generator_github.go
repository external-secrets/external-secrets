/*
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

type GithubAccessTokenSpec struct {
	// URL configures the Github instance URL. Defaults to https://github.com/.
	URL       string `json:"url,omitempty"`
	AppID     string `json:"appID"`
	InstallID string `json:"installID"`
	// Auth configures how ESO authenticates with a Github instance.
	Auth GithubAuth `json:"auth"`
}

type GithubAuth struct {
	PrivateKey GithubSecretRef `json:"privateKey"`
}

type GithubSecretRef struct {
	SecretRef esmeta.SecretKeySelector `json:"secretRef"`
}

// GithubAccessToken generates ghs_ accessToken
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={githubaccesstoken},shortName=githubaccesstoken
type GithubAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GithubAccessTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// GithubAccessToken contains a list of ExternalSecret resources.
type GithubAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubAccessToken `json:"items"`
}
