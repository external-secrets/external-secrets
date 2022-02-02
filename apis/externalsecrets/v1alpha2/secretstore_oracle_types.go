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

package v1alpha2

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

	// Tenancy is the tenancy OCID where secret is located.
	Tenancy string `json:"tenancy,omitempty"`

	// Region is the region where secret is located.
	Region string `json:"region,omitempty"`

	// Vault is the vault's OCID of the specific vault where secret is located.
	Vault string `json:"vault,omitempty"`
}

type OracleAuth struct {
	// SecretRef to pass through sensitive information.
	SecretRef OracleSecretRef `json:"secretRef"`
}

type OracleSecretRef struct {
	// PrivateKey is the user's API Signing Key in PEM format, used for authentication.
	PrivateKey esmeta.SecretKeySelector `json:"privatekey,omitempty"`

	// Fingerprint is the fingerprint of the API private key.
	Fingerprint esmeta.SecretKeySelector `json:"fingerprint,omitempty"`
}
