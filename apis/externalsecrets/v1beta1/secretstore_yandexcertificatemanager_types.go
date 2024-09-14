//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type YandexCertificateManagerAuth struct {
	// The authorized key used for authentication
	// +optional
	AuthorizedKey esmeta.SecretKeySelector `json:"authorizedKeySecretRef,omitempty"`
}

type YandexCertificateManagerCAProvider struct {
	Certificate esmeta.SecretKeySelector `json:"certSecretRef,omitempty"`
}

// YandexCertificateManagerProvider Configures a store to sync secrets using the Yandex Certificate Manager provider.
type YandexCertificateManagerProvider struct {
	// Yandex.Cloud API endpoint (e.g. 'api.cloud.yandex.net:443')
	// +optional
	APIEndpoint string `json:"apiEndpoint,omitempty"`

	// Auth defines the information necessary to authenticate against Yandex Certificate Manager
	Auth YandexCertificateManagerAuth `json:"auth"`

	// The provider for the CA bundle to use to validate Yandex.Cloud server certificate.
	// +optional
	CAProvider *YandexCertificateManagerCAProvider `json:"caProvider,omitempty"`
}
