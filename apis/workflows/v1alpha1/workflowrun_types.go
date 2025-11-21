// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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

// Package v1alpha1 contains API Schema definitions for the workflows v1alpha1 API group
package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowRunSpec defines the desired state of WorkflowRun.
type WorkflowRunSpec struct {
	// TemplateRef is a reference to the template to use
	// +required
	TemplateRef TemplateRef `json:"templateRef"`

	// Arguments are the values for template parameters
	// Each argument corresponds to a parameter defined in the template
	// +optional
	Arguments apiextensionsv1.JSON `json:"arguments,omitempty"`
}

// TemplateRef is a reference to a WorkflowTemplate.
type TemplateRef struct {
	// Name of the template
	// +required
	Name string `json:"name"`

	// Namespace of the template
	// If not specified, the namespace of the WorkflowRun is used
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// WorkflowRunStatus defines the observed state of WorkflowRun.
type WorkflowRunStatus struct {
	// WorkflowRef is a reference to the created workflow
	// +optional
	WorkflowRef *WorkflowRef `json:"workflowRef,omitempty"`

	// SensitiveValuesSecrets is a list of secret names containing sensitive values
	// +optional
	SensitiveValuesSecrets []string `json:"sensitiveValuesSecrets,omitempty"`

	// Conditions represent the latest available observations of the WorkflowRun's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current phase of the WorkflowRun
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
	// +optional
	Phase Phase `json:"phase,omitempty"`

	// StartTime represents when the WorkflowRun started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime represents when the WorkflowRun completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	// ExecutionTimeNanos represents the duration between the start and completion of the WorkflowRun, in nanoseconds
	// +optional
	ExecutionTimeNanos *int64 `json:"executionTimeNanos,omitempty"`
}

// WorkflowRef is a reference to a Workflow.
type WorkflowRef struct {
	// Name of the workflow
	// +required
	Name string `json:"name"`

	// Namespace of the workflow
	// +required
	Namespace string `json:"namespace"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PHASE",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="TEMPLATE",type="string",JSONPath=".spec.templateRef.name"
// +kubebuilder:printcolumn:name="WORKFLOW",type="string",JSONPath=".status.workflowRef.name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// WorkflowRun is the Schema for the workflowruns API.
type WorkflowRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowRunSpec   `json:"spec"`
	Status WorkflowRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkflowRunList contains a list of WorkflowRun.
type WorkflowRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkflowRun{}, &WorkflowRunList{})
}
