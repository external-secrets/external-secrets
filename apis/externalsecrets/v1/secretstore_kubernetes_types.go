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

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// KubernetesServer defines configuration for connecting to a Kubernetes API server.
type KubernetesServer struct {
	// configures the Kubernetes server Address.
	// +kubebuilder:default=kubernetes.default
	// +optional
	URL string `json:"url,omitempty"`

	// CABundle is a base64-encoded CA certificate
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// see: https://external-secrets.io/v0.4.1/spec/#external-secrets.io/v1alpha1.CAProvider
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`
}

// KubernetesProvider configures a store to sync secrets with a Kubernetes instance.
type KubernetesProvider struct {
	// configures the Kubernetes server Address.
	// +optional
	Server KubernetesServer `json:"server,omitempty"`

	// Auth configures how secret-manager authenticates with a Kubernetes instance.
	// +optional
	Auth *KubernetesAuth `json:"auth,omitempty"`

	// A reference to a secret that contains the auth information.
	// +optional
	AuthRef *esmeta.SecretKeySelector `json:"authRef,omitempty"`

	// Remote namespace to fetch the secrets from
	// +optional
	// +kubebuilder:default=default
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	RemoteNamespace string `json:"remoteNamespace,omitempty"`
}

// KubernetesAuth defines authentication options for connecting to a Kubernetes cluster.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type KubernetesAuth struct {
	// has both clientCert and clientKey as secretKeySelector
	// +optional
	Cert *CertAuth `json:"cert,omitempty"`

	// use static token to authenticate with
	// +optional
	Token *TokenAuth `json:"token,omitempty"`

	// points to a service account that should be used for authentication
	// +optional
	ServiceAccount *esmeta.ServiceAccountSelector `json:"serviceAccount,omitempty"`
}

// CertAuth defines certificate-based authentication configuration for Kubernetes.
type CertAuth struct {
	ClientCert esmeta.SecretKeySelector `json:"clientCert,omitempty"`
	ClientKey  esmeta.SecretKeySelector `json:"clientKey,omitempty"`
}

// TokenAuth defines token-based authentication configuration for Kubernetes.
type TokenAuth struct {
	BearerToken esmeta.SecretKeySelector `json:"bearerToken,omitempty"`
}
