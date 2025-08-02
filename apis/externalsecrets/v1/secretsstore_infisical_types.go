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

package v1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type UniversalAuthCredentials struct {
	// +kubebuilder:validation:Required
	ClientID esmeta.SecretKeySelector `json:"clientId"`
	// +kubebuilder:validation:Required
	ClientSecret esmeta.SecretKeySelector `json:"clientSecret"`
}

type AzureAuthCredentials struct {
	// +kubebuilder:validation:Required
	IdentityID esmeta.SecretKeySelector `json:"identityId"`
	// +optional
	Resource esmeta.SecretKeySelector `json:"resource"`
}

type GcpIdTokenAuthCredentials struct {
	// +kubebuilder:validation:Required
	IdentityID esmeta.SecretKeySelector `json:"identityId"`
}

type GcpIamAuthCredentials struct {
	// +kubebuilder:validation:Required
	IdentityID esmeta.SecretKeySelector `json:"identityId"`
	// +kubebuilder:validation:Required
	ServiceAccountKeyFilePath esmeta.SecretKeySelector `json:"serviceAccountKeyFilePath"`
}

type JwtAuthCredentials struct {
	// +kubebuilder:validation:Required
	IdentityID esmeta.SecretKeySelector `json:"identityId"`
	// +kubebuilder:validation:Required
	JWT esmeta.SecretKeySelector `json:"jwt"`
}

type LdapAuthCredentials struct {
	// +kubebuilder:validation:Required
	IdentityID esmeta.SecretKeySelector `json:"identityId"`
	// +kubebuilder:validation:Required
	LDAPPassword esmeta.SecretKeySelector `json:"ldapPassword"`
	// +kubebuilder:validation:Required
	LDAPUsername esmeta.SecretKeySelector `json:"ldapUsername"`
}

type OciAuthCredentials struct {
	// +kubebuilder:validation:Required
	IdentityID esmeta.SecretKeySelector `json:"identityId"`
	// +kubebuilder:validation:Required
	PrivateKey esmeta.SecretKeySelector `json:"privateKey"`
	// +optional
	PrivateKeyPassphrase esmeta.SecretKeySelector `json:"privateKeyPassphrase"`
	// +kubebuilder:validation:Required
	Fingerprint esmeta.SecretKeySelector `json:"fingerprint"`
	// +kubebuilder:validation:Required
	UserID esmeta.SecretKeySelector `json:"userId"`
	// +kubebuilder:validation:Required
	TenancyID esmeta.SecretKeySelector `json:"tenancyId"`
	// +kubebuilder:validation:Required
	Region esmeta.SecretKeySelector `json:"region"`
}

type KubernetesAuthCredentials struct {
	// +kubebuilder:validation:Required
	IdentityID esmeta.SecretKeySelector `json:"identityId"`
	// +optional
	ServiceAccountTokenPath esmeta.SecretKeySelector `json:"serviceAccountTokenPath"`
}

type AwsAuthCredentials struct {
	// +kubebuilder:validation:Required
	IdentityID esmeta.SecretKeySelector `json:"identityId"`
}

type TokenAuthCredentials struct {
	// +kubebuilder:validation:Required
	AccessToken esmeta.SecretKeySelector `json:"accessToken"`
}

type InfisicalAuth struct {
	// +optional
	UniversalAuthCredentials *UniversalAuthCredentials `json:"universalAuthCredentials,omitempty"`
	// +optional
	AzureAuthCredentials *AzureAuthCredentials `json:"azureAuthCredentials,omitempty"`
	// +optional
	GcpIdTokenAuthCredentials *GcpIdTokenAuthCredentials `json:"gcpIdTokenAuthCredentials,omitempty"`
	// +optional
	GcpIamAuthCredentials *GcpIamAuthCredentials `json:"gcpIamAuthCredentials,omitempty"`
	// +optional
	JwtAuthCredentials *JwtAuthCredentials `json:"jwtAuthCredentials,omitempty"`
	// +optional
	LdapAuthCredentials *LdapAuthCredentials `json:"ldapAuthCredentials,omitempty"`
	// +optional
	OciAuthCredentials *OciAuthCredentials `json:"ociAuthCredentials,omitempty"`
	// +optional
	KubernetesAuthCredentials *KubernetesAuthCredentials `json:"kubernetesAuthCredentials,omitempty"`
	// +optional
	AwsAuthCredentials *AwsAuthCredentials `json:"awsAuthCredentials,omitempty"`
	// +optional
	TokenAuthCredentials *TokenAuthCredentials `json:"tokenAuthCredentials,omitempty"`
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
