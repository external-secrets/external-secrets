//Copyright External Secrets Inc. All Rights Reserved

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FakeSpec contains the static data.
type FakeSpec struct {
	// Used to select the correct ESO controller (think: ingress.ingressClassName)
	// The ESO controller is instantiated with a specific controller name and filters VDS based on this property
	// +optional
	Controller string `json:"controller,omitempty"`

	// Data defines the static data returned
	// by this generator.
	Data map[string]string `json:"data,omitempty"`
}

// Fake generator is used for testing. It lets you define
// a static set of credentials that is always returned.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={fake},shortName=fake
type Fake struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FakeSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// FakeList contains a list of ExternalSecret resources.
type FakeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Fake `json:"items"`
}
