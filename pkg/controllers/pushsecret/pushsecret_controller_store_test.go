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

package pushsecret

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

func TestRemoveUnmanagedStoresWithMixedKinds(t *testing.T) {
	clusterRef := esapi.PushSecretStoreRef{Name: "shared", Kind: esv1.ClusterSecretStoreKind}
	foreignRef := esapi.PushSecretStoreRef{Name: "foreign", Kind: esv1.SecretStoreKind}
	managedRef := esapi.PushSecretStoreRef{Name: "managed", Kind: esv1.SecretStoreKind}
	shared := &esv1.ClusterSecretStore{}
	foreign := &esv1.SecretStore{Spec: esv1.SecretStoreSpec{Controller: "other"}}
	managed := &esv1.SecretStore{Spec: esv1.SecretStoreSpec{Controller: "mine"}}
	stores := map[esapi.PushSecretStoreRef]esv1.GenericStore{
		clusterRef: shared, foreignRef: foreign, managedRef: managed,
	}

	filtered := removeUnmanagedStores(&Reconciler{ControllerClass: "mine"}, stores)
	if len(filtered) != 2 || filtered[clusterRef] != shared || filtered[managedRef] != managed {
		t.Errorf("managed stores = %v, want shared cluster and managed namespaced stores", filtered)
	}
	if _, ok := filtered[foreignRef]; ok {
		t.Error("foreign store was not filtered")
	}
}

func TestSkipDisabledClusterSecretStore(t *testing.T) {
	clusterRef := esapi.PushSecretStoreRef{Name: "shared", Kind: esv1.ClusterSecretStoreKind}
	clusterSelectorRef := esapi.PushSecretStoreRef{Kind: esv1.ClusterSecretStoreKind, LabelSelector: &metav1.LabelSelector{}}
	localRef := esapi.PushSecretStoreRef{Name: "local", Kind: esv1.SecretStoreKind}
	tests := []struct {
		name    string
		refs    []esapi.PushSecretStoreRef
		dataTo  []esapi.PushSecretDataTo
		synced  esapi.SyncedPushSecretsMap
		enabled bool
		skip    bool
	}{
		{name: "cluster store", refs: []esapi.PushSecretStoreRef{clusterRef}, skip: true},
		{name: "mixed stores", refs: []esapi.PushSecretStoreRef{localRef, clusterRef}, skip: true},
		{name: "cluster selector", refs: []esapi.PushSecretStoreRef{clusterSelectorRef}, skip: true},
		{name: "dataTo cluster store", refs: []esapi.PushSecretStoreRef{localRef}, dataTo: []esapi.PushSecretDataTo{{StoreRef: &clusterRef}}, skip: true},
		{name: "dataTo cluster selector",
			refs: []esapi.PushSecretStoreRef{localRef}, dataTo: []esapi.PushSecretDataTo{{StoreRef: &clusterSelectorRef}}, skip: true},
		{name: "dataTo without storeRef", refs: []esapi.PushSecretStoreRef{localRef}, dataTo: []esapi.PushSecretDataTo{{}}, skip: false},
		{name: "previously synced cluster store", refs: []esapi.PushSecretStoreRef{localRef}, synced: esapi.SyncedPushSecretsMap{"ClusterSecretStore/shared": {}}, skip: true},
		{name: "namespaced store", refs: []esapi.PushSecretStoreRef{localRef}, skip: false},
		{name: "previously synced namespaced store", synced: esapi.SyncedPushSecretsMap{"SecretStore/local": {}}, skip: false},
		{name: "enabled cluster store", refs: []esapi.PushSecretStoreRef{clusterRef}, enabled: true, skip: false},
		{name: "enabled dataTo cluster store", dataTo: []esapi.PushSecretDataTo{{StoreRef: &clusterRef}}, enabled: true, skip: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &esapi.PushSecret{Spec: esapi.PushSecretSpec{SecretStoreRefs: tt.refs, DataTo: tt.dataTo}, Status: esapi.PushSecretStatus{SyncedPushSecrets: tt.synced}}
			r := &Reconciler{ClusterSecretStoreEnabled: tt.enabled}
			if got := r.shouldSkipClusterSecretStore(ps); got != tt.skip {
				t.Errorf("shouldSkipClusterSecretStore() = %t, want %t", got, tt.skip)
			}
		})
	}
}
