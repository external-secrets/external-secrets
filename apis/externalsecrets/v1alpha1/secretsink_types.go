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

type SecretSinkStoreRef struct {
	// Name of the SecretStore resource
	Name string `json:"name"`

	Status string `json:"status"`

	// Kind of the SecretStore resource (SecretStore or ClusterSecretStore)
	// Defaults to `SecretStore`
	// +optional
	Kind string `json:"kind,omitempty"`
}

// SecretSinkSpec configures the behavior of the SecretSink.
type SecretSinkSpec struct {
	SecretStoreRefs []SecretSinkStoreRef `json:"secretStoreRefs"`
	Selector        SecretSinkSelector   `json:"selector"`
	Data            []SecretSinkData     `json:"data,omitempty"`
}

type SecretSinkSecret struct {
	Name string `json:"name"`
}

type SecretSinkSelector struct {
	Secret SecretSinkSecret `json:"secret"`
}

type SecretSinkRemoteRefs struct {
	RemoteKey string `json:"remoteKey"`
}

type SecretSinkMatch struct {
	SecretKey  string                 `json:"secretKey"`
	RemoteRefs []SecretSinkRemoteRefs `json:"remoteRefs"`
}

type SecretSinkData struct {
	Match []SecretSinkMatch `json:"match"`
}

// SecretSinkConditionType indicates the condition of the SecretSink.
type SecretSinkConditionType string

const (
	SecretSinkNotImplemented SecretSinkConditionType = "NotImplemented"
)

// SecretSinkStatusCondition indicates the status of the SecretSink.
type SecretSinkStatusCondition struct {
	Type   SecretSinkConditionType `json:"type"`
	Status corev1.ConditionStatus  `json:"status"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// SecretSinkStatus indicates the history of the status of SecretSink.
type SecretSinkStatus struct {
	// +nullable
	// refreshTime is the time and date the external secret was fetched and
	// the target secret updated
	RefreshTime metav1.Time `json:"refreshTime,omitempty"`

	// SyncedResourceVersion keeps track of the last synced version.
	SyncedResourceVersion string `json:"syncedResourceVersion,omitempty"`

	// +optional
	Conditions []SecretSinkStatusCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// Secretsinks is the Schema for the secretsinks API.
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={secretsinks}

type SecretSink struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretSinkSpec   `json:"spec,omitempty"`
	Status SecretSinkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretSinkList contains a list of SecretSink resources.
type SecretSinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretSink `json:"items"`
}
