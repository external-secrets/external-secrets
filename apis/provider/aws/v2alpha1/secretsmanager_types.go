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

package v2alpha1

import (
	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretsManagerSpec defines the desired state of SecretsManager.
type SecretsManagerSpec struct {
	// Auth defines the information necessary to authenticate against AWS
	// if not set aws sdk will infer credentials from your environment
	// see: https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
	// +optional
	Auth v1.AWSAuth `json:"auth,omitempty"`

	// Role is a Role ARN which the provider will assume
	// +optional
	Role string `json:"role,omitempty"`

	// AWS Region to be used for the provider
	Region string `json:"region"`

	// AdditionalRoles is a chained list of Role ARNs which the provider will sequentially assume before assuming the Role
	// +optional
	AdditionalRoles []string `json:"additionalRoles,omitempty"`

	// AWS External ID set on assumed IAM roles
	ExternalID string `json:"externalID,omitempty"`

	// AWS STS assume role session tags
	// +optional
	SessionTags []*v1.Tag `json:"sessionTags,omitempty"`

	// SecretsManager defines how the provider behaves when interacting with AWS SecretsManager
	// +optional
	SecretsManager *v1.SecretsManager `json:"secretsManager,omitempty"`

	// AWS STS assume role transitive session tags. Required when multiple rules are used with the provider
	// +optional
	TransitiveTagKeys []string `json:"transitiveTagKeys,omitempty"`

	// Prefix adds a prefix to all retrieved values.
	// +optional
	Prefix string `json:"prefix,omitempty"`
}

// SecretsManagerStatus defines the observed state of SecretsManager.
type SecretsManagerStatus struct {
	// Conditions represent the latest available observations of the resource's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={externalsecrets},shortName=sm
// +kubebuilder:printcolumn:name="Region",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SecretsManager is the Schema for AWS Secrets Manager provider configuration.
type SecretsManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretsManagerSpec   `json:"spec,omitempty"`
	Status SecretsManagerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretsManagerList contains a list of SecretsManager.
type SecretsManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretsManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretsManager{}, &SecretsManagerList{})
}
