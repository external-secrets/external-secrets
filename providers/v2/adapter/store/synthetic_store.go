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

package store

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// SyntheticStore implements GenericStore to wrap provider config JSON
// for use with v1 providers. This allows v1 NewClient() methods to work
// with provider config passed as JSON bytes from v2 architecture.
type SyntheticStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	spec              *esv1.SecretStoreSpec
	status            esv1.SecretStoreStatus
	gvk               schema.GroupVersionKind
}

// NewSyntheticStore creates a new SyntheticStore from provider config JSON.
// The providerConfig JSON should contain the provider-specific configuration
// (e.g., KubernetesProvider, AWSProvider, etc.).
func NewSyntheticStore(spec *esv1.SecretStoreSpec, namespace string) (*SyntheticStore, error) {
	if spec == nil {
		return nil, fmt.Errorf("spec cannot be empty")
	}

	store := &SyntheticStore{
		TypeMeta: metav1.TypeMeta{
			APIVersion: esv1.SchemeGroupVersion.String(),
			Kind:       esv1.SecretStoreKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "synthetic-store",
			Namespace: namespace,
		},
		spec:   spec,
		status: esv1.SecretStoreStatus{},
		gvk: schema.GroupVersionKind{
			Group:   esv1.SchemeGroupVersion.Group,
			Version: esv1.SchemeGroupVersion.Version,
			Kind:    esv1.SecretStoreKind,
		},
	}
	store.SetGroupVersionKind(store.gvk)
	return store, nil
}

// GetSpec returns the SecretStoreSpec containing the provider configuration.
func (s *SyntheticStore) GetSpec() *esv1.SecretStoreSpec {
	return s.spec
}

// GetObjectKind returns the GroupVersionKind of this object.
func (s *SyntheticStore) GetObjectKind() schema.ObjectKind {
	return s
}

// DeepCopyObject creates a deep copy of the SyntheticStore.
func (s *SyntheticStore) DeepCopyObject() runtime.Object {
	return &SyntheticStore{
		TypeMeta:   s.TypeMeta,
		ObjectMeta: *s.ObjectMeta.DeepCopy(),
		spec:       s.spec.DeepCopy(),
		status:     s.status,
		gvk:        s.gvk,
	}
}

// GroupVersionKind returns the GVK of the synthetic store.
func (s *SyntheticStore) GroupVersionKind() schema.GroupVersionKind {
	return s.gvk
}

// SetGroupVersionKind sets the GVK of the synthetic store.
func (s *SyntheticStore) SetGroupVersionKind(gvk schema.GroupVersionKind) {
	s.gvk = gvk
}

// GetNamespacedName returns the name and namespace of the store.
func (s *SyntheticStore) GetNamespacedName() string {
	return fmt.Sprintf("%s/%s", s.Namespace, s.Name)
}

// GetStatus returns the status of the store.
func (s *SyntheticStore) GetStatus() esv1.SecretStoreStatus {
	return s.status
}

// SetStatus sets the status of the store.
func (s *SyntheticStore) SetStatus(status esv1.SecretStoreStatus) {
	s.status = status
}

// GetKind returns the kind of the store.
func (s *SyntheticStore) GetKind() string {
	return s.Kind
}

// GetObjectMeta returns the ObjectMeta of the store.
func (s *SyntheticStore) GetObjectMeta() *metav1.ObjectMeta {
	return &s.ObjectMeta
}

// GetTypeMeta returns the TypeMeta of the store.
func (s *SyntheticStore) GetTypeMeta() *metav1.TypeMeta {
	return &s.TypeMeta
}

// Copy returns a deep copy of the store.
func (s *SyntheticStore) Copy() esv1.GenericStore {
	return s.DeepCopyObject().(esv1.GenericStore)
}
