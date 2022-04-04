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

package v1alpha1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type VaultKVStoreVersion string

const (
	VaultKVStoreV1 VaultKVStoreVersion = "v1"
	VaultKVStoreV2 VaultKVStoreVersion = "v2"
)

type CAProviderType string

const (
	CAProviderTypeSecret    CAProviderType = "Secret"
	CAProviderTypeConfigMap CAProviderType = "ConfigMap"
)

// Defines a location to fetch the cert for the vault provider from.
type CAProvider struct {
	// The type of provider to use such as "Secret", or "ConfigMap".
	// +kubebuilder:validation:Enum="Secret";"ConfigMap"
	Type CAProviderType `json:"type"`

	// The name of the object located at the provider type.
	Name string `json:"name"`

	// The key the value inside of the provider type to use, only used with "Secret" type
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`

	// The namespace the Provider type is in.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

// Configures an store to sync secrets using a HashiCorp Vault
// KV backend.
type VaultProvider struct {
	// Auth configures how secret-manager authenticates with the Vault server.
	Auth VaultAuth `json:"auth"`

	// Server is the connection address for the Vault server, e.g: "https://vault.example.com:8200".
	Server string `json:"server"`

	// Path is the mount path of the Vault KV backend endpoint, e.g:
	// "secret". The v2 KV secret engine version specific "/data" path suffix
	// for fetching secrets from Vault is optional and will be appended
	// if not present in specified path.
	// +optional
	Path *string `json:"path"`

	// Version is the Vault KV secret engine version. This can be either "v1" or
	// "v2". Version defaults to "v2".
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum="v1";"v2"
	// +kubebuilder:default:="v2"
	Version VaultKVStoreVersion `json:"version"`

	// Name of the vault namespace. Namespaces is a set of features within Vault Enterprise that allows
	// Vault environments to support Secure Multi-tenancy. e.g: "ns1".
	// More about namespaces can be found here https://www.vaultproject.io/docs/enterprise/namespaces
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// PEM encoded CA bundle used to validate Vault server certificate. Only used
	// if the Server URL is using HTTPS protocol. This parameter is ignored for
	// plain HTTP protocol connection. If not set the system root certificates
	// are used to validate the TLS connection.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// The provider for the CA bundle to use to validate Vault server certificate.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`

	// ReadYourWrites ensures isolated read-after-write semantics by
	// providing discovered cluster replication states in each request.
	// More information about eventual consistency in Vault can be found here
	// https://www.vaultproject.io/docs/enterprise/consistency
	// +optional
	ReadYourWrites bool `json:"readYourWrites,omitempty"`

	// ForwardInconsistent tells Vault to forward read-after-write requests to the Vault
	// leader instead of simply retrying within a loop. This can increase performance if
	// the option is enabled serverside.
	// https://www.vaultproject.io/docs/configuration/replication#allow_forwarding_via_header
	// +optional
	ForwardInconsistent bool `json:"forwardInconsistent,omitempty"`
}

// VaultAuth is the configuration used to authenticate with a Vault server.
// Only one of `tokenSecretRef`, `appRole`,  `kubernetes`, `ldap`, `jwt` or `cert`
// can be specified.
type VaultAuth struct {
	// TokenSecretRef authenticates with Vault by presenting a token.
	// +optional
	TokenSecretRef *esmeta.SecretKeySelector `json:"tokenSecretRef,omitempty"`

	// AppRole authenticates with Vault using the App Role auth mechanism,
	// with the role and secret stored in a Kubernetes Secret resource.
	// +optional
	AppRole *VaultAppRole `json:"appRole,omitempty"`

	// Kubernetes authenticates with Vault by passing the ServiceAccount
	// token stored in the named Secret resource to the Vault server.
	// +optional
	Kubernetes *VaultKubernetesAuth `json:"kubernetes,omitempty"`

	// Ldap authenticates with Vault by passing username/password pair using
	// the LDAP authentication method
	// +optional
	Ldap *VaultLdapAuth `json:"ldap,omitempty"`

	// Jwt authenticates with Vault by passing role and JWT token using the
	// JWT/OIDC authentication method
	// +optional
	Jwt *VaultJwtAuth `json:"jwt,omitempty"`

	// Cert authenticates with TLS Certificates by passing client certificate, private key and ca certificate
	// Cert authentication method
	// +optional
	Cert *VaultCertAuth `json:"cert,omitempty"`
}

// VaultAppRole authenticates with Vault using the App Role auth mechanism,
// with the role and secret stored in a Kubernetes Secret resource.
type VaultAppRole struct {
	// Path where the App Role authentication backend is mounted
	// in Vault, e.g: "approle"
	// +kubebuilder:default=approle
	Path string `json:"path"`

	// RoleID configured in the App Role authentication backend when setting
	// up the authentication backend in Vault.
	RoleID string `json:"roleId"`

	// Reference to a key in a Secret that contains the App Role secret used
	// to authenticate with Vault.
	// The `key` field must be specified and denotes which entry within the Secret
	// resource is used as the app role secret.
	SecretRef esmeta.SecretKeySelector `json:"secretRef"`
}

// Authenticate against Vault using a Kubernetes ServiceAccount token stored in
// a Secret.
type VaultKubernetesAuth struct {
	// Path where the Kubernetes authentication backend is mounted in Vault, e.g:
	// "kubernetes"
	// +kubebuilder:default=kubernetes
	Path string `json:"mountPath"`

	// Optional service account field containing the name of a kubernetes ServiceAccount.
	// If the service account is specified, the service account secret token JWT will be used
	// for authenticating with Vault. If the service account selector is not supplied,
	// the secretRef will be used instead.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`

	// Optional secret field containing a Kubernetes ServiceAccount JWT used
	// for authenticating with Vault. If a name is specified without a key,
	// `token` is the default. If one is not specified, the one bound to
	// the controller will be used.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`

	// A required field containing the Vault Role to assume. A Role binds a
	// Kubernetes ServiceAccount with a set of Vault policies.
	Role string `json:"role"`
}

// VaultLdapAuth authenticates with Vault using the LDAP authentication method,
// with the username and password stored in a Kubernetes Secret resource.
type VaultLdapAuth struct {
	// Path where the LDAP authentication backend is mounted
	// in Vault, e.g: "ldap"
	// +kubebuilder:default=ldap
	Path string `json:"path"`

	// Username is a LDAP user name used to authenticate using the LDAP Vault
	// authentication method
	Username string `json:"username"`

	// SecretRef to a key in a Secret resource containing password for the LDAP
	// user used to authenticate with Vault using the LDAP authentication
	// method
	SecretRef esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// VaultKubernetesServiceAccountTokenAuth authenticates with Vault using a temporary
// Kubernetes service account token retrieved by the `TokenRequest` API.
type VaultKubernetesServiceAccountTokenAuth struct {
	// Service account field containing the name of a kubernetes ServiceAccount.
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef"`

	// Optional audiences field that will be used to request a temporary Kubernetes service
	// account token for the service account referenced by `serviceAccountRef`.
	// Defaults to a single audience `vault` it not specified.
	// +optional
	Audiences *[]string `json:"audiences,omitempty"`

	// Optional expiration time in seconds that will be used to request a temporary
	// Kubernetes service account token for the service account referenced by
	// `serviceAccountRef`.
	// Defaults to 10 minutes.
	// +optional
	ExpirationSeconds *int64 `json:"expirationSeconds,omitempty"`
}

// VaultJwtAuth authenticates with Vault using the JWT/OIDC authentication
// method, with the role name and a token stored in a Kubernetes Secret resource or
// a Kubernetes service account token retrieved via `TokenRequest`.
type VaultJwtAuth struct {
	// Path where the JWT authentication backend is mounted
	// in Vault, e.g: "jwt"
	// +kubebuilder:default=jwt
	Path string `json:"path"`

	// Role is a JWT role to authenticate using the JWT/OIDC Vault
	// authentication method
	// +optional
	Role string `json:"role"`

	// Optional SecretRef that refers to a key in a Secret resource containing JWT token to
	// authenticate with Vault using the JWT/OIDC authentication method.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`

	// Optional ServiceAccountToken specifies the Kubernetes service account for which to request
	// a token for with the `TokenRequest` API.
	// +optional
	KubernetesServiceAccountToken *VaultKubernetesServiceAccountTokenAuth `json:"kubernetesServiceAccountToken,omitempty"`
}

// VaultJwtAuth authenticates with Vault using the JWT/OIDC authentication
// method, with the role name and token stored in a Kubernetes Secret resource.
type VaultCertAuth struct {
	// ClientCert is a certificate to authenticate using the Cert Vault
	// authentication method
	// +optional
	ClientCert esmeta.SecretKeySelector `json:"clientCert,omitempty"`

	// SecretRef to a key in a Secret resource containing client private key to
	// authenticate with Vault using the Cert authentication method
	SecretRef esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}
