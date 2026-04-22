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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
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

func TestBuildCompatibilityStoreSerializesStore(t *testing.T) {
	store := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "aws-prod",
			Namespace:  "team-a",
			UID:        types.UID("uid-1"),
			Generation: 7,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{Kind: "ClusterProviderClass", Name: "aws"},
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{Data: []esv1.FakeProviderData{{Key: "db", Value: "s3cr3t"}}},
			},
		},
	}

	out, err := buildCompatibilityStore(store)
	if err != nil {
		t.Fatalf("buildCompatibilityStore() error = %v", err)
	}
	if out.StoreName != "aws-prod" || out.StoreNamespace != "team-a" || out.StoreKind != esv1.SecretStoreKind {
		t.Fatalf("unexpected compatibility store metadata: %#v", out)
	}
	if out.StoreGeneration != 7 || out.StoreUid != "uid-1" {
		t.Fatalf("unexpected compatibility store identity: %#v", out)
	}
	if !strings.Contains(string(out.StoreSpecJson), "\"runtimeRef\"") || !strings.Contains(string(out.StoreSpecJson), "\"fake\"") {
		t.Fatalf("expected serialized store spec, got %s", string(out.StoreSpecJson))
	}
}

func TestGetFromStoreReturnsErrorWhenRuntimeClassMissing(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))

	store := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "aws-prod",
			Namespace:  "team-a",
			UID:        types.UID("uid-1"),
			Generation: 7,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{Kind: "ClusterProviderClass", Name: "aws"},
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{Data: []esv1.FakeProviderData{{Key: "db", Value: "s3cr3t"}}},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(store).
		Build()

	mgr := NewManager(kubeClient, "", false)

	_, err := mgr.GetFromStore(context.Background(), store, "team-a")
	if err == nil || !strings.Contains(err.Error(), "ClusterProviderClass") {
		t.Fatalf("expected missing ClusterProviderClass error, got %v", err)
	}
}

func TestGetFromStoreRuntimeRefCacheHitSkipsRuntimeLookup(t *testing.T) {
	scheme := newManagerTestScheme(t)
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))

	store := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "aws-prod",
			Namespace:  "team-a",
			UID:        types.UID("uid-1"),
			Generation: 7,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{Kind: esv1.StoreRuntimeRefKindProviderClass, Name: "aws-runtime"},
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{Data: []esv1.FakeProviderData{{Key: "db", Value: "s3cr3t"}}},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(store).Build()
	manager := NewManager(kubeClient, "", false)

	cacheKey := runtimeRefStoreKey(store, "team-a")
	cachedClient := &MockFakeClient{id: "cached"}
	manager.clientMap[cacheKey] = &clientVal{
		client: cachedClient,
		store:  store,
	}

	got, err := manager.GetFromStore(context.Background(), store, "team-a")
	require.NoError(t, err)
	assert.Same(t, cachedClient, got)
}

func TestManagerGetRoutesProviderStoreKinds(t *testing.T) {
	scheme := newManagerTestScheme(t)
	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	manager := NewManager(client, "", false)

	_, err := manager.Get(context.Background(), esv1.SecretStoreRef{Name: "missing", Kind: "ProviderStore"}, "team-a", nil)
	if err == nil || !strings.Contains(err.Error(), "failed to get ProviderStore") {
		t.Fatalf("expected ProviderStore lookup error, got %v", err)
	}

	_, err = manager.Get(context.Background(), esv1.SecretStoreRef{Name: "missing", Kind: "ClusterProviderStore"}, "team-a", nil)
	if err == nil || !strings.Contains(err.Error(), "failed to get ClusterProviderStore") {
		t.Fatalf("expected ClusterProviderStore lookup error, got %v", err)
	}
}

func TestGetStoreDefaultsToSecretStoreForUnknownKind(t *testing.T) {
	scheme := newManagerTestScheme(t)

	store := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-prod",
			Namespace: "team-a",
		},
		Spec: esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{Fake: &esv1.FakeProvider{}}},
	}

	kubeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(store).Build()
	manager := NewManager(kubeClient, "", false)

	got, err := manager.getStore(context.Background(), &esv1.SecretStoreRef{Name: "aws-prod", Kind: "OtherKind"}, "team-a")
	require.NoError(t, err)
	assert.Equal(t, esv1.SecretStoreKind, got.GetKind())
}

func TestGetFromStoreDefaultsRuntimeRefKindToProviderClass(t *testing.T) {
	resetGlobalV2ConnectionPoolForTest(t)

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))

	server, address, tlsSecret := newRecordingProviderServer(t)

	store := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "aws-prod",
			Namespace:  "team-a",
			UID:        types.UID("uid-1"),
			Generation: 7,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{Name: "aws-runtime"},
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{Data: []esv1.FakeProviderData{{Key: "db", Value: "s3cr3t"}}},
			},
		},
	}

	runtimeClass := &esv1alpha1.ProviderClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-runtime",
			Namespace: "team-a",
		},
		Spec: esv1alpha1.ProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: "",
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "", false)

	client, err := mgr.GetFromStore(context.Background(), store, "team-a")
	require.NoError(t, err)

	result, err := client.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)
	assert.Equal(t, 1, server.ValidateCallCount())
}

func TestGetFromStoreRuntimeRefProviderClassUsesStoreNamespace(t *testing.T) {
	resetGlobalV2ConnectionPoolForTest(t)

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))

	server, address, tlsSecret := newRecordingProviderServer(t)

	store := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "aws-prod",
			Namespace:  "team-a",
			UID:        types.UID("uid-1"),
			Generation: 7,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{Kind: esv1.StoreRuntimeRefKindProviderClass, Name: "aws-runtime"},
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{Data: []esv1.FakeProviderData{{Key: "db", Value: "s3cr3t"}}},
			},
		},
	}

	runtimeClass := &esv1alpha1.ProviderClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-runtime",
			Namespace: "team-a",
		},
		Spec: esv1alpha1.ProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: "",
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "", false)

	client, err := mgr.GetFromStore(context.Background(), store, "team-b")
	require.NoError(t, err)

	result, err := client.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)
	assert.Equal(t, 1, server.ValidateCallCount())
	assert.Equal(t, []string{"team-b"}, server.ValidateSourceNamespaces())
}

func TestGetFromStoreRuntimeRefProviderClassMissingReturnsKindedError(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))

	store := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "aws-prod",
			Namespace:  "team-a",
			UID:        types.UID("uid-1"),
			Generation: 7,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{Kind: esv1.StoreRuntimeRefKindProviderClass, Name: "aws"},
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{Data: []esv1.FakeProviderData{{Key: "db", Value: "s3cr3t"}}},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(store).
		Build()

	mgr := NewManager(kubeClient, "", false)

	_, err := mgr.GetFromStore(context.Background(), store, "team-a")
	if err == nil || !strings.Contains(err.Error(), "failed to get ProviderClass") {
		t.Fatalf("expected missing ProviderClass error, got %v", err)
	}
}

func TestGetFromStoreRuntimeRefClusterStoreRejectsProviderClass(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))

	store := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "aws-prod",
			UID:        types.UID("uid-1"),
			Generation: 7,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{Kind: esv1.StoreRuntimeRefKindProviderClass, Name: "aws-runtime"},
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{Data: []esv1.FakeProviderData{{Key: "db", Value: "s3cr3t"}}},
			},
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(store).
		Build()

	mgr := NewManager(kubeClient, "", false)

	_, err := mgr.GetFromStore(context.Background(), store, "team-a")
	if err == nil || !strings.Contains(err.Error(), "ClusterSecretStore runtimeRef.kind must not be \"ProviderClass\"") {
		t.Fatalf("expected ClusterSecretStore ProviderClass error, got %v", err)
	}
}

func TestGetFromStoreWithRuntimeRefReturnsClientThatValidates(t *testing.T) {
	resetGlobalV2ConnectionPoolForTest(t)

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))

	server, address, tlsSecret := newRecordingProviderServer(t)

	store := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "aws-prod",
			Namespace:  "team-a",
			UID:        types.UID("uid-1"),
			Generation: 7,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{Kind: "ClusterProviderClass", Name: "aws-runtime"},
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{Data: []esv1.FakeProviderData{{Key: "db", Value: "s3cr3t"}}},
			},
		},
	}

	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aws-runtime",
		},
		Spec: esv1alpha1.ClusterProviderClassSpec{
			Address: address,
		},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: "",
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "", false)

	client, err := mgr.GetFromStore(context.Background(), store, "team-a")
	require.NoError(t, err)

	result, err := client.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)
	assert.Equal(t, 1, server.ValidateCallCount())
	require.Len(t, mgr.v2PooledConnections, 0)
}

func TestGetFromStoreWithRuntimeRefReusesCachedClient(t *testing.T) {
	resetGlobalV2ConnectionPoolForTest(t)

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))

	server, address, tlsSecret := newRecordingProviderServer(t)

	store := &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "aws-prod",
			Namespace:  "team-a",
			UID:        types.UID("uid-1"),
			Generation: 7,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{Kind: "ClusterProviderClass", Name: "aws-runtime"},
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{Data: []esv1.FakeProviderData{{Key: "db", Value: "s3cr3t"}}},
			},
		},
	}

	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-runtime"},
		Spec:       esv1alpha1.ClusterProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: "",
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "", false)

	firstClient, err := mgr.GetFromStore(context.Background(), store, "team-a")
	require.NoError(t, err)

	secondClient, err := mgr.GetFromStore(context.Background(), store, "team-a")
	require.NoError(t, err)

	assert.Same(t, firstClient, secondClient)
	require.Len(t, mgr.v2PooledConnections, 0)

	result, err := firstClient.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)
	assert.Equal(t, 1, server.ValidateCallCount())
}

func TestGetFromStoreWithRuntimeRefDoesNotReuseClientAcrossSourceNamespaces(t *testing.T) {
	resetGlobalV2ConnectionPoolForTest(t)

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))

	server, address, tlsSecret := newRecordingProviderServer(t)

	store := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "aws-prod",
			UID:        types.UID("uid-1"),
			Generation: 7,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: &esv1.StoreRuntimeRef{Kind: "ClusterProviderClass", Name: "aws-runtime"},
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{Data: []esv1.FakeProviderData{{Key: "db", Value: "s3cr3t"}}},
			},
		},
	}

	runtimeClass := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-runtime"},
		Spec:       esv1alpha1.ClusterProviderClassSpec{Address: address},
	}

	kubeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			store,
			runtimeClass,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "external-secrets-provider-tls",
					Namespace: "",
				},
				Data: tlsSecret,
			},
		).
		Build()

	mgr := NewManager(kubeClient, "", false)

	firstClient, err := mgr.GetFromStore(context.Background(), store, "team-a")
	require.NoError(t, err)

	secondClient, err := mgr.GetFromStore(context.Background(), store, "team-b")
	require.NoError(t, err)

	assert.NotSame(t, firstClient, secondClient)

	result, err := firstClient.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)

	result, err = secondClient.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)

	assert.Equal(t, []string{"team-a", "team-b"}, server.ValidateSourceNamespaces())
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

func (s *recordingProviderServer) ValidateSourceNamespaces() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]string, 0, len(s.validateRequests))
	for _, req := range s.validateRequests {
		out = append(out, req.GetSourceNamespace())
	}

	return out
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
