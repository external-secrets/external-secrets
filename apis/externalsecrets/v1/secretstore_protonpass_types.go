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

// ProtonPassProvider configures a store to sync secrets using Proton Pass via a
// Personal Access Token (PAT). The provider talks directly to the Proton Pass
// HTTP API; no CLI or sidecar is required.
type ProtonPassProvider struct {
	// Auth configures how the operator authenticates with Proton Pass.
	Auth ProtonPassAuth `json:"auth"`

	// Vaults optionally restricts the Proton Pass vaults this store uses to the
	// named vaults (an allow-list). An item title that is ambiguous across the
	// in-scope vaults is always a hard error (never a silent pick) — address such
	// items by id:<ItemID>. When empty, every vault the token can access is used.
	// +optional
	Vaults []string `json:"vaults,omitempty"`
}

// ProtonPassAuth contains the authentication configuration for Proton Pass.
type ProtonPassAuth struct {
	// PersonalAccessTokenSecretRef references a Secret holding the full Proton Pass
	// Personal Access Token string, in the form "pst_<token>::<key>". A viewer-role
	// token yields a read-only store; an editor/manager token enables PushSecret.
	PersonalAccessTokenSecretRef esmeta.SecretKeySelector `json:"personalAccessTokenSecretRef"`
}
