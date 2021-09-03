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

package v1alpha1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Configures an store to sync secrets using a Oracle Vault
// backend.
type OracleProvider struct {
	// Auth configures how secret-manager authenticates with the Oracle Vault.
	Auth OracleAuth `json:"auth"`

	// User is an access OCID specific to the account.
	User string `json:"user,omitempty"`

	// projectID is an access token specific to the secret.
	Tenancy string `json:"tenancy,omitempty"`

	// projectID is an access token specific to the secret.
	Region string `json:"region,omitempty"`
}

type OracleAuth struct {
	// SecretRef to pass through sensitive information.
	SecretRef OracleSecretRef `json:"secretRef"`
}

type OracleSecretRef struct {
	// The Access Token is used for authentication
	PrivateKey esmeta.SecretKeySelector `json:"privatekey,omitempty"`

	// projectID is an access token specific to the secret.
	Fingerprint esmeta.SecretKeySelector `json:"fingerprint,omitempty"`
}
