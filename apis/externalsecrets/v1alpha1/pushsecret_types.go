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
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// PushSecret condition reasons.
const (
	// ReasonSynced indicates that the push secret was successfully synced to the provider.
	ReasonSynced = "Synced"
	// ReasonErrored indicates that the push secret encountered an error during sync.
	ReasonErrored = "Errored"
)

// PushSecretStoreRef contains a reference on how to sync to a SecretStore.
type PushSecretStoreRef struct {
	// Optionally, sync to the SecretStore of the given name
	// +optional
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name,omitempty"`

	// Optionally, sync to secret stores with label selector
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// Kind of the SecretStore resource (SecretStore or ClusterSecretStore)
	// +optional
	// +kubebuilder:default="SecretStore"
	// +kubebuilder:validation:Enum=SecretStore;ClusterSecretStore
	Kind string `json:"kind,omitempty"`
}

// PushSecretUpdatePolicy defines how push secrets are updated in the provider.
// +kubebuilder:validation:Enum=Replace;IfNotExists
type PushSecretUpdatePolicy string

const (
	// PushSecretUpdatePolicyReplace replaces existing secrets in the provider.
	PushSecretUpdatePolicyReplace PushSecretUpdatePolicy = "Replace"
	// PushSecretUpdatePolicyIfNotExists only creates secrets that don't exist in the provider.
	PushSecretUpdatePolicyIfNotExists PushSecretUpdatePolicy = "IfNotExists"
)

// PushSecretDeletionPolicy defines how push secrets are deleted in the provider.
// +kubebuilder:validation:Enum=Delete;None
type PushSecretDeletionPolicy string

const (
	// PushSecretDeletionPolicyDelete deletes secrets from the provider when the PushSecret is deleted.
	PushSecretDeletionPolicyDelete PushSecretDeletionPolicy = "Delete"
	// PushSecretDeletionPolicyNone keeps secrets in the provider when the PushSecret is deleted.
	PushSecretDeletionPolicyNone PushSecretDeletionPolicy = "None"
)

// PushSecretConversionStrategy defines how secret values are converted when pushed to providers.
// +kubebuilder:validation:Enum=None;ReverseUnicode
type PushSecretConversionStrategy string

const (
	// PushSecretConversionNone indicates no conversion will be performed on the secret value.
	PushSecretConversionNone PushSecretConversionStrategy = "None"
	// PushSecretConversionReverseUnicode indicates that unicode escape sequences will be reversed.
	PushSecretConversionReverseUnicode PushSecretConversionStrategy = "ReverseUnicode"
)

// PushSecretSpec configures the behavior of the PushSecret.
type PushSecretSpec struct {
	// The Interval to which External Secrets will try to push a secret definition
	// +kubebuilder:default="1h"
	RefreshInterval *metav1.Duration `json:"refreshInterval,omitempty"`

	SecretStoreRefs []PushSecretStoreRef `json:"secretStoreRefs"`

	// UpdatePolicy to handle Secrets in the provider.
	// +kubebuilder:default="Replace"
	// +optional
	UpdatePolicy PushSecretUpdatePolicy `json:"updatePolicy,omitempty"`

	// Deletion Policy to handle Secrets in the provider.
	// +kubebuilder:default="None"
	// +optional
	DeletionPolicy PushSecretDeletionPolicy `json:"deletionPolicy,omitempty"`

	// The Secret Selector (k8s source) for the Push Secret
	Selector PushSecretSelector `json:"selector"`

	// Secret Data that should be pushed to providers
	Data []PushSecretData `json:"data,omitempty"`

	// Template defines a blueprint for the created Secret resource.
	// +optional
	Template *esv1.ExternalSecretTemplate `json:"template,omitempty"`
}

// PushSecretSecret defines a Secret that will be used as a source for pushing to providers.
type PushSecretSecret struct {
	// Name of the Secret.
	// The Secret must exist in the same namespace as the PushSecret manifest.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	// +optional
	Name string `json:"name,omitempty"`

	// Selector chooses secrets using a labelSelector.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// PushSecretSelector defines criteria for selecting the source Secret for pushing to providers.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type PushSecretSelector struct {
	// Select a Secret to Push.
	// +optional
	Secret *PushSecretSecret `json:"secret,omitempty"`

	// Point to a generator to create a Secret.
	// +optional
	GeneratorRef *esv1.GeneratorRef `json:"generatorRef,omitempty"`
}

// PushSecretRemoteRef defines the location of the secret in the provider.
type PushSecretRemoteRef struct {
	// Name of the resulting provider secret.
	RemoteKey string `json:"remoteKey"`

	// Name of the property in the resulting secret
	// +optional
	Property string `json:"property,omitempty"`
}

// GetRemoteKey returns the RemoteKey of this reference.
func (r PushSecretRemoteRef) GetRemoteKey() string {
	return r.RemoteKey
}

// GetProperty returns the Property of this reference.
func (r PushSecretRemoteRef) GetProperty() string {
	return r.Property
}

// PushSecretMatch defines how a source Secret key maps to a destination in the provider.
type PushSecretMatch struct {
	// Secret Key to be pushed
	// +optional
	SecretKey string `json:"secretKey,omitempty"`
	// Remote Refs to push to providers.
	RemoteRef PushSecretRemoteRef `json:"remoteRef"`
}

// PushSecretData defines data to be pushed to the provider and associated metadata.
type PushSecretData struct {
	// Match a given Secret Key to be pushed to the provider.
	Match PushSecretMatch `json:"match"`
	// Metadata is metadata attached to the secret.
	// The structure of metadata is provider specific, please look it up in the provider documentation.
	// +optional
	Metadata *apiextensionsv1.JSON `json:"metadata,omitempty"`
	// +optional
	// Used to define a conversion Strategy for the secret keys
	// +kubebuilder:default="None"
	ConversionStrategy PushSecretConversionStrategy `json:"conversionStrategy,omitempty"`
}

// GetMetadata returns the metadata of the PushSecretData.
func (d PushSecretData) GetMetadata() *apiextensionsv1.JSON {
	return d.Metadata
}

// GetSecretKey returns the secret key from the PushSecretData match.
func (d PushSecretData) GetSecretKey() string {
	return d.Match.SecretKey
}

// GetRemoteKey returns the remote key from the PushSecretData match.
func (d PushSecretData) GetRemoteKey() string {
	return d.Match.RemoteRef.RemoteKey
}

// GetProperty returns the property from the PushSecretData match.
func (d PushSecretData) GetProperty() string {
	return d.Match.RemoteRef.Property
}

// PushSecretConditionType indicates the condition of the PushSecret.
type PushSecretConditionType string

const (
	// PushSecretReady indicates the PushSecret resource is ready.
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

// SyncedPushSecretsMap is a map that tracks which PushSecretData was stored to which secret store.
// The outer map's key is the secret store name, and the inner map's key is the remote key name.
type SyncedPushSecretsMap map[string]map[string]PushSecretData

// PushSecretStatus indicates the history of the status of PushSecret.
type PushSecretStatus struct {
	// +nullable
	// refreshTime is the time and date the external secret was fetched and
	// the target secret updated
	RefreshTime metav1.Time `json:"refreshTime,omitempty"`

	// SyncedResourceVersion keeps track of the last synced version.
	SyncedResourceVersion string `json:"syncedResourceVersion,omitempty"`
	// Synced PushSecrets, including secrets that already exist in provider.
	// Matches secret stores to PushSecretData that was stored to that secret store.
	// +optional
	SyncedPushSecrets SyncedPushSecretsMap `json:"syncedPushSecrets,omitempty"`
	// +optional
	Conditions []PushSecretStatusCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets},shortName=ps

// PushSecret is the Schema for the PushSecrets API that enables pushing Kubernetes secrets to external secret providers.
type PushSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PushSecretSpec   `json:"spec,omitempty"`
	Status PushSecretStatus `json:"status,omitempty"`
}

// PushSecretList contains a list of PushSecret resources.
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
type PushSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PushSecret `json:"items"`
}

// ClusterPushSecretCondition used to refine PushSecrets to specific namespaces and names.
type ClusterPushSecretCondition struct {
	// Choose namespace using a labelSelector
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Choose namespaces by name
	// +optional
	// +kubebuilder:validation:items:MinLength:=1
	// +kubebuilder:validation:items:MaxLength:=63
	// +kubebuilder:validation:items:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Namespaces []string `json:"namespaces,omitempty"`
}

// PushSecretMetadata defines metadata fields for the PushSecret generated by the ClusterPushSecret.
type PushSecretMetadata struct {
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// ClusterPushSecretSpec defines the configuration for a ClusterPushSecret resource.
type ClusterPushSecretSpec struct {
	// PushSecretSpec defines what to do with the secrets.
	PushSecretSpec PushSecretSpec `json:"pushSecretSpec"`
	// The time in which the controller should reconcile its objects and recheck namespaces for labels.
	RefreshInterval *metav1.Duration `json:"refreshTime,omitempty"`
	// The name of the push secrets to be created.
	// Defaults to the name of the ClusterPushSecret
	// +optional
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	PushSecretName string `json:"pushSecretName,omitempty"`

	// The metadata of the external secrets to be created
	// +optional
	PushSecretMetadata PushSecretMetadata `json:"pushSecretMetadata,omitempty"`

	// A list of labels to select by to find the Namespaces to create the ExternalSecrets in. The selectors are ORed.
	// +optional
	NamespaceSelectors []*metav1.LabelSelector `json:"namespaceSelectors,omitempty"`
}

// ClusterPushSecretNamespaceFailure represents a failed namespace deployment and it's reason.
type ClusterPushSecretNamespaceFailure struct {

	// Namespace is the namespace that failed when trying to apply an PushSecret
	Namespace string `json:"namespace"`

	// Reason is why the PushSecret failed to apply to the namespace
	// +optional
	Reason string `json:"reason,omitempty"`
}

// ClusterPushSecretStatus contains the status information for the ClusterPushSecret resource.
type ClusterPushSecretStatus struct {
	// Failed namespaces are the namespaces that failed to apply an PushSecret
	// +optional
	FailedNamespaces []ClusterPushSecretNamespaceFailure `json:"failedNamespaces,omitempty"`

	// ProvisionedNamespaces are the namespaces where the ClusterPushSecret has secrets
	// +optional
	ProvisionedNamespaces []string `json:"provisionedNamespaces,omitempty"`
	PushSecretName        string   `json:"pushSecretName,omitempty"`

	// +optional
	Conditions []PushSecretStatusCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Cluster,categories={external-secrets}

// ClusterPushSecret is the Schema for the ClusterPushSecrets API that enables cluster-wide management of pushing Kubernetes secrets to external providers.
type ClusterPushSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterPushSecretSpec   `json:"spec,omitempty"`
	Status ClusterPushSecretStatus `json:"status,omitempty"`
}

// ClusterPushSecretList contains a list of ClusterPushSecret resources.
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
type ClusterPushSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPushSecret `json:"items"`
}
