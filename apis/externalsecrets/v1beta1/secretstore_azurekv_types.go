/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// AzureAuthType describes how to authenticate to the Azure Keyvault.
// Only one of the following auth types may be specified.
// If none of the following auth type is specified, the default one
// is ServicePrincipal.
// +kubebuilder:validation:Enum=ServicePrincipal;ManagedIdentity;WorkloadIdentity
type AzureAuthType string

const (
	// AzureServicePrincipal uses service principal to authenticate, which needs a tenantId, a clientId and a clientSecret.
	AzureServicePrincipal AzureAuthType = "ServicePrincipal"

	// AzureManagedIdentity uses Managed Identity to authenticate. Used with aad-pod-identity installed in the cluster.
	AzureManagedIdentity AzureAuthType = "ManagedIdentity"

	// AzureWorkloadIdentity uses Workload Identity service accounts to authenticate.
	AzureWorkloadIdentity AzureAuthType = "WorkloadIdentity"
)

// AzureEnvironmentType specifies the Azure cloud environment endpoints to use for
// connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
// The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152
// PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud
// +kubebuilder:validation:Enum=PublicCloud;USGovernmentCloud;ChinaCloud;GermanCloud
type AzureEnvironmentType string

const (
	// AzureEnvironmentPublicCloud represents the Azure public cloud environment.
	AzureEnvironmentPublicCloud AzureEnvironmentType = "PublicCloud"
	// AzureEnvironmentUSGovernmentCloud represents the Azure US government cloud environment.
	AzureEnvironmentUSGovernmentCloud AzureEnvironmentType = "USGovernmentCloud"
	// AzureEnvironmentChinaCloud represents the Azure China cloud environment.
	AzureEnvironmentChinaCloud AzureEnvironmentType = "ChinaCloud"
	// AzureEnvironmentGermanCloud represents the Azure German cloud environment.
	AzureEnvironmentGermanCloud AzureEnvironmentType = "GermanCloud"
)

// AzureKVProvider configures a store to sync secrets using Azure Key Vault.
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

// AzureKVAuth defines configuration for authentication with Azure Key Vault.
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
