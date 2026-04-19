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

// GiteaProvider provides access and authentication to a Gitea instance.
type GiteaProvider struct {
	// URL is the base URL of the Gitea instance (e.g. https://gitea.example.com).
	// This field is required.
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Auth contains credentials for authenticating with the Gitea API.
	Auth GiteaAuth `json:"auth"`

	// Organization is the Gitea organization to sync secrets into.
	// Required when targeting organization-scoped or repository-scoped secrets.
	// +optional
	Organization string `json:"organization,omitempty"`

	// Repository scopes secrets to a specific repository within the Organization.
	// Requires Organization to also be set.
	// +optional
	Repository string `json:"repository,omitempty"`
}

// GiteaAuth configures authentication for the Gitea API.
type GiteaAuth struct {
	// SecretRef references a Kubernetes Secret containing the Gitea personal access token.
	SecretRef esmeta.SecretKeySelector `json:"secretRef"`
}
