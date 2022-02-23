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
