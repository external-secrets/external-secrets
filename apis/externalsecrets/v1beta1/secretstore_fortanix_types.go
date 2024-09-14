// Copyright External Secrets Inc. All Rights Reserved
package v1beta1

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

type FortanixProvider struct {
	// APIURL is the URL of SDKMS API. Defaults to `sdkms.fortanix.com`.
	APIURL string `json:"apiUrl,omitempty"`

	// APIKey is the API token to access SDKMS Applications.
	APIKey *FortanixProviderSecretRef `json:"apiKey,omitempty"`
}

type FortanixProviderSecretRef struct {
	// SecretRef is a reference to a secret containing the SDKMS API Key.
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}
