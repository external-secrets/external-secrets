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

// Configures an store to sync secrets using Azure KV.
type AzureKVProvider struct {
	// Vault Url from which the secrets to be fetched from.
	VaultURL *string `json:"vaultUrl"`
	// TenantID configures the Azure Tenant to send requests to.
	TenantID *string `json:"tenantId"`
	// Auth configures how the operator authenticates with Azure.
	AuthSecretRef *AzureKVAuth `json:"authSecretRef"`
	// ActiveDirectoryEndpoint configures which Active Directory Endpoint to use when not on default public cloud
	ActiveDirectoryEndpoint *string `json:"activeDirectoryEndpoint,omitempty"`
	// ActiveDirectoryResourceID configures which Active Directory Resource ID to use when not on default public cloud
	ActiveDirectoryResourceID *string `json:"activeDirectoryResourceID,omitempty"`
}

// Configuration used to authenticate with Azure.
type AzureKVAuth struct {
	// The Azure clientId of the service principle used for authentication.
	ClientID *smmeta.SecretKeySelector `json:"clientId"`
	// The Azure ClientSecret of the service principle used for authentication.
	ClientSecret *smmeta.SecretKeySelector `json:"clientSecret"`
}
