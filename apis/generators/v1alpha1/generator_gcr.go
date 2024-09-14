//Copyright External Secrets Inc. All Rights Reserved

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
