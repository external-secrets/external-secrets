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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StoreProvider string

const (
	AWSSM StoreProvider = "AWSSM"
	GCPSM StoreProvider = "GCPSM"
	Vault StoreProvider = "VAULT"
)

// SecretStoreSpec defines the desired state of SecretStore.
type SecretStoreSpec struct {
	// Used to select the correct KES controller (think: ingress.ingressClassName)
	// The KES controller is instantiated with a specific controller name and filters ES based on this property
	// +optional
	Controller string `json:"controller"`

	// Used to configure the provider. Only one provider may be set
	Provider *SecretStoreProvider `json:"provider"`
}

// SecretStoreProvider contains the provider-specific configration.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type SecretStoreProvider struct {
	// AWSSM configures this store to sync secrets using AWS Secret Manager provider
	// +optional
	AWSSM *AWSSMProvider `json:"awssm,omitempty"`
}

type SecretStoreStatusPhase string

const (
	// E.g. referenced Secret containing credentials is missing.
	SecretStorePending SecretStoreStatusPhase = "Pending"

	// All dependencies are met, sync.
	SecretStoreRunning SecretStoreStatusPhase = "Running"
)

type SecretStoreConditionType string

const (
	Ready SecretStoreConditionType = "Ready"
)

type SecretStoreStatusCondition struct {
	Type   SecretStoreConditionType `json:"type"`
	Status corev1.ConditionStatus   `json:"status"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// SecretStoreStatus defines the observed state of the SecretStore.
type SecretStoreStatus struct {
	// +optional
	Phase SecretStoreStatusPhase `json:"phase"`

	// +optional
	Conditions []SecretStoreStatusCondition `json:"conditions"`
}

// +kubebuilder:object:root=true

// SecretStore is the Schema for the secretstores API.
type SecretStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretStoreSpec   `json:"spec,omitempty"`
	Status SecretStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretStoreList contains a list of SecretStore.
type SecretStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretStore{}, &SecretStoreList{})
}
