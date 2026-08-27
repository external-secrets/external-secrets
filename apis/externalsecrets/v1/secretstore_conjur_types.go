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

// ConjurProvider provides access to a Conjur provider.
type ConjurProvider struct {
	// URL is the endpoint of the Conjur instance.
	// +required
	URL string `json:"url"`

	// CABundle is a PEM encoded CA bundle that will be used to validate the Conjur server certificate.
	// +optional
	CABundle string `json:"caBundle,omitempty"`

	// Used to provide custom certificate authority (CA) certificates
	// for a secret store. The CAProvider points to a Secret or ConfigMap resource
	// that contains a PEM-encoded certificate.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`

	// Defines authentication settings for connecting to Conjur.
	// +required
	Auth ConjurAuth `json:"auth"`
}

// ConjurAuth is the way to provide authentication credentials to the ConjurProvider.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type ConjurAuth struct {
	// Authenticates with Conjur using an API key.
	// +optional
	APIKey *ConjurAPIKey `json:"apikey,omitempty"`

	// Jwt enables JWT authentication using Kubernetes service account tokens.
	// +optional
	Jwt *ConjurJWT `json:"jwt,omitempty"`

	// Cert enables certificate-based authentication using a client certificate and key.
	// +optional
	Cert *ConjurCert `json:"cert,omitempty"`

	// IAM enables authentication to Conjur via the authn-iam authenticator.
	// +optional
	IAM *ConjurIAM `json:"iam,omitempty"`

	// Azure enables authentication to Conjur via the authn-azure authenticator.
	// +optional
	Azure *ConjurAzure `json:"azure,omitempty"`

	// GCP enables authentication to Conjur via the authn-gcp authenticator.
	// +optional
	GCP *ConjurGCP `json:"gcp,omitempty"`
}

// ConjurAPIKey contains references to a Secret resource that holds
// the Conjur username and API key.
type ConjurAPIKey struct {
	// Account is the Conjur organization account name.
	// +required
	Account string `json:"account"`

	// A reference to a specific 'key' containing the Conjur username
	// within a Secret resource. In some instances, `key` is a required field.
	// +required
	UserRef *esmeta.SecretKeySelector `json:"userRef"`

	// A reference to a specific 'key' containing the Conjur API key
	// within a Secret resource. In some instances, `key` is a required field.
	// +required
	APIKeyRef *esmeta.SecretKeySelector `json:"apiKeyRef"`
}

// ConjurJWT defines the JWT authentication configuration for Conjur provider.
type ConjurJWT struct {
	// Account is the Conjur organization account name.
	// +required
	Account string `json:"account"`

	// The conjur authn jwt webservice id
	// +required
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

// ConjurCert defines the Cert authentication configuration for Conjur provider.
type ConjurCert struct {
	// Account is the Conjur organization account name.
	// +required
	Account string `json:"account"`

	// The conjur authn cert webservice id
	// +required
	ServiceID string `json:"serviceID"`

	// Optional HostID for cert authentication (can be omitted when using 'spiffe' mode).
	// +optional
	HostID string `json:"hostId,omitempty"`

	// ClientCertRef is a reference to a specific 'key' containing the client certificate
	// within a Secret resource. The certificate must be PEM-encoded.
	// +required
	ClientCertRef *esmeta.SecretKeySelector `json:"clientCertRef"`

	// ClientKeyRef is a reference to a specific 'key' containing the private RSA client key
	// within a Secret resource. The key must be PEM-encoded.
	// +required
	ClientKeyRef *esmeta.SecretKeySelector `json:"clientKeyRef"`
}

// ConjurIAM configures authentication to Conjur via the authn-iam authenticator.
// It uses the AWS STS GetCallerIdentity endpoint to authenticate.
type ConjurIAM struct {
	// Account is the Conjur organization account name.
	Account string `json:"account"`

	// ServiceID is the Conjur authn-iam webservice identifier (e.g. "prod").
	ServiceID string `json:"serviceID"`

	// HostID is the Conjur host mapped to the AWS IAM role
	// (e.g. "data/myapp/123456789012/MyRole").
	HostID string `json:"hostId"`

	// SecretRef holds optional references to Kubernetes Secrets containing explicit
	// AWS credentials. If omitted, the default AWS SDK credential chain is used
	// (IRSA, instance metadata, environment variables, etc.).
	// +optional
	SecretRef *ConjurIAMSecretRef `json:"secretRef,omitempty"`
}

// ConjurIAMSecretRef holds secret selectors for explicit AWS credentials.
type ConjurIAMSecretRef struct {
	// A reference to a Secret key containing the AWS Access Key ID.
	AccessKeyIDSecretRef esmeta.SecretKeySelector `json:"accessKeyIDSecretRef"`

	// A reference to a Secret key containing the AWS Secret Access Key.
	SecretAccessKeySecretRef esmeta.SecretKeySelector `json:"secretAccessKeySecretRef"`

	// A reference to a Secret key containing the AWS Session Token.
	// Required only when using temporary credentials.
	// +optional
	SessionTokenSecretRef *esmeta.SecretKeySelector `json:"sessionTokenSecretRef,omitempty"`
}

// ConjurAzure configures authentication to Conjur via the authn-azure authenticator.
// It uses an Azure JWT token to authenticate — either fetched from the Azure Instance
// Metadata Service (IMDS) automatically, or sourced from a Kubernetes ServiceAccount token.
type ConjurAzure struct {
	// Account is the Conjur organization account name.
	Account string `json:"account"`

	// ServiceID is the Conjur authn-azure webservice identifier (e.g. "prod").
	ServiceID string `json:"serviceID"`

	// HostID is the Conjur host mapped to the Azure managed identity
	// (e.g. "data/myapp/myhost").
	HostID string `json:"hostId"`

	// ClientID is the Azure managed identity client ID. Required for user-assigned
	// managed identities; omit for system-assigned identities.
	// +optional
	ClientID string `json:"clientId,omitempty"`

	// ServiceAccountRef specifies the Kubernetes service account for which to request
	// a token via the TokenRequest API. That token is used as the Azure JWT for Conjur
	// authn-azure. If omitted, the token is fetched from the Azure IMDS endpoint instead.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

// ConjurGCP configures authentication to Conjur via the authn-gcp authenticator.
// It uses a GCP identity token to authenticate — either fetched from the GCP Metadata
// Service automatically (GKE Workload Identity or GCE instance), or sourced from a
// Kubernetes Secret.
type ConjurGCP struct {
	// Account is the Conjur organization account name.
	Account string `json:"account"`

	// ServiceID is the Conjur authn-gcp webservice identifier (e.g. "prod").
	// Note: Conjur's authn-gcp authenticator does not include the service ID in the
	// authentication URL; this field is reserved for future use.
	// +optional
	ServiceID string `json:"serviceID,omitempty"`

	// HostID is the Conjur host mapped to the GCP service account
	// (e.g. "data/myapp/myhost").
	HostID string `json:"hostId"`

	// SecretRef holds a reference to a Kubernetes Secret containing a pre-obtained
	// GCP identity token. If omitted, the token is fetched from the GCP Metadata
	// Service automatically (requires GKE Workload Identity or a GCE/GKE node).
	// +optional
	SecretRef *ConjurGCPSecretRef `json:"secretRef,omitempty"`
}

// ConjurGCPSecretRef holds a reference to a Kubernetes Secret containing a GCP identity token.
type ConjurGCPSecretRef struct {
	// JWT is a reference to the Kubernetes Secret key holding the GCP identity token.
	JWT esmeta.SecretKeySelector `json:"jwt"`
}
