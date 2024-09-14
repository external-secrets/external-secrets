//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Configures a store to sync secrets with a Password Depot instance.
type PasswordDepotProvider struct {
	// URL configures the Password Depot instance URL.
	Host string `json:"host"`

	// Database to use as source
	Database string `json:"database"`

	// Auth configures how secret-manager authenticates with a Password Depot instance.
	Auth PasswordDepotAuth `json:"auth"`
}

type PasswordDepotAuth struct {
	SecretRef PasswordDepotSecretRef `json:"secretRef"`
}

type PasswordDepotSecretRef struct {
	// Username / Password is used for authentication.
	// +optional
	Credentials esmeta.SecretKeySelector `json:"credentials,omitempty"`
}
