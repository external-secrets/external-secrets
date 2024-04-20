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

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

type ConjurProvider struct {
	URL string `json:"url"`
	// +optional
	CABundle string `json:"caBundle,omitempty"`
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`
	Auth       ConjurAuth  `json:"auth"`
}

type ConjurAuth struct {
	// +optional
	APIKey *ConjurAPIKey `json:"apikey,omitempty"`
	// +optional
	Jwt *ConjurJWT `json:"jwt,omitempty"`
}

type ConjurAPIKey struct {
	Account   string                    `json:"account"`
	UserRef   *esmeta.SecretKeySelector `json:"userRef"`
	APIKeyRef *esmeta.SecretKeySelector `json:"apiKeyRef"`
}

type ConjurJWT struct {
	Account string `json:"account"`

	// The conjur authn jwt webservice id
	ServiceID string `json:"serviceID"`

	// Optional HostID for JWT authentication. This may be used depending
	// on how the Conjur JWT authenticator policy is configured.
	// +optional
	HostID string `json:"hostId"`

	// Optional SecretRef that refers to a key in a Secret resource containing JWT token to
	// authenticate with Conjur using the JWT authentication method.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`

	// Optional ServiceAccountRef specifies the Kubernetes service account for which to request
	// a token for with the `TokenRequest` API.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}
