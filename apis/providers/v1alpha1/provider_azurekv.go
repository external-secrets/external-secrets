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

	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// AuthType describes how to authenticate to the AzureKvKeyvault
// Only one of the following auth types may be specified.
// If none of the following auth type is specified, the default one
// is ServicePrincipal.
// +kubebuilder:validation:Enum=ServicePrincipal;ManagedIdentity;WorkloadIdentity
type AzureAuthType string

const (
	// Using service principal to authenticate, which needs a tenantId, a clientId and a clientSecret.
	AzureServicePrincipal AzureAuthType = "ServicePrincipal"

	// Using Managed Identity to authenticate. Used with aad-pod-identity installed in the cluster.
	AzureManagedIdentity AzureAuthType = "ManagedIdentity"

	// Using Workload Identity service accounts to authenticate.
	AzureWorkloadIdentity AzureAuthType = "WorkloadIdentity"
)

// AzureEnvironmentType specifies the AzureKvcloud environment endpoints to use for
// connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
// The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152
// PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud
// +kubebuilder:validation:Enum=PublicCloud;USGovernmentCloud;ChinaCloud;GermanCloud
type AzureEnvironmentType string

const (
	AzureEnvironmentPublicCloud       AzureEnvironmentType = "PublicCloud"
	AzureEnvironmentUSGovernmentCloud AzureEnvironmentType = "USGovernmentCloud"
	AzureEnvironmentChinaCloud        AzureEnvironmentType = "ChinaCloud"
	AzureEnvironmentGermanCloud       AzureEnvironmentType = "GermanCloud"
)

// Configures an store to sync secrets using AzureKvKV.
type AzureKVSpec struct {
	// Used to select the correct ESO controller (think: ingress.ingressClassName)
	// The ESO controller is instantiated with a specific controller name and filters ES based on this property
	// +optional
	Controller string `json:"controller,omitempty"`

	// Used to configure http retries if failed
	// +optional
	RetrySettings *smmeta.RetrySettings `json:"retrySettings,omitempty"`

	// Used to configure store refresh interval in seconds. Empty or 0 will default to the controller config.
	// +optional
	RefreshInterval int `json:"refreshInterval,omitempty"`
	// Auth type defines how to authenticate to the keyvault service.
	// Valid values are:
	// - "ServicePrincipal" (default): Using a service principal (tenantId, clientId, clientSecret)
	// - "ManagedIdentity": Using Managed Identity assigned to the pod (see aad-pod-identity)
	// +optional
	// +kubebuilder:default=ServicePrincipal
	AuthType *AzureAuthType `json:"authType,omitempty"`

	// Vault Url from which the secrets to be fetched from.
	VaultURL *string `json:"vaultUrl"`

	// TenantID configures the AzureKvTenant to send requests to. Required for ServicePrincipal auth type. Optional for WorkloadIdentity.
	// +optional
	TenantID *string `json:"tenantId,omitempty"`

	// EnvironmentType specifies the AzureKvcloud environment endpoints to use for
	// connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
	// The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152
	// PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud
	// +kubebuilder:default=PublicCloud
	EnvironmentType AzureEnvironmentType `json:"environmentType,omitempty"`

	// Auth configures how the operator authenticates with Azure. Required for ServicePrincipal auth type. Optional for WorkloadIdentity.
	// +optional
	AuthSecretRef *AzureKVAuth `json:"authSecretRef,omitempty"`

	// ServiceAccountRef specified the service account
	// that should be used when authenticating with WorkloadIdentity.
	// +optional
	ServiceAccountRef *smmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`

	// If multiple Managed Identity is assigned to the pod, you can select the one to be used
	// +optional
	IdentityID *string `json:"identityId,omitempty"`
}

// Configuration used to authenticate with Azure.
type AzureKVAuth struct {
	// The AzureKvclientId of the service principle or managed identity used for authentication.
	// +optional
	ClientID *smmeta.SecretKeySelector `json:"clientId,omitempty"`

	// The AzureKvtenantId of the managed identity used for authentication.
	// +optional
	TenantID *smmeta.SecretKeySelector `json:"tenantId,omitempty"`

	// The AzureKvClientSecret of the service principle used for authentication.
	// +optional
	ClientSecret *smmeta.SecretKeySelector `json:"clientSecret,omitempty"`

	// The AzureKvClientCertificate of the service principle used for authentication.
	// +optional
	ClientCertificate *smmeta.SecretKeySelector `json:"clientCertificate,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={azure},shortName=azure
type AzureKv struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AzureKVSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// AzureList contains a list of ExternalSecret resources.
type AzureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureKv `json:"items"`
}

func init() {
}

// AzureKv type metadata.
var (
	AzureKind             = reflect.TypeOf(AzureKv{}).Name()
	AzureGroupKind        = schema.GroupKind{Group: Group, Kind: AzureKind}.String()
	AzureKindAPIVersion   = AzureKind + "." + SchemeGroupVersion.String()
	AzureGroupVersionKind = SchemeGroupVersion.WithKind(AzureKind)
)
