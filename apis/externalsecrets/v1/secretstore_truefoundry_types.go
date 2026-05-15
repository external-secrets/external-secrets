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

// TrueFoundryProvider configures a store to sync secrets using the TrueFoundry provider.
// Secrets are fetched from TrueFoundry secret groups via its REST API.
// The TrueFoundry API requires a two-step lookup: a secret group is located by its
// fully-qualified name (FQN, "<tenant>:<group>") and the value for each associated
// secret is then fetched by its ID. See docs/provider/truefoundry.md for details.
type TrueFoundryProvider struct {
	// BaseURL is the TrueFoundry control-plane URL, e.g. https://app.truefoundry.com.
	// The client appends /api/svc to this when calling the secrets API.
	// +kubebuilder:validation:MinLength=1
	BaseURL string `json:"baseURL"`

	// Tenant is the TrueFoundry tenant name. It is used together with the secret group
	// name to build the FQN passed to the search endpoint ("<tenant>:<group>").
	// +kubebuilder:validation:MinLength=1
	Tenant string `json:"tenant"`

	// Auth configures how the Operator authenticates with the TrueFoundry API.
	Auth TrueFoundryAuth `json:"auth"`
}

// TrueFoundryAuth configures authentication against the TrueFoundry API.
// Only Bearer-token authentication (via a Kubernetes Secret) is supported today.
type TrueFoundryAuth struct {
	// SecretRef authenticates using a TrueFoundry Personal Access Token stored in
	// a Kubernetes Secret. The token is sent as `Authorization: Bearer <token>`.
	SecretRef TrueFoundryAuthSecretRef `json:"secretRef"`
}

// TrueFoundryAuthSecretRef contains the SecretKeySelector that points to the
// Kubernetes Secret holding the TrueFoundry API key.
type TrueFoundryAuthSecretRef struct {
	// APIKey references a key inside a Kubernetes Secret that contains a
	// TrueFoundry Personal Access Token.
	APIKey esmeta.SecretKeySelector `json:"apiKey"`
}
