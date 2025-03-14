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
	// SecretsPath specifies the path to the secrets within the workspace. Defaults to "/" if not provided.
	// +kubebuilder:default="/"
	// +optional
	SecretsPath string `json:"secretsPath,omitempty"`
	// Recursive indicates whether the secrets should be fetched recursively. Defaults to false if not provided.
	// +kubebuilder:default=false
	// +optional
	Recursive bool `json:"recursive,omitempty"`
	// EnvironmentSlug is the required slug identifier for the environment.
	// +kubebuilder:validation:Required
	EnvironmentSlug string `json:"environmentSlug"`
	// ProjectSlug is the required slug identifier for the project.
	// +kubebuilder:validation:Required
	ProjectSlug string `json:"projectSlug"`
	// ExpandSecretReferences indicates whether secret references should be expanded. Defaults to true if not provided.
	// +kubebuilder:default=true
	// +optional
	ExpandSecretReferences bool `json:"expandSecretReferences,omitempty"`
}

// InfisicalProvider configures a store to sync secrets using the Infisical provider.
type InfisicalProvider struct {
	// Auth configures how the Operator authenticates with the Infisical API
	// +kubebuilder:validation:Required
	Auth InfisicalAuth `json:"auth"`
	// SecretsScope defines the scope of the secrets within the workspace
	// +kubebuilder:validation:Required
	SecretsScope MachineIdentityScopeInWorkspace `json:"secretsScope"`
	// HostAPI specifies the base URL of the Infisical API. If not provided, it defaults to "https://app.infisical.com/api".
	// +kubebuilder:default="https://app.infisical.com/api"
	// +optional
	HostAPI string `json:"hostAPI,omitempty"`
}
