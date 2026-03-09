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

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// SecretServerProviderRef references a value that can be specified directly or via a secret
// for a SecretServerProvider.
type SecretServerProviderRef struct {

	// Value can be specified directly to set a value without using a secret.
	// +optional
	Value string `json:"value,omitempty"`

	// SecretRef references a key in a secret that will be used as value.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// SecretServerProvider provides access to authenticate to a secrets provider server.
// See: https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go.
type SecretServerProvider struct {
	// Username is the secret server account username.
	// +required
	Username *SecretServerProviderRef `json:"username"`

	// Password is the secret server account password.
	// +required
	Password *SecretServerProviderRef `json:"password"`

	// Domain is the secret server domain.
	// +optional
	Domain string `json:"domain,omitempty"`

	// ServerURL
	// URL to your secret server installation
	// +required
	ServerURL string `json:"serverURL"`

	// PEM/base64 encoded CA bundle used to validate Secret ServerURL. Only used
	// if the ServerURL URL is using HTTPS protocol. If not set the system root certificates
	// are used to validate the TLS connection.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// The provider for the CA bundle to use to validate Secret ServerURL certificate.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`
}
