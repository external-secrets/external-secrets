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

package v1

import smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// AuthType describes how to authenticate to the Azure Keyvault
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

// AzureEnvironmentType specifies the Azure cloud environment endpoints to use for
// connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
// The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152
// PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud, AzureStackCloud
// +kubebuilder:validation:Enum=PublicCloud;USGovernmentCloud;ChinaCloud;GermanCloud;AzureStackCloud
type AzureEnvironmentType string

const (
	AzureEnvironmentPublicCloud       AzureEnvironmentType = "PublicCloud"
	AzureEnvironmentUSGovernmentCloud AzureEnvironmentType = "USGovernmentCloud"
	AzureEnvironmentChinaCloud        AzureEnvironmentType = "ChinaCloud"
	AzureEnvironmentGermanCloud       AzureEnvironmentType = "GermanCloud"
	AzureEnvironmentAzureStackCloud   AzureEnvironmentType = "AzureStackCloud"
)

// AzureCustomCloudConfig specifies custom cloud configuration for private Azure environments
// IMPORTANT: Custom cloud configuration is ONLY supported when UseAzureSDK is true.
// The legacy go-autorest SDK does not support custom cloud endpoints.
type AzureCustomCloudConfig struct {
	// ActiveDirectoryEndpoint is the AAD endpoint for authentication
	// Required when using custom cloud configuration
	// +kubebuilder:validation:Required
	ActiveDirectoryEndpoint string `json:"activeDirectoryEndpoint"`

	// KeyVaultEndpoint is the Key Vault service endpoint
	// +optional
	KeyVaultEndpoint *string `json:"keyVaultEndpoint,omitempty"`

	// KeyVaultDNSSuffix is the DNS suffix for Key Vault URLs
	// +optional
	KeyVaultDNSSuffix *string `json:"keyVaultDNSSuffix,omitempty"`

	// ResourceManagerEndpoint is the Azure Resource Manager endpoint
	// +optional
	ResourceManagerEndpoint *string `json:"resourceManagerEndpoint,omitempty"`
}

// Configures an store to sync secrets using Azure KV.
type AzureKVProvider struct {
	// Auth type defines how to authenticate to the keyvault service.
	// Valid values are:
	// - "ServicePrincipal" (default): Using a service principal (tenantId, clientId, clientSecret)
	// - "ManagedIdentity": Using Managed Identity assigned to the pod (see aad-pod-identity)
	// +optional
	// +kubebuilder:default=ServicePrincipal
	AuthType *AzureAuthType `json:"authType,omitempty"`

	// Vault Url from which the secrets to be fetched from.
	VaultURL *string `json:"vaultUrl"`

	// TenantID configures the Azure Tenant to send requests to. Required for ServicePrincipal auth type. Optional for WorkloadIdentity.
	// +optional
	TenantID *string `json:"tenantId,omitempty"`

	// EnvironmentType specifies the Azure cloud environment endpoints to use for
	// connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
	// The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152
	// PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud, AzureStackCloud
	// Use AzureStackCloud when you need to configure custom Azure Stack Hub or Azure Stack Edge endpoints.
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

	// UseAzureSDK enables the use of the new Azure SDK for Go (azcore-based) instead of the legacy go-autorest SDK.
	// This is experimental and may have behavioral differences. Defaults to false (legacy SDK).
	// +optional
	// +kubebuilder:default=false
	UseAzureSDK *bool `json:"useAzureSDK,omitempty"`

	// CustomCloudConfig defines custom Azure Stack Hub or Azure Stack Edge endpoints.
	// Required when EnvironmentType is AzureStackCloud.
	// IMPORTANT: This feature REQUIRES UseAzureSDK to be set to true. Custom cloud
	// configuration is not supported with the legacy go-autorest SDK.
	// +optional
	CustomCloudConfig *AzureCustomCloudConfig `json:"customCloudConfig,omitempty"`
}

// Configuration used to authenticate with Azure.
type AzureKVAuth struct {
	// The Azure clientId of the service principle or managed identity used for authentication.
	// +optional
	ClientID *smmeta.SecretKeySelector `json:"clientId,omitempty"`

	// The Azure tenantId of the managed identity used for authentication.
	// +optional
	TenantID *smmeta.SecretKeySelector `json:"tenantId,omitempty"`

	// The Azure ClientSecret of the service principle used for authentication.
	// +optional
	ClientSecret *smmeta.SecretKeySelector `json:"clientSecret,omitempty"`

	// The Azure ClientCertificate of the service principle used for authentication.
	// +optional
	ClientCertificate *smmeta.SecretKeySelector `json:"clientCertificate,omitempty"`
}
