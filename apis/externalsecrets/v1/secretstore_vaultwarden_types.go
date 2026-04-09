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

package v1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// VaultwardenProvider configures a store to sync secrets with a self-hosted Vaultwarden instance.
type VaultwardenProvider struct {
	// URL is the base URL of the Vaultwarden instance, e.g. https://vault.example.com
	URL string `json:"url"`
	// Auth configures how ESO authenticates with Vaultwarden using a personal API key.
	Auth VaultwardenAuth `json:"auth"`
}

// VaultwardenAuth holds references to the Kubernetes secrets containing Vaultwarden credentials.
type VaultwardenAuth struct {
	SecretRef VaultwardenSecretRef `json:"secretRef"`
}

// VaultwardenSecretRef contains selectors for the three credentials needed to access Vaultwarden.
type VaultwardenSecretRef struct {
	// ClientID is the OAuth2 client_id of the Vaultwarden personal API key.
	ClientID esmeta.SecretKeySelector `json:"clientID"`
	// ClientSecret is the OAuth2 client_secret of the Vaultwarden personal API key.
	ClientSecret esmeta.SecretKeySelector `json:"clientSecret"`
	// MasterPassword is the Vaultwarden account master password used to decrypt vault items.
	MasterPassword esmeta.SecretKeySelector `json:"masterPassword"`
}
