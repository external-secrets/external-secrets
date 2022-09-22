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
// +kubebuilder:validation:Enum=Owner;Orphan;Merge;None
type ExternalSecretCreationPolicy string

const (
	// Owner creates the Secret and sets .metadata.ownerReferences to the ExternalSecret resource.
	CreatePolicyOwner ExternalSecretCreationPolicy = "Owner"

	// Orphan creates the Secret and does not set the ownerReference.
	// I.e. it will be orphaned after the deletion of the ExternalSecret.
	CreatePolicyOrphan ExternalSecretCreationPolicy = "Orphan"

	// Merge does not create the Secret, but merges the data fields to the Secret.
	CreatePolicyMerge ExternalSecretCreationPolicy = "Merge"

	// None does not create a Secret (future use with injector).
	CreatePolicyNone ExternalSecretCreationPolicy = "None"
)

// ExternalSecretDeletionPolicy defines rules on how to delete the resulting Secret.
// +kubebuilder:validation:Enum=Delete;Merge;Retain
type ExternalSecretDeletionPolicy string

const (
	// Delete deletes the secret if all provider secrets are deleted.
	// If a secret gets deleted on the provider side and is not accessible
	// anymore this is not considered an error and the ExternalSecret
	// does not go into SecretSyncedError status.
	DeletionPolicyDelete ExternalSecretDeletionPolicy = "Delete"

	// Merge removes keys in the secret, but not the secret itself.
	// If a secret gets deleted on the provider side and is not accessible
	// anymore this is not considered an error and the ExternalSecret
	// does not go into SecretSyncedError status.
	DeletionPolicyMerge ExternalSecretDeletionPolicy = "Merge"

	// Retain will retain the secret if all provider secrets have been deleted.
	// If a provider secret does not exist the ExternalSecret gets into the
	// SecretSyncedError status.
	DeletionPolicyRetain ExternalSecretDeletionPolicy = "Retain"
)

// ExternalSecretTemplateMetadata defines metadata fields for the Secret blueprint.
type ExternalSecretTemplateMetadata struct {
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// ExternalSecretTemplate defines a blueprint for the created Secret resource.
// we can not use native corev1.Secret, it will have empty ObjectMeta values: https://github.com/kubernetes-sigs/controller-tools/issues/448
type ExternalSecretTemplate struct {
	// +optional
	Type corev1.SecretType `json:"type,omitempty"`

	// EngineVersion specifies the template engine version
	// that should be used to compile/execute the
	// template specified in .data and .templateFrom[].
	// +kubebuilder:default="v2"

	EngineVersion TemplateEngineVersion `json:"engineVersion,omitempty"`
	// +optional
	Metadata ExternalSecretTemplateMetadata `json:"metadata,omitempty"`

	// +optional
	Data map[string]string `json:"data,omitempty"`

	// +optional
	TemplateFrom []TemplateFrom `json:"templateFrom,omitempty"`
}

type TemplateEngineVersion string

const (
	TemplateEngineV1 TemplateEngineVersion = "v1"
	TemplateEngineV2 TemplateEngineVersion = "v2"
)

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type TemplateFrom struct {
	ConfigMap *TemplateRef `json:"configMap,omitempty"`
	Secret    *TemplateRef `json:"secret,omitempty"`
}

type TemplateRef struct {
	Name  string            `json:"name"`
	Items []TemplateRefItem `json:"items"`
}

type TemplateRefItem struct {
	Key string `json:"key"`
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
	// +kubebuilder:default="Owner"
	CreationPolicy ExternalSecretCreationPolicy `json:"creationPolicy,omitempty"`
	// DeletionPolicy defines rules on how to delete the resulting Secret
	// Defaults to 'Retain'
	// +optional
	// +kubebuilder:default="Retain"
	DeletionPolicy ExternalSecretDeletionPolicy `json:"deletionPolicy,omitempty"`
	// Template defines a blueprint for the created Secret resource.
	// +optional
	Template *ExternalSecretTemplate `json:"template,omitempty"`

	// Immutable defines if the final secret will be immutable
	// +optional
	Immutable bool `json:"immutable,omitempty"`
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

	// +optional
	// Policy for fetching tags/labels from provider secrets, possible options are Fetch, None. Defaults to None
	MetadataPolicy ExternalSecretMetadataPolicy `json:"metadataPolicy,omitempty"`

	// +optional
	// Used to select a specific property of the Provider value (if a map), if supported
	Property string `json:"property,omitempty"`

	// +optional
	// Used to select a specific version of the Provider value, if supported
	Version string `json:"version,omitempty"`

	// +optional
	// Used to define a conversion Strategy
	// +kubebuilder:default="Default"
	ConversionStrategy ExternalSecretConversionStrategy `json:"conversionStrategy,omitempty"`

	// +optional
	// Used to define a decoding Strategy
	// +kubebuilder:default="None"
	DecodingStrategy ExternalSecretDecodingStrategy `json:"decodingStrategy,omitempty"`
}

type ExternalSecretMetadataPolicy string

const (
	ExternalSecretMetadataPolicyNone  ExternalSecretMetadataPolicy = "None"
	ExternalSecretMetadataPolicyFetch ExternalSecretMetadataPolicy = "Fetch"
)

type ExternalSecretConversionStrategy string

const (
	ExternalSecretConversionDefault ExternalSecretConversionStrategy = "Default"
	ExternalSecretConversionUnicode ExternalSecretConversionStrategy = "Unicode"
)

type ExternalSecretDecodingStrategy string

const (
	ExternalSecretDecodeAuto      ExternalSecretDecodingStrategy = "Auto"
	ExternalSecretDecodeBase64    ExternalSecretDecodingStrategy = "Base64"
	ExternalSecretDecodeBase64URL ExternalSecretDecodingStrategy = "Base64URL"
	ExternalSecretDecodeNone      ExternalSecretDecodingStrategy = "None"
)

type ExternalSecretDataFromRemoteRef struct {
	// Used to extract multiple key/value pairs from one secret
	// +optional
	Extract *ExternalSecretDataRemoteRef `json:"extract,omitempty"`
	// Used to find secrets based on tags or regular expressions
	// +optional
	Find *ExternalSecretFind `json:"find,omitempty"`

	// Used to rewrite secret Keys after getting them from the secret Provider
	// Multiple Rewrite operations can be provided. They are applied in a layered order (first to last)
	// +optional
	Rewrite []ExternalSecretRewrite `json:"rewrite,omitempty"`
}

type ExternalSecretRewrite struct {
	// Used to rewrite with regular expressions.
	// The resulting key will be the output of a regexp.ReplaceAll operation.
	// +optional
	Regexp *ExternalSecretRewriteRegexp `json:"regexp,omitempty"`
}

type ExternalSecretRewriteRegexp struct {
	// Used to define the regular expression of a re.Compiler.
	Source string `json:"source"`
	// Used to define the target pattern of a ReplaceAll operation.
	Target string `json:"target"`
}
type ExternalSecretFind struct {
	// A root path to start the find operations.
	// +optional
	Path *string `json:"path,omitempty"`
	// Finds secrets based on the name.
	// +optional
	Name *FindName `json:"name,omitempty"`

	// Find secrets based on tags.
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// +optional
	// Used to define a conversion Strategy
	// +kubebuilder:default="Default"
	ConversionStrategy ExternalSecretConversionStrategy `json:"conversionStrategy,omitempty"`

	// +optional
	// Used to define a decoding Strategy
	// +kubebuilder:default="None"
	DecodingStrategy ExternalSecretDecodingStrategy `json:"decodingStrategy,omitempty"`
}

type FindName struct {
	// Finds secrets base
	// +optional
	RegExp string `json:"regexp,omitempty"`
}

// ExternalSecretSpec defines the desired state of ExternalSecret.
type ExternalSecretSpec struct {
	SecretStoreRef SecretStoreRef `json:"secretStoreRef"`
	// +kubebuilder:default={creationPolicy:Owner,deletionPolicy:Retain}
	// +optional
	Target ExternalSecretTarget `json:"target,omitempty"`

	// RefreshInterval is the amount of time before the values are read again from the SecretStore provider
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h"
	// May be set to zero to fetch and create it once. Defaults to 1h.
	// +kubebuilder:default="1h"
	RefreshInterval *metav1.Duration `json:"refreshInterval,omitempty"`

	// Data defines the connection between the Kubernetes Secret keys and the Provider data
	// +optional
	Data []ExternalSecretData `json:"data,omitempty"`

	// DataFrom is used to fetch all properties from a specific Provider data
	// If multiple entries are specified, the Secret keys are merged in the specified order
	// +optional
	DataFrom []ExternalSecretDataFromRemoteRef `json:"dataFrom,omitempty"`
}

type ExternalSecretConditionType string

const (
	ExternalSecretReady   ExternalSecretConditionType = "Ready"
	ExternalSecretDeleted ExternalSecretConditionType = "Deleted"
)

type ExternalSecretStatusCondition struct {
	Type   ExternalSecretConditionType `json:"type"`
	Status corev1.ConditionStatus      `json:"status"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

const (
	// ConditionReasonSecretSynced indicates that the secrets was synced.
	ConditionReasonSecretSynced = "SecretSynced"
	// ConditionReasonSecretSyncedError indicates that there was an error syncing the secret.
	ConditionReasonSecretSyncedError = "SecretSyncedError"
	// ConditionReasonSecretDeleted indicates that the secret has been deleted.
	ConditionReasonSecretDeleted = "SecretDeleted"

	ReasonInvalidStoreRef      = "InvalidStoreRef"
	ReasonUnavailableStore     = "UnavailableStore"
	ReasonProviderClientConfig = "InvalidProviderClientConfig"
	ReasonUpdateFailed         = "UpdateFailed"
	ReasonDeprecated           = "ParameterDeprecated"
	ReasonUpdated              = "Updated"
	ReasonDeleted              = "Deleted"
)

type ExternalSecretStatus struct {
	// +nullable
	// refreshTime is the time and date the external secret was fetched and
	// the target secret updated
	RefreshTime metav1.Time `json:"refreshTime,omitempty"`

	// SyncedResourceVersion keeps track of the last synced version
	SyncedResourceVersion string `json:"syncedResourceVersion,omitempty"`

	// +optional
	Conditions []ExternalSecretStatusCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// ExternalSecret is the Schema for the external-secrets API.
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={externalsecrets},shortName=es
// +kubebuilder:printcolumn:name="Store",type=string,JSONPath=`.spec.secretStoreRef.name`
// +kubebuilder:printcolumn:name="Refresh Interval",type=string,JSONPath=`.spec.refreshInterval`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
type ExternalSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalSecretSpec   `json:"spec,omitempty"`
	Status ExternalSecretStatus `json:"status,omitempty"`
}

const (
	// AnnotationDataHash is used to ensure consistency.
	AnnotationDataHash = "reconcile.external-secrets.io/data-hash"
)

// +kubebuilder:object:root=true

// ExternalSecretList contains a list of ExternalSecret resources.
type ExternalSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalSecret `json:"items"`
}
