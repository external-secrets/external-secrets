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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AssumeRoleRequestParameters contains parameters for the STS AssumeRole call.
type AssumeRoleRequestParameters struct {
	// SessionDuration The duration, in seconds, of the role session.
	// The value can range from 900 seconds (15 minutes) to the maximum session
	// duration setting for the role. If not specified, the default is 1 hour.
	// +optional
	SessionDuration *int32 `json:"sessionDuration,omitempty"`

	// ExternalID is a unique identifier that might be required when you assume a
	// role in another account. If the administrator of the account to which the
	// role belongs provided you with an external ID, then provide that value.
	// +optional
	ExternalID *string `json:"externalID,omitempty"`
}

// STSAssumeRoleTokenSpec defines the desired state to generate temporary AWS credentials
// via sts:AssumeRole. Unlike STSSessionToken, this generator works with both long-term
// credentials and temporary credentials (e.g. IRSA / pod identity).
type STSAssumeRoleTokenSpec struct {
	// Region specifies the AWS region to operate in.
	Region string `json:"region"`

	// Role is the ARN of the IAM role to assume.
	// +kubebuilder:validation:MinLength=1
	Role string `json:"role"`

	// Auth defines how to authenticate with AWS.
	// +optional
	Auth AWSAuth `json:"auth,omitempty"`

	// RequestParameters contains optional parameters for the AssumeRole call.
	// +optional
	RequestParameters *AssumeRoleRequestParameters `json:"requestParameters,omitempty"`
}

// STSAssumeRoleToken uses sts:AssumeRole to obtain temporary AWS credentials.
// Unlike STSSessionToken (which calls GetSessionToken), this generator works with
// both long-term IAM credentials and temporary credentials such as IRSA pod identity,
// making it suitable for any environment including on-premises clusters.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type STSAssumeRoleToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec STSAssumeRoleTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// STSAssumeRoleTokenList contains a list of STSAssumeRoleToken resources.
type STSAssumeRoleTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []STSAssumeRoleToken `json:"items"`
}
