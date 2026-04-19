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

package store

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

// CompatibilityStoreToSyntheticStore materializes a v1 SecretStore/ClusterSecretStore payload
// into a GenericStore that existing v1 providers can consume.
func CompatibilityStoreToSyntheticStore(store *pb.CompatibilityStore) (*SyntheticStore, error) {
	if store == nil {
		return nil, fmt.Errorf("compatibility store is nil")
	}
	if len(store.GetStoreSpecJson()) == 0 {
		return nil, fmt.Errorf("compatibility store spec is empty")
	}

	spec := &esv1.SecretStoreSpec{}
	if err := json.Unmarshal(store.GetStoreSpecJson(), spec); err != nil {
		return nil, fmt.Errorf("decode compatibility store spec: %w", err)
	}
	if spec.Provider == nil {
		return nil, fmt.Errorf("compatibility store provider config is required")
	}

	switch store.GetStoreKind() {
	case esv1.SecretStoreKind, esv1.ClusterSecretStoreKind:
	default:
		return nil, fmt.Errorf("unsupported compatibility store kind %q", store.GetStoreKind())
	}

	gvk := schema.GroupVersionKind{
		Group:   esv1.SchemeGroupVersion.Group,
		Version: esv1.SchemeGroupVersion.Version,
		Kind:    store.GetStoreKind(),
	}
	syntheticStore := &SyntheticStore{
		TypeMeta: metav1.TypeMeta{
			APIVersion: esv1.SchemeGroupVersion.String(),
			Kind:       store.GetStoreKind(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       store.GetStoreName(),
			Namespace:  store.GetStoreNamespace(),
			UID:        types.UID(store.GetStoreUid()),
			Generation: store.GetStoreGeneration(),
		},
		spec:   spec,
		status: esv1.SecretStoreStatus{},
		gvk:    gvk,
	}
	syntheticStore.SetGroupVersionKind(gvk)

	return syntheticStore, nil
}
