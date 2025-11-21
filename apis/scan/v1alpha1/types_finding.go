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

// FindingSpec defines the desired state of Finding.
type FindingSpec struct {
	// ID is the external ID of the finding.
	ID string `json:"id"`
	// DisplayName is the display name of the finding.
	DisplayName string `json:"displayName,omitempty"`
	// Hash is the hash of the finding (salted).
	Hash string `json:"hash,omitempty"`
	// RunTemplateRef is a reference to the run template.
	RunTemplateRef *RunTemplateReference `json:"runTemplateRef,omitempty"`
}

// RunTemplateReference defines a reference to a run template.
type RunTemplateReference struct {
	// Name is the name of the run template.
	Name string `json:"name"`
}

// FindingStatus defines the observed state of Finding.
type FindingStatus struct {
	// Locations is a list of SecretInStoreRef.
	Locations []SecretInStoreRef `json:"locations,omitempty"`
}

// Finding is the schema to store duplicate findings from a job
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-scan}
type Finding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FindingSpec   `json:"spec,omitempty"`
	Status            FindingStatus `json:"status,omitempty"`
}

// FindingList contains a list of Job resources.
// +kubebuilder:object:root=true
type FindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Finding `json:"items"`
}
