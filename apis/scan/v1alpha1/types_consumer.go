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

// ConsumerConditionType defines the type of a Consumer condition.
type ConsumerConditionType string

const (
	// ConsumerLatestVersion indicates that the consumer is using the latest version of the external-secrets-enterprise.
	ConsumerLatestVersion ConsumerConditionType = "UsingLatestVersion"
)

const (
	// ConsumerLocationsUpToDate indicates that the consumer is using the latest version of the external-secrets-enterprise.
	ConsumerLocationsUpToDate ConsumerConditionType = "LocationsUpToDate"
	// ConsumerLocationsOutOfDate indicates that the consumer is using the latest version of the external-secrets-enterprise.
	ConsumerLocationsOutOfDate ConsumerConditionType = "LocationsOutOfDate"
	// ConsumerWorkloadReady indicates that the consumer is using the latest version of the external-secrets-enterprise.
	ConsumerWorkloadReady ConsumerConditionType = "WorkloadReady"
	// ConsumerWorkloadNotReady indicates that the consumer is using the latest version of the external-secrets-enterprise.
	ConsumerWorkloadNotReady ConsumerConditionType = "WorkloadNotReady"
	// ConsumerNotReady indicates that the consumer is using the latest version of the external-secrets-enterprise.
	ConsumerNotReady ConsumerConditionType = "ConsumerNotReady"
)

// ConsumerSpec defines the desired state of Consumer.
type ConsumerSpec struct {
	Target TargetReference `json:"target"`

	// Type discriminates which payload below is populated.
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// A stable ID for correlation across scans.
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id"`

	// Human readable name for UIs.
	DisplayName string `json:"displayName"`

	// Exactly one of the following should be set according to Type.
	// +kubebuilder:validation:Required
	Attributes ConsumerAttrs `json:"attributes"`
}

// ConsumerAttrs defines the attributes of a Consumer.
type ConsumerAttrs struct {
	// VMProcess defines the attributes of a VM process.
	VMProcess *VMProcessSpec `json:"vmProcess,omitempty"`
	// GitHubActor defines the attributes of a GitHub actor.
	GitHubActor *GitHubActorSpec `json:"gitHubActor,omitempty"`
	// K8sWorkload defines the attributes of a Kubernetes workload.
	K8sWorkload *K8sWorkloadSpec `json:"k8sWorkload,omitempty"`
}

// ConsumerStatus defines the observed state of Consumer.
type ConsumerStatus struct {
	// ObservedIndex is a map of secret names to SecretUpdateRecord.
	ObservedIndex map[string]SecretUpdateRecord `json:"observedIndex,omitempty"`
	// Locations is a list of SecretInStoreRef.
	Locations []SecretInStoreRef `json:"locations,omitempty"`
	// Conditions is a list of metav1.Condition.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// TargetReference defines a reference to a target.
type TargetReference struct {
	// Name is the name of the target.
	Name string `json:"name"`
	// Namespace is the namespace of the target.
	Namespace string `json:"namespace"`
}

// VMProcessSpec describes a process on a VM/host.
type VMProcessSpec struct {
	// RUID is the real user ID.
	RUID int64 `json:"ruid"`
	// EUID is the effective user ID.
	EUID int64 `json:"euid"`
	// Executable is the path to the executable.
	Executable string `json:"executable,omitempty"`
	// Cmdline is the command line arguments.
	Cmdline string `json:"cmdline,omitempty"`
}

// GitHubActorSpec describes who/what is interacting with a repo.
type GitHubActorSpec struct {
	// Repo slug "owner/name" for context (e.g., "acme/api").
	Repository string `json:"repository"`
	// ActorType: "User" | "App" | "Bot" (GitHub notions)
	// +kubebuilder:validation:Enum=User;App;Bot
	ActorType string `json:"actorType"`
	// ActorLogin is the login of the actor.
	ActorLogin string `json:"actorLogin,omitempty"` // "octocat"
	// ActorID is the stable numeric id of the actor if known.
	ActorID string `json:"actorID,omitempty"` // stable numeric id if known
	// Optional context that led to detection (push/clone/workflow).
	Event string `json:"event,omitempty"` // "clone","workflow","push"
	// Optional: workflow/job id when usage came from Actions
	WorkflowRunID string `json:"workflowRunID,omitempty"`
}

// K8sWorkloadSpec describes the workload that is interacting with a kubernetes target.
type K8sWorkloadSpec struct {
	// ClusterName is the name of the cluster.
	ClusterName string `json:"clusterName,omitempty"`

	// Namespace is the namespace of the workload.
	Namespace string `json:"namespace"`

	// Workload identity (top controller or naked Pod as fallback)
	// e.g., Kind="Deployment", Group="apps", Version="v1", Name="api"
	WorkloadKind string `json:"workloadKind"`
	// WorkloadGroup is the group of the workload.
	WorkloadGroup string `json:"workloadGroup,omitempty"`
	// WorkloadVersion is the version of the workload.
	WorkloadVersion string `json:"workloadVersion,omitempty"`
	// WorkloadName is the name of the workload.
	WorkloadName string `json:"workloadName"`
	// WorkloadUID is the UID of the workload.
	WorkloadUID string `json:"workloadUID,omitempty"`

	// Convenience string for UIs: "deployment.apps/api"
	Controller string `json:"controller,omitempty"`
}

// ConsumerFinding defines a finding from a consumer.
type ConsumerFinding struct {
	// ObservedIndex is a map of secret names to SecretUpdateRecord.
	ObservedIndex SecretUpdateRecord `json:"observedIndex"`
	// Location is a SecretInStoreRef.
	Location SecretInStoreRef `json:"location"`
	// Type is the type of the finding.
	Type string `json:"type"`
	// ID is the external ID of the finding.
	ID string `json:"externalID"`
	// DisplayName is the display name of the finding.
	DisplayName string `json:"displayName,omitempty"`
	// Attributes are the attributes of the finding.
	Attributes ConsumerAttrs `json:"attributes"`
}

// Consumer is the schema to store duplicate findings from a job
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-scan}
type Consumer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ConsumerSpec   `json:"spec,omitempty"`
	Status            ConsumerStatus `json:"status,omitempty"`
}

// ConsumerList contains a list of Consumer resources.
// +kubebuilder:object:root=true
type ConsumerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Consumer `json:"items"`
}
