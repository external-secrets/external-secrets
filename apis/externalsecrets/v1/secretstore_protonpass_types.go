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

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// ProtonPassAuth contains auth for Proton Pass.
// +kubebuilder:validation:XValidation:rule="has(self.personalAccessTokenSecretRef.name) && size(self.personalAccessTokenSecretRef.name) > 0 && has(self.personalAccessTokenSecretRef.key) && size(self.personalAccessTokenSecretRef.key) > 0",message="personalAccessTokenSecretRef.name and personalAccessTokenSecretRef.key are required"
//
//nolint:lll // Kubebuilder validation markers cannot be split across multiple lines.
type ProtonPassAuth struct {
	// PersonalAccessTokenSecretRef references a secret whose value is the
	// complete Proton Pass PAT (formatted as "pst_[token]::[key]"), including
	// its 32-byte AES key half. The key half is used client-side to decrypt
	// vault content.
	PersonalAccessTokenSecretRef esmeta.SecretKeySelector `json:"personalAccessTokenSecretRef"`
}

// ProtonPassProvider configures a store to sync secrets using the Proton Pass provider (read-only).
type ProtonPassProvider struct {
	// Auth defines the PAT credential.
	Auth *ProtonPassAuth `json:"auth"`
}
