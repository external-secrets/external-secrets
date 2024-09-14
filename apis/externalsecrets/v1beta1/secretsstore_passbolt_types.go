//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Passbolt contains a secretRef for the passbolt credentials.
type PassboltAuth struct {
	PasswordSecretRef   *esmeta.SecretKeySelector `json:"passwordSecretRef"`
	PrivateKeySecretRef *esmeta.SecretKeySelector `json:"privateKeySecretRef"`
}

type PassboltProvider struct {
	// Auth defines the information necessary to authenticate against Passbolt Server
	Auth *PassboltAuth `json:"auth"`
	// Host defines the Passbolt Server to connect to
	Host string `json:"host"`
}
