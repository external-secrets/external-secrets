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

// SecretStoreRef defines which SecretStore to fetch the ExternalSecret data.
type SecretStoreRef struct {
	// Name of the SecretStore resource
	Name string `json:"name"`

	// Kind of the SecretStore resource (SecretStore or ClusterSecretStore)
	// Defaults to `SecretStore`
	// +optional
	Kind string `json:"kind,omitempty"`
}

// ExternalSecretCreationPolicy defines rules on how to create the resulting Secret.
type ExternalSecretCreationPolicy string

const (
	// Owner creates the Secret and sets .metadata.ownerReferences to the ExternalSecret resource.
	Owner ExternalSecretCreationPolicy = "Owner"

	// Merge does not create the Secret, but merges the data fields to the Secret.
	Merge ExternalSecretCreationPolicy = "Merge"

	// None does not create a Secret (future use with injector).
	None ExternalSecretCreationPolicy = "None"
)

// ExternalSecretTemplateMetadata defines metadata fields for the Secret blueprint.
type ExternalSecretTemplateMetadata struct {
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// ExternalSecretTemplate defines a blueprint for the created Secret resource.
type ExternalSecretTemplate struct {
	// +optional
	Type corev1.SecretType `json:"type,omitempty"`

	// +optional
	Metadata ExternalSecretTemplateMetadata `json:"metadata,omitempty"`
}

// ExternalSecretTarget defines the Kubernetes Secret to be created
// There can be only one target per ExternalSecret.
type ExternalSecretTarget struct {
	// Name defines the name of the Secret resource to be managed
	// This field is immutable
	// Defaults to the .metadata.name of the ExternalSecret resource
	// +optional
	Name string `json:"name,omitempty"`

	// CreationPolicy defines rules on how to create the resulting Secret
	// Defaults to 'Owner'
	// +optional
	CreationPolicy ExternalSecretCreationPolicy `json:"creationPolicy,omitempty"`
}

// ExternalSecretData defines the connection between the Kubernetes Secret key (spec.data.<key>) and the Provider data.
type ExternalSecretData struct {
	SecretKey string `json:"secretKey"`

	RemoteRef ExternalSecretDataRemoteRef `json:"remoteRef"`
}

// ExternalSecretDataRemoteRef defines Provider data location.
type ExternalSecretDataRemoteRef struct {
	// Key is the key used in the Provider, mandatory
	Key string `json:"key"`

	// Used to select a specific version of the Provider value, if supported
	// +optional
	Version string `json:"version,omitempty"`

	// +optional
	// Used to select a specific property of the Provider value (if a map), if supported
	Property string `json:"property,omitempty"`
}

// ExternalSecretSpec defines the desired state of ExternalSecret.
type ExternalSecretSpec struct {
	SecretStoreRef SecretStoreRef `json:"secretStoreRef"`

	Target ExternalSecretTarget `json:"target"`

	// RefreshInterval is the amount of time before the values reading again from the SecretStore provider
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h" (from time.ParseDuration)
	// May be set to zero to fetch and create it once
	// TODO: Default to some value?
	// +optional
	RefreshInterval string `json:"refreshInterval,omitempty"`

	// Data defines the connection between the Kubernetes Secret keys and the Provider data
	// +optional
	Data []ExternalSecretData `json:"data,omitempty"`

	// DataFrom is used to fetch all properties from a specific Provider data
	// If multiple entries are specified, the Secret keys are merged in the specified order
	// +optional
	DataFrom []ExternalSecretDataRemoteRef `json:"dataFrom,omitempty"`
}

// ExternalSecretStatusPhase represents the current phase of the Secret sync.
type ExternalSecretStatusPhase string

const (
	// ExternalSecret created, controller did not yet sync the ExternalSecret or other dependencies are missing (e.g. secret store or configmap template).
	ExternalSecretPending ExternalSecretStatusPhase = "Pending"

	// ExternalSecret is being actively synced according to spec.
	ExternalSecretSyncing ExternalSecretStatusPhase = "Syncing"

	// ExternalSecret can not be synced, this might require user intervention.
	ExternalSecretFailing ExternalSecretStatusPhase = "Failing"

	// ExternalSecret can not be synced right now and will not able to.
	ExternalSecretFailed ExternalSecretStatusPhase = "Failed"

	// ExternalSecret was synced successfully (one-time use only).
	ExternalSecretCompleted ExternalSecretStatusPhase = "Completed"
)

type ExternalSecretConditionType string

const (
	InSync ExternalSecretConditionType = "InSync"
)

type ExternalSecretStatusCondition struct {
	Type   SecretStoreConditionType `json:"type"`
	Status corev1.ConditionStatus   `json:"status"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// +optional
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`
}

type ExternalSecretStatus struct {
	// +optional
	Phase ExternalSecretStatusPhase `json:"phase"`

	// +optional
	Conditions []ExternalSecretStatusCondition `json:"conditions"`
}

// +kubebuilder:object:root=true

// ExternalSecret is the Schema for the external-secrets API.
type ExternalSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalSecretSpec   `json:"spec,omitempty"`
	Status ExternalSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ExternalSecretList contains a list of ExternalSecret resources.
type ExternalSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalSecret `json:"items"`
}
