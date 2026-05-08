/*
Copyright © The ESO Authors

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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// RDSIAMAuthTokenSpec defines the desired state to generate an AWS RDS IAM authentication token.
type RDSIAMAuthTokenSpec struct {
	// Used to select the correct ESO controller (think: ingress.ingressClassName).
	// +optional
	Controller string `json:"controller,omitempty"`

	// Region specifies the AWS region where the database is hosted.
	Region string `json:"region"`

	// Hostname is the RDS endpoint hostname.
	Hostname string `json:"hostname"`

	// Port is the RDS endpoint port.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int `json:"port"`

	// Username is the database user to authenticate as.
	Username string `json:"username"`

	// Auth defines how to authenticate with AWS.
	// +optional
	Auth AWSAuth `json:"auth,omitempty"`

	// You can assume a role before building the RDS IAM auth token.
	// +optional
	Role string `json:"role,omitempty"`
}

// RDSIAMAuthToken uses AWS credentials to build an IAM authentication token for RDS.
// The token can be used as the database password for IAM database authentication.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type RDSIAMAuthToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RDSIAMAuthTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// RDSIAMAuthTokenList contains a list of RDSIAMAuthToken resources.
type RDSIAMAuthTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RDSIAMAuthToken `json:"items"`
}
