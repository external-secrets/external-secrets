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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type GCRAccessTokenSpec struct {
	// Auth defines the means for authenticating with GCP
	Auth GCPSMAuth `json:"auth"`
	// ProjectID defines which project to use to authenticate with
	ProjectID string `json:"projectID"`
}

type GCPSMAuth struct {
	// +optional
	SecretRef *GCPSMAuthSecretRef `json:"secretRef,omitempty"`
	// +optional
	WorkloadIdentity *GCPWorkloadIdentity `json:"workloadIdentity,omitempty"`
}

type GCPSMAuthSecretRef struct {
	// The SecretAccessKey is used for authentication
	// +optional
	SecretAccessKey esmeta.SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`
}

type GCPWorkloadIdentity struct {
	ServiceAccountRef     esmeta.ServiceAccountSelector `json:"serviceAccountRef"`
	ClusterLocation       string                        `json:"clusterLocation"`
	ClusterName           string                        `json:"clusterName"`
	ClusterProjectID      string                        `json:"clusterProjectID,omitempty"`
	ClusterMembershipName string                        `json:"clusterMembershipName,omitempty"`
}

// GCRAccessToken generates an GCP access token
// that can be used to authenticate with GCR.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={gcraccesstoken},shortName=gcraccesstoken
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
