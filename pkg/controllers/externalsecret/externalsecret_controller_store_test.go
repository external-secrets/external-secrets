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

package externalsecret

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestShouldSkipUnmanagedStoreAfterClusterStore(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := esv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	const namespace = "team-a"
	clusterRef := esv1.SecretStoreRef{Name: "shared-cluster-store", Kind: esv1.ClusterSecretStoreKind}
	foreignRef := esv1.SecretStoreRef{Name: "foreign-ss", Kind: esv1.SecretStoreKind}
	clusterStore := &esv1.ClusterSecretStore{}
	clusterStore.SetName(clusterRef.Name)
	foreignStore := &esv1.SecretStore{Spec: esv1.SecretStoreSpec{Controller: "other-eso"}}
	foreignStore.SetName(foreignRef.Name)
	foreignStore.SetNamespace(namespace)
	kube := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(clusterStore, foreignStore).Build()
	r := &Reconciler{Client: kube, ControllerClass: "my-eso"}

	tests := []struct {
		name     string
		top      esv1.SecretStoreRef
		source   *esv1.SecretStoreRef
		dataFrom bool
		wantSkip bool
	}{
		{name: "cluster store then foreign data source", top: clusterRef, source: &foreignRef, wantSkip: true},
		{name: "cluster store then foreign dataFrom source", top: clusterRef, source: &foreignRef, dataFrom: true, wantSkip: true},
		{name: "foreign store then cluster data source", top: foreignRef, source: &clusterRef, wantSkip: true},
		{name: "cluster store only", top: clusterRef, wantSkip: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &esv1.ExternalSecret{Spec: esv1.ExternalSecretSpec{SecretStoreRef: tt.top}}
			if tt.source != nil {
				if tt.dataFrom {
					es.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{{
						SourceRef: &esv1.StoreGeneratorSourceRef{SecretStoreRef: tt.source},
					}}
				} else {
					es.Spec.Data = []esv1.ExternalSecretData{{
						SourceRef: &esv1.StoreSourceRef{SecretStoreRef: *tt.source},
					}}
				}
			}

			skip, err := shouldSkipUnmanagedStore(context.Background(), namespace, r, es)
			if err != nil {
				t.Fatal(err)
			}
			if skip != tt.wantSkip {
				t.Errorf("shouldSkipUnmanagedStore() = %t, want %t", skip, tt.wantSkip)
			}
		})
	}
}
