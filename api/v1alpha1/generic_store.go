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
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil

// GenericStore is a common interface for interacting with ClusterSecretStore
// or a namespaced SecretStore
type GenericStore interface {
	runtime.Object
	metav1.Object
	GetProvider() *SecretStoreProvider
}

// +kubebuilder:object:root:false
// +kubebuilder:object:generate:false
var _ GenericStore = &SecretStore{}

// GetProvider returns the underlying provider
func (c *SecretStore) GetProvider() *SecretStoreProvider {
	return c.Spec.Provider
}

// Copy returns a DeepCopy of the Store
func (c *SecretStore) Copy() GenericStore {
	return c.DeepCopy()
}
