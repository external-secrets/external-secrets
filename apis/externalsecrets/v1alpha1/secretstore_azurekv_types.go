package v1alpha1

import smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// Configures an store to sync secrets using Azure KV.
type AzureKVProvider struct {
	// TenantID configures the Azure Tenant to send requests to.
	TenantID *string `json:"tenantid"`
	// Auth configures how the operator authenticates with Azure.
	AuthSecretRef *AzureKVAuth `json:"authSecretRef"`
}

// Configuration used to authenticate with Azure.
type AzureKVAuth struct {
	// The Azure clientId of the service principle used for authentication.
	ClientID *smmeta.SecretKeySelector `json:"clientID"`
	// The Azure ClientSecret of the service principle used for authentication.
	ClientSecret *smmeta.SecretKeySelector `json:"clientSecret"`
}
