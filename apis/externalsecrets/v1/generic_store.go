/*
Copyright © 2025 ESO Maintainer Team

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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil

// GenericStore is a common interface for interacting with ClusterSecretStore
// or a namespaced SecretStore.
type GenericStore interface {
	runtime.Object
	metav1.Object

	GetObjectMeta() *metav1.ObjectMeta
	GetTypeMeta() *metav1.TypeMeta
	GetKind() string

	GetSpec() *SecretStoreSpec
	GetNamespacedName() string
	GetStatus() SecretStoreStatus
	SetStatus(status SecretStoreStatus)
	Copy() GenericStore
}

// +kubebuilder:object:root:false
// +kubebuilder:object:generate:false
var _ GenericStore = &SecretStore{}

// GetObjectMeta returns the ObjectMeta of the SecretStore.
func (c *SecretStore) GetObjectMeta() *metav1.ObjectMeta {
	return &c.ObjectMeta
}

// GetTypeMeta returns the TypeMeta of the SecretStore.
func (c *SecretStore) GetTypeMeta() *metav1.TypeMeta {
	return &c.TypeMeta
}

// GetSpec returns the Spec of the SecretStore.
func (c *SecretStore) GetSpec() *SecretStoreSpec {
	return &c.Spec
}

// GetStatus returns the Status of the SecretStore.
func (c *SecretStore) GetStatus() SecretStoreStatus {
	return c.Status
}

// SetStatus sets the Status of the SecretStore.
func (c *SecretStore) SetStatus(status SecretStoreStatus) {
	c.Status = status
}

// GetNamespacedName returns the namespaced name of the SecretStore in the format "namespace/name".
func (c *SecretStore) GetNamespacedName() string {
	return fmt.Sprintf("%s/%s", c.Namespace, c.Name)
}

// GetKind returns the kind of the SecretStore.
func (c *SecretStore) GetKind() string {
	return SecretStoreKind
}

// Copy returns a deep copy of the SecretStore.
func (c *SecretStore) Copy() GenericStore {
	return c.DeepCopy()
}

// +kubebuilder:object:root:false
// +kubebuilder:object:generate:false
var _ GenericStore = &ClusterSecretStore{}

// GetObjectMeta returns the ObjectMeta of the ClusterSecretStore.
func (c *ClusterSecretStore) GetObjectMeta() *metav1.ObjectMeta {
	return &c.ObjectMeta
}

// GetTypeMeta returns the TypeMeta of the ClusterSecretStore.
func (c *ClusterSecretStore) GetTypeMeta() *metav1.TypeMeta {
	return &c.TypeMeta
}

// GetSpec returns the Spec of the ClusterSecretStore.
func (c *ClusterSecretStore) GetSpec() *SecretStoreSpec {
	return &c.Spec
}

// Copy returns a deep copy of the ClusterSecretStore.
func (c *ClusterSecretStore) Copy() GenericStore {
	return c.DeepCopy()
}

// GetStatus returns the Status of the ClusterSecretStore.
func (c *ClusterSecretStore) GetStatus() SecretStoreStatus {
	return c.Status
}

// SetStatus sets the Status of the ClusterSecretStore.
func (c *ClusterSecretStore) SetStatus(status SecretStoreStatus) {
	c.Status = status
}

// GetNamespacedName returns the namespaced name of the ClusterSecretStore in the format "namespace/name".
func (c *ClusterSecretStore) GetNamespacedName() string {
	return fmt.Sprintf("%s/%s", c.Namespace, c.Name)
}

// GetKind returns the kind of the ClusterSecretStore.
func (c *ClusterSecretStore) GetKind() string {
	return ClusterSecretStoreKind
}
