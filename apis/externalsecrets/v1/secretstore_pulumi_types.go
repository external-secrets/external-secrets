/*
Copyright © 2025 ESO Maintainer Team

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

// PulumiProvider defines configuration for accessing secrets from Pulumi ESC.
type PulumiProvider struct {
	// APIURL is the URL of the Pulumi API.
	// +kubebuilder:default="https://api.pulumi.com/api/esc"
	APIURL string `json:"apiUrl,omitempty"`

	// Auth configures how the Operator authenticates with the Pulumi API.
	// Either auth or the deprecated accessToken field must be specified.
	// +optional
	Auth *PulumiAuth `json:"auth,omitempty"`

	// Organization are a space to collaborate on shared projects and stacks.
	// To create a new organization, visit https://app.pulumi.com/ and click "New Organization".
	Organization string `json:"organization"`

	// Project is the name of the Pulumi ESC project the environment belongs to.
	Project string `json:"project"`
	// Environment are YAML documents composed of static key-value pairs, programmatic expressions,
	// dynamically retrieved values from supported providers including all major clouds,
	// and other Pulumi ESC environments.
	// To create a new environment, visit https://www.pulumi.com/docs/esc/environments/ for more information.
	Environment string `json:"environment"`

	// AccessToken is the access tokens to sign in to the Pulumi Cloud Console.
	// Deprecated: Use auth.accessToken instead.
	// +optional
	AccessToken *PulumiProviderSecretRef `json:"accessToken,omitempty"`
}

// PulumiAuth configures authentication with the Pulumi API.
// Exactly one of accessToken or oidcConfig must be specified.
// +kubebuilder:validation:XValidation:rule="(has(self.accessToken) && !has(self.oidcConfig)) || (!has(self.accessToken) && has(self.oidcConfig))",message="Exactly one of 'accessToken' or 'oidcConfig' must be specified"
type PulumiAuth struct {
	// AccessToken authenticates using a Pulumi access token stored in a Kubernetes Secret.
	// +optional
	AccessToken *PulumiProviderSecretRef `json:"accessToken,omitempty"`

	// OIDCConfig authenticates using Kubernetes ServiceAccount tokens via OIDC.
	// +optional
	OIDCConfig *PulumiOIDCAuth `json:"oidcConfig,omitempty"`
}

// PulumiProviderSecretRef contains the secret reference for Pulumi authentication.
type PulumiProviderSecretRef struct {
	// SecretRef is a reference to a secret containing the Pulumi API token.
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// PulumiOIDCAuth configures OIDC authentication with Pulumi using Kubernetes ServiceAccount tokens.
type PulumiOIDCAuth struct {
	// Organization is the name of the Pulumi organization configured for OIDC authentication.
	Organization string `json:"organization"`

	// ServiceAccountRef specifies the Kubernetes ServiceAccount to use for authentication.
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef"`

	// ExpirationSeconds sets the ServiceAccount token validity duration.
	// Defaults to 10 minutes.
	// +kubebuilder:default=600
	// +optional
	ExpirationSeconds *int64 `json:"expirationSeconds,omitempty"`
}
