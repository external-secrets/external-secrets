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
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// AWSSMAuth contains a secretRef for credentials.
type AWSSMAuth struct {
	SecretRef AWSSMAuthSecretRef `json:"secretRef"`
}

// AWSSMAuthSecretRef holds secret references for aws credentials
// both AccessKeyID and SecretAccessKey must be defined in order to properly authenticate.
type AWSSMAuthSecretRef struct {
	// The AccessKeyID is used for authentication
	AccessKeyID esmeta.SecretKeySelector `json:"accessKeyIDSecretRef,omitempty"`

	// The SecretAccessKey is used for authentication
	SecretAccessKey esmeta.SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`
}

// AWSSMProvider configures a store to sync secrets using the AWS Secret Manager provider.
type AWSSMProvider struct {
	// Auth defines the information necessary to authenticate against AWS
	// if not set aws sdk will infer credentials from your environment
	// see: https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
	// +nullable
	// +optional
	Auth *AWSSMAuth `json:"auth"`

	// Role is a Role ARN which the SecretManager provider will assume
	// +optional
	Role string `json:"role,omitempty"`

	// AWS Region to be used for the provider
	Region string `json:"region"`
}
