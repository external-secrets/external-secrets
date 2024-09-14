//Copyright External Secrets Inc. All Rights Reserved

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
