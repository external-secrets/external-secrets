//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

type SecretServerProviderRef struct {

	// Value can be specified directly to set a value without using a secret.
	// +optional
	Value string `json:"value,omitempty"`

	// SecretRef references a key in a secret that will be used as value.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// See https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go.
type SecretServerProvider struct {

	// Username is the secret server account username.
	// +required
	Username *SecretServerProviderRef `json:"username"`

	// Password is the secret server account password.
	// +required
	Password *SecretServerProviderRef `json:"password"`

	// ServerURL
	// URL to your secret server installation
	// +required
	ServerURL string `json:"serverURL"`
}
