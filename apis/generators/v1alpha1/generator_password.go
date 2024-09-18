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

// PasswordSpec controls the behavior of the password generator.
type PasswordSpec struct {
	// Length of the password to be generated.
	// Defaults to 24
	// +kubebuilder:default=24
	Length int `json:"length"`

	// Digits specifies the number of digits in the generated
	// password. If omitted it defaults to 25% of the length of the password
	Digits *int `json:"digits,omitempty"`

	// Symbols specifies the number of symbol characters in the generated
	// password. If omitted it defaults to 25% of the length of the password
	Symbols *int `json:"symbols,omitempty"`

	// SymbolCharacters specifies the special characters that should be used
	// in the generated password.
	SymbolCharacters *string `json:"symbolCharacters,omitempty"`

	// Set NoUpper to disable uppercase characters
	// +kubebuilder:default=false
	NoUpper bool `json:"noUpper"`

	// set AllowRepeat to true to allow repeating characters.
	// +kubebuilder:default=false
	AllowRepeat bool `json:"allowRepeat"`
}

// Password generates a random password based on the
// configuration parameters in spec.
// You can specify the length, characterset and other attributes.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={password},shortName=password
type Password struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PasswordSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// PasswordList contains a list of ExternalSecret resources.
type PasswordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Password `json:"items"`
}
