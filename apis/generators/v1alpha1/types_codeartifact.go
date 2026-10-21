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

// CodeArtifactAuthorizationTokenSpec defines the desired state to generate an AWS CodeArtifact authorization token.
type CodeArtifactAuthorizationTokenSpec struct {
	// Region specifies the region to operate in.
	// +kubebuilder:validation:MinLength=1
	Region string `json:"region"`

	// Auth defines how to authenticate with AWS
	// +optional
	Auth AWSAuth `json:"auth,omitempty"`

	// You can assume a role before making calls to the
	// desired AWS service.
	// +optional
	Role string `json:"role,omitempty"`

	// Domain is the name of the CodeArtifact domain.
	// +kubebuilder:validation:MinLength=1
	Domain string `json:"domain"`

	// DomainOwner is the AWS account ID that owns the CodeArtifact domain.
	// +kubebuilder:validation:MinLength=1
	DomainOwner string `json:"domainOwner"`

	// DurationSeconds is the time, in seconds, that the generated authorization token is valid.
	// Valid values are 0 and any number between 900 (15 minutes) and 43200 (12 hours).
	// A value of 0 sets the expiration to match the expiration of the caller's temporary credentials.
	// When omitted, AWS applies its default of 43200 (12 hours).
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == 0 || (self >= 900 && self <= 43200)",message="durationSeconds must be 0 or between 900 and 43200"
	DurationSeconds *int64 `json:"durationSeconds,omitempty"`
}

// CodeArtifactAuthorizationToken uses the GetAuthorizationToken API to retrieve an
// authorization token for AWS CodeArtifact.
// The authorization token is a temporary bearer token that can be used to authenticate
// package manager clients (pip, npm, maven, gradle, etc.) against a CodeArtifact repository.
// For more information, see:
// https://docs.aws.amazon.com/codeartifact/latest/ug/tokens-authentication.html
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type CodeArtifactAuthorizationToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CodeArtifactAuthorizationTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// CodeArtifactAuthorizationTokenList contains a list of CodeArtifactAuthorizationToken resources.
type CodeArtifactAuthorizationTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CodeArtifactAuthorizationToken `json:"items"`
}
