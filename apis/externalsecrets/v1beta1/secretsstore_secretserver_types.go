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

package v1beta1

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

type SecretServerProviderRef struct {

	// Value can be specified directly to set a value without using a secret.
	// +optional
	Value string `json:"value,omitempty"`

	// SecretRef references a key in a secret that will be used as value.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// See https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go.
type SecretServerProvider struct {

	// Username is the secret server account username.
	// +required
	Username *SecretServerProviderRef `json:"username"`

	// Password is the secret server account password.
	// +required
	Password *SecretServerProviderRef `json:"password"`

	// ServerURL
	// URL to your secret server installation
	// +required
	ServerURL string `json:"serverURL"`
}
