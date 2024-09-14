//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Device42Provider configures a store to sync secrets with a Device42 instance.
type Device42Provider struct {
	// URL configures the Device42 instance URL.
	Host string `json:"host"`

	// Auth configures how secret-manager authenticates with a Device42 instance.
	Auth Device42Auth `json:"auth"`
}

type Device42Auth struct {
	SecretRef Device42SecretRef `json:"secretRef"`
}

type Device42SecretRef struct {
	// Username / Password is used for authentication.
	// +optional
	Credentials esmeta.SecretKeySelector `json:"credentials,omitempty"`
}
