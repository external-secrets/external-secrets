// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package v1alpha1 implements Github targets
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GithubTargetKind is the kind name for GithubRepository resources.
var GithubTargetKind = "GithubRepository"

// GithubRepositorySpec contains the GithubRepository spec.
type GithubRepositorySpec struct {
	// Owner of the repository (user or organization).
	Owner string `json:"owner"`

	// Repository name.
	Repository string `json:"repository"`

	// Branch to target (optional, defaults to default branch).
	// +optional
	Branch string `json:"branch,omitempty"`

	// GitHub Enterprise endpoint. Format should be http(s)://[hostname]/api/v3/
	// or it will always return the 406 status code
	// If empty, default GitHub client will be configured
	// +optional
	EnterpriseURL string `json:"enterpriseUrl,omitempty"`

	// GitHub Enterprise upload endpoint. The upload URL format should be http(s)://[hostname]/api/uploads/
	// or it will always return the 406 status code
	// If empty, default GitHub client will be configured
	// +optional
	UploadURL string `json:"uploadUrl,omitempty"`

	// Paths to scan or push secrets to (relative to repo root).
	Paths []string `json:"paths,omitempty"`

	// CABundle is an optional PEM encoded CA bundle for HTTPS verification (for GitHub Enterprise).
	CABundle string `json:"caBundle,omitempty"`

	// Auth method to access the repository.
	Auth *GithubTargetAuth `json:"auth"`
}

// GithubTargetAuth contains the Github target auth spec.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type GithubTargetAuth struct {
	// Use a personal access token.
	Token *esmeta.SecretKeySelector `json:"token,omitempty"`

	// GitHub App authentication (JWT).
	AppAuth *GithubAppAuth `json:"appAuth,omitempty"`
}

// GithubAppAuth contains the Github App authentication spec.
type GithubAppAuth struct {
	AppID      string                   `json:"appID"`
	InstallID  string                   `json:"installID"`
	PrivateKey esmeta.SecretKeySelector `json:"privateKey"`
}

// GithubRepository is the schema for a GitHub target.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets,external-secrets-target}
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Capabilities",type=string,JSONPath=`.status.capabilities`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:subresource:status
type GithubRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GithubRepositorySpec `json:"spec,omitempty"`
	Status            TargetStatus         `json:"status,omitempty"`
}

// GithubRepositoryList contains a list of GithubRepository resources.
// +kubebuilder:object:root=true
type GithubRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubRepository `json:"items"`
}

// GetObjectMeta returns the object meta.
func (c *GithubRepository) GetObjectMeta() *metav1.ObjectMeta {
	return &c.ObjectMeta
}

// GetTypeMeta returns the type meta.
func (c *GithubRepository) GetTypeMeta() *metav1.TypeMeta {
	return &c.TypeMeta
}

// GetSpec returns the spec of the object.
func (c *GithubRepository) GetSpec() *esv1.SecretStoreSpec {
	return &esv1.SecretStoreSpec{}
}

// GetStatus returns the status of the object.
func (c *GithubRepository) GetStatus() esv1.SecretStoreStatus {
	return *TargetToSecretStoreStatus(&c.Status)
}

// SetStatus sets the status of the object.
func (c *GithubRepository) SetStatus(status esv1.SecretStoreStatus) {
	convertedStatus := SecretStoreToTargetStatus(&status)
	c.Status.Capabilities = convertedStatus.Capabilities
	c.Status.Conditions = convertedStatus.Conditions
}

// GetNamespacedName returns the namespaced name of the object.
func (c *GithubRepository) GetNamespacedName() string {
	return fmt.Sprintf("%s/%s", c.Namespace, c.Name)
}

// GetKind returns the kind of the object.
func (c *GithubRepository) GetKind() string {
	return GithubTargetKind
}

// Copy returns a copy of the object.
func (c *GithubRepository) Copy() esv1.GenericStore {
	return c.DeepCopy()
}

// GetTargetStatus returns the target status.
func (c *GithubRepository) GetTargetStatus() TargetStatus {
	return c.Status
}

// SetTargetStatus sets the target status.
func (c *GithubRepository) SetTargetStatus(status TargetStatus) {
	c.Status = status
}

// CopyTarget returns a copy of the target.
func (c *GithubRepository) CopyTarget() GenericTarget {
	return c.DeepCopy()
}

func init() {
	RegisterObjKind(GithubTargetKind, &GithubRepository{})
}
