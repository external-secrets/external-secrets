package v1alpha1

import smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// Configures an store to sync secrets using Azure KV.
type AzureKVProvider struct {
	// Vault Url from which the secrets to be fetched from.
	VaultUrl *string `json:"vaultUrl"`
	// TenantID configures the Azure Tenant to send requests to.
	TenantID *string `json:"tenantId"`
	// Auth configures how the operator authenticates with Azure.
	AuthSecretRef *AzureKVAuth `json:"authSecretRef"`
}

// Configuration used to authenticate with Azure.
type AzureKVAuth struct {
	// The Azure clientId of the service principle used for authentication.
	ClientID *smmeta.SecretKeySelector `json:"clientId"`
	// The Azure ClientSecret of the service principle used for authentication.
	ClientSecret *smmeta.SecretKeySelector `json:"clientSecret"`
}
