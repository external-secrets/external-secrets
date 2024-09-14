//Copyright External Secrets Inc. All Rights Reserved

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
