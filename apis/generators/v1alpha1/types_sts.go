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
)

// RequestParameters contains parameters that can be passed to the STS service.
type RequestParameters struct {
	// SessionDuration The duration, in seconds, that the credentials should remain valid. Acceptable durations for
	// IAM user sessions range from 900 seconds (15 minutes) to 129,600 seconds (36 hours), with 43,200 seconds
	// (12 hours) as the default.
	// +optional
	SessionDuration *int64 `json:"sessionDuration,omitempty"`
	// SerialNumber is the identification number of the MFA device that is associated with the IAM user who is making
	// the GetSessionToken call.
	// Possible values: hardware device (such as GAHT12345678) or an Amazon Resource Name (ARN) for a virtual device
	// (such as arn:aws:iam::123456789012:mfa/user)
	// +optional
	SerialNumber *string `json:"serialNumber,omitempty"`
	// TokenCode is the value provided by the MFA device, if MFA is required.
	// +optional
	TokenCode *string `json:"tokenCode,omitempty"`
}

type STSSessionTokenSpec struct {
	// Region specifies the region to operate in.
	Region string `json:"region"`

	// Auth defines how to authenticate with AWS
	// +optional
	Auth AWSAuth `json:"auth,omitempty"`

	// You can assume a role before making calls to the
	// desired AWS service.
	// +optional
	Role string `json:"role,omitempty"`

	// RequestParameters contains parameters that can be passed to the STS service.
	// +optional
	RequestParameters *RequestParameters `json:"requestParameters,omitempty"`
}

// STSSessionToken uses the GetSessionToken API to retrieve an authorization token.
// The authorization token is valid for 12 hours.
// The authorizationToken returned is a base64 encoded string that can be decoded.
// For more information, see GetSessionToken (https://docs.aws.amazon.com/STS/latest/APIReference/API_GetSessionToken.html).
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type STSSessionToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec STSSessionTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// STSSessionTokenList contains a list of STSSessionToken resources.
type STSSessionTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []STSSessionToken `json:"items"`
}
