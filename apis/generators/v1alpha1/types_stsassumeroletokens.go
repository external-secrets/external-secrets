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

// RoleAssumptionParameters contains optional parameters for the STS AssumeRole call.
type RoleAssumptionParameters struct {
	// SessionDuration is the duration, in seconds, of the role session.
	// Acceptable durations range from 900 seconds (15 minutes) to the maximum
	// session duration configured for the role (default: 3600 seconds / 1 hour).
	// +optional
	SessionDuration *int32 `json:"sessionDuration,omitempty"`

	// ExternalID is a unique identifier required by some roles for cross-account
	// access. It is used to protect against the confused deputy problem.
	// +optional
	ExternalID *string `json:"externalId,omitempty"`

	// RoleSessionName is an identifier for the assumed role session.
	// +optional
	RoleSessionName *string `json:"roleSessionName,omitempty"`
}

// STSAssumeRoleTokenSpec defines the desired state to generate temporary AWS
// credentials via AssumeRole (or AssumeRoleWithWebIdentity for IRSA).
type STSAssumeRoleTokenSpec struct {
	// Region specifies the AWS region to operate in.
	Region string `json:"region"`

	// Auth defines how to authenticate with AWS.
	// Supports both SecretRef (static IAM credentials) and JWTAuth (IRSA / service-account tokens).
	// +optional
	Auth AWSAuth `json:"auth,omitempty"`

	// Role is the ARN of the IAM role to assume.
	// When set, AssumeRole is called and the resulting temporary credentials are
	// returned. This is the primary use-case for IRSA: the service-account token
	// obtains short-term credentials via AssumeRoleWithWebIdentity, and those
	// credentials are then used to assume this role.
	// When omitted, the credentials from the configured auth source are returned
	// directly (e.g. the IRSA web-identity credentials themselves).
	// +optional
	Role string `json:"role,omitempty"`

	// RoleAssumptionParameters contains optional parameters passed to the
	// AssumeRole API call. Only used when Role is set.
	// +optional
	RoleAssumptionParameters *RoleAssumptionParameters `json:"roleAssumptionParameters,omitempty"`
}

// STSAssumeRoleToken returns temporary AWS credentials obtained via the STS
// AssumeRole (or AssumeRoleWithWebIdentity) API.
//
// Unlike STSSessionToken, this generator does NOT call GetSessionToken.
// GetSessionToken requires long-term IAM user credentials and cannot be called
// with temporary credentials (e.g. from IRSA). Use this generator instead when
// your auth source already provides temporary credentials (service-account /
// IRSA tokens) or when you need to assume a role.
//
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
