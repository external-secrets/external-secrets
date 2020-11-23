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

type AWSSMAuth struct {
	SecretRef AWSSMAuthSecretRef `json:"secretRef"`
}

type AWSSMAuthSecretRef struct {
	// The AccessKeyID is used for authentication
	// +optional
	AccessKeyID SecretKeySelector `json:"accessKeyIDSecretRef,omitempty"`

	// The SecretAccessKey is used for authentication
	// +optional
	SecretAccessKey SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`
}

// Configures a store to sync secrets using the AWS Secret Manager provider
type AWSSMProvider struct {
	// Auth defines the information necessary to authenticate agains AWS
	Auth AWSSMAuth `json:"auth"`

	// Role is a Role ARN which the SecretManager provider will assume
	// +optional
	Role string `json:"role,omitempty"`

	// AWS Region to be used for the provider
	Region string `json:"region"`
}
