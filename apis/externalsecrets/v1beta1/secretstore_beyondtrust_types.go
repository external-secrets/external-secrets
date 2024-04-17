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
	// Required - BeyondTrust Password Safe API URL. https://example.com:443/beyondtrust/api/public/V3.
	APIURL string `json:"apiurl"`
	// Required - API OAuth Client ID.
	Clientid *BeyondTrustProviderSecretRef `json:"clientid"`
	// Required - API OAuth Client Secret.
	Clientsecret *BeyondTrustProviderSecretRef `json:"clientsecret"`
	// Content of the certificate (cert.pem) for use when authenticating with an OAuth client Id using a Client Certificate.
	Certificate *BeyondTrustProviderSecretRef `json:"certificate,omitempty"`
	// Certificate private key (key.pem). For use when authenticating with an OAuth client Id
	Certificatekey *BeyondTrustProviderSecretRef `json:"certificatekey,omitempty"`
	// The secret retrieval type. SECRET = Secrets Safe (credential, text, file). MANAGED_ACCOUNT = Password Safe account associated with a system.
	Retrievaltype string `json:"retrievaltype,omitempty"`
	// A character that separates the folder names.
	Separator string `json:"separator,omitempty"`
	// Required - Indicates whether to verify the certificate authority on the Secrets Safe instance. Warning - false is insecure, instructs the BT provider not to verify the certificate authority.
	VerifyCA bool `json:"verifyca"`
	// Timeout specifies a time limit for requests made by this Client. The timeout includes connection time, any redirects, and reading the response body. Defaults to 45 seconds.
	Clienttimeoutseconds int `json:"clienttimeoutseconds,omitempty"`
}
