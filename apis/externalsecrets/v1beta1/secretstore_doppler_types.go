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

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Set DOPPLER_BASE_URL and DOPPLER_VERIFY_TLS environment variables to override defaults

type DopplerAuth struct {
	SecretRef DopplerAuthSecretRef `json:"secretRef"`
}

type DopplerAuthSecretRef struct {
	// The DopplerToken is used for authentication.
	// See https://docs.doppler.com/reference/api#authentication for auth token types.
	// The Key attribute defaults to dopplerToken if not specified.
	DopplerToken esmeta.SecretKeySelector `json:"dopplerToken"`
}

// DopplerProvider configures a store to sync secrets using the Doppler provider.
// Project and Config are required if not using a Service Token.
type DopplerProvider struct {
	// Auth configures how the Operator authenticates with the Doppler API
	Auth *DopplerAuth `json:"auth"`

	// Doppler project (required if not using a Service Token)
	// +optional
	Project string `json:"project,omitempty"`

	// Doppler config (required if not using a Service Token)
	// +optional
	Config string `json:"config,omitempty"`

	// Environment variable compatible name transforms that change secret names to a different format
	// +kubebuilder:validation:Enum=upper-camel;camel;lower-snake;tf-var;dotnet-env;lower-kebab
	// +optional
	NameTransformer string `json:"nameTransformer,omitempty"`

	// Format enables the downloading of secrets as a file (string)
	// +kubebuilder:validation:Enum=json;dotnet-json;env;yaml;docker
	// +optional
	Format string `json:"format,omitempty"`
}
