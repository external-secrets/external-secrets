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

// PreviderProvider configures a store to sync secrets using the Previder Secret Manager provider.
type PreviderProvider struct {
	Auth PreviderAuth `json:"auth"`
	// +optional
	BaseURI string `json:"baseUri,omitempty"`
}

// PreviderAuth contains a secretRef for credentials.
type PreviderAuth struct {
	// +optional
	SecretRef *PreviderAuthSecretRef `json:"secretRef,omitempty"`
}

// PreviderAuthSecretRef holds secret references for Previder Vault credentials.
type PreviderAuthSecretRef struct {
	// The AccessToken is used for authentication
	AccessToken esmeta.SecretKeySelector `json:"accessToken"`
}
