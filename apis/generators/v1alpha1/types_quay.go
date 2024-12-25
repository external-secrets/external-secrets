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

type QuayAccessTokenSpec struct {
	// URL configures the Quay instance URL. Defaults to quay.io.
	URL string `json:"url,omitempty"`
	// Name of the robot account you are federating with
	RobotAccount string `json:"robotAccount"`
	// Name of the service account you are federating with
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef"`
}

// QuayAccessToken generates Quay oauth token for pulling/pushing images
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type QuayAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec QuayAccessTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// QuayAccessTokenList contains a list of ExternalSecret resources.
type QuayAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QuayAccessToken `json:"items"`
}
