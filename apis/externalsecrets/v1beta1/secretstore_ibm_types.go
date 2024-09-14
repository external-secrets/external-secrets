//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Configures an store to sync secrets using a IBM Cloud Secrets Manager
// backend.
type IBMProvider struct {
	// Auth configures how secret-manager authenticates with the IBM secrets manager.
	Auth IBMAuth `json:"auth"`

	// ServiceURL is the Endpoint URL that is specific to the Secrets Manager service instance
	ServiceURL *string `json:"serviceUrl,omitempty"`
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type IBMAuth struct {
	SecretRef     *IBMAuthSecretRef     `json:"secretRef,omitempty"`
	ContainerAuth *IBMAuthContainerAuth `json:"containerAuth,omitempty"`
}

type IBMAuthSecretRef struct {
	// The SecretAccessKey is used for authentication
	SecretAPIKey esmeta.SecretKeySelector `json:"secretApiKeySecretRef,omitempty"`
}

// IBM Container-based auth with IAM Trusted Profile.
type IBMAuthContainerAuth struct {
	// the IBM Trusted Profile
	Profile string `json:"profile"`

	// Location the token is mounted on the pod
	TokenLocation string `json:"tokenLocation,omitempty"`

	IAMEndpoint string `json:"iamEndpoint,omitempty"`
}
