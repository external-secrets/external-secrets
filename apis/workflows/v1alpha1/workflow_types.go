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

	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	generatorsv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

// Workflow is the Schema for the workflows API.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=wf
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Completed",type="date",JSONPath=".status.completionTime"
// +kubebuilder:resource:scope=Namespaced,categories={workflows}
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec"`
	Status WorkflowStatus `json:"status,omitempty"`
}

// WorkflowSpec defines the desired state of Workflow.
type WorkflowSpec struct {
	// +kubebuilder:validation:Required
	Version string `json:"version"`
	// +kubebuilder:validation:Required
	Name      string               `json:"name"`
	Variables apiextensionsv1.JSON `json:"variables,omitempty"`
	// +kubebuilder:validation:Required
	Jobs map[string]Job `json:"jobs"`
}

// Phase types for workflow state machine.
type Phase string

const (
	// PhasePending indicates the workflow is pending execution.
	PhasePending Phase = "Pending"
	// PhaseRunning indicates the workflow is running.
	PhaseRunning Phase = "Running"
	// PhaseSucceeded indicates the workflow has succeeded.
	PhaseSucceeded Phase = "Succeeded"
	// PhaseFailed indicates the workflow has failed.
	PhaseFailed Phase = "Failed"
)

// JobPhase types for job state machine.
type JobPhase string

const (
	// JobPhasePending indicates the job is pending execution.
	JobPhasePending JobPhase = "Pending"
	// JobPhaseRunning indicates the job is running.
	JobPhaseRunning JobPhase = "Running"
	// JobPhaseSucceeded indicates the job has succeeded.
	JobPhaseSucceeded JobPhase = "Succeeded"
	// JobPhaseFailed indicates the job has failed.
	JobPhaseFailed JobPhase = "Failed"
)

// StepPhase types for step state machine.
type StepPhase string

const (
	// StepPhasePending indicates the step is pending execution.
	StepPhasePending StepPhase = "Pending"
	// StepPhaseRunning indicates the step is running.
	StepPhaseRunning StepPhase = "Running"
	// StepPhaseSucceeded indicates the step has succeeded.
	StepPhaseSucceeded StepPhase = "Succeeded"
	// StepPhaseFailed indicates the step has failed.
	StepPhaseFailed StepPhase = "Failed"
)

// WorkflowStatus defines the observed state of Workflow.
type WorkflowStatus struct {
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
	Phase Phase `json:"phase,omitempty"`

	JobStatuses    map[string]JobStatus `json:"jobStatuses"`
	ExecutionOrder []string             `json:"executionOrder,omitempty"`

	StartTime          *metav1.Time `json:"startTime,omitempty"`
	CompletionTime     *metav1.Time `json:"completionTime,omitempty"`
	ExecutionTimeNanos *int64       `json:"executionTimeNanos,omitempty"`
}

// Job defines a unit of work in a workflow.
type Job struct {
	DependsOn []string          `json:"dependsOn,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`

	// Standard job configuration
	// +kubebuilder:validation:Optional
	Standard *StandardJob `json:"standard,omitempty"`

	// Loop job configuration
	// +kubebuilder:validation:Optional
	Loop *LoopJob `json:"loop,omitempty"`
	// Switch job configuration
	// +kubebuilder:validation:Optional
	Switch *SwitchJob `json:"switch,omitempty"`
}

// StandardJob is the default job type that executes a sequence of steps.
type StandardJob struct {
	// +kubebuilder:validation:MinItems=1
	Steps []Step `json:"steps"`
}

// LoopJob defines a job that executes its steps in a loop.
type LoopJob struct {
	// Concurrency specifies how many iterations can run in parallel
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Required
	Concurrency int `json:"concurrency"`

	// Range is a template string that resolves to a map of strings
	// The steps will be executed for each key-value pair in the map
	// +kubebuilder:validation:Required
	Range string `json:"range"`

	// Steps contains the list of steps to execute in each iteration
	// +kubebuilder:validation:MinItems=1
	Steps []Step `json:"steps"`
}

// SwitchJob defines a job that executes different cases of steps based on conditions.
type SwitchJob struct {
	// Cases contains the different cases of steps to execute based on conditions
	// +kubebuilder:validation:MinItems=1
	Cases []SwitchCase `json:"cases"`
}

// SwitchCase defines a case of steps with a condition.
type SwitchCase struct {
	// Condition is a template string that resolves to a boolean value
	// If the condition evaluates to true, this branch will be executed
	// +kubebuilder:validation:Required
	Condition string `json:"condition"`

	// Steps contains the list of steps to execute if the condition is true
	// +kubebuilder:validation:MinItems=1
	Steps []Step `json:"steps"`
}

// OutputType defines the type of an output variable
// +kubebuilder:validation:Enum=bool;number;time;map;string
type OutputType string

const (
	// OutputTypeBool represents a boolean output type
	// This is used for true/false values.
	OutputTypeBool OutputType = "bool"

	// OutputTypeNumber represents a float64 output type
	// This is used for numeric values.
	OutputTypeNumber OutputType = "number"

	// OutputTypeTime represents a time.Time output type
	// This is used for timestamp values.
	OutputTypeTime OutputType = "time"

	// OutputTypeMap represents a map output type
	// This is used for complex data structures like JSON objects.
	OutputTypeMap OutputType = "map"

	// OutputTypeString represents a string output type
	// This is used for text values.
	OutputTypeString OutputType = "string"
)

// OutputDefinition defines an output variable from a workflow step
// This allows workflow authors to explicitly define what outputs a step provides,
// including the name, type, and sensitivity of each output.
type OutputDefinition struct {
	// Name is the name of the output variable
	// This is the key that will be used to access the output in templates
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type is the data type of the output variable
	// Supported types are: bool, number, time, and map
	// +kubebuilder:validation:Required
	Type OutputType `json:"type"`

	// Sensitive indicates whether the output should be masked in the workflow status
	// If true, the output value will be replaced with asterisks (********)
	// +kubebuilder:validation:Optional
	Sensitive bool `json:"sensitive,omitempty"`
}

// Step defines a single step in a workflow job.
type Step struct {

	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Pull *PullStep `json:"pull,omitempty"`
	// +kubebuilder:validation:Optional
	Push *PushStep `json:"push,omitempty"`
	// +kubebuilder:validation:Optional
	Debug *DebugStep `json:"debug,omitempty"`
	// +kubebuilder:validation:Optional
	Transform *TransformStep `json:"transform,omitempty"`
	// +kubebuilder:validation:Optional
	Generator *GeneratorStep `json:"generator,omitempty"`
	// +kubebuilder:validation:Optional
	JavaScript *JavaScriptStep `json:"javascript,omitempty"`
	// +kubebuilder:validation:Optional
	// Outputs defines the expected outputs from this step
	// Only values explicitly defined here will be saved in the step outputs
	Outputs []OutputDefinition `json:"outputs,omitempty"`
	// +kubebuilder:validation:Optional
}

// JavaScriptStep defines a step that executes JavaScript code with access to step input data.
type JavaScriptStep struct {
	// Script contains the JavaScript code to execute
	// +kubebuilder:validation:Required
	Script string `json:"script"`
}

// GeneratorStep defines a step that generates secrets using a configured generator.
type GeneratorStep struct {
	// GeneratorRef points to a generator custom resource.
	// +optional
	GeneratorRef *v1.GeneratorRef `json:"generatorRef,omitempty"`

	// Kind specifies the kind of generator to use when using inline generator configuration.
	// Required when using inline generator configuration.
	// +optional
	Kind generatorsv1alpha1.GeneratorKind `json:"kind,omitempty"`

	// Generator contains the inline generator configuration.
	// Required when using inline generator configuration.
	// +optional
	Generator *generatorsv1alpha1.GeneratorSpec `json:"spec,omitempty"`

	// Rewrite contains rules for rewriting the generated keys.
	// +optional
	Rewrite []v1.ExternalSecretRewrite `json:"rewrite,omitempty"`

	// AutoCleanup indicates whether to delete the old generated secrets at when creating a new one.
	// +optional
	// +kubebuilder:default=true
	AutoCleanup bool `json:"autoCleanup,omitempty"`
}

// PullStep defines a step that pulls secrets from a secret store.
type PullStep struct {
	// Source allows you to fetch secrets from a SecretStore.
	Source v1.StoreSourceRef `json:"source"`

	// Data allows you to fetch specific data from the secret.
	// +optional
	Data []v1.ExternalSecretData `json:"data,omitempty"`

	// DataFrom allows you to find multiple secrets in a store or extract structured data from a secret.
	// +optional
	DataFrom []v1.ExternalSecretDataFromRemoteRef `json:"dataFrom,omitempty"`
}

// PushStep defines a step that pushes secrets to a destination.
type PushStep struct {
	// SecretSource defines the source map in the workflow context,
	// indicating where to retrieve the secret values
	SecretSource string `json:"secretSource,omitempty"`

	Destination DestinationRef `json:"destination,omitempty"`

	Data []v1alpha1.PushSecretData `json:"data,omitempty"`
}

// TransformStep defines a step that transforms data.
type TransformStep struct {
	// Transform is a map of key-value pairs, where the value is a template string
	// that will be dynamically resolved at runtime against the workflow data.
	Mappings map[string]string `json:"mappings,omitempty"`

	// Full YAML template to be generated during transformation
	Template string `json:"template,omitempty"`
}

// DestinationRef allows you to override the SecretStore destination
// where the secret will be pushed to.
// +kubebuilder:validation:MaxProperties=1
type DestinationRef struct {
	SecretStoreRef v1.SecretStoreRef `json:"storeRef,omitempty"`
}

// DebugStep defines a step that outputs debug information.
type DebugStep struct {
	// +kubebuilder:validation:Required
	Message string `json:"message"`
}

// JobStatus defines the observed state of a Job.
type JobStatus struct {
	// +kubebuilder:validation:Enum=Pending;Waiting;Running;Succeeded;Failed
	Phase JobPhase `json:"phase,omitempty"`

	StepStatuses       map[string]StepStatus `json:"stepStatuses"`
	StartTime          *metav1.Time          `json:"startTime,omitempty"`
	CompletionTime     *metav1.Time          `json:"completionTime,omitempty"`
	ExecutionTimeNanos *int64                `json:"executionTimeNanos,omitempty"`
}

// StepStatus defines the observed state of a Step.
type StepStatus struct {
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
	Phase StepPhase `json:"phase,omitempty"`
	// +optional
	Outputs            map[string]string `json:"outputs,omitempty"`
	Message            string            `json:"message,omitempty"`
	StartTime          *metav1.Time      `json:"startTime,omitempty"`
	CompletionTime     *metav1.Time      `json:"completionTime,omitempty"`
	ExecutionTimeNanos *int64            `json:"executionTimeNanos,omitempty"`
}

// WorkflowList contains a list of Workflow.
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}
