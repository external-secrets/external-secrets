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

type OnboardbaseAuth struct {
	// The OnboardbaseToken is used for authentication.
	// See https://docs.doppler.com/reference/api#authentication for auth token types.
	// The Key attribute defaults to dopplerToken if not specified.
	OnboardbaseAPIKey esmeta.SecretKeySelector `json:"onboardbaseAPIKey"`
	OnboardbasePasscode esmeta.SecretKeySelector `json:"onboardbasePasscode"`
}

// OnboardbaseProvider configures a store to sync secrets using the Onboardbase provider.
// Project and Config are required if not using a Service Token.
type OnboardbaseProvider struct {
	// Auth configures how the Operator authenticates with the Onboardbase API
	Auth *OnboardbaseAuth `json:"auth"`

	Environment string `json:"onboardbaseEnvironment"`
	Project string `json:"onboardbaseProject"`
}
