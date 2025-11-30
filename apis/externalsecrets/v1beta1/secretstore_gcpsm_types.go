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

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// GCPSMAuth defines the authentication methods for the GCP Secret Manager provider.
type GCPSMAuth struct {
	// +optional
	SecretRef *GCPSMAuthSecretRef `json:"secretRef,omitempty"`
	// +optional
	WorkloadIdentity *GCPWorkloadIdentity `json:"workloadIdentity,omitempty"`
}

// GCPSMAuthSecretRef defines a reference to a secret containing credentials for the GCP Secret Manager provider.
type GCPSMAuthSecretRef struct {
	// The SecretAccessKey is used for authentication
	// +optional
	SecretAccessKey esmeta.SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`
}

// GCPWorkloadIdentity defines configuration for using GCP Workload Identity authentication.
type GCPWorkloadIdentity struct {
	// +kubebuilder:validation:Required
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef"`
	// ClusterLocation is the location of the cluster
	// If not specified, it fetches information from the metadata server
	// +optional
	ClusterLocation string `json:"clusterLocation,omitempty"`
	// ClusterName is the name of the cluster
	// If not specified, it fetches information from the metadata server
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// ClusterProjectID is the project ID of the cluster
	// If not specified, it fetches information from the metadata server
	// +optional
	ClusterProjectID string `json:"clusterProjectID,omitempty"`
}

// GCPSMProvider Configures a store to sync secrets using the GCP Secret Manager provider.
type GCPSMProvider struct {
	// Auth defines the information necessary to authenticate against GCP
	// +optional
	Auth GCPSMAuth `json:"auth,omitempty"`

	// ProjectID project where secret is located
	ProjectID string `json:"projectID,omitempty"`

	// Location optionally defines a location for a secret
	Location string `json:"location,omitempty"`
}
