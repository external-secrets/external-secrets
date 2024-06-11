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

type UniversalAuthCredentials struct {
	// +kubebuilder:validation:Required
	ClientID esmeta.SecretKeySelector `json:"clientId"`
	// +kubebuilder:validation:Required
	ClientSecret esmeta.SecretKeySelector `json:"clientSecret"`
}

type InfisicalAuth struct {
	// +optional
	UniversalAuthCredentials *UniversalAuthCredentials `json:"universalAuthCredentials,omitempty"`
}

type MachineIdentityScopeInWorkspace struct {
	// +kubebuilder:default="/"
	// +optional
	SecretsPath string `json:"secretsPath,omitempty"`
	// +kubebuilder:validation:Required
	EnvironmentSlug string `json:"environmentSlug"`
	// +kubebuilder:validation:Required
	ProjectSlug string `json:"projectSlug"`
}

// InfisicalProvider configures a store to sync secrets using the Infisical provider.
type InfisicalProvider struct {
	// Auth configures how the Operator authenticates with the Infisical API
	// +kubebuilder:validation:Required
	Auth InfisicalAuth `json:"auth"`
	// +kubebuilder:validation:Required
	SecretsScope MachineIdentityScopeInWorkspace `json:"secretsScope"`
	// +kubebuilder:default="https://app.infisical.com/api"
	// +optional
	HostAPI string `json:"hostAPI,omitempty"`
}
