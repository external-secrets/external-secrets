//Copyright External Secrets Inc. All Rights Reserved

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
