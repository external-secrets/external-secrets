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

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// NebiusAuth defines the authentication method for the Nebius provider.
// +kubebuilder:validation:XValidation:rule="(has(self.serviceAccountCredsSecretRef) && !has(self.tokenSecretRef) && !has(self.serviceAccountRef)) || (!has(self.serviceAccountCredsSecretRef) && has(self.tokenSecretRef) && !has(self.serviceAccountRef)) || (!has(self.serviceAccountCredsSecretRef) && !has(self.tokenSecretRef) && has(self.serviceAccountRef))",message="exactly one of serviceAccountCredsSecretRef, tokenSecretRef or serviceAccountRef must be set"
// +kubebuilder:validation:XValidation:rule="!has(self.serviceAccountRef) || self.iamServiceAccountID != ''",message="iamServiceAccountID must be set when serviceAccountRef is used"
// +kubebuilder:validation:XValidation:rule="has(self.serviceAccountRef) || !has(self.iamServiceAccountID)",message="iamServiceAccountID can only be used with serviceAccountRef"
type NebiusAuth struct {
	// ServiceAccountCreds references a Kubernetes Secret key that contains a JSON
	// document with service account credentials used to get an IAM token.
	//
	// Expected JSON structure:
	// {
	//   "subject-credentials": {
	//     "alg": "RS256",
	//     "private-key": "-----BEGIN PRIVATE KEY-----\n<private-key>\n-----END PRIVATE KEY-----\n",
	//     "kid": "<public-key-id>",
	//     "iss": "<issuer-service-account-id>",
	//     "sub": "<subject-service-account-id>"
	//   }
	// }
	// +optional
	ServiceAccountCreds esmeta.SecretKeySelector `json:"serviceAccountCredsSecretRef,omitempty"`
	// Token authenticates with Nebius Mysterybox by presenting a token.
	// +optional
	Token esmeta.SecretKeySelector `json:"tokenSecretRef,omitempty"`

	// ServiceAccountRef references a Kubernetes ServiceAccount used to request a
	// temporary JWT via the TokenRequest API. The JWT is then exchanged for a
	// Nebius IAM token using workload federation.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`

	// IAMServiceAccountID is the Nebius IAM service account identifier that the
	// federated Kubernetes service account should impersonate during token exchange.
	// This field is required when serviceAccountRef is used.
	// +optional
	IAMServiceAccountID string `json:"iamServiceAccountID,omitempty"`
}

// NebiusCAProvider The provider for the CA bundle to use to validate Nebius server certificate.
type NebiusCAProvider struct {
	// +optional
	Certificate esmeta.SecretKeySelector `json:"certSecretRef,omitempty"`
}

// NebiusMysteryboxProvider Configures a store to sync secrets using the Nebius Mysterybox provider.
type NebiusMysteryboxProvider struct {

	// NebiusMysterybox API endpoint
	APIDomain string `json:"apiDomain"`

	// Auth defines parameters to authenticate in MysteryBox
	Auth NebiusAuth `json:"auth"`

	// The provider for the CA bundle to use to validate NebiusMysterybox server certificate.
	// +optional
	CAProvider *NebiusCAProvider `json:"caProvider,omitempty"`
}
