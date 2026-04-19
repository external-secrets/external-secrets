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

package providerstore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv2alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v2alpha1"
)

func TestProviderStoreReconcileMarksReadyWhenValidateSucceeds(t *testing.T) {
	_, address, tlsSecret := newProviderStoreGRPCServer(t)

	store := &esv2alpha1.ProviderStore{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-prod", Namespace: "tenant-a"},
		Spec: esv2alpha1.ProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "aws"},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: "aws.external-secrets.io/v1alpha1",
				Kind:       "SecretsManagerStore",
				Name:       "prod",
			},
		},
	}

	runtimeClass := readyRuntimeClass("aws")
	runtimeClass.Spec.Address = address

	reconciler := newProviderStoreReconciler(
		t,
		store,
		runtimeClass,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "external-secrets-provider-tls",
				Namespace: store.Namespace,
			},
			Data: tlsSecret,
		},
	)

	_, err := reconciler.Reconcile(context.Background(), reconcileRequest(store))
	require.NoError(t, err)

	updated := &esv2alpha1.ProviderStore{}
	err = reconciler.Get(context.Background(), client.ObjectKeyFromObject(store), updated)
	require.NoError(t, err)
	if !providerStoreReady(updated.Status) {
		t.Fatalf("expected Ready condition, got %#v", updated.Status.Conditions)
	}
}

func TestClusterProviderStoreReconcileUsesRuntimeOnlyValidationWhenBackendNamespaceIsOmitted(t *testing.T) {
	store := &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-shared"},
		Spec: esv2alpha1.ClusterProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "aws"},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: "aws.external-secrets.io/v1alpha1",
				Kind:       "SecretsManagerStore",
				Name:       "shared",
			},
		},
	}

	reconciler := newClusterProviderStoreReconciler(t, store, readyRuntimeClass("aws"))

	_, err := reconciler.Reconcile(context.Background(), reconcileRequest(store))
	require.NoError(t, err)

	updated := &esv2alpha1.ClusterProviderStore{}
	err = reconciler.Get(context.Background(), client.ObjectKeyFromObject(store), updated)
	require.NoError(t, err)
	if !providerStoreReady(updated.Status) {
		t.Fatalf("expected Ready condition, got %#v", updated.Status.Conditions)
	}
}
