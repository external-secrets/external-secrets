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

// YandexCertificateManagerProvider Configures a store to sync secrets using the Yandex Certificate Manager provider.
type YandexCertificateManagerProvider struct {
	// Yandex.Cloud API endpoint (e.g. 'api.cloud.yandex.net:443')
	// +optional
	APIEndpoint string `json:"apiEndpoint,omitempty"`

	// Auth defines the information necessary to authenticate against Yandex.Cloud
	Auth YandexAuth `json:"auth"`

	// The provider for the CA bundle to use to validate Yandex.Cloud server certificate.
	// +optional
	CAProvider *YandexCAProvider `json:"caProvider,omitempty"`

	// FetchingPolicy configures the provider to interpret the `data.secretKey.remoteRef.key` field in ExternalSecret as certificate ID or certificate name
	// +optional
	FetchingPolicy *FetchingPolicy `json:"fetching,omitempty"`
}
