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

package clientmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv2alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v2alpha1"
)

func TestManagerGetProviderStoreUsesStoreNamespaceForBackendRef(t *testing.T) {
	resetGlobalV2ConnectionPoolForTest(t)

	const callerNamespace = "tenant-a"

	server, address, tlsSecret := newRecordingProviderServer(t)
	store := &esv2alpha1.ProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-prod",
			Namespace: callerNamespace,
		},
		Spec: esv2alpha1.ProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "aws"},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: "aws.external-secrets.io/v1alpha1",
				Kind:       "SecretsManagerStore",
				Name:       "prod",
			},
		},
		Status: readyProviderStoreStatus(),
	}
	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "aws"},
		Spec:       esv1alpha1.ClusterProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(providerStoreTestScheme(t)).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: callerNamespace,
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "default", true)
	defer func() {
		_ = mgr.Close(context.Background())
	}()

	client, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: store.Name,
		Kind: esv1.ProviderStoreKindStr,
	}, callerNamespace, nil)
	require.NoError(t, err)

	result, err := client.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)

	req := server.LastValidateRequest()
	require.NotNil(t, req)
	require.NotNil(t, req.ProviderRef)
	assert.Equal(t, callerNamespace, req.ProviderRef.Namespace)
	assert.Equal(t, esv1.ProviderStoreKindStr, req.ProviderRef.StoreRefKind)
	assert.Equal(t, callerNamespace, req.SourceNamespace)
}

func TestManagerGetClusterProviderStoreDefaultsBackendNamespaceToCallerNamespace(t *testing.T) {
	resetGlobalV2ConnectionPoolForTest(t)

	const callerNamespace = "tenant-a"

	server, address, tlsSecret := newRecordingProviderServer(t)
	store := &esv2alpha1.ClusterProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aws-shared",
		},
		Spec: esv2alpha1.ClusterProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "aws"},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: "aws.external-secrets.io/v1alpha1",
				Kind:       "SecretsManagerStore",
				Name:       "shared",
			},
		},
		Status: readyProviderStoreStatus(),
	}
	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "aws"},
		Spec:       esv1alpha1.ClusterProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(providerStoreTestScheme(t)).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: callerNamespace,
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "default", true)
	defer func() {
		_ = mgr.Close(context.Background())
	}()

	client, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: store.Name,
		Kind: esv1.ClusterProviderStoreKindStr,
	}, callerNamespace, nil)
	require.NoError(t, err)

	result, err := client.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)

	req := server.LastValidateRequest()
	require.NotNil(t, req)
	require.NotNil(t, req.ProviderRef)
	assert.Equal(t, callerNamespace, req.ProviderRef.Namespace)
	assert.Equal(t, esv1.ClusterProviderStoreKindStr, req.ProviderRef.StoreRefKind)
	assert.Equal(t, callerNamespace, req.SourceNamespace)
}

func TestManagerGetProviderStoreRejectsNotReadyStoreWhenFloodgateEnabled(t *testing.T) {
	resetGlobalV2ConnectionPoolForTest(t)

	const callerNamespace = "tenant-a"

	_, address, tlsSecret := newRecordingProviderServer(t)

	store := &esv2alpha1.ProviderStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-prod",
			Namespace: callerNamespace,
		},
		Spec: esv2alpha1.ProviderStoreSpec{
			RuntimeRef: esv2alpha1.StoreRuntimeRef{Name: "aws"},
			BackendRef: esv2alpha1.BackendObjectReference{
				APIVersion: "aws.external-secrets.io/v1alpha1",
				Kind:       "SecretsManagerStore",
				Name:       "prod",
			},
		},
		Status: esv2alpha1.ProviderStoreStatus{},
	}
	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "aws"},
		Spec:       esv1alpha1.ClusterProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(providerStoreTestScheme(t)).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: callerNamespace,
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "default", true)
	defer func() {
		_ = mgr.Close(context.Background())
	}()

	_, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: store.Name,
		Kind: esv1.ProviderStoreKindStr,
	}, callerNamespace, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "ProviderStore")
	assert.ErrorContains(t, err, "is not ready")
}

func providerStoreTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := newManagerTestScheme(t)
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))
	utilruntime.Must(esv2alpha1.AddToScheme(scheme))
	return scheme
}

func readyProviderStoreStatus() esv2alpha1.ProviderStoreStatus {
	return esv2alpha1.ProviderStoreStatus{
		Conditions: []esv2alpha1.ProviderStoreCondition{
			{
				Type:   esv2alpha1.ProviderStoreReady,
				Status: corev1.ConditionTrue,
			},
		},
	}
}
