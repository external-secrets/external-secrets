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
)

// UUIDSpec controls the behavior of the uuid generator.
type UUIDSpec struct{}

// UUID generates a version 1 UUID (e56657e3-764f-11ef-a397-65231a88c216).
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type UUID struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec UUIDSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// UUIDList contains a list of ExternalSecret resources.
type UUIDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UUID `json:"items"`
}
