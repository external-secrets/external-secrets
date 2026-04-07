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

	// DomainOwner is the 12-digit AWS account ID that owns the CodeArtifact domain.
	// +kubebuilder:validation:Pattern=`^[0-9]{12}$`
	DomainOwner string `json:"domainOwner"`
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
