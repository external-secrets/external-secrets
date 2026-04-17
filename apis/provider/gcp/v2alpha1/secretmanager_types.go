/*
Copyright © The ESO Authors

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// SecretManagerSpec defines the desired state of SecretManager.
type SecretManagerSpec struct {
	ProjectID string       `json:"projectID,omitempty"`
	Location  string       `json:"location,omitempty"`
	Auth      v1.GCPSMAuth `json:"auth,omitempty"`
}

// SecretManagerStatus defines the observed state of SecretManager.
type SecretManagerStatus struct {
	// Conditions represent the latest available observations of the resource's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets},shortName=gcpsm

// SecretManager is the Schema for GCP Secret Manager provider configuration.
type SecretManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretManagerSpec   `json:"spec,omitempty"`
	Status SecretManagerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretManagerList contains a list of SecretManager resources.
type SecretManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretManager{}, &SecretManagerList{})
}
