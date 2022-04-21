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

	// Using Managed Identity to authenticate. Used with aad-pod-identity installed in the clister.
	AzureManagedIdentity AzureAuthType = "ManagedIdentity"

	// Using Workload Identity service accounts to authenticate.
	AzureWorkloadIdentity AzureAuthType = "WorkloadIdentity"
)

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

	// TenantID configures the Azure Tenant to send requests to. Required for ServicePrincipal auth type.
	// +optional
	TenantID *string `json:"tenantId,omitempty"`

	// Auth configures how the operator authenticates with Azure. Required for ServicePrincipal auth type.
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
	// The Azure clientId of the service principle used for authentication.
	// +optional
	ClientID *smmeta.SecretKeySelector `json:"clientId,omitempty"`

	// The Azure ClientSecret of the service principle used for authentication.
	// +optional
	ClientSecret *smmeta.SecretKeySelector `json:"clientSecret,omitempty"`
}
