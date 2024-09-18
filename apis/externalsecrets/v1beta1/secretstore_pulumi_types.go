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

type PulumiProvider struct {
	// APIURL is the URL of the Pulumi API.
	// +kubebuilder:default="https://api.pulumi.com/api/preview"
	APIURL string `json:"apiUrl,omitempty"`

	// AccessToken is the access tokens to sign in to the Pulumi Cloud Console.
	AccessToken *PulumiProviderSecretRef `json:"accessToken"`

	// Organization are a space to collaborate on shared projects and stacks.
	// To create a new organization, visit https://app.pulumi.com/ and click "New Organization".
	Organization string `json:"organization"`

	// Environment are YAML documents composed of static key-value pairs, programmatic expressions,
	// dynamically retrieved values from supported providers including all major clouds,
	// and other Pulumi ESC environments.
	// To create a new environment, visit https://www.pulumi.com/docs/esc/environments/ for more information.
	Environment string `json:"environment"`
}

type PulumiProviderSecretRef struct {
	// SecretRef is a reference to a secret containing the Pulumi API token.
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}
