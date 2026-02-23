/*
Copyright © 2026 ESO Maintainer Team

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

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// ProtonPassAuth contains authentication configuration for Proton Pass.
type ProtonPassAuth struct {
	// SecretRef contains the secret references for authentication.
	SecretRef ProtonPassAuthSecretRef `json:"secretRef"`
}

// ProtonPassAuthSecretRef contains the secret references for Proton Pass authentication.
type ProtonPassAuthSecretRef struct {
	// Password is a reference to the Proton account password.
	Password esmeta.SecretKeySelector `json:"password"`

	// TOTP is a reference to the TOTP secret for two-factor authentication.
	// This should be the TOTP secret key (not the generated code).
	// +optional
	TOTP *esmeta.SecretKeySelector `json:"totp,omitempty"`

	// ExtraPassword is a reference to an extra password if configured on the account.
	// +optional
	ExtraPassword *esmeta.SecretKeySelector `json:"extraPassword,omitempty"`
}

// ProtonPassProvider configures a store to sync secrets using the Proton Pass provider.
type ProtonPassProvider struct {
	// Auth configures how the operator authenticates with Proton Pass.
	Auth *ProtonPassAuth `json:"auth"`

	// Username is the Proton account username (email).
	Username string `json:"username"`

	// Vault is the name of the Proton Pass vault to use.
	// If not specified, items from all vaults will be accessible.
	// +optional
	Vault string `json:"vault,omitempty"`
}
