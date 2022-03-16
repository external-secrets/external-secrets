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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterExternalSecretSpec defines the desired state of ClusterExternalSecret.
type ClusterExternalSecretSpec struct {
	// The spec for the ExternalSecrets to be created
	ExternalSecretSpec ExternalSecretSpec `json:"externalSecretSpec"`

	// The name of the external secrets to be created defaults to the name of the ClusterExternalSecret
	// +optional
	ExternalSecretName string `json:"externalSecretName"`

	// The labels to select by to find the Namespaces to create the ExternalSecrets in.
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector"`

	// The time in which the controller should reconcile it's objects and recheck namespaces for labels.
	RefreshInterval *metav1.Duration `json:"refreshTime,omitempty"`
}

type ClusterExternalSecretConditionType string

const (
	ClusterExternalSecretReady          ClusterExternalSecretConditionType = "Ready"
	ClusterExternalSecretPartiallyReady ClusterExternalSecretConditionType = "PartiallyReady"
	ClusterExternalSecretNotReady       ClusterExternalSecretConditionType = "NotReady"
)

type ClusterExternalSecretStatusCondition struct {
	Type   ClusterExternalSecretConditionType `json:"type"`
	Status corev1.ConditionStatus             `json:"status"`

	// +optional
	Message string `json:"message,omitempty"`
}

// ClusterExternalSecretNamespaceFailure represents a failed namespace deployment and it's reason.
type ClusterExternalSecretNamespaceFailure struct {

	// Namespace is the namespace that failed when trying to apply an ExternalSecret
	Namespace string `json:"namespace"`

	// Reason is why the ExternalSecret failed to apply to the namespace
	// +optional
	Reason string `json:"reason,omitempty"`
}

// ClusterExternalSecretStatus defines the observed state of ClusterExternalSecret.
type ClusterExternalSecretStatus struct {
	// Failed namespaces are the namespaces that failed to apply an ExternalSecret
	// +optional
	FailedNamespaces []ClusterExternalSecretNamespaceFailure `json:"failedNamespaces,omitempty"`

	// ProvisionedNamespaces are the namespaces where the ClusterExternalSecret has secrets
	// +optional
	ProvisionedNamespaces []string `json:"provisionedNamespaces,omitempty"`

	// +optional
	Conditions []ClusterExternalSecretStatusCondition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster,categories={externalsecrets},shortName=ces
//+kubebuilder:subresource:status
// ClusterExternalSecret is the Schema for the clusterexternalsecrets API.
type ClusterExternalSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterExternalSecretSpec   `json:"spec,omitempty"`
	Status ClusterExternalSecretStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterExternalSecretList contains a list of ClusterExternalSecret.
type ClusterExternalSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterExternalSecret `json:"items"`
}
