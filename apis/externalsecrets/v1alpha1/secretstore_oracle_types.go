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

// +kubebuilder:validation:Enum="";UserPrincipal;InstancePrincipal;Workload
type OraclePrincipalType string

const (
	// UserPrincipal represents a user principal.
	UserPrincipal OraclePrincipalType = "UserPrincipal"
	// InstancePrincipal represents a instance principal.
	InstancePrincipal OraclePrincipalType = "InstancePrincipal"
	// WorkloadPrincipal represents a workload principal.
	WorkloadPrincipal OraclePrincipalType = "Workload"
)

// Configures an store to sync secrets using a Oracle Vault
// backend.
type OracleProvider struct {
	// Region is the region where vault is located.
	Region string `json:"region"`

	// Vault is the vault's OCID of the specific vault where secret is located.
	Vault string `json:"vault"`

	// Compartment is the vault compartment OCID.
	// Required for PushSecret
	// +optional
	Compartment string `json:"compartment,omitempty"`

	// EncryptionKey is the OCID of the encryption key within the vault.
	// Required for PushSecret
	// +optional
	EncryptionKey string `json:"encryptionKey,omitempty"`

	// The type of principal to use for authentication. If left blank, the Auth struct will
	// determine the principal type. This optional field must be specified if using
	// workload identity.
	// +optional
	PrincipalType OraclePrincipalType `json:"principalType,omitempty"`

	// Auth configures how secret-manager authenticates with the Oracle Vault.
	// If empty, instance principal is used. Optionally, the authenticating principal type
	// and/or user data may be supplied for the use of workload identity and user principal.
	// +optional
	Auth *OracleAuth `json:"auth,omitempty"`

	// ServiceAccountRef specified the service account
	// that should be used when authenticating with WorkloadIdentity.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

type OracleAuth struct {
	// Tenancy is the tenancy OCID where user is located.
	Tenancy string `json:"tenancy"`

	// User is an access OCID specific to the account.
	User string `json:"user"`

	// SecretRef to pass through sensitive information.
	SecretRef OracleSecretRef `json:"secretRef"`
}

type OracleSecretRef struct {
	// PrivateKey is the user's API Signing Key in PEM format, used for authentication.
	PrivateKey esmeta.SecretKeySelector `json:"privatekey"`

	// Fingerprint is the fingerprint of the API private key.
	Fingerprint esmeta.SecretKeySelector `json:"fingerprint"`
}
