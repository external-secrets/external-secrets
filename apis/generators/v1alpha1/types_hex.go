/*
Copyright © 2025 ESO Maintainer Team

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
)

// HexSpec controls the behavior of the hex string generator.
type HexSpec struct {
	// Length of the hex string to be generated.
	// Defaults to 16 (8 bytes)
	// +kubebuilder:default=16
	Length int `json:"length"`

	// Uppercase specifies whether to use uppercase letters (A-F) instead of lowercase (a-f)
	// +kubebuilder:default=false
	Uppercase bool `json:"uppercase"`

	// Prefix specifies an optional prefix to add to the hex string (e.g., "0x")
	// +optional
	Prefix string `json:"prefix,omitempty"`
}

// Hex generates a random hexadecimal string based on the
// configuration parameters in spec.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type Hex struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec HexSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// HexList contains a list of Hex resources.
type HexList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Hex `json:"items"`
}
