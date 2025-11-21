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

// Package v1alpha1 contains API Schema definitions for the scan v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobSpec defines the desired state of Job.
type JobSpec struct {
	// Constrains this job to a given set of SecretStores / Targets.
	// By default it will run against all SecretStores / Targets on the Job namespace.
	Constraints *JobConstraints `json:"constraints,omitempty"`
	// Defines the RunPolicy for this job (Poll/OnChange/Once)
	// +kubebuilder:validation:Enum=Poll;OnChange;Once
	RunPolicy JobRunPolicy `json:"runPolicy,omitempty"`
	// Defines the interval for this job if Policy is Poll(Poll/OnChange/Once)
	Interval metav1.Duration `json:"interval,omitempty"`
	// TODO - also implement Cron Schedulingf
	// Define the interval to wait before forcing reconcile if job froze at running state
	// +kubebuilder:default="10m"
	JobTimeout metav1.Duration `json:"jobTimeout,omitempty"`
}

// JobRunPolicy defines the run policy for a job.
type JobRunPolicy string

const (
	// JobRunPolicyPull defines the run policy for a job.
	JobRunPolicyPull JobRunPolicy = "Poll"
	// JobRunPolicyOnChange defines the run policy for a job.
	JobRunPolicyOnChange JobRunPolicy = "OnChange"
	// JobRunPolicyOnce defines the run policy for a job.
	JobRunPolicyOnce JobRunPolicy = "Once"
)

// JobConstraints defines the constraints for a job.
type JobConstraints struct {
	// SecretStoreConstraints defines the constraints for a job.
	SecretStoreConstraints []SecretStoreConstraint `json:"secretStoreConstraints,omitempty"`
	// TargetConstraints defines the constraints for a job.
	TargetConstraints []TargetConstraint `json:"targetConstraints,omitempty"`
}

// SecretStoreConstraint defines the constraints for a job.
type SecretStoreConstraint struct {
	// MatchExpressions defines the constraints for a job.
	MatchExpressions []metav1.LabelSelector `json:"matchExpression,omitempty"`
	// MatchLabels defines the constraints for a job.
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// TargetConstraint defines the constraints for a job.
type TargetConstraint struct {
	// Kind defines the kind of the target.
	Kind string `json:"kind,omitempty"`
	// APIVersion defines the API version of the target.
	APIVersion string `json:"apiVersion,omitempty"`
	// MatchExpressions defines the constraints for a job.
	MatchExpressions []metav1.LabelSelector `json:"matchExpression,omitempty"`
	// MatchLabels defines the constraints for a job.
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// JobStatus defines the observed state of a Job.
type JobStatus struct {
	// ObservedSecretStoresDigest is a digest of the SecretStores that were used in the last run.
	// +optional
	ObservedSecretStoresDigest string `json:"observedSecretStoresDigest,omitempty"`
	// ObservedTargetsDigest is a digest of the Targets that were used in the last run.
	// +optional
	ObservedTargetsDigest string `json:"observedTargetsDigest,omitempty"`
	// LastRunTime is the time when the job was last run.
	LastRunTime metav1.Time `json:"lastRunTime,omitempty"`
	// RunStatus is the status of the job.
	RunStatus JobRunStatus `json:"runStatus,omitempty"`
	// Conditions is a list of metav1.Condition.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// JobRunStatus defines the status of a job.
type JobRunStatus string

const (
	// JobRunStatusRunning defines the status of a job.
	JobRunStatusRunning JobRunStatus = "Running"
	// JobRunStatusSucceeded defines the status of a job.
	JobRunStatusSucceeded JobRunStatus = "Succeeded"
	// JobRunStatusFailed defines the status of a job.
	JobRunStatusFailed JobRunStatus = "Failed"
)

// Job is the schema to run a scan job over targets
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-scan}
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JobSpec   `json:"spec,omitempty"`
	Status            JobStatus `json:"status,omitempty"`
}

// JobList contains a list of Job resources.
// +kubebuilder:object:root=true
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Job `json:"items"`
}
