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

// Auth contains a secretRef for credentials.
type BeyondTrustProviderSecretRef struct {

	// Value can be specified directly to set a value without using a secret.
	// +optional
	Value string `json:"value,omitempty"`

	// SecretRef references a key in a secret that will be used as value.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// Configures a store to sync secrets using BeyondTrust Password Safe.
type BeyondtrustProvider struct {
	APIURL         string                        `json:"apiurl"`
	Clientid       *BeyondTrustProviderSecretRef `json:"clientid"`
	Clientsecret   *BeyondTrustProviderSecretRef `json:"clientsecret"`
	Certificate    *BeyondTrustProviderSecretRef `json:"certificate,omitempty"`
	Certificatekey *BeyondTrustProviderSecretRef `json:"certificatekey,omitempty"`
	Retrievaltype  string                        `json:"retrievaltype,omitempty"`
}
