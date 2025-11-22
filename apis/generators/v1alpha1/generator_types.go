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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ControllerClassResource defines a resource that can be assigned to a specific controller class.
type ControllerClassResource struct {
	Spec struct {
		ControllerClass string `json:"controller"`
	} `json:"spec"`
}

const (
	// IdleCleanupPolicy indicates that secrets should be cleaned up when idle.
	IdleCleanupPolicy = "idle"
	// RetainLatestPolicy indicates that only the latest secret should be retained.
	RetainLatestPolicy = "retainLatest"
)

// CleanupPolicy defines the cleanup policy for generated secrets.
type CleanupPolicy struct {
	// Type of the cleanup policy. Supported values: "idle", "retainLatest".
	// idle: delete the secret if it has not been used for a while
	// retainLatest: delete older secrets when a new one is created
	// +kubebuilder:validation:Enum=idle;retainLatest
	// +kubebuilder:default=retainLatest
	Type string `json:"type"`

	// IdleTimeout Indicates how long without activity a secret is considered inactive and can be removed.
	// Used only when type is "idle".
	// +optional
	// +kubebuilder:validation:Format=duration
	// +kubebuilder:default="24h"
	IdleTimeout metav1.Duration `json:"idleTimeout,omitempty"`

	// GracePeriod is the amount of time to wait before deleting a secret.
	// +optional
	// +kubebuilder:validation:Format=duration
	// +kubebuilder:default="2m"
	GracePeriod metav1.Duration `json:"gracePeriod,omitempty"`
}
