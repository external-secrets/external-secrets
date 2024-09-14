//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// BitwardenSecretsManagerProvider configures a store to sync secrets with a Bitwarden Secrets Manager instance.
type BitwardenSecretsManagerProvider struct {
	APIURL                string `json:"apiURL,omitempty"`
	IdentityURL           string `json:"identityURL,omitempty"`
	BitwardenServerSDKURL string `json:"bitwardenServerSDKURL,omitempty"`
	// Base64 encoded certificate for the bitwarden server sdk. The sdk MUST run with HTTPS to make sure no MITM attack
	// can be performed.
	// +optional
	CABundle string `json:"caBundle,omitempty"`
	// see: https://external-secrets.io/latest/spec/#external-secrets.io/v1alpha1.CAProvider
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`
	// OrganizationID determines which organization this secret store manages.
	OrganizationID string `json:"organizationID"`
	// ProjectID determines which project this secret store manages.
	ProjectID string `json:"projectID"`
	// Auth configures how secret-manager authenticates with a bitwarden machine account instance.
	// Make sure that the token being used has permissions on the given secret.
	Auth BitwardenSecretsManagerAuth `json:"auth"`
}

// BitwardenSecretsManagerAuth contains the ref to the secret that contains the machine account token.
type BitwardenSecretsManagerAuth struct {
	SecretRef BitwardenSecretsManagerSecretRef `json:"secretRef"`
}

// BitwardenSecretsManagerSecretRef contains the credential ref to the bitwarden instance.
type BitwardenSecretsManagerSecretRef struct {
	// AccessToken used for the bitwarden instance.
	// +required
	Credentials esmeta.SecretKeySelector `json:"credentials"`
}
