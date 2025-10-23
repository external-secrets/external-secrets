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

package v2alpha1

import (
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Fake defines the configuration for the Fake provider.
// This provider returns static key-value pairs for testing purposes.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets},shortName=fake
// +genclient.
type Fake struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec esv1.FakeProvider `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// FakeList contains a list of Fake resources.
type FakeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Fake `json:"items"`
}

// FakeProviderSpec defines the desired state of Fake provider.
// It matches the structure of v1.FakeProvider for easy conversion.
// +kubebuilder:object:generate=true
type FakeProviderSpec struct {
	// Data defines the static key-value pairs to return.
	Data []FakeProviderData `json:"data"`

	// ValidationResult optionally specifies the validation result for testing.
	// +optional
	ValidationResult *string `json:"validationResult,omitempty"`
}

// FakeProviderData defines a key-value pair with optional version.
// +kubebuilder:object:generate=true
type FakeProviderData struct {
	// Key is the secret key.
	Key string `json:"key"`

	// Value is the secret value.
	Value string `json:"value"`

	// Version is an optional version identifier.
	// +optional
	Version string `json:"version,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Fake{}, &FakeList{})
}
