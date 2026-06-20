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
	Auth *OpenBaoAuth `json:"auth,omitempty"`

	// PEM encoded CA bundle used to validate the OpenBao server certificate. If
	// this and `caProvider` are not set the system root certificates are used
	// to validate the TLS connection.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// The provider for the CA bundle to use to validate OpenBao server
	// certificate. If this and `caBundle` are not set the system root
	// certificates are used to validate the TLS connection.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`

	// Server is the connection address for the OpenBao server, e.g: `https://openbao.example.com:8200`.
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
}

// OpenBaoAuth is the configuration used to authenticate with an OpenBao server.
// Currently only token-based authentication is supported via `tokenSecretRef`.
// Additional authentication methods are planned for future releases.
//
// +kubebuilder:validation:MaxProperties=1
type OpenBaoAuth struct {
	// TokenSecretRef authenticates with OpenBao by presenting a token.
	// +optional
	TokenSecretRef *esmeta.SecretKeySelector `json:"tokenSecretRef,omitempty"`

	// UserPass authenticates with OpenBao by passing a username/password pair
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
