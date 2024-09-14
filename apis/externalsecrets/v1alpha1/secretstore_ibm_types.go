//Copyright External Secrets Inc. All Rights Reserved

package v1alpha1

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

type IBMAuth struct {
	SecretRef IBMAuthSecretRef `json:"secretRef"`
}

type IBMAuthSecretRef struct {
	// The SecretAccessKey is used for authentication
	// +optional
	SecretAPIKey esmeta.SecretKeySelector `json:"secretApiKeySecretRef,omitempty"`
}
