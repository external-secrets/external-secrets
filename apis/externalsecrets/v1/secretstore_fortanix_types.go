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

// FortanixProvider provides access to Fortanix SDKMS API using the provided credentials.
type FortanixProvider struct {
	// APIURL is the URL of SDKMS API. Defaults to `sdkms.fortanix.com`.
	APIURL string `json:"apiUrl,omitempty"`

	// APIKey is the API token to access SDKMS Applications.
	APIKey *FortanixProviderSecretRef `json:"apiKey,omitempty"`
}

// FortanixProviderSecretRef is a secret reference containing the SDKMS API Key.
type FortanixProviderSecretRef struct {
	// SecretRef is a reference to a secret containing the SDKMS API Key.
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}
