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

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// OnePasswordAuth contains a secretRef for credentials.
type OnePasswordAuth struct {
	SecretRef *OnePasswordAuthSecretRef `json:"secretRef"`
}

// OnePasswordAuthSecretRef holds secret references for 1Password credentials.
type OnePasswordAuthSecretRef struct {
	// The ConnectToken is used for authentication to a 1Password Connect Server.
	ConnectToken esmeta.SecretKeySelector `json:"connectTokenSecretRef"`
}

// OnePasswordProvider configures a store to sync secrets using the 1Password Secret Manager provider.
type OnePasswordProvider struct {
	// Auth defines the information necessary to authenticate against OnePassword Connect Server
	Auth *OnePasswordAuth `json:"auth"`
	// ConnectHost defines the OnePassword Connect Server to connect to
	ConnectHost string `json:"connectHost"`
	// Vaults defines which OnePassword vaults to search in which order
	Vaults map[string]int `json:"vaults"`
}
