package v1alpha1

import (
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkflowSpec struct {
	// RefreshInterval is the amount of time before the workflow is being reconciled.
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h"
	// May be set to zero to fetch and create it once. Defaults to 1h.
	// +kubebuilder:default="1h"
	RefreshInterval *metav1.Duration `json:"refreshInterval,omitempty"`

	// Workflows are a list of workflows that are being executed in order.
	// +optional
	Workflows []WorkflowItem `json:"workflows"`
}
type WorkflowItem struct {
	// Name of the workflow.
	// It will be used as the index in the workflows data map.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Steps of the workflow, they are executed in order.
	// +optional
	Steps []WorkflowStep `json:"steps,omitempty"`
}

type WorkflowStep struct {
	// Name of the workflow step.
	Name string `json:"name"`

	// Pull allows you to fetch secrets from a SecretStore.
	// The secret data will be stored in the workflow data map.
	// +optional
	Pull *WorkflowStepPull `json:"pull,omitempty"`

	// Push allows you to push secrets to a SecretStore.
	// The secret data will be read from the workflow data map.
	// +optional
	Push *WorkflowStepPush `json:"push,omitempty"`

	// Template allows you to compose data from the workflow.
	// The result will be stored in the workflow data map.
	// +optional
	Template *WorkflowTemplate `json:"template,omitempty"`

	// Manifests allows you to apply manifests to the cluster. The manifests are applied in order.
	// The manifests can be templated and have access to the workflow data map.
	// +optional
	Manifests []string `json:"manifests,omitempty"`
}

type WorkflowTemplate struct {
	// Metadata allows you to set metadata on the workflow data map.
	// +optional
	Metadata v1beta1.ExternalSecretTemplateMetadata `json:"metadata,omitempty"`

	// Data allows you to compose data from the workflow. It is stored in the workflow data map.
	// Previous data can be accessed from the workflow data map.
	Data map[string]string `json:"data,omitempty"`
}

type WorkflowStepPull struct {
	// Source allows you to fetch secrets from a SecretStore.
	Source v1beta1.StoreSourceRef `json:"source"`

	// Data allows you to fetch specific data from the secret.
	// +optional
	Data []v1beta1.ExternalSecretData `json:"data,omitempty"`

	// DataFrom allows you to find multiple secrets in a store or extract structured data from a secret.
	// +optional
	DataFrom []v1beta1.ExternalSecretDataFromRemoteRef `json:"dataFrom,omitempty"`
}

type WorkflowStepPush struct {
	Destination DestinationRef   `json:"destination,omitempty"`
	Data        []PushSecretData `json:"data,omitempty"`
}

// DestinationRef allows you to override the SecretStore destination
// where the secret will be pushed to.
// +kubebuilder:validation:MaxProperties=1
type DestinationRef struct {
	// +optional
	SecretStoreRef v1beta1.SecretStoreRef `json:"storeRef,omitempty"`
}

type WorkflowStatus struct {
	Conditions []WorkflowStatusCondition `json:"conditions,omitempty"`
}

type WorkflowStatusCondition struct {
	Type   WorkflowConditionType  `json:"type"`
	Status corev1.ConditionStatus `json:"status"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

type WorkflowConditionType string

const (
	WorkflowReady WorkflowConditionType = "Ready"
)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={workflows}

type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}
