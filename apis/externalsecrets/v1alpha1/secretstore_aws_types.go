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
}

// Authenticate against AWS using service account tokens.
type AWSJWTAuth struct {
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

// AWSServiceType is a enum that defines the service/API that is used to fetch the secrets.
// +kubebuilder:validation:Enum=SecretsManager;ParameterStore
type AWSServiceType string

const (
	// AWSServiceSecretsManager is the AWS SecretsManager.
	// see: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html
	AWSServiceSecretsManager AWSServiceType = "SecretsManager"
	// AWSServiceParameterStore is the AWS SystemsManager ParameterStore.
	// see: https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html
	AWSServiceParameterStore AWSServiceType = "ParameterStore"
)

// AWSProvider configures a store to sync secrets with AWS.
type AWSProvider struct {
	// Service defines which service should be used to fetch the secrets
	Service AWSServiceType `json:"service"`

	// Auth defines the information necessary to authenticate against AWS
	// if not set aws sdk will infer credentials from your environment
	// see: https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
	// +optional
	Auth AWSAuth `json:"auth"`

	// Role is a Role ARN which the SecretManager provider will assume
	// +optional
	Role string `json:"role,omitempty"`

	// AWS Region to be used for the provider
	Region string `json:"region"`
}
