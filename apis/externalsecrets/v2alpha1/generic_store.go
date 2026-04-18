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

package v2alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil

// GenericStore is a common interface for interacting with ProviderStore
// or ClusterProviderStore resources.
type GenericStore interface {
	runtime.Object
	metav1.Object

	GetKind() string
	GetRuntimeRef() StoreRuntimeRef
	GetBackendRef() BackendObjectReference
	GetConditions() []StoreNamespaceCondition
	GetStoreStatus() ProviderStoreStatus
	SetStoreStatus(ProviderStoreStatus)
	Copy() GenericStore
}

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
var _ GenericStore = &ProviderStore{}

func (s *ProviderStore) GetKind() string {
	return ProviderStoreKindStr
}

func (s *ProviderStore) GetRuntimeRef() StoreRuntimeRef {
	return s.Spec.RuntimeRef
}

func (s *ProviderStore) GetBackendRef() BackendObjectReference {
	return s.Spec.BackendRef
}

func (s *ProviderStore) GetConditions() []StoreNamespaceCondition {
	return nil
}

func (s *ProviderStore) GetStoreStatus() ProviderStoreStatus {
	return s.Status
}

func (s *ProviderStore) SetStoreStatus(status ProviderStoreStatus) {
	s.Status = status
}

func (s *ProviderStore) Copy() GenericStore {
	return s.DeepCopy()
}

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
var _ GenericStore = &ClusterProviderStore{}

func (s *ClusterProviderStore) GetKind() string {
	return ClusterProviderStoreKindStr
}

func (s *ClusterProviderStore) GetRuntimeRef() StoreRuntimeRef {
	return s.Spec.RuntimeRef
}

func (s *ClusterProviderStore) GetBackendRef() BackendObjectReference {
	return s.Spec.BackendRef
}

func (s *ClusterProviderStore) GetConditions() []StoreNamespaceCondition {
	return s.Spec.Conditions
}

func (s *ClusterProviderStore) GetStoreStatus() ProviderStoreStatus {
	return s.Status
}

func (s *ClusterProviderStore) SetStoreStatus(status ProviderStoreStatus) {
	s.Status = status
}

func (s *ClusterProviderStore) Copy() GenericStore {
	return s.DeepCopy()
}
