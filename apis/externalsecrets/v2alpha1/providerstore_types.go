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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StoreRuntimeRef struct {
	// +kubebuilder:validation:Enum=ClusterProviderClass
	// +kubebuilder:default=ClusterProviderClass
	// +optional
	Kind string `json:"kind,omitempty"`

	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name"`
}

type BackendObjectReference struct {
	// APIVersion of the referenced backend resource.
	// +kubebuilder:validation:MinLength:=1
	APIVersion string `json:"apiVersion"`

	// Kind of the referenced backend resource.
	// +kubebuilder:validation:MinLength:=1
	Kind string `json:"kind"`

	// Name of the referenced backend resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name"`

	// Namespace of the referenced backend resource.
	// +optional
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Namespace string `json:"namespace,omitempty"`
}

// StoreNamespaceCondition describes conditions that constrain where a cluster store can be used from.
type StoreNamespaceCondition struct {
	// Choose namespace using a labelSelector.
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Choose namespaces by name.
	// +optional
	// +kubebuilder:validation:items:MinLength:=1
	// +kubebuilder:validation:items:MaxLength:=63
	// +kubebuilder:validation:items:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Namespaces []string `json:"namespaces,omitempty"`

	// Choose namespaces by using regex matching.
	// +optional
	NamespaceRegexes []string `json:"namespaceRegexes,omitempty"`
}

type ProviderStoreConditionType string

const (
	// ProviderStoreReady indicates that the store is ready and able to serve requests.
	ProviderStoreReady ProviderStoreConditionType = "Ready"
)

// ProviderStoreCondition describes the state of a store at a certain point.
type ProviderStoreCondition struct {
	Type   ProviderStoreConditionType `json:"type"`
	Status corev1.ConditionStatus     `json:"status"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`
}

// ProviderStoreStatus defines the observed state of a provider store.
type ProviderStoreStatus struct {
	// +optional
	Conditions []ProviderStoreCondition `json:"conditions,omitempty"`
}

// ProviderStoreSpec defines the desired state of ProviderStore.
type ProviderStoreSpec struct {
	// RuntimeRef points to the runtime configuration used by this store.
	RuntimeRef StoreRuntimeRef `json:"runtimeRef"`

	// BackendRef references the provider-owned backend configuration object.
	BackendRef BackendObjectReference `json:"backendRef"`
}

// ClusterProviderStoreSpec defines the desired state of ClusterProviderStore.
type ClusterProviderStoreSpec struct {
	// RuntimeRef points to the runtime configuration used by this store.
	RuntimeRef StoreRuntimeRef `json:"runtimeRef"`

	// BackendRef references the provider-owned backend configuration object.
	BackendRef BackendObjectReference `json:"backendRef"`

	// Conditions constrain where this ClusterProviderStore can be used from.
	// +optional
	Conditions []StoreNamespaceCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={externalsecrets},shortName=pstore
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Runtime",type=string,JSONPath=`.spec.runtimeRef.name`
// +kubebuilder:printcolumn:name="Backend",type=string,JSONPath=`.spec.backendRef.name`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ProviderStore is the namespaced clean store API.
type ProviderStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderStoreSpec   `json:"spec,omitempty"`
	Status ProviderStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderStoreList contains a list of ProviderStore.
type ProviderStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderStore `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={externalsecrets},shortName=cpstore
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Runtime",type=string,JSONPath=`.spec.runtimeRef.name`
// +kubebuilder:printcolumn:name="Backend",type=string,JSONPath=`.spec.backendRef.name`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ClusterProviderStore is the cluster-scoped clean store API.
type ClusterProviderStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterProviderStoreSpec `json:"spec,omitempty"`
	Status ProviderStoreStatus      `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterProviderStoreList contains a list of ClusterProviderStore.
type ClusterProviderStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterProviderStore `json:"items"`
}
