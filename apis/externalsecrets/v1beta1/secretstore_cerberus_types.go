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

// CerberusProvider configures a store to sync secrets with Cerberus.
type CerberusProvider struct {
	// Auth defines the information necessary to authenticate against AWS
	// if not set aws sdk will infer credentials from your environment
	// see: https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
	// +optional
	Auth AWSAuth `json:"auth,omitempty"`

	// Role is a Role ARN which the provider will assume
	// +optional
	Role string `json:"role,omitempty"`

	// AdditionalRoles is a chained list of Role ARNs which the SecretManager provider will sequentially assume before assuming Role
	// +optional
	AdditionalRoles []string `json:"additionalRoles,omitempty"`

	// AWS STS assume role session tags
	// +optional
	SessionTags []*Tag `json:"sessionTags,omitempty"`

	// AWS STS assume role transitive session tags. Required when multiple rules are used with SecretStore
	// +optional
	TransitiveTagKeys []*string `json:"transitiveTagKeys,omitempty"`

	// AWS External ID set on assumed IAM roles
	ExternalID string `json:"externalID,omitempty"`

	// Region to be used for the provider
	Region string `json:"region"`

	// URL of Cerberus crypto-vault
	CerberusURL string `json:"cerberusURL"`

	// Name of the Safe Deposit Box
	SDB string `json:"sdb"`
}
