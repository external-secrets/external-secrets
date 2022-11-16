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

	// InheritFromGroups specifies whether parent groups should be discovered and checked for secrets.
	InheritFromGroups bool `json:"inheritFromGroups,omitempty"`

	// GroupIDs specify, which gitlab groups to pull secrets from. Group secrets are read from left to right followed by the project variables.
	GroupIDs []string `json:"groupIDs,omitempty"`

	// Environment environment_scope of gitlab CI/CD variables (Please see https://docs.gitlab.com/ee/ci/environments/#create-a-static-environment on how to create environments)
	Environment string `json:"environment,omitempty"`
}

type GitlabAuth struct {
	SecretRef GitlabSecretRef `json:"SecretRef"`
}

type GitlabSecretRef struct {
	// AccessToken is used for authentication.
	AccessToken esmeta.SecretKeySelector `json:"accessToken,omitempty"`
}
