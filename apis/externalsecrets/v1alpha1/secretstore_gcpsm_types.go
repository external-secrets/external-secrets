//Copyright External Secrets Inc. All Rights Reserved

package v1alpha1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

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

// GCPSMProvider Configures a store to sync secrets using the GCP Secret Manager provider.
type GCPSMProvider struct {
	// Auth defines the information necessary to authenticate against GCP
	// +optional
	Auth GCPSMAuth `json:"auth,omitempty"`

	// ProjectID project where secret is located
	ProjectID string `json:"projectID,omitempty"`
}
