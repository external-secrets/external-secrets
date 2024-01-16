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

package v1beta1

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
	ClusterLocation   string                        `json:"clusterLocation,omitempty"`
	ClusterName       string                        `json:"clusterName,omitempty"`
	ClusterProjectID  string                        `json:"clusterProjectID,omitempty"`

	// ClusterMembershipName is the name of the cluster membership if authenticating
	// using fleet identity. ClusterName and ClusterLocation must be left blank because
	// they take precedence.
	ClusterMembershipName string `json:"clusterMembershipName,omitempty"`
}

// GCPSMProvider Configures a store to sync secrets using the GCP Secret Manager provider.
type GCPSMProvider struct {
	// Auth defines the information necessary to authenticate against GCP
	// +optional
	Auth GCPSMAuth `json:"auth,omitempty"`

	// ProjectID project where secret is located
	ProjectID string `json:"projectID,omitempty"`
}
