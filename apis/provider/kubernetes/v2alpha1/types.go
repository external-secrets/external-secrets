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
package v2alpha1

import (
	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Kubernetes defines the configuration for the Kubernetes Secret provider.
// This provider fetches secrets from Kubernetes Secrets in the same cluster.
// It's primarily useful for testing and migration scenarios.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets}
type Kubernetes struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec v1.KubernetesProvider `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// KubernetesList contains a list of Kubernetes resources.
type KubernetesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kubernetes `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kubernetes{}, &KubernetesList{})
}
