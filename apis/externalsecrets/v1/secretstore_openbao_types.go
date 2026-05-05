/*
Copyright © The ESO Authors

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

// OpenBaoKVStoreVersion represents the version of the OpenBao KV secret engine.
type OpenBaoKVStoreVersion string

// These are the currently supported OpenBaoKVStoreVersion.
const (
	OpenBaoKVStoreV1 OpenBaoKVStoreVersion = "v1"
	OpenBaoKVStoreV2 OpenBaoKVStoreVersion = "v2"
)

// OpenBaoProvider configures a store to sync secrets using an OpenBao KV backend.
type OpenBaoProvider struct {
	// Auth configures how secret-manager authenticates with the OpenBao server.
	Auth *OpenBaoAuth `json:"auth,omitempty"`

	// Server is the connection address for the OpenBao server, e.g: "https://openbao.example.com:8200".
	Server string `json:"server"`

	// Path is the mount path of the OpenBao KV backend endpoint, e.g:
	// "secret". The v2 KV secret engine version specific "/data" path suffix
	// for fetching secrets from OpenBao is optional and will be appended
	// if not present in specified path.
	// +optional
	Path *string `json:"path,omitempty"`

	// Version is the OpenBao KV secret engine version. This can be either "v1" or
	// "v2". Version defaults to "v2".
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum="v1";"v2"
	// +kubebuilder:default:="v2"
	Version OpenBaoKVStoreVersion `json:"version"`

	// Name of the OpenBao namespace. Namespaces is a set of features within OpenBao that allows
	// OpenBao environments to support Secure Multi-tenancy. e.g: "ns1".
	// More about namespaces can be found here https://openbao.org/docs/concepts/namespaces/
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// PEM encoded CA bundle used to validate OpenBao server certificate. Only used
	// if the Server URL is using HTTPS protocol. This parameter is ignored for
	// plain HTTP protocol connection. If not set the system root certificates
	// are used to validate the TLS connection.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// The configuration used for client side related TLS communication, when the OpenBao server
	// requires mutual authentication. Only used if the Server URL is using HTTPS protocol.
	// This parameter is ignored for plain HTTP protocol connection.
	// It's worth noting this configuration is different from the "TLS certificates auth method",
	// which is available under the `auth.cert` section.
	// +optional
	ClientTLS OpenBaoClientTLS `json:"tls,omitempty"`

	// The provider for the CA bundle to use to validate OpenBao server certificate.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`

	// Headers to be added in OpenBao request
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// CheckAndSet defines the Check-And-Set (CAS) settings for PushSecret operations.
	// Only applies to OpenBao KV v2 stores. When enabled, write operations must include
	// the current version of the secret to prevent unintentional overwrites.
	// +optional
	CheckAndSet *OpenBaoCheckAndSet `json:"checkAndSet,omitempty"`
}

// OpenBaoClientTLS is the configuration used for client side related TLS communication,
// when the OpenBao server requires mutual authentication.
type OpenBaoClientTLS struct {
	// CertSecretRef is a certificate added to the transport layer
	// when communicating with the OpenBao server.
	// If no key for the Secret is specified, external-secret will default to 'tls.crt'.
	// +optional
	CertSecretRef *esmeta.SecretKeySelector `json:"certSecretRef,omitempty"`

	// KeySecretRef to a key in a Secret resource containing client private key
	// added to the transport layer when communicating with the OpenBao server.
	// If no key for the Secret is specified, external-secret will default to 'tls.key'.
	// +optional
	KeySecretRef *esmeta.SecretKeySelector `json:"keySecretRef,omitempty"`
}

// OpenBaoAuth is the configuration used to authenticate with an OpenBao server.
// Only one of `tokenSecretRef`, `appRole`,  `kubernetes`, `ldap`, `userPass`, `jwt`, `cert`, `iam` or `gcp`
// can be specified. A namespace to authenticate against can optionally be specified.
type OpenBaoAuth struct {
	// Name of the OpenBao namespace to authenticate to. This can be different than the namespace your secret is in.
	// Namespaces is a set of features within OpenBao that allows
	// OpenBao environments to support Secure Multi-tenancy. e.g: "ns1".
	// More about namespaces can be found here https://openbao.org/docs/concepts/namespaces/
	// This will default to OpenBao.Namespace field if set, or empty otherwise
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// TokenSecretRef authenticates with OpenBao by presenting a token.
	// +optional
	TokenSecretRef *esmeta.SecretKeySelector `json:"tokenSecretRef,omitempty"`

	// AppRole authenticates with OpenBao using the App Role auth mechanism,
	// with the role and secret stored in a Kubernetes Secret resource.
	// +optional
	AppRole *OpenBaoAppRole `json:"appRole,omitempty"`

	// Kubernetes authenticates with OpenBao by passing the ServiceAccount
	// token stored in the named Secret resource to the OpenBao server.
	// +optional
	Kubernetes *OpenBaoKubernetesAuth `json:"kubernetes,omitempty"`

	// Ldap authenticates with OpenBao by passing username/password pair using
	// the LDAP authentication method
	// +optional
	Ldap *OpenBaoLdapAuth `json:"ldap,omitempty"`

	// Jwt authenticates with OpenBao by passing role and JWT token using the
	// JWT/OIDC authentication method
	// +optional
	Jwt *OpenBaoJwtAuth `json:"jwt,omitempty"`

	// Cert authenticates with TLS Certificates by passing client certificate, private key and ca certificate
	// Cert authentication method
	// +optional
	Cert *OpenBaoCertAuth `json:"cert,omitempty"`

	// Iam authenticates with OpenBao by passing a special AWS request signed with AWS IAM credentials
	// AWS IAM authentication method
	// +optional
	Iam *OpenBaoIamAuth `json:"iam,omitempty"`

	// UserPass authenticates with OpenBao by passing username/password pair
	// +optional
	UserPass *OpenBaoUserPassAuth `json:"userPass,omitempty"`

	// Gcp authenticates with OpenBao using Google Cloud Platform authentication method
	// GCP authentication method
	// +optional
	GCP *OpenBaoGCPAuth `json:"gcp,omitempty"`
}

// OpenBaoAppRole authenticates with OpenBao using the App Role auth mechanism,
// with the role and secret stored in a Kubernetes Secret resource.
type OpenBaoAppRole struct {
	// Path where the App Role authentication backend is mounted
	// in OpenBao, e.g: "approle"
	// +kubebuilder:default=approle
	Path string `json:"path"`

	// RoleID configured in the App Role authentication backend when setting
	// up the authentication backend in OpenBao.
	//+optional
	RoleID string `json:"roleId,omitempty"`

	// Reference to a key in a Secret that contains the App Role ID used
	// to authenticate with OpenBao.
	// The `key` field must be specified and denotes which entry within the Secret
	// resource is used as the app role id.
	//+optional
	RoleRef *esmeta.SecretKeySelector `json:"roleRef,omitempty"`

	// Reference to a key in a Secret that contains the App Role secret used
	// to authenticate with OpenBao.
	// The `key` field must be specified and denotes which entry within the Secret
	// resource is used as the app role secret.
	SecretRef esmeta.SecretKeySelector `json:"secretRef"`
}

// OpenBaoKubernetesAuth authenticates against OpenBao using a Kubernetes ServiceAccount token stored in
// a Secret.
type OpenBaoKubernetesAuth struct {
	// Path where the Kubernetes authentication backend is mounted in OpenBao, e.g:
	// "kubernetes"
	// +kubebuilder:default=kubernetes
	Path string `json:"mountPath"`

	// Optional service account field containing the name of a kubernetes ServiceAccount.
	// If the service account is specified, the service account secret token JWT will be used
	// for authenticating with OpenBao. If the service account selector is not supplied,
	// the secretRef will be used instead.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`

	// Optional secret field containing a Kubernetes ServiceAccount JWT used
	// for authenticating with OpenBao. If a name is specified without a key,
	// `token` is the default. If one is not specified, the one bound to
	// the controller will be used.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`

	// A required field containing the OpenBao Role to assume. A Role binds a
	// Kubernetes ServiceAccount with a set of OpenBao policies.
	Role string `json:"role"`
}

// OpenBaoLdapAuth authenticates with OpenBao using the LDAP authentication method,
// with the username and password stored in a Kubernetes Secret resource.
type OpenBaoLdapAuth struct {
	// Path where the LDAP authentication backend is mounted
	// in OpenBao, e.g: "ldap"
	// +kubebuilder:default=ldap
	Path string `json:"path"`

	// Username is an LDAP username used to authenticate using the LDAP OpenBao
	// authentication method
	Username string `json:"username"`

	// SecretRef to a key in a Secret resource containing password for the LDAP
	// user used to authenticate with OpenBao using the LDAP authentication
	// method
	// +optional
	SecretRef esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// OpenBaoAwsAuth tells the controller how to do authentication with aws.
// Only one of secretRef or jwt can be specified.
// if none is specified the controller will try to load credentials from its own service account assuming it is IRSA enabled.
type OpenBaoAwsAuth struct {
	// +optional
	SecretRef *OpenBaoAwsAuthSecretRef `json:"secretRef,omitempty"`
	// +optional
	JWTAuth *OpenBaoAwsJWTAuth `json:"jwt,omitempty"`
}

// OpenBaoAwsAuthSecretRef holds secret references for AWS credentials
// both AccessKeyID and SecretAccessKey must be defined in order to properly authenticate.
type OpenBaoAwsAuthSecretRef struct {
	// The AccessKeyID is used for authentication
	// +optional
	AccessKeyID esmeta.SecretKeySelector `json:"accessKeyIDSecretRef,omitempty"`

	// The SecretAccessKey is used for authentication
	// +optional
	SecretAccessKey esmeta.SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`

	// The SessionToken used for authentication
	// This must be defined if AccessKeyID and SecretAccessKey are temporary credentials
	// see: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_use-resources.html
	// +optional
	SessionToken *esmeta.SecretKeySelector `json:"sessionTokenSecretRef,omitempty"`
}

// OpenBaoAwsJWTAuth Authenticate against AWS using service account tokens.
type OpenBaoAwsJWTAuth struct {
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

// OpenBaoJwtAuth authenticates with OpenBao using the JWT/OIDC authentication
// method, with the role name and a token stored in a Kubernetes Secret resource or
// a Kubernetes service account token retrieved via `TokenRequest`.
type OpenBaoJwtAuth struct {
	// Path where the JWT authentication backend is mounted
	// in OpenBao, e.g: "jwt"
	// +kubebuilder:default=jwt
	Path string `json:"path"`

	// Role is a JWT role to authenticate using the JWT/OIDC OpenBao
	// authentication method
	// +optional
	Role string `json:"role,omitempty"`

	// Optional SecretRef that refers to a key in a Secret resource containing JWT token to
	// authenticate with OpenBao using the JWT/OIDC authentication method.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`

	// Optional ServiceAccountRef specifies the Kubernetes service account for which to request
	// a token for with the `TokenRequest` API.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

// OpenBaoCertAuth authenticates with OpenBao using the JWT/OIDC authentication
// method, with the role name and token stored in a Kubernetes Secret resource.
type OpenBaoCertAuth struct {
	// Path where the Certificate authentication backend is mounted
	// in OpenBao, e.g: "cert"
	// +kubebuilder:default=cert
	// +optional
	Path string `json:"path"`

	// OpenBaoRole specifies the OpenBao role to use for TLS certificate authentication.
	// +optional
	OpenBaoRole string `json:"openBaoRole,omitempty"`

	// ClientCert is a certificate to authenticate using the Cert OpenBao
	// authentication method
	// +optional
	ClientCert esmeta.SecretKeySelector `json:"clientCert,omitempty"`

	// SecretRef to a key in a Secret resource containing client private key to
	// authenticate with OpenBao using the Cert authentication method
	// +optional
	SecretRef esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// OpenBaoIamAuth authenticates with OpenBao using the OpenBao's AWS IAM authentication method.
//
// When JWTAuth and SecretRef are not specified, the provider will use the controller pod's
// identity to authenticate with AWS. This supports both IRSA and EKS Pod Identity.
type OpenBaoIamAuth struct {
	// Path where the AWS auth method is enabled in OpenBao, e.g: "aws"
	// +optional
	Path string `json:"path,omitempty"`
	// AWS region
	// +optional
	Region string `json:"region,omitempty"`
	// This is the AWS role to be assumed before talking to OpenBao
	// +optional
	AWSIAMRole string `json:"role,omitempty"`
	// OpenBao Role. In OpenBao, a role describes an identity with a set of permissions, groups, or policies you want to attach a user of the secrets engine
	Role string `json:"openBaoRole"`
	// AWS External ID set on assumed IAM roles
	ExternalID string `json:"externalID,omitempty"`
	// X-Vault-AWS-IAM-Server-ID is an additional header used by OpenBao IAM auth method to mitigate against different types of replay attacks.
	// +optional
	VaultAWSIAMServerID string `json:"vaultAwsIamServerID,omitempty"`
	// Specify credentials in a Secret object
	// +optional
	SecretRef *OpenBaoAwsAuthSecretRef `json:"secretRef,omitempty"`
	// Specify a service account with IRSA enabled
	// +optional
	JWTAuth *OpenBaoAwsJWTAuth `json:"jwt,omitempty"`
}

// OpenBaoUserPassAuth authenticates with OpenBao using UserPass authentication method,
// with the username and password stored in a Kubernetes Secret resource.
type OpenBaoUserPassAuth struct {
	// Path where the UserPassword authentication backend is mounted
	// in OpenBao, e.g: "userpass"
	// +kubebuilder:default=userpass
	Path string `json:"path"`

	// Username is a username used to authenticate using the UserPass OpenBao
	// authentication method
	Username string `json:"username"`

	// SecretRef to a key in a Secret resource containing password for the
	// user used to authenticate with OpenBao using the UserPass authentication
	// method
	// +optional
	SecretRef esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// OpenBaoGCPAuth authenticates with OpenBao using Google Cloud Platform authentication method.
// Refer: https://github.com/openbao/openbao-plugins/blob/main/auth/gcp/docs/index.md
//
// When ServiceAccountRef, SecretRef and WorkloadIdentity are not specified, the provider will use the controller pod's
// identity to authenticate with GCP. This supports both GKE Workload Identity and service account keys.
type OpenBaoGCPAuth struct {
	// Path where the GCP auth method is enabled in OpenBao, e.g: "gcp"
	// +kubebuilder:default=gcp
	// +optional
	Path string `json:"path,omitempty"`

	// OpenBao Role. In OpenBao, a role describes an identity with a set of permissions, groups, or policies you want to attach to a user of the secrets engine.
	//+required
	Role string `json:"role"`

	// Project ID of the Google Cloud Platform project
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// Location optionally defines a location/region for the secret
	// +optional
	Location string `json:"location,omitempty"`

	// Specify credentials in a Secret object
	// +optional
	SecretRef *GCPSMAuthSecretRef `json:"secretRef,omitempty"`

	// Specify a service account with Workload Identity
	// +optional
	WorkloadIdentity *GCPWorkloadIdentity `json:"workloadIdentity,omitempty"`

	// ServiceAccountRef to a service account for impersonation
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

// OpenBaoCheckAndSet defines the Check-And-Set (CAS) settings for OpenBao KV v2 PushSecret operations.
type OpenBaoCheckAndSet struct {
	// Required when true, all write operations must include a check-and-set parameter.
	// This helps prevent unintentional overwrites of secrets.
	// +optional
	Required bool `json:"required,omitempty"`
}
