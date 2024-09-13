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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IAMKeysSpec struct {
	// Region specifies the region to operate in.
	Region string `json:"region"`

	// Auth defines how to authenticate with AWS
	// +optional
	Auth AWSAuth `json:"auth,omitempty"`

	// You can assume a role before making calls to the
	// desired AWS service.
	// +optional
	Role string `json:"role,omitempty"`

	IAMRef IAMRef `json:"iamRef"`
}

type IAMRef struct {
	Username string `json:"username"`
	MaxKeys  int    `json:"maxKeys"`
}

// IAMKeys uses the CreateAccessKey API to retrieve an
// access key. It also rotates the key by making sure only X keys exist on a given user.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={awsiamkeys},shortName=awsiamkeys
type AWSIAMKeys struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IAMKeysSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// IAMKeysList contains a list of IAMKeys resources.
type AWSIAMKeysList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSIAMKeys `json:"items"`
}
