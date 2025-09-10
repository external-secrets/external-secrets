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

// SSHKeySpec controls the behavior of the ssh key generator.
type SSHKeySpec struct {
	// KeyType specifies the SSH key type (rsa, ed25519)
	// +kubebuilder:validation:Enum=rsa;ed25519
	// +kubebuilder:default="rsa"
	KeyType string `json:"keyType,omitempty"`

	// KeySize specifies the key size for RSA keys (default: 2048)
	// For RSA keys: 2048, 3072, 4096
	// Ignored for ed25519 keys
	// +kubebuilder:validation:Minimum=256
	// +kubebuilder:validation:Maximum=8192
	KeySize *int `json:"keySize,omitempty"`

	// Comment specifies an optional comment for the SSH key
	Comment string `json:"comment,omitempty"`
}

// SSHKey generates SSH key pairs.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type SSHKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SSHKeySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// SSHKeyList contains a list of SSHKey resources.
type SSHKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SSHKey `json:"items"`
}
