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

const (
	ReasonSynced  = "Synced"
	ReasonErrored = "Errored"
)

type PushSecretStoreRef struct {
	// Optionally, sync to the SecretStore of the given name
	// +optional
	Name string `json:"name"`
	// Optionally, sync to secret stores with label selector
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector"`
	// Kind of the SecretStore resource (SecretStore or ClusterSecretStore)
	// Defaults to `SecretStore`
	// +optional
	Kind string `json:"kind,omitempty"`
}

type PushSecretDeletionPolicy string

const (
	PushSecretDeletionPolicyDelete PushSecretDeletionPolicy = "Delete"
	PushSecretDeletionPolicyNone   PushSecretDeletionPolicy = "None"
)

// PushSecretSpec configures the behavior of the PushSecret.
type PushSecretSpec struct {
	// The Interval to which External Secrets will try to push a secret definition
	RefreshInterval *metav1.Duration     `json:"refreshInterval,omitempty"`
	SecretStoreRefs []PushSecretStoreRef `json:"secretStoreRefs"`
	// Deletion Policy to handle Secrets in the provider. Possible Values: "Delete/None". Defaults to "None".
	// +kubebuilder:default="None"
	DeletionPolicy PushSecretDeletionPolicy `json:"deletionPolicy"`
	// The Secret Selector (k8s source) for the Push Secret
	Selector PushSecretSelector `json:"selector"`
	// Secret Data that should be pushed to providers
	Data []PushSecretData `json:"data,omitempty"`
}

type PushSecretSecret struct {
	// Name of the Secret. The Secret must exist in the same namespace as the PushSecret manifest.
	Name string `json:"name"`
}

type PushSecretSelector struct {
	// Select a Secret to Push.
	Secret PushSecretSecret `json:"secret"`
}

type PushSecretRemoteRef struct {
	// Name of the resulting provider secret.
	RemoteKey string `json:"remoteKey"`
}

func (r PushSecretRemoteRef) GetRemoteKey() string {
	return r.RemoteKey
}

type PushSecretMatch struct {
	// Secret Key to be pushed
	SecretKey string `json:"secretKey"`
	// Remote Refs to push to providers.
	RemoteRef PushSecretRemoteRef `json:"remoteRef"`
}

type PushSecretData struct {
	// Match a given Secret Key to be pushed to the provider.
	Match PushSecretMatch `json:"match"`
}

// PushSecretConditionType indicates the condition of the PushSecret.
type PushSecretConditionType string

const (
	PushSecretReady PushSecretConditionType = "Ready"
)

// PushSecretStatusCondition indicates the status of the PushSecret.
type PushSecretStatusCondition struct {
	Type   PushSecretConditionType `json:"type"`
	Status corev1.ConditionStatus  `json:"status"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}
type SyncedPushSecretsMap map[string]map[string]PushSecretData

// PushSecretStatus indicates the history of the status of PushSecret.
type PushSecretStatus struct {
	// +nullable
	// refreshTime is the time and date the external secret was fetched and
	// the target secret updated
	RefreshTime metav1.Time `json:"refreshTime,omitempty"`

	// SyncedResourceVersion keeps track of the last synced version.
	SyncedResourceVersion string `json:"syncedResourceVersion,omitempty"`
	// Synced Push Secrets for later deletion. Matches Secret Stores to PushSecretData that was stored to that secretStore.
	// +optional
	SyncedPushSecrets SyncedPushSecretsMap `json:"syncedPushSecrets,omitempty"`
	// +optional
	Conditions []PushSecretStatusCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// PushSecrets is the Schema for the PushSecrets API.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={pushsecrets}

type PushSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PushSecretSpec   `json:"spec,omitempty"`
	Status PushSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// PushSecretList contains a list of PushSecret resources.
type PushSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PushSecret `json:"items"`
}
