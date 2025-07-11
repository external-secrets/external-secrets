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

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// OnePasswordSDKAuth contains a secretRef for the service account token.
type OnePasswordSDKAuth struct {
	// ServiceAccountSecretRef points to the secret containing the token to access 1Password vault.
	ServiceAccountSecretRef esmeta.SecretKeySelector `json:"serviceAccountSecretRef"`
}

// IntegrationInfo specifies the name and version of the integration built using the 1Password Go SDK.
type IntegrationInfo struct {
	// Name defaults to "1Password SDK".
	// +kubebuilder:default="1Password SDK"
	Name string `json:"name,omitempty"`
	// Version defaults to "v1.0.0".
	// +kubebuilder:default="v1.0.0"
	Version string `json:"version,omitempty"`
}

// OnePasswordSDKProvider configures a store to sync secrets using the 1Password sdk.
type OnePasswordSDKProvider struct {
	// Vault defines the vault's name or uuid to access. Do NOT add op:// prefix. This will be done automatically.
	Vault string `json:"vault"`
	// IntegrationInfo specifies the name and version of the integration built using the 1Password Go SDK.
	// If you don't know which name and version to use, use `DefaultIntegrationName` and `DefaultIntegrationVersion`, respectively.
	// +optional
	IntegrationInfo *IntegrationInfo `json:"integrationInfo,omitempty"`
	// Auth defines the information necessary to authenticate against OnePassword API.
	Auth *OnePasswordSDKAuth `json:"auth"`
}
