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
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil
type StatefulResource interface {
	runtime.Object
	metav1.Object
}

const (
	// The owner key points to the resource which created the generator state.
	// It is used in the garbage collection process to identify all states
	// that belong to a specific resource.
	GeneratorStateLabelOwnerKey = "generators.external-secrets.io/owner-key"
)

type GeneratorStateSpec struct {
	// GarbageCollectionDeadline is the time after which the generator state
	// will be deleted.
	// It is set by the controller which creates the generator state and
	// can be set configured by the user.
	// If the garbage collection deadline is not set the generator state will not be deleted.
	GarbageCollectionDeadline *metav1.Time `json:"garbageCollectionDeadline,omitempty"`

	// Resource is the generator manifest that produced the state.
	// It is a snapshot of the generator manifest at the time the state was produced.
	// This manifest will be used to delete the resource. Any configuration that is referenced
	// in the manifest should be available at the time of garbage collection. If that is not the case deletion will
	// be blocked by a finalizer.
	Resource *apiextensions.JSON `json:"resource"`
	// State is the state that was produced by the generator implementation.
	State *apiextensions.JSON `json:"state"`
}

type GeneratorStateConditionType string

const (
	GeneratorStateReady GeneratorStateConditionType = "Ready"
)

type GeneratorStateStatusCondition struct {
	Type   GeneratorStateConditionType `json:"type"`
	Status corev1.ConditionStatus      `json:"status"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

const (
	ConditionReasonCreated = "Created"
	ConditionReasonError   = "Error"
)

type GeneratorStateStatus struct {
	Conditions []GeneratorStateStatusCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:printcolumn:name="GC Deadline",type="string",JSONPath=".spec.garbageCollectionDeadline"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators},shortName=gs
type GeneratorState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GeneratorStateSpec   `json:"spec,omitempty"`
	Status GeneratorStateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GeneratorStateList contains a list of ExternalSecret resources.
type GeneratorStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GeneratorState `json:"items"`
}
