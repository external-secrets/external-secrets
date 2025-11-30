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

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// GCRAccessTokenSpec defines the desired state to generate a Google Container Registry access token.
type GCRAccessTokenSpec struct {
	// Auth defines the means for authenticating with GCP
	Auth GCPSMAuth `json:"auth"`
	// ProjectID defines which project to use to authenticate with
	ProjectID string `json:"projectID"`
}

// GCPSMAuth defines the authentication methods for Google Cloud Platform.
type GCPSMAuth struct {
	// +optional
	SecretRef *GCPSMAuthSecretRef `json:"secretRef,omitempty"`
	// +optional
	WorkloadIdentity *GCPWorkloadIdentity `json:"workloadIdentity,omitempty"`
	// +optional
	WorkloadIdentityFederation *esv1.GCPWorkloadIdentityFederation `json:"workloadIdentityFederation,omitempty"`
}

// GCPSMAuthSecretRef defines the reference to a secret containing Google Cloud Platform credentials.
type GCPSMAuthSecretRef struct {
	// The SecretAccessKey is used for authentication
	// +optional
	SecretAccessKey esmeta.SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`
}

// GCPWorkloadIdentity defines the configuration for using GCP Workload Identity authentication.
type GCPWorkloadIdentity struct {
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef"`
	ClusterLocation   string                        `json:"clusterLocation"`
	ClusterName       string                        `json:"clusterName"`
	ClusterProjectID  string                        `json:"clusterProjectID,omitempty"`
}

// GCRAccessToken generates an GCP access token
// that can be used to authenticate with GCR.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type GCRAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GCRAccessTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// GCRAccessTokenList contains a list of ExternalSecret resources.
type GCRAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GCRAccessToken `json:"items"`
}
