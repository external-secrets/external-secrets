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

	// corev1 "k8s.io/api/core/v1".
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// Secretsinks is the Schema for the secretsinks API.
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={secretsinks}

type SecretSink struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   string `json:"spec,omitempty"`
	Status string `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretSinkList contains a list of SecretSink resources.
type SecretSinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretSink `json:"items"`
}
