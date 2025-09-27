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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretStoreRef defines which SecretStore to fetch the ExternalSecret data.
type SecretStoreRef struct {
	// Name of the SecretStore resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name,omitempty"`

	// Kind of the SecretStore resource (SecretStore or ClusterSecretStore)
	// Defaults to `SecretStore`
	// +optional
	// +kubebuilder:validation:Enum=SecretStore;ClusterSecretStore
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

	// +optional
	Finalizers []string `json:"finalizers,omitempty"`
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

	// +kubebuilder:default="Replace"
	MergePolicy TemplateMergePolicy `json:"mergePolicy,omitempty"`

	// +optional
	Data map[string]string `json:"data,omitempty"`

	// +optional
	TemplateFrom []TemplateFrom `json:"templateFrom,omitempty"`
}

// +kubebuilder:validation:Enum=Replace;Merge
type TemplateMergePolicy string

const (
	MergePolicyReplace TemplateMergePolicy = "Replace"
	MergePolicyMerge   TemplateMergePolicy = "Merge"
)

// +kubebuilder:validation:Enum=v2
type TemplateEngineVersion string

const (
	TemplateEngineV2 TemplateEngineVersion = "v2"
)

type TemplateFrom struct {
	ConfigMap *TemplateRef `json:"configMap,omitempty"`
	Secret    *TemplateRef `json:"secret,omitempty"`

	// +optional
	// +kubebuilder:default="Data"
	Target TemplateTarget `json:"target,omitempty"`

	// +optional
	Literal *string `json:"literal,omitempty"`
}

// +kubebuilder:validation:Enum=Values;KeysAndValues
type TemplateScope string

const (
	TemplateScopeValues        TemplateScope = "Values"
	TemplateScopeKeysAndValues TemplateScope = "KeysAndValues"
)

// +kubebuilder:validation:Enum=Data;Annotations;Labels
type TemplateTarget string

const (
	TemplateTargetData        TemplateTarget = "Data"
	TemplateTargetAnnotations TemplateTarget = "Annotations"
	TemplateTargetLabels      TemplateTarget = "Labels"
)

type TemplateRef struct {
	// The name of the ConfigMap/Secret resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name"`

	// A list of keys in the ConfigMap/Secret to use as templates for Secret data
	Items []TemplateRefItem `json:"items"`
}

type TemplateRefItem struct {
	// A key in the ConfigMap/Secret
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[-._a-zA-Z0-9]+$
	Key string `json:"key"`

	// +kubebuilder:default="Values"
	TemplateAs TemplateScope `json:"templateAs,omitempty"`
}

// ExternalSecretTarget defines the Kubernetes Secret to be created
// There can be only one target per ExternalSecret.
type ExternalSecretTarget struct {
	// The name of the Secret resource to be managed.
	// Defaults to the .metadata.name of the ExternalSecret resource
	// +optional
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name,omitempty"`

	// CreationPolicy defines rules on how to create the resulting Secret.
	// Defaults to "Owner"
	// +optional
	// +kubebuilder:default="Owner"
	CreationPolicy ExternalSecretCreationPolicy `json:"creationPolicy,omitempty"`

	// DeletionPolicy defines rules on how to delete the resulting Secret.
	// Defaults to "Retain"
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
	// The key in the Kubernetes Secret to store the value.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[-._a-zA-Z0-9]+$
	SecretKey string `json:"secretKey"`

	// RemoteRef points to the remote secret and defines
	// which secret (version/property/..) to fetch.
	RemoteRef ExternalSecretDataRemoteRef `json:"remoteRef"`

	// SourceRef allows you to override the source
	// from which the value will be pulled.
	SourceRef *StoreSourceRef `json:"sourceRef,omitempty"`
}

// ExternalSecretDataRemoteRef defines Provider data location.
type ExternalSecretDataRemoteRef struct {
	// Key is the key used in the Provider, mandatory
	Key string `json:"key"`

	// +optional
	// Policy for fetching tags/labels from provider secrets, possible options are Fetch, None. Defaults to None
	// +kubebuilder:default="None"
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

// +kubebuilder:validation:Enum=None;Fetch
type ExternalSecretMetadataPolicy string

const (
	ExternalSecretMetadataPolicyNone  ExternalSecretMetadataPolicy = "None"
	ExternalSecretMetadataPolicyFetch ExternalSecretMetadataPolicy = "Fetch"
)

// +kubebuilder:validation:Enum=Default;Unicode
type ExternalSecretConversionStrategy string

const (
	ExternalSecretConversionDefault ExternalSecretConversionStrategy = "Default"
	ExternalSecretConversionUnicode ExternalSecretConversionStrategy = "Unicode"
)

// +kubebuilder:validation:Enum=Auto;Base64;Base64URL;None
type ExternalSecretDecodingStrategy string

const (
	ExternalSecretDecodeAuto      ExternalSecretDecodingStrategy = "Auto"
	ExternalSecretDecodeBase64    ExternalSecretDecodingStrategy = "Base64"
	ExternalSecretDecodeBase64URL ExternalSecretDecodingStrategy = "Base64URL"
	ExternalSecretDecodeNone      ExternalSecretDecodingStrategy = "None"
)

type ExternalSecretDataFromRemoteRef struct {
	// Used to extract multiple key/value pairs from one secret
	// Note: Extract does not support sourceRef.Generator or sourceRef.GeneratorRef.
	// +optional
	Extract *ExternalSecretDataRemoteRef `json:"extract,omitempty"`
	// Used to find secrets based on tags or regular expressions
	// Note: Find does not support sourceRef.Generator or sourceRef.GeneratorRef.
	// +optional
	Find *ExternalSecretFind `json:"find,omitempty"`

	// Used to rewrite secret Keys after getting them from the secret Provider
	// Multiple Rewrite operations can be provided. They are applied in a layered order (first to last)
	// +optional
	Rewrite []ExternalSecretRewrite `json:"rewrite,omitempty"`

	// SourceRef points to a store or generator
	// which contains secret values ready to use.
	// Use this in combination with Extract or Find pull values out of
	// a specific SecretStore.
	// When sourceRef points to a generator Extract or Find is not supported.
	// The generator returns a static map of values
	SourceRef *StoreGeneratorSourceRef `json:"sourceRef,omitempty"`
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type ExternalSecretRewrite struct {

	// Used to merge key/values in one single Secret
	// The resulting key will contain all values from the specified secrets
	// +optional
	Merge *ExternalSecretRewriteMerge `json:"merge,omitempty"`

	// Used to rewrite with regular expressions.
	// The resulting key will be the output of a regexp.ReplaceAll operation.
	// +optional
	Regexp *ExternalSecretRewriteRegexp `json:"regexp,omitempty"`

	// Used to apply string transformation on the secrets.
	// The resulting key will be the output of the template applied by the operation.
	// +optional
	Transform *ExternalSecretRewriteTransform `json:"transform,omitempty"`
}

type ExternalSecretRewriteMerge struct {
	// Used to define the target key of the merge operation.
	// Required if strategy is JSON. Ignored otherwise.
	// +optional
	// +kubebuilder:default=""
	Into string `json:"into,omitempty"`

	// Used to define key priority in conflict resolution.
	// +optional
	Priority []string `json:"priority,omitempty"`

	// Used to define the policy when a key in the priority list does not exist in the input.
	// +optional
	// +kubebuilder:default="Strict"
	PriorityPolicy ExternalSecretRewriteMergePriorityPolicy `json:"priorityPolicy,omitempty"`

	// Used to define the policy to use in conflict resolution.
	// +optional
	// +kubebuilder:default="Error"
	ConflictPolicy ExternalSecretRewriteMergeConflictPolicy `json:"conflictPolicy,omitempty"`

	// Used to define the strategy to use in the merge operation.
	// +optional
	// +kubebuilder:default="Extract"
	Strategy ExternalSecretRewriteMergeStrategy `json:"strategy,omitempty"`
}

// +kubebuilder:validation:Enum=Ignore;Error
type ExternalSecretRewriteMergeConflictPolicy string

const (
	ExternalSecretRewriteMergeConflictPolicyIgnore ExternalSecretRewriteMergeConflictPolicy = "Ignore"
	ExternalSecretRewriteMergeConflictPolicyError  ExternalSecretRewriteMergeConflictPolicy = "Error"
)

// +kubebuilder:validation:Enum=IgnoreNotFound;Strict
type ExternalSecretRewriteMergePriorityPolicy string

const (
	ExternalSecretRewriteMergePriorityPolicyIgnoreNotFound ExternalSecretRewriteMergePriorityPolicy = "IgnoreNotFound"
	ExternalSecretRewriteMergePriorityPolicyStrict         ExternalSecretRewriteMergePriorityPolicy = "Strict"
)

// +kubebuilder:validation:Enum=Extract;JSON
type ExternalSecretRewriteMergeStrategy string

const (
	ExternalSecretRewriteMergeStrategyExtract ExternalSecretRewriteMergeStrategy = "Extract"
	ExternalSecretRewriteMergeStrategyJSON    ExternalSecretRewriteMergeStrategy = "JSON"
)

type ExternalSecretRewriteRegexp struct {
	// Used to define the regular expression of a re.Compiler.
	Source string `json:"source"`
	// Used to define the target pattern of a ReplaceAll operation.
	Target string `json:"target"`
}

type ExternalSecretRewriteTransform struct {
	// Used to define the template to apply on the secret name.
	// `.value ` will specify the secret name in the template.
	Template string `json:"template"`
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

// +kubebuilder:validation:Enum=CreatedOnce;Periodic;OnChange
type ExternalSecretRefreshPolicy string

const (
	RefreshPolicyCreatedOnce ExternalSecretRefreshPolicy = "CreatedOnce"
	RefreshPolicyPeriodic    ExternalSecretRefreshPolicy = "Periodic"
	RefreshPolicyOnChange    ExternalSecretRefreshPolicy = "OnChange"
)

// ExternalSecretSpec defines the desired state of ExternalSecret.
type ExternalSecretSpec struct {
	// +optional
	SecretStoreRef SecretStoreRef `json:"secretStoreRef,omitempty"`

	// +kubebuilder:default={creationPolicy:Owner,deletionPolicy:Retain}
	// +optional
	Target ExternalSecretTarget `json:"target,omitempty"`

	// RefreshPolicy determines how the ExternalSecret should be refreshed:
	// - CreatedOnce: Creates the Secret only if it does not exist and does not update it thereafter
	// - Periodic: Synchronizes the Secret from the external source at regular intervals specified by refreshInterval.
	//   No periodic updates occur if refreshInterval is 0.
	// - OnChange: Only synchronizes the Secret when the ExternalSecret's metadata or specification changes
	// +optional
	RefreshPolicy ExternalSecretRefreshPolicy `json:"refreshPolicy,omitempty"`

	// RefreshInterval is the amount of time before the values are read again from the SecretStore provider,
	// specified as Golang Duration strings.
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h"
	// Example values: "1h", "2h30m", "10s"
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

// StoreSourceRef allows you to override the SecretStore source
// from which the secret will be pulled from.
// You can define at maximum one property.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type StoreSourceRef struct {
	// +optional
	SecretStoreRef SecretStoreRef `json:"storeRef,omitempty"`

	// GeneratorRef points to a generator custom resource.
	//
	// Deprecated: The generatorRef is not implemented in .data[].
	// this will be removed with v1.
	GeneratorRef *GeneratorRef `json:"generatorRef,omitempty"`
}

// StoreGeneratorSourceRef allows you to override the source
// from which the secret will be pulled from.
// You can define at maximum one property.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type StoreGeneratorSourceRef struct {
	// +optional
	SecretStoreRef *SecretStoreRef `json:"storeRef,omitempty"`

	// GeneratorRef points to a generator custom resource.
	// +optional
	GeneratorRef *GeneratorRef `json:"generatorRef,omitempty"`
}

// GeneratorRef points to a generator custom resource.
type GeneratorRef struct {
	// Specify the apiVersion of the generator resource
	// +kubebuilder:default="generators.external-secrets.io/v1alpha1"
	APIVersion string `json:"apiVersion,omitempty"`

	// Specify the Kind of the generator resource
	// +kubebuilder:validation:Enum=ACRAccessToken;ClusterGenerator;CloudsmithAccessToken;ECRAuthorizationToken;Fake;GCRAccessToken;GithubAccessToken;QuayAccessToken;Password;SSHKey;STSSessionToken;UUID;VaultDynamicSecret;Webhook;Grafana;MFA
	Kind string `json:"kind"`

	// Specify the name of the generator resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name"`
}

// +kubebuilder:validation:Enum=Ready;Deleted
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
	// ConditionReasonSecretMissing indicates that the secret is missing.
	ConditionReasonSecretMissing = "SecretMissing"

	ReasonUpdateFailed          = "UpdateFailed"
	ReasonDeprecated            = "ParameterDeprecated"
	ReasonCreated               = "Created"
	ReasonUpdated               = "Updated"
	ReasonDeleted               = "Deleted"
	ReasonMissingProviderSecret = "MissingProviderSecret"
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

	// Binding represents a servicebinding.io Provisioned Service reference to the secret
	Binding corev1.LocalObjectReference `json:"binding,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// ExternalSecret is the Schema for the external-secrets API.
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets},shortName=es
// +kubebuilder:printcolumn:name="StoreType",type=string,JSONPath=`.spec.secretStoreRef.kind`
// +kubebuilder:printcolumn:name="Store",type=string,JSONPath=`.spec.secretStoreRef.name`
// +kubebuilder:printcolumn:name="Refresh Interval",type=string,JSONPath=`.spec.refreshInterval`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:selectablefield:JSONPath=`.spec.secretStoreRef.name`
// +kubebuilder:selectablefield:JSONPath=`.spec.secretStoreRef.kind`
// +kubebuilder:selectablefield:JSONPath=`.spec.target.name`
// +kubebuilder:selectablefield:JSONPath=`.spec.refreshInterval`
type ExternalSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalSecretSpec   `json:"spec,omitempty"`
	Status ExternalSecretStatus `json:"status,omitempty"`
}

const (
	// AnnotationDataHash all secrets managed by an ExternalSecret have this annotation with the hash of their data.
	AnnotationDataHash = "reconcile.external-secrets.io/data-hash"
	// AnnotationForceSync all ExternalSecrets managed by a ClusterExternalSecret mirror the state and value of this annotation.
	AnnotationForceSync = "external-secrets.io/force-sync"

	// LabelManaged all secrets managed by an ExternalSecret will have this label equal to "true".
	LabelManaged      = "reconcile.external-secrets.io/managed"
	LabelManagedValue = "true"

	// LabelOwner points to the owning ExternalSecret resource when CreationPolicy=Owner.
	LabelOwner = "reconcile.external-secrets.io/created-by"
)

// +kubebuilder:object:root=true

// ExternalSecretList contains a list of ExternalSecret resources.
type ExternalSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalSecret `json:"items"`
}
