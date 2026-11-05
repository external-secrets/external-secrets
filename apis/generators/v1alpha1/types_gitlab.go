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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// GitlabDeployTokenScope is a scope that can be granted to a GitLab deploy token.
// +kubebuilder:validation:Enum=read_repository;read_registry;write_registry;read_package_registry;write_package_registry;read_virtual_registry;write_virtual_registry
type GitlabDeployTokenScope string

const (
	// GitlabDeployTokenScopeReadRepository allows read access to the repository.
	GitlabDeployTokenScopeReadRepository GitlabDeployTokenScope = "read_repository"
	// GitlabDeployTokenScopeReadRegistry allows read access to the container registry.
	GitlabDeployTokenScopeReadRegistry GitlabDeployTokenScope = "read_registry"
	// GitlabDeployTokenScopeWriteRegistry allows write access to the container registry.
	GitlabDeployTokenScopeWriteRegistry GitlabDeployTokenScope = "write_registry"
	// GitlabDeployTokenScopeReadPackageRegistry allows read access to the package registry.
	GitlabDeployTokenScopeReadPackageRegistry GitlabDeployTokenScope = "read_package_registry"
	// GitlabDeployTokenScopeWritePackageRegistry allows write access to the package registry.
	GitlabDeployTokenScopeWritePackageRegistry GitlabDeployTokenScope = "write_package_registry"
	// GitlabDeployTokenScopeReadVirtualRegistry allows read access to the virtual registry (projects only).
	GitlabDeployTokenScopeReadVirtualRegistry GitlabDeployTokenScope = "read_virtual_registry"
	// GitlabDeployTokenScopeWriteVirtualRegistry allows write access to the virtual registry (projects only).
	GitlabDeployTokenScopeWriteVirtualRegistry GitlabDeployTokenScope = "write_virtual_registry"
)

// GitlabDeployTokenSpec defines the desired state to generate a GitLab deploy token.
// +kubebuilder:validation:XValidation:rule="has(self.projectID) != has(self.groupID)",message="exactly one of projectID or groupID must be set"
// +kubebuilder:validation:XValidation:rule="!(has(self.expiresAt) && has(self.expiresAfter))",message="expiresAt and expiresAfter are mutually exclusive"
type GitlabDeployTokenSpec struct {
	// URL configures the GitLab instance URL. Defaults to https://gitlab.com.
	// +optional
	URL string `json:"url,omitempty"`

	// ProjectID is the numeric ID or unescaped path (e.g. group/project) of the
	// project to create the deploy token in. The generator URL-escapes paths before
	// calling the GitLab API, so do not pre-encode. Mutually exclusive with groupID.
	// +optional
	// +kubebuilder:validation:MinLength=1
	ProjectID string `json:"projectID,omitempty"`

	// GroupID is the numeric ID or unescaped path (e.g. parent/group) of the group to
	// create the deploy token in. The generator URL-escapes paths before calling the
	// GitLab API, so do not pre-encode. Mutually exclusive with projectID.
	// +optional
	// +kubebuilder:validation:MinLength=1
	GroupID string `json:"groupID,omitempty"`

	// Name of the deploy token.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Scopes granted to the deploy token. At least one scope is required.
	// +kubebuilder:validation:MinItems=1
	Scopes []GitlabDeployTokenScope `json:"scopes"`

	// ExpiresAt is an optional expiry for the deploy token. If omitted the token does
	// not expire on the GitLab side and is revoked only when the generator state is
	// cleaned up (on regeneration or when the consuming ExternalSecret is deleted).
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// ExpiresAfter sets a relative expiry, computed as now + expiresAfter on
	// each generation, so every token carries a fresh expiry with nothing to
	// bump by hand. metav1.Duration syntax, largest unit hours (e.g. "720h").
	// Mutually exclusive with expiresAt; minimum 24h. Keep it larger than the
	// consuming ExternalSecret refreshInterval so a token cannot expire before
	// the next rotation replaces it.
	// +optional
	// +kubebuilder:validation:XValidation:rule="duration(self) >= duration('24h')",message="expiresAfter must be at least 24h"
	ExpiresAfter *metav1.Duration `json:"expiresAfter,omitempty"`

	// Username is an optional username for the deploy token. GitLab defaults it to
	// gitlab+deploy-token-{n} when omitted.
	// +optional
	Username string `json:"username,omitempty"`

	// Auth configures how ESO authenticates with the GitLab API.
	Auth GitlabDeployTokenAuth `json:"auth"`
}

// GitlabDeployTokenAuth defines the authentication configuration for the GitLab API.
type GitlabDeployTokenAuth struct {
	// Token references a secret containing a GitLab access token (personal, group, or
	// project) with the api scope and at least the Maintainer role on the target.
	Token GitlabDeployTokenSecretRef `json:"token"`
}

// GitlabDeployTokenSecretRef references a secret containing a GitLab access token.
type GitlabDeployTokenSecretRef struct {
	SecretRef esmeta.SecretKeySelector `json:"secretRef"`
}

// GitlabDeployToken generates a GitLab deploy token.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type GitlabDeployToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GitlabDeployTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// GitlabDeployTokenList contains a list of GitlabDeployToken resources.
type GitlabDeployTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitlabDeployToken `json:"items"`
}
