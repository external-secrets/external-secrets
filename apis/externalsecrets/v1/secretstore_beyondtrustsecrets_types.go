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

package v1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// BeyondTrustSecretAuthSecretRef defines a reference to a secret containing credentials for the BeyondTrust Secrets Manager provider.
// The nested structure supports multiple authentication methods (currently only API token is supported).
// For more information on authentication, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#authentication
type BeyondTrustSecretAuthSecretRef struct {
	// Token references the Kubernetes secret containing the BeyondTrust Secrets Manager API token.
	// The secret should contain the API key used to authenticate with BeyondTrust Secrets Manager.
	// Create an API token in your BeyondTrust Secrets Manager console and store it in a Kubernetes secret.
	// For details on creating API tokens, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#authentication
	Token esmeta.SecretKeySelector `json:"token"`
}

// BeyondTrustSecretAuth defines the authentication method for the BeyondTrust Secrets Manager provider.
// Currently supports API key authentication via Kubernetes secret reference.
// For authentication documentation, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#authentication
type BeyondTrustSecretAuth struct {
	// APIKey configures API token authentication for BeyondTrust Secrets Manager.
	// The token is retrieved from a Kubernetes secret and used as a Bearer token for API requests.
	APIKey BeyondTrustSecretAuthSecretRef `json:"apikey"`
}

// BeyondTrustSecretsServer defines connection configuration for BeyondTrust Secrets Manager.
// For API reference documentation, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api
type BeyondTrustSecretsServer struct {
	// APIURL is the base URL of your BeyondTrust Secrets Manager API server.
	// This should be the full URL to your BeyondTrust instance.
	// Example: https://example.secretsmanager.cyberark.cloud
	// For more information, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#base-url
	// +required
	APIURL string `json:"apiUrl"`

	// SiteID is your BeyondTrust Secrets Manager site identifier (UUID format).
	// This identifier is unique to your BeyondTrust Secrets Manager instance.
	// You can find your Site ID in the BeyondTrust Secrets Manager admin console.
	// Example: a1b2c3d4-e5f6-7890-abcd-ef1234567890
	// For more information, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api
	// +required
	SiteID string `json:"siteId"`
}

// BeyondTrustSecretsProvider configures a store to sync secrets using the BeyondTrust Secrets Manager provider.
// BeyondTrust Secrets Manager provides secure storage for static secrets and dynamic credential generation.
// This provider supports reading secrets and generating dynamic credentials (e.g., temporary AWS credentials).
// For complete documentation, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api
type BeyondTrustSecretsProvider struct {
	// Auth configures how the Operator authenticates with the BeyondTrust Secrets Manager API.
	// Currently supports API key authentication via Kubernetes secret reference.
	// For authentication setup, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#authentication
	// +required
	Auth *BeyondTrustSecretAuth `json:"auth"`

	// Server configures the BeyondTrust Secrets Manager server connection details.
	// Includes the API URL and Site ID for your BeyondTrust instance.
	// For API reference, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api
	// +required
	Server *BeyondTrustSecretsServer `json:"server"`

	// FolderPath specifies the default folder path for secret retrieval.
	// Secrets will be fetched from this folder unless overridden in the ExternalSecret spec.
	// Example: "production/database" or "dev/api-keys"
	// Leave empty to retrieve secrets from the root folder.
	// For folder organization, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#folders
	// +optional
	FolderPath string `json:"folderPath,omitempty"`

	// CABundle is a base64-encoded CA certificate used to validate the BeyondTrust Secrets Manager API TLS certificate.
	// Use this when your BeyondTrust instance uses a self-signed certificate or internal CA.
	// If not set, the system's trusted root certificates are used.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// CAProvider points to a Secret or ConfigMap containing a PEM-encoded CA certificate.
	// This is used to validate the BeyondTrust Secrets Manager API TLS certificate.
	// Use this as an alternative to CABundle when you want to reference an existing Kubernetes resource.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`
}
