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

// BeyondtrustSecretAuthSecretRef defines a reference to a secret containing credentials for the BeyondtrustSecret provider.
type BeyondtrustSecretAuthSecretRef struct {
	// The token is used for authentication.
	Token esmeta.SecretKeySelector `json:"token"`
}

// BeyondtrustSecretAuth defines the authentication method for the BeyondtrustSecret provider.
type BeyondtrustSecretAuth struct {
	APIKey BeyondtrustSecretAuthSecretRef `json:"apikey"`
}

// BeyondtrustSecretsServer defines configuration for connecting to BeyondtrustSecrets server.
type BeyondtrustSecretsServer struct {
	// +required
	APIURL string `json:"apiUrl"`
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
	// +required
	SiteID string `json:"siteId"`
}

// BeyondtrustSecretsProvider configures a store to sync secrets using the BeyondtrustSecrets provider.
type BeyondtrustSecretsProvider struct {
	// Auth configures how the Operator authenticates with the BeyondtrustSecret API
	// +required
	Auth *BeyondtrustSecretAuth `json:"auth"`

	// Server configures the BeyondtrustSecret server connection details
	// +required
	Server *BeyondtrustSecretsServer `json:"server"`

	// Folder path to retrieve secret from
	// +optional
	FolderPath string `json:"folderPath,omitempty"`

	// CABundle is a base64-encoded CA certificate used to validate the BeyondtrustSecrets API TLS certificate. If not set, system roots are used.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// CAProvider points to a Secret or ConfigMap containing a PEM-encoded certificate used to validate the BeyondtrustSecrets API TLS certificate.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`
}
