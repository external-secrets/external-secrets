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

type ECRAuthorizationTokenSpec struct {
	// Region specifies the region to operate in.
	Region string `json:"region"`

	// Auth defines how to authenticate with AWS
	// +optional
	Auth AWSAuth `json:"auth,omitempty"`

	// You can assume a role before making calls to the
	// desired AWS service.
	// +optional
	Role string `json:"role,omitempty"`
}

// AWSAuth tells the controller how to do authentication with aws.
// Only one of secretRef or jwt can be specified.
// if none is specified the controller will load credentials using the aws sdk defaults.
type AWSAuth struct {
	// +optional
	SecretRef *AWSAuthSecretRef `json:"secretRef,omitempty"`
	// +optional
	JWTAuth *AWSJWTAuth `json:"jwt,omitempty"`
}

// AWSAuthSecretRef holds secret references for AWS credentials
// both AccessKeyID and SecretAccessKey must be defined in order to properly authenticate.
type AWSAuthSecretRef struct {
	// The AccessKeyID is used for authentication
	AccessKeyID esmeta.SecretKeySelector `json:"accessKeyIDSecretRef,omitempty"`

	// The SecretAccessKey is used for authentication
	SecretAccessKey esmeta.SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`

	// The SessionToken used for authentication
	// This must be defined if AccessKeyID and SecretAccessKey are temporary credentials
	// see: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_use-resources.html
	// +Optional
	SessionToken *esmeta.SecretKeySelector `json:"sessionTokenSecretRef,omitempty"`
}

// Authenticate against AWS using service account tokens.
type AWSJWTAuth struct {
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

// ECRAuthorizationTokenSpec uses the GetAuthorizationToken API to retrieve an
// authorization token.
// The authorization token is valid for 12 hours.
// The authorizationToken returned is a base64 encoded string that can be decoded
// and used in a docker login command to authenticate to a registry.
// For more information, see Registry authentication (https://docs.aws.amazon.com/AmazonECR/latest/userguide/Registries.html#registry_auth) in the Amazon Elastic Container Registry User Guide.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={ecrauthorizationtoken},shortName=ecrauthorizationtoken
type ECRAuthorizationToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ECRAuthorizationTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ECRAuthorizationTokenList contains a list of ExternalSecret resources.
type ECRAuthorizationTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ECRAuthorizationToken `json:"items"`
}
