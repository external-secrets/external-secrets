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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	prov "github.com/external-secrets/external-secrets/apis/providers/v1alpha1"
)

// ACRAccessTokenSpec defines how to generate the access token
// e.g. how to authenticate and which registry to use.
// see: https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md#overview
type ACRAccessTokenSpec struct {
	Auth ACRAuth `json:"auth"`
	// TenantID configures the Azure Tenant to send requests to. Required for ServicePrincipal auth type.
	TenantID string `json:"tenantId,omitempty"`

	// the domain name of the ACR registry
	// e.g. foobarexample.azurecr.io
	ACRRegistry string `json:"registry"`

	// Define the scope for the access token, e.g. pull/push access for a repository.
	// if not provided it will return a refresh token that has full scope.
	// Note: you need to pin it down to the repository level, there is no wildcard available.
	//
	// examples:
	// repository:my-repository:pull,push
	// repository:my-repository:pull
	//
	// see docs for details: https://docs.docker.com/registry/spec/auth/scope/
	// +optional
	Scope string `json:"scope,omitempty"`

	// EnvironmentType specifies the Azure cloud environment endpoints to use for
	// connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
	// The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152
	// PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud
	// +kubebuilder:default=PublicCloud
	EnvironmentType prov.AzureEnvironmentType `json:"environmentType,omitempty"`
}

type ACRAuth struct {
	// ServicePrincipal uses Azure Service Principal credentials to authenticate with Azure.
	// +optional
	ServicePrincipal *AzureACRServicePrincipalAuth `json:"servicePrincipal,omitempty"`

	// ManagedIdentity uses Azure Managed Identity to authenticate with Azure.
	// +optional
	ManagedIdentity *AzureACRManagedIdentityAuth `json:"managedIdentity,omitempty"`

	// WorkloadIdentity uses Azure Workload Identity to authenticate with Azure.
	// +optional
	WorkloadIdentity *AzureACRWorkloadIdentityAuth `json:"workloadIdentity,omitempty"`
}

type AzureACRServicePrincipalAuth struct {
	SecretRef AzureACRServicePrincipalAuthSecretRef `json:"secretRef"`
}

type AzureACRManagedIdentityAuth struct {
	// If multiple Managed Identity is assigned to the pod, you can select the one to be used
	IdentityID string `json:"identityId,omitempty"`
}

type AzureACRWorkloadIdentityAuth struct {
	// ServiceAccountRef specified the service account
	// that should be used when authenticating with WorkloadIdentity.
	// +optional
	ServiceAccountRef *smmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

// Configuration used to authenticate with Azure using static
// credentials stored in a Kind=Secret.
type AzureACRServicePrincipalAuthSecretRef struct {
	// The Azure clientId of the service principle used for authentication.
	ClientID smmeta.SecretKeySelector `json:"clientId,omitempty"`
	// The Azure ClientSecret of the service principle used for authentication.
	ClientSecret smmeta.SecretKeySelector `json:"clientSecret,omitempty"`
}

// ACRAccessToken returns a Azure Container Registry token
// that can be used for pushing/pulling images.
// Note: by default it will return an ACR Refresh Token with full access
// (depending on the identity).
// This can be scoped down to the repository level using .spec.scope.
// In case scope is defined it will return an ACR Access Token.
//
// See docs: https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md
//
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={acraccesstoken},shortName=acraccesstoken
type ACRAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ACRAccessTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ACRAccessTokenList contains a list of ExternalSecret resources.
type ACRAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ACRAccessToken `json:"items"`
}
