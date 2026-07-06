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

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// OpenBaoKVStoreVersion represents the version of the OpenBao KV secret engine.
type OpenBaoKVStoreVersion string

// These are the currently supported OpenBaoKVStoreVersion.
const (
	OpenBaoKVStoreV1 OpenBaoKVStoreVersion = "v1"
	OpenBaoKVStoreV2 OpenBaoKVStoreVersion = "v2"
)

// OpenBaoProvider configures a store to sync secrets using an OpenBao KV backend.
// +kubebuilder:validation:AtMostOneOf=caBundle;caProvider
type OpenBaoProvider struct {
	// Auth configures how secret-manager authenticates with the OpenBao server.
	//
	// +optional
	Auth *OpenBaoAuth `json:"auth,omitempty"`

	// PEM encoded CA bundle used to validate the OpenBao server certificate. If
	// this and `caProvider` are not set the system root certificates are used
	// to validate the TLS connection.
	//
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// The provider for the CA bundle to use to validate OpenBao server
	// certificate. If this and `caBundle` are not set the system root
	// certificates are used to validate the TLS connection.
	//
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`

	// Name of the [OpenBao Namespace]. Namespaces is a set of features within
	// OpenBao that allows OpenBao environments to support secure multi-tenancy.
	// e.g: "ns1".
	//
	// +optional
	//
	// [OpenBao Namespace]: https://openbao.org/docs/concepts/namespaces/
	Namespace *string `json:"namespace,omitempty"`

	// Server is the connection address for the OpenBao server, e.g: `https://openbao.example.com:8200`.
	Server string `json:"server"`

	// Path is the mount path of the OpenBao KV backend endpoint, e.g:
	// "secret". The v2 KV secret engine version specific "/data" path suffix
	// for fetching secrets from OpenBao is optional and will be appended
	// if not present in specified path.
	//
	// +optional
	Path *string `json:"path,omitempty"`

	// Version is the OpenBao KV secret engine version. This can be either "v1" or
	// "v2". Version defaults to "v2".
	//
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum="v1";"v2"
	// +kubebuilder:default:="v2"
	Version OpenBaoKVStoreVersion `json:"version"`
}

// OpenBaoAuth is the configuration used to authenticate with an OpenBao server.
// Currently the following authentication methods are supported: [AppRole],
// [Kubernetes], [Token] and [UserPass]
//
// Additional authentication methods are planned for future releases.
//
// +kubebuilder:validation:ExactlyOneOf=appRole;tokenSecretRef;userPass;kubernetes
//
// [AppRole]: https://openbao.org/docs/auth/approle/
// [Token]: https://openbao.org/docs/auth/token/
// [UserPass]: https://openbao.org/docs/auth/userpass/
// [Kubernetes]: https://openbao.org/docs/auth/kubernetes/
type OpenBaoAuth struct {
	// AppRole authenticates with OpenBao using the [App Role auth mechanism],
	// with the role and secret stored in a Kubernetes Secret resource.
	//
	// [App Role auth mechanism]: https://openbao.org/docs/auth/approle/
	//
	// +optional
	AppRole *OpenBaoAppRole `json:"appRole,omitempty"`

	// Kubernetes authenticates with OpenBao by passing a ServiceAccount
	// token to the [Kubernetes auth mechanism].
	//
	// +optional
	//
	// [Kubernetes auth mechanism]: https://openbao.org/docs/auth/kubernetes/
	Kubernetes *OpenBaoKubernetesAuth `json:"kubernetes,omitempty"`

	// Name of the [OpenBao Namespace] to authenticate to. This can be different
	// than the namespace your secret is in. Namespaces is a set of features
	// within OpenBao that allows OpenBao environments to support secure
	// multi-tenancy. e.g: "ns1". This will default to OpenBao.Namespace field
	// if set, or empty otherwise
	//
	// +optional
	//
	// [OpenBao Namespace]: https://openbao.org/docs/concepts/namespaces/
	Namespace *string `json:"namespace,omitempty"`

	// TokenSecretRef authenticates with OpenBao by presenting a token.
	//
	// +optional
	TokenSecretRef *esmeta.SecretKeySelector `json:"tokenSecretRef,omitempty"`

	// UserPass authenticates with OpenBao by passing a username/password pair
	//
	// +optional
	UserPass *OpenBaoUserPassAuth `json:"userPass,omitempty"`
}

// OpenBaoUserPassAuth authenticates with OpenBao using [UserPass authentication
// method], with the username and password stored in a Kubernetes Secret
// resource.
//
// [UserPass authentication method]: https://openbao.org/docs/auth/userpass/
type OpenBaoUserPassAuth struct {
	// Path where the UserPassword authentication backend is mounted
	// in OpenBao, e.g: "userpass"
	//
	// +kubebuilder:default=userpass
	Path string `json:"path"`

	// Username is a username used to authenticate using the [UserPass
	// authentication method]
	//
	// [UserPass authentication method]: https://openbao.org/docs/auth/userpass/
	Username string `json:"username"`

	// SecretRef to a key in a Secret resource containing password for the user
	// used to authenticate with OpenBao using the [UserPass authentication
	// method]
	//
	// [UserPass authentication method]: https://openbao.org/docs/auth/userpass/
	SecretRef esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// OpenBaoAppRole authenticates with OpenBao using the [App Role auth
// mechanism], with the role and secret stored in a Kubernetes Secret resource.
// The role ID has to be specified either inline via `roleId` or by referencing
// a secret via `roleRef`.
//
// +kubebuilder:validation:ExactlyOneOf=roleId;roleRef
//
// [App Role auth mechanism]: https://openbao.org/docs/auth/approle/
type OpenBaoAppRole struct {
	// Path where the App Role authentication backend is mounted
	// in OpenBao, e.g: "approle"
	//
	// +kubebuilder:default=approle
	Path string `json:"path"`

	// RoleID configured in the App Role authentication backend when setting
	// up the authentication backend in OpenBao.
	//
	// +optional
	// +kubebuilder:validation:MinLength=1
	RoleID string `json:"roleId,omitempty"`

	// Reference to a key in a Secret that contains the App Role ID used
	// to authenticate with OpenBao.
	// The `key` field must be specified and denotes which entry within the Secret
	// resource is used as the app role id.
	//
	// +optional
	RoleRef *esmeta.SecretKeySelector `json:"roleRef,omitempty"`

	// Reference to a key in a Secret that contains the App Role secret used
	// to authenticate with OpenBao.
	// The `key` field must be specified and denotes which entry within the Secret
	// resource is used as the app role secret.
	SecretRef esmeta.SecretKeySelector `json:"secretRef"`
}

// OpenBaoKubernetesAuth authenticates with OpenBao using the [Kubernetes
// auth mechanism] with a ServiceAccount token. The ServiceAccount token can be
// sourced from a ServiceAccount via `ServiceAccountRef` or from a secret
// via `SecretRef`.
// Using the controller pod's ServiceAccount token is not supported.
//
// +kubebuilder:validation:ExactlyOneOf=serviceAccountRef;secretRef
//
// [Kubernetes auth mechanism]: https://openbao.org/docs/auth/kubernetes/
type OpenBaoKubernetesAuth struct {
	// Path where the Kubernetes authentication backend is mounted in OpenBao, e.g:
	// "kubernetes"
	// +kubebuilder:default=kubernetes
	Path string `json:"path"`

	// Optional service account field containing the name of a kubernetes ServiceAccount.
	// If the service account is specified, the service account secret token JWT will be used
	// for authenticating with OpenBao.
	//
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`

	// Optional secret field containing a Kubernetes ServiceAccount JWT used
	// for authenticating with OpenBao. If a name is specified without a key,
	// `token` is the default.
	//
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`

	// A required field containing the OpenBao Role to assume. A Role binds a
	// Kubernetes ServiceAccount with a set of OpenBao policies.
	Role string `json:"role"`
}
