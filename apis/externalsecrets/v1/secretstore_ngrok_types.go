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

// NgrokProvider configures a store to sync secrets with a ngrok vault to use in traffic policies.
// See: https://ngrok.com/blog-post/secrets-for-traffic-policy
type NgrokProvider struct {
	// APIURL is the URL of the ngrok API.
	// +kubebuilder:default="https://api.ngrok.com"
	APIURL string `json:"apiUrl,omitempty"`

	// Auth configures how the ngrok provider authenticates with the ngrok API.
	// +kubebuilder:validation:Required
	Auth NgrokAuth `json:"auth"`

	// Vault configures the ngrok vault to sync secrets with.
	// +kubebuilder:validation:Required
	Vault NgrokVault `json:"vault"`
}

// NgrokAuth configures the authentication method for the ngrok provider.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type NgrokAuth struct {
	// APIKey is the API Key used to authenticate with ngrok. See https://ngrok.com/docs/api/#authentication
	// +optional
	APIKey *NgrokProviderSecretRef `json:"apiKey,omitempty"`
}

// NgrokVault configures the ngrok vault to sync secrets with.
type NgrokVault struct {
	// Name is the name of the ngrok vault to sync secrets with.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// NgrokProviderSecretRef contains the secret reference for the ngrok provider.
type NgrokProviderSecretRef struct {
	// SecretRef is a reference to a secret containing the ngrok API key.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}
