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
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ECRAuthorizationTokenSpec uses the GetAuthorizationToken API to retrieve an
// authorization token.
// The authorization token is valid for 12 hours.
// The authorizationToken returned is a base64 encoded string that can be decoded
// and used in a docker login command to authenticate to a registry.
// For more information, see Registry authentication (https://docs.aws.amazon.com/AmazonECR/latest/userguide/Registries.html#registry_auth) in the Amazon Elastic Container Registry User Guide.
type ECRAuthorizationTokenSpec struct {
	// +optional
	Auth esv1beta1.AWSAuth `json:"auth"`
	// +optional
	Role   string `json:"role"`
	Region string `json:"region"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
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
