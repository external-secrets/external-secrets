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

package secretstore

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

func TestManagerGet(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = esv1beta1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)

	// We have a test provider to control
	// the behavior of the NewClient func.
	fakeProvider := &WrapProvider{}
	esv1beta1.ForceRegister(fakeProvider, &esv1beta1.SecretStoreProvider{
		AWS: &esv1beta1.AWSProvider{},
	})

	// fake clients are re-used to compare the
	// in-memory reference
	clientA := &MockFakeClient{id: "1"}
	clientB := &MockFakeClient{id: "2"}

	const testNamespace = "foo"

	readyStatus := esv1beta1.SecretStoreStatus{
		Conditions: []esv1beta1.SecretStoreStatusCondition{
			{
				Type:   esv1beta1.SecretStoreReady,
				Status: corev1.ConditionTrue,
			},
		},
	}

	fakeSpec := esv1beta1.SecretStoreSpec{
		Provider: &esv1beta1.SecretStoreProvider{
			AWS: &esv1beta1.AWSProvider{},
		},
	}

	defaultStore := &esv1beta1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1beta1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: testNamespace,
		},
		Spec:   fakeSpec,
		Status: readyStatus,
	}

	otherStore := &esv1beta1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1beta1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other",
			Namespace: testNamespace,
		},
		Spec:   fakeSpec,
		Status: readyStatus,
	}

	var mgr *Manager

	provKey := clientKey{
		providerType: "*secretstore.WrapProvider",
	}

	type fields struct {
		client    client.Client
		clientMap map[clientKey]*clientVal
	}
	type args struct {
		storeRef  esv1beta1.SecretStoreRef
		namespace string
		sourceRef *esv1beta1.StoreGeneratorSourceRef
	}
	tests := []struct {
		name              string
		fields            fields
		args              args
		clientConstructor func(
			ctx context.Context,
			store esv1beta1.GenericStore,
			kube client.Client,
			namespace string) (esv1beta1.SecretsClient, error)
		verify     func(esv1beta1.SecretsClient)
		afterClose func()
		want       esv1beta1.SecretsClient
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
				storeRef: esv1beta1.SecretStoreRef{
					Name: defaultStore.Name,
					Kind: esv1beta1.SecretStoreKind,
				},
				namespace: defaultStore.Namespace,
				sourceRef: nil,
			},
			clientConstructor: func(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
				return clientA, nil
			},
			verify: func(sc esv1beta1.SecretsClient) {
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
				storeRef: esv1beta1.SecretStoreRef{
					Name: defaultStore.Name,
					Kind: esv1beta1.SecretStoreKind,
				},
				// this should take precedence
				sourceRef: &esv1beta1.StoreGeneratorSourceRef{
					SecretStoreRef: &esv1beta1.SecretStoreRef{
						Name: otherStore.Name,
						Kind: esv1beta1.SecretStoreKind,
					},
				},
				namespace: defaultStore.Namespace,
			},
			clientConstructor: func(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
				return clientB, nil
			},
			verify: func(sc esv1beta1.SecretsClient) {
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
				storeRef: esv1beta1.SecretStoreRef{
					Name: defaultStore.Name,
					Kind: esv1beta1.SecretStoreKind,
				},
				namespace: defaultStore.Namespace,
				sourceRef: nil,
			},
			clientConstructor: func(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
				// constructor should not be called,
				// the client from the cache should be returned instead
				t.Fail()
				return nil, nil
			},
			verify: func(sc esv1beta1.SecretsClient) {
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
				storeRef: esv1beta1.SecretStoreRef{
					Name: otherStore.Name,
					Kind: esv1beta1.SecretStoreKind,
				},
				namespace: otherStore.Namespace,
				sourceRef: nil,
			},
			clientConstructor: func(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
				// because there is a store mismatch
				// we create a new client
				return clientB, nil
			},
			verify: func(sc esv1beta1.SecretsClient) {
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

type WrapProvider struct {
	newClientFunc func(
		context.Context,
		esv1beta1.GenericStore,
		client.Client,
		string) (esv1beta1.SecretsClient, error)
}

// NewClient constructs a SecretsManager Provider.
func (f *WrapProvider) NewClient(
	ctx context.Context,
	store esv1beta1.GenericStore,
	kube client.Client,
	namespace string) (esv1beta1.SecretsClient, error) {
	return f.newClientFunc(ctx, store, kube, namespace)
}

func (f *WrapProvider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

// ValidateStore checks if the provided store is valid.
func (f *WrapProvider) ValidateStore(_ esv1beta1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

type MockFakeClient struct {
	id          string
	closeCalled bool
}

func (c *MockFakeClient) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return nil
}

func (c *MockFakeClient) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return nil
}

func (c *MockFakeClient) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, nil
}

func (c *MockFakeClient) GetSecret(_ context.Context, _ esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, nil
}

func (c *MockFakeClient) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (c *MockFakeClient) GetSecretMap(_ context.Context, _ esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, nil
}

// GetAllSecrets returns multiple k/v pairs from the provider.
func (c *MockFakeClient) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, nil
}

func (c *MockFakeClient) Close(_ context.Context) error {
	c.closeCalled = true
	return nil
}
