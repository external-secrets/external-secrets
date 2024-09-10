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
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

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

// AWSServiceType is a enum that defines the service/API that is used to fetch the secrets.
// +kubebuilder:validation:Enum=SecretsManager;ParameterStore
type AWSServiceType string

const (
	// AWSServiceSecretsManager is the AWS SecretsManager service.
	// see: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html
	AWSServiceSecretsManager AWSServiceType = "SecretsManager"
	// AWSServiceParameterStore is the AWS SystemsManager ParameterStore service.
	// see: https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html
	AWSServiceParameterStore AWSServiceType = "ParameterStore"
)

// SecretsManager defines how the provider behaves when interacting with AWS
// SecretsManager. Some of these settings are only applicable to controlling how
// secrets are deleted, and hence only apply to PushSecret (and only when
// deletionPolicy is set to Delete).
type SecretsManager struct {
	// Specifies whether to delete the secret without any recovery window. You
	// can't use both this parameter and RecoveryWindowInDays in the same call.
	// If you don't use either, then by default Secrets Manager uses a 30 day
	// recovery window.
	// see: https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html#SecretsManager-DeleteSecret-request-ForceDeleteWithoutRecovery
	// +optional
	ForceDeleteWithoutRecovery bool `json:"forceDeleteWithoutRecovery,omitempty"`
	// The number of days from 7 to 30 that Secrets Manager waits before
	// permanently deleting the secret. You can't use both this parameter and
	// ForceDeleteWithoutRecovery in the same call. If you don't use either,
	// then by default Secrets Manager uses a 30 day recovery window.
	// see: https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html#SecretsManager-DeleteSecret-request-RecoveryWindowInDays
	// +optional
	RecoveryWindowInDays int64 `json:"recoveryWindowInDays,omitempty"`
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// AWSProvider configures a store to sync secrets with AWS.
type AWSSpec struct {
	// Used to select the correct ESO controller (think: ingress.ingressClassName)
	// The ESO controller is instantiated with a specific controller name and filters ES based on this property
	// +optional
	Controller string `json:"controller,omitempty"`

	// Used to configure http retries if failed
	// +optional
	RetrySettings *esmeta.RetrySettings `json:"retrySettings,omitempty"`

	// Used to configure store refresh interval in seconds. Empty or 0 will default to the controller config.
	// +optional
	RefreshInterval int `json:"refreshInterval,omitempty"`
	// Service defines which service should be used to fetch the secrets
	Service AWSServiceType `json:"service"`

	// Auth defines the information necessary to authenticate against AWS
	// if not set aws sdk will infer credentials from your environment
	// see: https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
	// +optional
	Auth AWSAuth `json:"auth,omitempty"`

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
	SessionTags []*Tag `json:"sessionTags,omitempty"`

	// SecretsManager defines how the provider behaves when interacting with AWS SecretsManager
	// +optional
	SecretsManager *SecretsManager `json:"secretsManager,omitempty"`

	// AWS STS assume role transitive session tags. Required when multiple rules are used with the provider
	// +optional
	TransitiveTagKeys []*string `json:"transitiveTagKeys,omitempty"`

	// Prefix adds a prefix to all retrieved values.
	// +optional
	Prefix string `json:"prefix,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={aws},shortName=aws
type AWS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AWSSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// AWSList contains a list of ExternalSecret resources.
type AWSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWS `json:"items"`
}

func init() {
}

// aws type metadata.
var (
	AWSKind             = reflect.TypeOf(AWS{}).Name()
	AWSGroupKind        = schema.GroupKind{Group: Group, Kind: AWSKind}.String()
	AWSKindAPIVersion   = AWSKind + "." + SchemeGroupVersion.String()
	AWSGroupVersionKind = SchemeGroupVersion.WithKind(AWSKind)
)
