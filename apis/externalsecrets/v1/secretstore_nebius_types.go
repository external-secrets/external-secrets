// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package v1

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// NebiusAuth defines the authentication method for the Nebius provider.
// +kubebuilder:validation:XValidation:rule="has(self.serviceAccountCredsSecretRef) || has(self.tokenSecretRef)",message="either serviceAccountCredsSecretRef or tokenSecretRef must be set"
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
}

// NebiusCAProvider The provider for the CA bundle to use to validate Nebius server certificate.
type NebiusCAProvider struct {
	// +optional
	Certificate esmeta.SecretKeySelector `json:"certSecretRef,omitempty"`
}

// NebiusMysteryboxProvider Configures a store to sync secrets using the Nebius Mysterybox provider.
type NebiusMysteryboxProvider struct {

	// NebiusMysterybox API endpoint
	APIDomain string `json:"apiDomain,omitempty"`

	// Auth defines parameters to authenticate in MysteryBox
	Auth NebiusAuth `json:"auth"`

	// The provider for the CA bundle to use to validate NebiusMysterybox server certificate.
	// +optional
	CAProvider *NebiusCAProvider `json:"caProvider,omitempty"`
}
