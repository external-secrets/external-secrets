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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProviderSpec defines the desired state of Provider.
type ProviderSpec struct {
	// Config contains configuration for connecting to the provider.
	Config ProviderConfig `json:"config"`
}

// ProviderConfig defines how to connect to a provider service.
type ProviderConfig struct {
	// Address is the gRPC address of the provider service.
	// Format: "hostname:port" (e.g., "aws-provider:8080")
	// +kubebuilder:validation:Required
	Address string `json:"address"`

	// ProviderRef references the provider-specific configuration resource.
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`
}

// ProviderReference references a provider-specific configuration resource.
type ProviderReference struct {
	// APIVersion of the referenced resource.
	// Example: "provider.aws.external-secrets.io/v2alpha1"
	// +kubebuilder:validation:Required
	APIVersion string `json:"apiVersion"`

	// Kind of the referenced resource.
	// Example: "AWSSecretsManager"
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Name of the referenced resource.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the referenced resource.
	// If empty, assumes the same namespace as the Provider.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// ProviderCapabilities defines the possible operations a Provider can do.
type ProviderCapabilities string

const (
	// ProviderReadOnly indicates the provider supports read-only operations.
	ProviderReadOnly ProviderCapabilities = "ReadOnly"
	// ProviderWriteOnly indicates the provider supports write-only operations.
	ProviderWriteOnly ProviderCapabilities = "WriteOnly"
	// ProviderReadWrite indicates the provider supports both read and write operations.
	ProviderReadWrite ProviderCapabilities = "ReadWrite"
)

// ProviderStatus defines the observed state of Provider.
type ProviderStatus struct {
	// Conditions represent the latest available observations of the Provider's state.
	// +optional
	Conditions []ProviderCondition `json:"conditions,omitempty"`

	// Capabilities indicates what operations this Provider supports.
	// +optional
	Capabilities ProviderCapabilities `json:"capabilities,omitempty"`
}

// ProviderCondition describes the state of a Provider at a certain point.
type ProviderCondition struct {
	// Type of the condition.
	Type ProviderConditionType `json:"type"`

	// Status of the condition, one of True, False, Unknown.
	Status metav1.ConditionStatus `json:"status"`

	// LastTransitionTime is the last time the condition transitioned.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason contains a programmatic identifier indicating the reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human-readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// ProviderConditionType defines the type of Provider condition.
type ProviderConditionType string

const (
	// ProviderReady indicates that the Provider is ready to serve requests.
	ProviderReady ProviderConditionType = "Ready"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={externalsecrets},shortName=prov
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.spec.config.address`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:storageversion

// Provider is the Schema for the providers API.
type Provider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderSpec   `json:"spec,omitempty"`
	Status ProviderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderList contains a list of Provider.
type ProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Provider `json:"items"`
}

// AuthenticationScope defines which namespace should be used for authentication.
type AuthenticationScope string

const (
	// AuthenticationScopeProviderNamespace uses the namespace from spec.config.providerRef.namespace
	// for authentication. This is the default.
	AuthenticationScopeProviderNamespace AuthenticationScope = "ProviderNamespace"

	// AuthenticationScopeManifestNamespace uses the namespace of the ExternalSecret/PushSecret
	// for authentication.
	AuthenticationScopeManifestNamespace AuthenticationScope = "ManifestNamespace"
)

// ClusterProviderSpec defines the desired state of ClusterProvider.
type ClusterProviderSpec struct {
	// Config contains configuration for connecting to the provider.
	Config ProviderConfig `json:"config"`

	// AuthenticationScope defines which namespace should be used for authentication.
	// ProviderNamespace (default): uses the namespace from spec.config.providerRef.namespace
	// ManifestNamespace: uses the namespace of the ExternalSecret/PushSecret
	// +kubebuilder:validation:Enum=ProviderNamespace;ManifestNamespace
	// +kubebuilder:default=ProviderNamespace
	// +optional
	AuthenticationScope AuthenticationScope `json:"authenticationScope,omitempty"`

	// Conditions constrain where this ClusterProvider can be used from.
	// Conditions are evaluated against the namespace of the ExternalSecret/PushSecret.
	// +optional
	Conditions []ClusterSecretStoreCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={externalsecrets},shortName=cprov
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.spec.config.address`
// +kubebuilder:printcolumn:name="AuthScope",type=string,JSONPath=`.spec.authenticationScope`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:storageversion

// ClusterProvider is the cluster-scoped variant of Provider.
// It can be referenced from ExternalSecrets and PushSecrets in any namespace.
type ClusterProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterProviderSpec `json:"spec,omitempty"`
	Status ProviderStatus      `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterProviderList contains a list of ClusterProvider.
type ClusterProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterProvider `json:"items"`
}

