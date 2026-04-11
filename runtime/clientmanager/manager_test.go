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

package clientmanager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	providergrpc "github.com/external-secrets/external-secrets/providers/v2/common/grpc"
)

func TestManagerGet(t *testing.T) {
	scheme := runtime.NewScheme()

	// add kubernetes schemes
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	// add external-secrets schemes
	utilruntime.Must(esv1.AddToScheme(scheme))

	// We have a test provider to control
	// the behavior of the NewClient func.
	fakeProvider := &WrapProvider{}
	esv1.ForceRegister(fakeProvider, &esv1.SecretStoreProvider{
		AWS: &esv1.AWSProvider{},
	}, esv1.MaintenanceStatusMaintained)

	// fake clients are re-used to compare the
	// in-memory reference
	clientA := &MockFakeClient{id: "1"}
	clientB := &MockFakeClient{id: "2"}

	const testNamespace = "foo"

	readyStatus := esv1.SecretStoreStatus{
		Conditions: []esv1.SecretStoreStatusCondition{
			{
				Type:   esv1.SecretStoreReady,
				Status: corev1.ConditionTrue,
			},
		},
	}

	fakeSpec := esv1.SecretStoreSpec{
		Provider: &esv1.SecretStoreProvider{
			AWS: &esv1.AWSProvider{},
		},
	}

	defaultStore := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: testNamespace,
		},
		Spec:   fakeSpec,
		Status: readyStatus,
	}

	otherStore := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other",
			Namespace: testNamespace,
		},
		Spec:   fakeSpec,
		Status: readyStatus,
	}

	var mgr *Manager

	provKey := storeKey(fakeProvider)

	type fields struct {
		client    client.Client
		clientMap map[clientKey]*clientVal
	}
	type args struct {
		storeRef  esv1.SecretStoreRef
		namespace string
		sourceRef *esv1.StoreGeneratorSourceRef
	}
	tests := []struct {
		name              string
		fields            fields
		args              args
		clientConstructor func(
			ctx context.Context,
			store esv1.GenericStore,
			kube client.Client,
			namespace string) (esv1.SecretsClient, error)
		verify     func(esv1.SecretsClient)
		afterClose func()
		want       esv1.SecretsClient
		wantErr    bool
	}{
		{
			name:    "creates a new client from storeRef and stores it",
			wantErr: false,
			fields: fields{
				client: fakeclient.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(defaultStore).
					Build(),
				clientMap: make(map[clientKey]*clientVal),
			},
			args: args{
				storeRef: esv1.SecretStoreRef{
					Name: defaultStore.Name,
					Kind: esv1.SecretStoreKind,
				},
				namespace: defaultStore.Namespace,
				sourceRef: nil,
			},
			clientConstructor: func(_ context.Context, _ esv1.GenericStore, _ client.Client, _ string) (esv1.SecretsClient, error) {
				return clientA, nil
			},
			verify: func(sc esv1.SecretsClient) {
				// we now must have this provider in the clientMap
				// and it mustbe the client defined in clientConstructor
				assert.NotNil(t, sc)
				c, ok := mgr.clientMap[provKey]
				require.True(t, ok)
				assert.Same(t, c.client, clientA)
			},

			afterClose: func() {
				v, ok := mgr.clientMap[provKey]
				assert.False(t, ok)
				assert.Nil(t, v)
			},
		},
		{
			name:    "creates a new client using both storeRef and sourceRef",
			wantErr: false,
			fields: fields{
				client: fakeclient.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(otherStore).
					Build(),
				clientMap: make(map[clientKey]*clientVal),
			},
			args: args{
				storeRef: esv1.SecretStoreRef{
					Name: defaultStore.Name,
					Kind: esv1.SecretStoreKind,
				},
				// this should take precedence
				sourceRef: &esv1.StoreGeneratorSourceRef{
					SecretStoreRef: &esv1.SecretStoreRef{
						Name: otherStore.Name,
						Kind: esv1.SecretStoreKind,
					},
				},
				namespace: defaultStore.Namespace,
			},
			clientConstructor: func(_ context.Context, _ esv1.GenericStore, _ client.Client, _ string) (esv1.SecretsClient, error) {
				return clientB, nil
			},
			verify: func(sc esv1.SecretsClient) {
				// we now must have this provider in the clientMap
				// and it mustbe the client defined in clientConstructor
				assert.NotNil(t, sc)
				c, ok := mgr.clientMap[provKey]
				assert.True(t, ok)
				assert.Same(t, c.client, clientB)
			},

			afterClose: func() {
				v, ok := mgr.clientMap[provKey]
				assert.False(t, ok)
				assert.True(t, clientB.closeCalled)
				assert.Nil(t, v)
			},
		},
		{
			name:    "retrieve cached client when store matches",
			wantErr: false,
			fields: fields{
				client: fakeclient.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(defaultStore).
					Build(),
				clientMap: map[clientKey]*clientVal{
					provKey: {
						client: clientA,
						store:  defaultStore,
					},
				},
			},
			args: args{
				storeRef: esv1.SecretStoreRef{
					Name: defaultStore.Name,
					Kind: esv1.SecretStoreKind,
				},
				namespace: defaultStore.Namespace,
				sourceRef: nil,
			},
			clientConstructor: func(_ context.Context, _ esv1.GenericStore, _ client.Client, _ string) (esv1.SecretsClient, error) {
				// constructor should not be called,
				// the client from the cache should be returned instead
				t.Fail()
				return nil, nil
			},
			verify: func(sc esv1.SecretsClient) {
				// verify that the secretsClient is the one from cache
				assert.NotNil(t, sc)
				c, ok := mgr.clientMap[provKey]
				assert.True(t, ok)
				assert.Same(t, c.client, clientA)
				assert.Same(t, sc, clientA)
			},

			afterClose: func() {
				v, ok := mgr.clientMap[provKey]
				assert.False(t, ok)
				assert.True(t, clientA.closeCalled)
				assert.Nil(t, v)
			},
		},
		{
			name:    "create new client when store doesn't match",
			wantErr: false,
			fields: fields{
				client: fakeclient.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(otherStore).
					Build(),
				clientMap: map[clientKey]*clientVal{
					provKey: {
						// we have clientA in cache pointing at defaultStore
						client: clientA,
						store:  defaultStore,
					},
				},
			},
			args: args{
				storeRef: esv1.SecretStoreRef{
					Name: otherStore.Name,
					Kind: esv1.SecretStoreKind,
				},
				namespace: otherStore.Namespace,
				sourceRef: nil,
			},
			clientConstructor: func(_ context.Context, _ esv1.GenericStore, _ client.Client, _ string) (esv1.SecretsClient, error) {
				// because there is a store mismatch
				// we create a new client
				return clientB, nil
			},
			verify: func(sc esv1.SecretsClient) {
				// verify that SecretsClient is NOT the one from cache
				assert.NotNil(t, sc)
				c, ok := mgr.clientMap[provKey]
				assert.True(t, ok)
				assert.Same(t, c.client, clientB)
				assert.Same(t, sc, clientB)
				assert.True(t, clientA.closeCalled)
			},
			afterClose: func() {
				v, ok := mgr.clientMap[provKey]
				assert.False(t, ok)
				assert.True(t, clientB.closeCalled)
				assert.Nil(t, v)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr = &Manager{
				log:             logr.Discard(),
				client:          tt.fields.client,
				enableFloodgate: true,
				clientMap:       tt.fields.clientMap,
			}
			fakeProvider.newClientFunc = tt.clientConstructor
			clientA.closeCalled = false
			clientB.closeCalled = false
			got, err := mgr.Get(context.Background(), tt.args.storeRef, tt.args.namespace, tt.args.sourceRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("Manager.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.verify(got)
			mgr.Close(context.Background())
			tt.afterClose()
		})
	}
}

func TestShouldProcessSecret(t *testing.T) {
	scheme := runtime.NewScheme()

	// add kubernetes schemes
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	// add external-secrets schemes
	utilruntime.Must(esv1.AddToScheme(scheme))

	testNamespace := "test-a"
	testCases := []struct {
		name       string
		conditions []esv1.ClusterSecretStoreCondition
		namespace  *corev1.Namespace
		wantErr    string
		want       bool
	}{
		{
			name: "processes a regex condition",
			conditions: []esv1.ClusterSecretStoreCondition{
				{
					NamespaceRegexes: []string{`test-*`},
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			},
			want: true,
		},
		{
			name: "process multiple regexes",
			conditions: []esv1.ClusterSecretStoreCondition{
				{
					NamespaceRegexes: []string{`nope`, `test-*`},
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			},
			want: true,
		},
		{
			name: "shouldn't process if nothing matches",
			conditions: []esv1.ClusterSecretStoreCondition{
				{
					NamespaceRegexes: []string{`nope`},
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			},
			want: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			fakeSpec := esv1.SecretStoreSpec{
				Conditions: tt.conditions,
			}

			defaultStore := &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: tt.namespace.Name,
				},
				Spec: fakeSpec,
			}

			client := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(defaultStore, tt.namespace).Build()
			clientMap := make(map[clientKey]*clientVal)
			mgr := &Manager{
				log:             logr.Discard(),
				client:          client,
				enableFloodgate: true,
				clientMap:       clientMap,
			}

			got, err := mgr.shouldProcessSecret(defaultStore, tt.namespace.Name)
			require.NoError(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetV2ProviderFeatureGate(t *testing.T) {
	previous := V2ProvidersEnabled()
	SetV2ProvidersEnabled(false)
	t.Cleanup(func() {
		SetV2ProvidersEnabled(previous)
	})

	mgr := NewManager(nil, "default", false)

	_, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: "example-provider",
		Kind: esv1.ProviderKindStr,
	}, "default", nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "v2 provider support is disabled")
}

func TestGetV2ProviderFeatureGateFromSourceRef(t *testing.T) {
	previous := V2ProvidersEnabled()
	SetV2ProvidersEnabled(false)
	t.Cleanup(func() {
		SetV2ProvidersEnabled(previous)
	})

	mgr := NewManager(nil, "default", false)

	_, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: "example-store",
		Kind: esv1.SecretStoreKind,
	}, "default", &esv1.StoreGeneratorSourceRef{
		SecretStoreRef: &esv1.SecretStoreRef{
			Name: "example-provider",
			Kind: esv1.ProviderKindStr,
		},
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "v2 provider support is disabled")
}

func TestGetV2ClusterProviderManifestScopeUsesManifestNamespaceForTLS(t *testing.T) {
	previous := V2ProvidersEnabled()
	SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		SetV2ProvidersEnabled(previous)
	})

	resetGlobalV2ConnectionPoolForTest(t)

	scheme := newManagerTestScheme(t)
	server, address, tlsSecret := newRecordingProviderServer(t)

	const manifestNamespace = "tenant-a"
	const referencedConfigNamespace = "provider-config-ns"

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-provider",
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "kubernetes-backend",
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeManifestNamespace,
			Conditions: []esv1.ClusterSecretStoreCondition{
				{
					Namespaces: []string{manifestNamespace},
				},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: manifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": manifestNamespace,
				},
			}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: referencedConfigNamespace}},
			clusterProvider,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: manifestNamespace,
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "default", false)

	client, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: clusterProvider.Name,
		Kind: esv1.ClusterProviderKindStr,
	}, manifestNamespace, nil)
	require.NoError(t, err)

	result, err := client.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)

	req := server.LastValidateRequest()
	require.NotNil(t, req)
	assert.Equal(t, manifestNamespace, req.SourceNamespace)
	require.NotNil(t, req.ProviderRef)
	assert.Equal(t, "kubernetes-backend", req.ProviderRef.Name)
	assert.Equal(t, "", req.ProviderRef.Namespace)
	assert.Equal(t, esv1.ClusterProviderKindStr, req.ProviderRef.StoreRefKind)

	require.Len(t, mgr.v2PooledConnections, 1)
}

func TestGetV2ClusterProviderRejectsProviderNamespaceWithoutNamespace(t *testing.T) {
	previous := V2ProvidersEnabled()
	SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		SetV2ProvidersEnabled(previous)
	})

	resetGlobalV2ConnectionPoolForTest(t)

	scheme := newManagerTestScheme(t)
	const manifestNamespace = "tenant-a"

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-provider",
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: "127.0.0.1:9443",
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "kubernetes-backend",
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeProviderNamespace,
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: manifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": manifestNamespace,
				},
			}},
			clusterProvider,
		).
		Build()

	mgr := NewManager(kubeClient, "default", false)

	_, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: clusterProvider.Name,
		Kind: esv1.ClusterProviderKindStr,
	}, manifestNamespace, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "authenticationScope=ProviderNamespace")
	assert.ErrorContains(t, err, "providerRef.namespace is empty")
}

func TestGetV2ClusterProviderProviderScopeUsesProviderNamespaceForTLS(t *testing.T) {
	previous := V2ProvidersEnabled()
	SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		SetV2ProvidersEnabled(previous)
	})

	resetGlobalV2ConnectionPoolForTest(t)

	scheme := newManagerTestScheme(t)
	server, address, tlsSecret := newRecordingProviderServer(t)

	const manifestNamespace = "tenant-a"
	const referencedConfigNamespace = "provider-config-ns"

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-provider",
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "kubernetes-backend",
					Namespace:  referencedConfigNamespace,
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeProviderNamespace,
			Conditions: []esv1.ClusterSecretStoreCondition{
				{
					Namespaces: []string{manifestNamespace},
				},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: manifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": manifestNamespace,
				},
			}},
			clusterProvider,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: referencedConfigNamespace,
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "default", false)

	client, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: clusterProvider.Name,
		Kind: esv1.ClusterProviderKindStr,
	}, manifestNamespace, nil)
	require.NoError(t, err)

	result, err := client.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)

	req := server.LastValidateRequest()
	require.NotNil(t, req)
	assert.Equal(t, referencedConfigNamespace, req.SourceNamespace)
	require.NotNil(t, req.ProviderRef)
	assert.Equal(t, "kubernetes-backend", req.ProviderRef.Name)
	assert.Equal(t, referencedConfigNamespace, req.ProviderRef.Namespace)
	assert.Equal(t, esv1.ClusterProviderKindStr, req.ProviderRef.StoreRefKind)

	require.Len(t, mgr.v2PooledConnections, 1)
}

func TestGetV2ClusterProviderRejectsDeniedNamespace(t *testing.T) {
	previous := V2ProvidersEnabled()
	SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		SetV2ProvidersEnabled(previous)
	})

	resetGlobalV2ConnectionPoolForTest(t)

	scheme := newManagerTestScheme(t)
	const manifestNamespace = "tenant-a"

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-provider",
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: "127.0.0.1:9443",
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "kubernetes-backend",
					Namespace:  "provider-config-ns",
				},
			},
			Conditions: []esv1.ClusterSecretStoreCondition{
				{
					NamespaceRegexes: []string{`other-.*`},
				},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: manifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": manifestNamespace,
				},
			}},
			clusterProvider,
		).
		Build()

	mgr := NewManager(kubeClient, "default", false)

	_, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: clusterProvider.Name,
		Kind: esv1.ClusterProviderKindStr,
	}, manifestNamespace, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "denied by spec.conditions")
}

func TestGetV2ProviderUsesManifestNamespaceAndDistinctCacheEntriesPerNamespace(t *testing.T) {
	previous := V2ProvidersEnabled()
	SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		SetV2ProvidersEnabled(previous)
	})

	registry := installGlobalV2ConnectionPoolForTest(t)

	scheme := newManagerTestScheme(t)
	server, address, tlsSecret := newRecordingProviderServer(t)

	const providerName = "provider"
	const firstManifestNamespace = "tenant-a"
	const secondManifestNamespace = "tenant-b"
	const referencedConfigNamespace = "provider-config-ns"

	providerA := &esv1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:       providerName,
			Namespace:  firstManifestNamespace,
			Generation: 1,
		},
		Spec: esv1.ProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "kubernetes-backend",
					Namespace:  referencedConfigNamespace,
				},
			},
		},
	}
	providerB := &esv1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:       providerName,
			Namespace:  secondManifestNamespace,
			Generation: 1,
		},
		Spec: esv1.ProviderSpec{
			Config: providerA.Spec.Config,
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: firstManifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": firstManifestNamespace,
				},
			}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: secondManifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": secondManifestNamespace,
				},
			}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: referencedConfigNamespace}},
			providerA,
			providerB,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: firstManifestNamespace,
				},
				Data: tlsSecret,
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: secondManifestNamespace,
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "default", false)

	firstClient, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: providerName,
		Kind: esv1.ProviderKindStr,
	}, firstManifestNamespace, nil)
	require.NoError(t, err)

	ready, err := firstClient.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, ready)

	firstReq := server.LastValidateRequest()
	require.NotNil(t, firstReq)
	assert.Equal(t, firstManifestNamespace, firstReq.SourceNamespace)

	secondClient, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: providerName,
		Kind: esv1.ProviderKindStr,
	}, secondManifestNamespace, nil)
	require.NoError(t, err)

	ready, err = secondClient.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, ready)

	secondReq := server.LastValidateRequest()
	require.NotNil(t, secondReq)
	assert.Equal(t, secondManifestNamespace, secondReq.SourceNamespace)

	assert.NotSame(t, firstClient, secondClient)
	require.Len(t, mgr.clientMap, 2)
	require.Len(t, mgr.v2PooledConnections, 2)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_active", address, true, 1)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_total", address, true, 1)

	require.NoError(t, mgr.Close(context.Background()))
	assert.Empty(t, mgr.clientMap)
	assert.Empty(t, mgr.v2PooledConnections)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_active", address, true, 0)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_idle", address, true, 1)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_total", address, true, 1)
	assert.Equal(t, 2, server.ValidateCallCount())
}

func TestGetV2ProviderInvalidatesGenerationCacheAndReleasesPoolReferences(t *testing.T) {
	previous := V2ProvidersEnabled()
	SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		SetV2ProvidersEnabled(previous)
	})

	registry := installGlobalV2ConnectionPoolForTest(t)

	scheme := newManagerTestScheme(t)
	server, address, tlsSecret := newRecordingProviderServer(t)

	const manifestNamespace = "tenant-a"
	const referencedConfigNamespace = "provider-config-ns"

	provider := &esv1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider",
			Namespace: manifestNamespace,
			Generation: 1,
		},
		Spec: esv1.ProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "kubernetes-backend",
					Namespace:  referencedConfigNamespace,
				},
			},
		},
	}

	tlsSecretObject := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-secrets-provider-tls",
			Namespace: manifestNamespace,
		},
		Data: tlsSecret,
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: manifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": manifestNamespace,
				},
			}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: referencedConfigNamespace}},
			provider,
			tlsSecretObject,
		).
		Build()

	mgr := NewManager(kubeClient, "default", false)

	firstClient, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: provider.Name,
		Kind: esv1.ProviderKindStr,
	}, manifestNamespace, nil)
	require.NoError(t, err)

	ready, err := firstClient.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, ready)

	req := server.LastValidateRequest()
	require.NotNil(t, req)
	assert.Equal(t, manifestNamespace, req.SourceNamespace)
	require.NotNil(t, req.ProviderRef)
	assert.Equal(t, "kubernetes-backend", req.ProviderRef.Name)
	assert.Equal(t, referencedConfigNamespace, req.ProviderRef.Namespace)
	assert.Equal(t, esv1.ProviderKindStr, req.ProviderRef.StoreRefKind)

	cachedClient, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: provider.Name,
		Kind: esv1.ProviderKindStr,
	}, manifestNamespace, nil)
	require.NoError(t, err)
	assert.Same(t, firstClient, cachedClient)
	require.Len(t, mgr.v2PooledConnections, 1)

	provider.Generation = 2
	require.NoError(t, kubeClient.Update(context.Background(), provider))

	secondClient, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: provider.Name,
		Kind: esv1.ProviderKindStr,
	}, manifestNamespace, nil)
	require.NoError(t, err)
	assert.NotSame(t, firstClient, secondClient)

	ready, err = secondClient.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, ready)

	require.Len(t, mgr.v2PooledConnections, 2)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_active", address, true, 1)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_total", address, true, 1)

	require.NoError(t, mgr.Close(context.Background()))
	assert.Empty(t, mgr.clientMap)
	assert.Empty(t, mgr.v2PooledConnections)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_active", address, true, 0)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_idle", address, true, 1)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_total", address, true, 1)

	assert.Equal(t, 2, server.ValidateCallCount())
}

func TestGetV2ClusterProviderInvalidatesGenerationCacheAndReleasesPoolReferences(t *testing.T) {
	previous := V2ProvidersEnabled()
	SetV2ProvidersEnabled(true)
	t.Cleanup(func() {
		SetV2ProvidersEnabled(previous)
	})

	registry := installGlobalV2ConnectionPoolForTest(t)

	scheme := newManagerTestScheme(t)
	server, address, tlsSecret := newRecordingProviderServer(t)

	const manifestNamespace = "tenant-a"
	const referencedConfigNamespace = "provider-config-ns"

	clusterProvider := &esv1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "cluster-provider",
			Generation: 1,
		},
		Spec: esv1.ClusterProviderSpec{
			Config: esv1.ProviderConfig{
				Address: address,
				ProviderRef: esv1.ProviderReference{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Kubernetes",
					Name:       "kubernetes-backend",
					Namespace:  referencedConfigNamespace,
				},
			},
			AuthenticationScope: esv1.AuthenticationScopeProviderNamespace,
			Conditions: []esv1.ClusterSecretStoreCondition{
				{
					Namespaces: []string{manifestNamespace},
				},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: manifestNamespace,
				Labels: map[string]string{
					"kubernetes.io/metadata.name": manifestNamespace,
				},
			}},
			clusterProvider,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: referencedConfigNamespace,
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "default", false)

	firstClient, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: clusterProvider.Name,
		Kind: esv1.ClusterProviderKindStr,
	}, manifestNamespace, nil)
	require.NoError(t, err)

	ready, err := firstClient.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, ready)

	firstReq := server.LastValidateRequest()
	require.NotNil(t, firstReq)
	assert.Equal(t, referencedConfigNamespace, firstReq.SourceNamespace)

	cachedClient, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: clusterProvider.Name,
		Kind: esv1.ClusterProviderKindStr,
	}, manifestNamespace, nil)
	require.NoError(t, err)
	assert.Same(t, firstClient, cachedClient)
	require.Len(t, mgr.v2PooledConnections, 1)

	clusterProvider.Generation = 2
	require.NoError(t, kubeClient.Update(context.Background(), clusterProvider))

	secondClient, err := mgr.Get(context.Background(), esv1.SecretStoreRef{
		Name: clusterProvider.Name,
		Kind: esv1.ClusterProviderKindStr,
	}, manifestNamespace, nil)
	require.NoError(t, err)
	assert.NotSame(t, firstClient, secondClient)

	ready, err = secondClient.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, ready)

	secondReq := server.LastValidateRequest()
	require.NotNil(t, secondReq)
	assert.Equal(t, referencedConfigNamespace, secondReq.SourceNamespace)

	require.Len(t, mgr.v2PooledConnections, 2)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_active", address, true, 1)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_total", address, true, 1)

	require.NoError(t, mgr.Close(context.Background()))
	assert.Empty(t, mgr.clientMap)
	assert.Empty(t, mgr.v2PooledConnections)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_active", address, true, 0)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_idle", address, true, 1)
	assertPoolMetricEventually(t, registry, "grpc_pool_connections_total", address, true, 1)
	assert.Equal(t, 2, server.ValidateCallCount())
}

type WrapProvider struct {
	newClientFunc func(
		context.Context,
		esv1.GenericStore,
		client.Client,
		string) (esv1.SecretsClient, error)
}

// NewClient constructs a SecretsManager Provider.
func (f *WrapProvider) NewClient(
	ctx context.Context,
	store esv1.GenericStore,
	kube client.Client,
	namespace string) (esv1.SecretsClient, error) {
	return f.newClientFunc(ctx, store, kube, namespace)
}

func (f *WrapProvider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// ValidateStore checks if the provided store is valid.
func (f *WrapProvider) ValidateStore(_ esv1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

type MockFakeClient struct {
	id          string
	closeCalled bool
}

func (c *MockFakeClient) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return nil
}

func (c *MockFakeClient) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return nil
}

func (c *MockFakeClient) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, nil
}

func (c *MockFakeClient) GetSecret(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, nil
}

func (c *MockFakeClient) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (c *MockFakeClient) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, nil
}

// GetAllSecrets returns multiple k/v pairs from the provider.
func (c *MockFakeClient) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, nil
}

func (c *MockFakeClient) Close(_ context.Context) error {
	c.closeCalled = true
	return nil
}

func newManagerTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	return scheme
}

func resetGlobalV2ConnectionPoolForTest(t *testing.T) {
	t.Helper()

	if globalV2ConnectionPool != nil {
		_ = globalV2ConnectionPool.Close()
	}
	globalV2ConnectionPool = nil
	globalV2ConnectionPoolOnce = sync.Once{}
	t.Cleanup(func() {
		if globalV2ConnectionPool != nil {
			_ = globalV2ConnectionPool.Close()
		}
		globalV2ConnectionPool = nil
		globalV2ConnectionPoolOnce = sync.Once{}
	})
}

func installGlobalV2ConnectionPoolForTest(t *testing.T) *prometheus.Registry {
	t.Helper()

	resetGlobalV2ConnectionPoolForTest(t)

	globalV2ConnectionPool = providergrpc.NewConnectionPool(providergrpc.PoolConfig{
		MaxIdleTime:         time.Minute,
		MaxLifetime:         time.Minute,
		HealthCheckInterval: 10 * time.Millisecond,
	})

	var once sync.Once
	once.Do(func() {})
	globalV2ConnectionPoolOnce = once

	registry := prometheus.NewRegistry()
	require.NoError(t, providergrpc.RegisterMetrics(registry))

	return registry
}

func assertPoolMetricEventually(t *testing.T, registry *prometheus.Registry, metricName, address string, tlsEnabled bool, want float64) {
	t.Helper()

	labelValue := "false"
	if tlsEnabled {
		labelValue = "true"
	}

	assert.Eventually(t, func() bool {
		got, ok := lookupPoolMetricValue(registry, metricName, address, labelValue)
		return ok && got == want
	}, time.Second, 10*time.Millisecond)
}

func lookupPoolMetricValue(registry *prometheus.Registry, metricName, address, tlsEnabled string) (float64, bool) {
	metricFamilies, err := registry.Gather()
	if err != nil {
		return 0, false
	}

	for _, metricFamily := range metricFamilies {
		if metricFamily.GetName() != metricName {
			continue
		}
		for _, metric := range metricFamily.GetMetric() {
			if metricLabelValue(metric.GetLabel(), "address") != address {
				continue
			}
			if metricLabelValue(metric.GetLabel(), "tls_enabled") != tlsEnabled {
				continue
			}
			if gauge := metric.GetGauge(); gauge != nil {
				return gauge.GetValue(), true
			}
		}
	}

	return 0, false
}

func metricLabelValue(labels []*dto.LabelPair, name string) string {
	for _, label := range labels {
		if label.GetName() == name {
			return label.GetValue()
		}
	}

	return ""
}

type recordingProviderServer struct {
	pb.UnimplementedSecretStoreProviderServer

	mu               sync.Mutex
	validateRequests []*pb.ValidateRequest
}

func newRecordingProviderServer(t *testing.T) (*recordingProviderServer, string, map[string][]byte) {
	t.Helper()

	serverCert, serverKey, clientCert, clientKey, caCert := newMutualTLSArtifacts(t, "127.0.0.1")

	caPool := x509.NewCertPool()
	require.True(t, caPool.AppendCertsFromPEM(caCert))

	tlsCert, err := tls.X509KeyPair(serverCert, serverKey)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	recorder := &recordingProviderServer{}
	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{tlsCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})))
	pb.RegisterSecretStoreProviderServer(grpcServer, recorder)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = lis.Close()
	})

	return recorder, lis.Addr().String(), map[string][]byte{
		"ca.crt":     caCert,
		"client.crt": clientCert,
		"client.key": clientKey,
	}
}

func (s *recordingProviderServer) Validate(_ context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.validateRequests = append(s.validateRequests, req)
	return &pb.ValidateResponse{Valid: true}, nil
}

func (s *recordingProviderServer) LastValidateRequest() *pb.ValidateRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.validateRequests) == 0 {
		return nil
	}
	return s.validateRequests[len(s.validateRequests)-1]
}

func (s *recordingProviderServer) ValidateCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.validateRequests)
}

func newMutualTLSArtifacts(t *testing.T, host string) (serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM, caCertPEM []byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "clientmanager-test-ca",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	serverCertPEM, serverKeyPEM = newSignedCertificateForTest(t, caCert, caKey, 2, host, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
	clientCertPEM, clientKeyPEM = newSignedCertificateForTest(t, caCert, caKey, 3, "client", []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth})
	caCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	return serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM, caCertPEM
}

func newSignedCertificateForTest(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey, serial int64, host string, usages []x509.ExtKeyUsage) ([]byte, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: usages,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return certPEM, keyPEM
}
