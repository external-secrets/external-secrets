//Copyright External Secrets Inc. All Rights Reserved

package v1alpha1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Configures a store to sync secrets with a GitLab instance.
type GitlabProvider struct {
	// URL configures the GitLab instance URL. Defaults to https://gitlab.com/.
	URL string `json:"url,omitempty"`

	// Auth configures how secret-manager authenticates with a GitLab instance.
	Auth GitlabAuth `json:"auth"`

	// ProjectID specifies a project where secrets are located.
	ProjectID string `json:"projectID,omitempty"`
}

type GitlabAuth struct {
	SecretRef GitlabSecretRef `json:"SecretRef"`
}

type GitlabSecretRef struct {
	// AccessToken is used for authentication.
	AccessToken esmeta.SecretKeySelector `json:"accessToken,omitempty"`
}
