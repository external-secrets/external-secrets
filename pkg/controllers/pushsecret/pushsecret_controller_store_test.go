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
