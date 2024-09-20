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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={gitlab},shortName=gitlab
type Gitlab struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GitlabSpec `json:"spec,omitempty"`
}

// Configures a store to sync secrets with a GitLab instance.
type GitlabSpec struct {
	// Used to select the correct ESO controller (think: ingress.ingressClassName)
	// The ESO controller is instantiated with a specific controller name and filters ES based on this property
	// +optional
	Controller string `json:"controller,omitempty"`

	// Used to configure http retries if failed
	// +optional
	RetrySettings *esmeta.RetrySettings `json:"retrySettings,omitempty"`

	// Used to configure store refresh interval in seconds. Empty or 0 will default to the controller config.
	// +optional
	RefreshInterval int `json:"refreshInterval,omitempty"`
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

// +kubebuilder:object:root=true

// FakeList contains a list of ExternalSecret resources.
type GitlabList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gitlab `json:"items"`
}

func init() {
}

// Gitlab type metadata.
var (
	GitlabKind             = reflect.TypeOf(Gitlab{}).Name()
	GitlabGroupKind        = schema.GroupKind{Group: Group, Kind: GitlabKind}.String()
	GitlabKindAPIVersion   = GitlabKind + "." + SchemeGroupVersion.String()
	GitlabGroupVersionKind = SchemeGroupVersion.WithKind(GitlabKind)
)
