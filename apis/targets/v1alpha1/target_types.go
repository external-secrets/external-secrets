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

// Package v1alpha1 contains API Schema definitions for the targets v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
)

// TargetConditionType defines the possible conditions a Target can have.
type TargetConditionType string

const (
	// TargetReady defines a Target that is ready to be used.
	TargetReady TargetConditionType = "Ready"
)

// TargetStatusCondition defines the status of a Target.
type TargetStatusCondition struct {
	Type   TargetConditionType    `json:"type"`
	Status corev1.ConditionStatus `json:"status"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// TargetCapabilities defines the possible operations a Target can do.
type TargetCapabilities string

const (
	// TargetReadOnly defines a Target that can only read secrets.
	TargetReadOnly TargetCapabilities = "ReadOnly"
	// TargetWriteOnly defines a Target that can only write secrets.
	TargetWriteOnly TargetCapabilities = "WriteOnly"
	// TargetReadWrite defines a Target that can read and write secrets.
	TargetReadWrite TargetCapabilities = "ReadWrite"
)

// TargetStatus defines the observed state of the Target.
type TargetStatus struct {
	// +optional
	Conditions []TargetStatusCondition `json:"conditions,omitempty"`
	// +optional
	Capabilities TargetCapabilities `json:"capabilities,omitempty"`
	// +optional
	PushIndex map[string][]scanv1alpha1.SecretUpdateRecord `json:"pushIndex,omitempty"`
}

// SecretStoreToTargetStatus converts a SecretStoreStatus into a TargetStatus.
func SecretStoreToTargetStatus(in *esv1.SecretStoreStatus) *TargetStatus {
	if in == nil {
		return &TargetStatus{}
	}
	return &TargetStatus{
		Conditions:   convertConditionsSecretStoreToTarget(in.Conditions),
		Capabilities: convertCapabilitiesSecretStoreToTarget(in.Capabilities),
		// No source for this on SecretStore:
		PushIndex: nil,
	}
}

// TargetToSecretStoreStatus converts a TargetStatus into a SecretStoreStatus.
func TargetToSecretStoreStatus(in *TargetStatus) *esv1.SecretStoreStatus {
	if in == nil {
		return &esv1.SecretStoreStatus{}
	}
	return &esv1.SecretStoreStatus{
		Conditions:   convertConditionsTargetToSecretStore(in.Conditions),
		Capabilities: convertCapabilitiesTargetToSecretStore(in.Capabilities),
	}
}

func convertConditionsSecretStoreToTarget(in []esv1.SecretStoreStatusCondition) []TargetStatusCondition {
	if len(in) == 0 {
		return nil
	}
	out := make([]TargetStatusCondition, 0, len(in))
	for _, c := range in {
		out = append(out, TargetStatusCondition{
			Type:               targetConditionTypeFromString(string(c.Type)),
			Status:             c.Status,
			Reason:             c.Reason,
			Message:            c.Message,
			LastTransitionTime: metav1.Time{Time: c.LastTransitionTime.Time},
		})
	}
	return out
}

func convertConditionsTargetToSecretStore(in []TargetStatusCondition) []esv1.SecretStoreStatusCondition {
	if len(in) == 0 {
		return nil
	}
	out := make([]esv1.SecretStoreStatusCondition, 0, len(in))
	for _, c := range in {
		out = append(out, esv1.SecretStoreStatusCondition{
			Type:               secretStoreConditionTypeFromString(string(c.Type)),
			Status:             c.Status,
			Reason:             c.Reason,
			Message:            c.Message,
			LastTransitionTime: metav1.Time{Time: c.LastTransitionTime.Time},
		})
	}
	return out
}

func convertCapabilitiesSecretStoreToTarget(in esv1.SecretStoreCapabilities) TargetCapabilities {
	switch string(in) {
	case "ReadOnly":
		return TargetReadOnly
	case "WriteOnly":
		return TargetWriteOnly
	case "ReadWrite":
		return TargetReadWrite
	default:
		return ""
	}
}

func convertCapabilitiesTargetToSecretStore(in TargetCapabilities) esv1.SecretStoreCapabilities {
	switch string(in) {
	case "ReadOnly":
		return esv1.SecretStoreReadOnly
	case "WriteOnly":
		return esv1.SecretStoreWriteOnly
	case "ReadWrite":
		return esv1.SecretStoreReadWrite
	default:
		return ""
	}
}

// Map condition type strings defensively (identical names today; future-proof if they diverge).
func targetConditionTypeFromString(s string) TargetConditionType {
	switch s {
	case "Ready":
		return TargetReady
	default:
		return TargetConditionType(s)
	}
}

func secretStoreConditionTypeFromString(s string) esv1.SecretStoreConditionType {
	switch s {
	case "Ready":
		return esv1.SecretStoreReady
	default:
		return esv1.SecretStoreConditionType(s)
	}
}
