//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type YandexLockboxAuth struct {
	// The authorized key used for authentication
	// +optional
	AuthorizedKey esmeta.SecretKeySelector `json:"authorizedKeySecretRef,omitempty"`
}

type YandexLockboxCAProvider struct {
	Certificate esmeta.SecretKeySelector `json:"certSecretRef,omitempty"`
}

// YandexLockboxProvider Configures a store to sync secrets using the Yandex Lockbox provider.
type YandexLockboxProvider struct {
	// Yandex.Cloud API endpoint (e.g. 'api.cloud.yandex.net:443')
	// +optional
	APIEndpoint string `json:"apiEndpoint,omitempty"`

	// Auth defines the information necessary to authenticate against Yandex Lockbox
	Auth YandexLockboxAuth `json:"auth"`

	// The provider for the CA bundle to use to validate Yandex.Cloud server certificate.
	// +optional
	CAProvider *YandexLockboxCAProvider `json:"caProvider,omitempty"`
}
