// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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

// BasicAuthSpec controls the behavior of the basic auth generator.
type BasicAuthSpec struct {
	Username UsernameSpec `json:"username,omitempty"`
	Password PasswordSpec `json:"password,omitempty"`
}

// len int,
// prefix string,
// sufix string,
// wordCount int,
// separator string,
// includeNumbers bool,

// UsernameSpec controls the behavior of the username generated.
type UsernameSpec struct {
	// Length of each word of the username to be generated.
	// Defaults to 8
	// +kubebuilder:default=8
	Length int `json:"length"`

	// Prefix specifies a prefix to be added to the generated
	// username. If omitted it defaults to empty
	Prefix *string `json:"prefix,omitempty"`

	// Sufix specifies a sufix to be added to the generated
	// username. If omitted it defaults to empty
	Sufix *string `json:"sufix,omitempty"`

	// WordCount specifies the number of words in the generated
	// username. If omitted it defaults to 1
	// +kubebuilder:default=1
	WordCount int `json:"wordCount,omitempty"`

	// Separator specifies the separator character that should be used
	// in the generated username. If omitted it defaults to "_"
	Separator *string `json:"separator,omitempty"`

	// set IncludeNumbers to add 4 numbers at the end of the username after the sufix.
	// +kubebuilder:default=false
	IncludeNumbers bool `json:"includeNumbers"`
}

// BasicAuth generates a random basic auth based on the
// configuration parameters in spec.
// You can specify the length, characterset and other attributes.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type BasicAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BasicAuthSpec   `json:"spec,omitempty"`
	Status GeneratorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BasicAuthList contains a list of ExternalSecret resources.
type BasicAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BasicAuth `json:"items"`
}
