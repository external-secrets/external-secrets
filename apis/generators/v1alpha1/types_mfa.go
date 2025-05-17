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
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MFASpec controls the behavior of the mfa generator.
type MFASpec struct {
	// Secret is a secret selector to a secret containing the seed secret to generate the TOTP value from.
	Secret smmeta.SecretKeySelector `json:"secret"`
	// Length defines the token length. Defaults to 6 characters.
	Length int `json:"length,omitempty"`
	// TimePeriod defines how long the token can be active. Defaults to 30 seconds.
	TimePeriod int `json:"timePeriod,omitempty"`
	// Algorithm to use for encoding. Defaults to SHA1 as per the RFC.
	Algorithm string `json:"algorithm,omitempty"`
	// When defines a time parameter that can be used to pin the origin time of the generated token.
	When *metav1.Time `json:"when,omitempty"`
}

// MFA generates a new TOTP token that is compliant with RFC 6238.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type MFA struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MFASpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// MFAList contains a list of MFA resources.
type MFAList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MFA `json:"items"`
}
