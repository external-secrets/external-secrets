/*
Copyright © The ESO Authors

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// AzureAccessTokenSpec defines how to generate a Microsoft Entra ID access token for a
// given Entra resource.
//
// All supported authentication methods are app-only (client-credentials) flows, so the
// token is always requested with the "<resource>/.default" scope. The resource selects
// what the token grants access to -- for example the Azure DevOps application id
// (499b84ac-1321-427f-aa17-267ca6975798), Azure Resource Manager
// (https://management.azure.com), or any other Entra resource id/URI.
//
// The Azure DevOps use-case is tracked in
// https://github.com/external-secrets/external-secrets/issues/5113
//
// +kubebuilder:validation:XValidation:rule="!has(self.auth.servicePrincipal) || (has(self.tenantId) && size(self.tenantId) > 0)",message="tenantId is required when servicePrincipal auth is used"
type AzureAccessTokenSpec struct {
	// Auth configures how ESO authenticates with Microsoft Entra ID.
	Auth AzureAuth `json:"auth"`

	// Resource is the Microsoft Entra resource id ("audience") that the issued access
	// token targets. The token is requested with the "<resource>/.default" scope.
	// Examples: "499b84ac-1321-427f-aa17-267ca6975798" (Azure DevOps),
	// "https://management.azure.com" (Azure Resource Manager),
	// "https://storage.azure.com" (Azure Storage).
	// +kubebuilder:validation:MinLength=1
	Resource string `json:"resource"`

	// TenantID configures the Azure Tenant to send requests to. Required for the
	// ServicePrincipal auth type.
	// +optional
	TenantID string `json:"tenantId,omitempty"`

	// EnvironmentType specifies the Azure cloud environment endpoints to use for
	// connecting and authenticating with Azure. By default, it points to the public cloud AAD endpoint.
	// The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152
	// PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud
	// +kubebuilder:default=PublicCloud
	// +optional
	EnvironmentType esv1.AzureEnvironmentType `json:"environmentType,omitempty"`
}

// AzureAuth defines the authentication methods for minting an Entra access token.
// Exactly one of the authentication methods must be configured.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type AzureAuth struct {
	// ServicePrincipal uses Azure Service Principal credentials (client secret or
	// client certificate) to authenticate with Microsoft Entra ID.
	// +optional
	ServicePrincipal *AzureServicePrincipalAuth `json:"servicePrincipal,omitempty"`

	// ManagedIdentity uses Azure Managed Identity to authenticate with Microsoft Entra ID.
	// +optional
	ManagedIdentity *AzureManagedIdentityAuth `json:"managedIdentity,omitempty"`

	// WorkloadIdentity uses Azure Workload Identity to authenticate with Microsoft Entra ID.
	// +optional
	WorkloadIdentity *AzureWorkloadIdentityAuth `json:"workloadIdentity,omitempty"`
}

// AzureServicePrincipalAuth defines the configuration for using Azure Service
// Principal authentication. Exactly one of ClientSecret or ClientCertificate must be set.
type AzureServicePrincipalAuth struct {
	SecretRef AzureServicePrincipalAuthSecretRef `json:"secretRef"`
}

// AzureManagedIdentityAuth defines the configuration for using Azure Managed Identity authentication.
type AzureManagedIdentityAuth struct {
	// If multiple Managed Identities are assigned to the pod, you can select the one to be used.
	// +optional
	IdentityID string `json:"identityId,omitempty"`
}

// AzureWorkloadIdentityAuth defines the configuration for using Azure Workload Identity authentication.
type AzureWorkloadIdentityAuth struct {
	// ServiceAccountRef specifies the service account
	// that should be used when authenticating with WorkloadIdentity.
	// +optional
	ServiceAccountRef *smmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

// AzureServicePrincipalAuthSecretRef defines the secret references for Azure Service
// Principal authentication. Provide either a client secret or a client certificate.
// +kubebuilder:validation:XValidation:rule="has(self.clientSecret) != has(self.clientCertificate)",message="exactly one of clientSecret or clientCertificate must be set"
type AzureServicePrincipalAuthSecretRef struct {
	// The Azure clientId of the service principal used for authentication.
	ClientID smmeta.SecretKeySelector `json:"clientId"`
	// The Azure ClientSecret of the service principal used for authentication.
	// Mutually exclusive with ClientCertificate.
	// +optional
	ClientSecret *smmeta.SecretKeySelector `json:"clientSecret,omitempty"`
	// The PEM-encoded certificate (certificate and private key) of the service principal
	// used for certificate based authentication. Mutually exclusive with ClientSecret.
	// +optional
	ClientCertificate *smmeta.SecretKeySelector `json:"clientCertificate,omitempty"`
}

// AzureAccessToken generates a Microsoft Entra ID access token scoped to a configurable
// resource. The token is returned under the "token" key and can be remapped (e.g. to
// AZP_TOKEN for Azure DevOps) via the consuming ExternalSecret template.
//
// See tracking issue: https://github.com/external-secrets/external-secrets/issues/5113
//
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type AzureAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AzureAccessTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// AzureAccessTokenList contains a list of AzureAccessToken resources.
type AzureAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureAccessToken `json:"items"`
}
