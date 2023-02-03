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

type ScalewayProviderSecretRef struct {

	// Value can be specified directly to set a value without using a secret.
	// +optional
	Value string `json:"value,omitempty"`

	// SecretNamespace is the namespace of the secret to use. If not specified, default
	// to the namespace of the SecretStore.
	// +optional
	SecretNamespace string `json:"secretNamespace,omitempty"`

	// SecretName is the name of the secret to use.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// SecretKey is the specific key in the secret to be used.
	// +optional
	SecretKey string `json:"secretKey,omitempty"`
}

type ScalewayProvider struct {

	// ApiUrl is the url of the api to use. Defaults to https://api.scaleway.com
	// +optional
	ApiUrl string `json:"apiUrl,omitempty"`

	Region string `json:"region"`

	ProjectId string `json:"projectId"`

	// AccessKey is the non-secret par of the api key.
	AccessKey *ScalewayProviderSecretRef `json:"accessKey"`

	// SecretKey is the non-secret par of the api key.
	SecretKey *ScalewayProviderSecretRef `json:"secretKey"`
}
