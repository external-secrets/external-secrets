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

package truefoundry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	testNamespace = "default"
	testBaseURL   = "https://app.truefoundry.com"
	testTenant    = "my-tenant"
	testSecretKey = "api-key"
)

func makeSecretStore(baseURL, tenant string, ref esmeta.SecretKeySelector) *esv1.SecretStore {
	return &esv1.SecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "tfy"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				TrueFoundry: &esv1.TrueFoundryProvider{
					BaseURL: baseURL,
					Tenant:  tenant,
					Auth: esv1.TrueFoundryAuth{
						SecretRef: esv1.TrueFoundryAuthSecretRef{APIKey: ref},
					},
				},
			},
		},
	}
}

func makeClusterSecretStore(baseURL, tenant string, ref esmeta.SecretKeySelector) *esv1.ClusterSecretStore {
	return &esv1.ClusterSecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{Name: "tfy"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				TrueFoundry: &esv1.TrueFoundryProvider{
					BaseURL: baseURL,
					Tenant:  tenant,
					Auth: esv1.TrueFoundryAuth{
						SecretRef: esv1.TrueFoundryAuthSecretRef{APIKey: ref},
					},
				},
			},
		},
	}
}

func validRef() esmeta.SecretKeySelector {
	return esmeta.SecretKeySelector{Name: "tfy-creds", Key: testSecretKey}
}

func TestValidateStore(t *testing.T) {
	p := &Provider{}

	cases := []struct {
		name      string
		store     esv1.GenericStore
		wantError string
	}{
		{
			name:      "nil store",
			store:     nil,
			wantError: "nil store",
		},
		{
			name: "missing provider config",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
				Spec:     esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{}},
			},
			wantError: "missing provider config",
		},
		{
			name:      "missing baseURL",
			store:     makeSecretStore("", testTenant, validRef()),
			wantError: "baseURL is required",
		},
		{
			name:      "non-http scheme",
			store:     makeSecretStore("ftp://example.com", testTenant, validRef()),
			wantError: "must be a valid http(s) URL",
		},
		{
			name:      "url with no host",
			store:     makeSecretStore("https://", testTenant, validRef()),
			wantError: "must be a valid http(s) URL",
		},
		{
			name:      "missing tenant",
			store:     makeSecretStore(testBaseURL, "  ", validRef()),
			wantError: "tenant is required",
		},
		{
			name:      "missing apiKey.name",
			store:     makeSecretStore(testBaseURL, testTenant, esmeta.SecretKeySelector{Key: testSecretKey}),
			wantError: "apiKey.name is required",
		},
		{
			name:      "missing apiKey.key",
			store:     makeSecretStore(testBaseURL, testTenant, esmeta.SecretKeySelector{Name: "tfy-creds"}),
			wantError: "apiKey.key is required",
		},
		{
			name: "SecretStore with cross-namespace SecretRef",
			store: func() esv1.GenericStore {
				otherNS := "other-ns"
				return makeSecretStore(testBaseURL, testTenant, esmeta.SecretKeySelector{
					Name: "tfy-creds", Key: testSecretKey, Namespace: &otherNS,
				})
			}(),
			wantError: "namespace should either be empty or match",
		},
		{
			name:  "happy path SecretStore",
			store: makeSecretStore(testBaseURL, testTenant, validRef()),
		},
		{
			name:  "happy path ClusterSecretStore",
			store: makeClusterSecretStore(testBaseURL, testTenant, validRef()),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := p.ValidateStore(tc.store)
			if tc.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantError)
		})
	}
}

func TestNewClient(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	t.Run("missing kube secret", func(t *testing.T) {
		kube := fake.NewClientBuilder().WithScheme(scheme).Build()
		p := &Provider{}
		_, err := p.NewClient(context.Background(), makeSecretStore(testBaseURL, testTenant, validRef()), kube, testNamespace)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to resolve truefoundry api key")
	})

	t.Run("nil provider config", func(t *testing.T) {
		kube := fake.NewClientBuilder().WithScheme(scheme).Build()
		p := &Provider{}
		bad := &esv1.SecretStore{
			TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
			Spec:     esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{}},
		}
		_, err := p.NewClient(context.Background(), bad, kube, testNamespace)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing provider config")
	})

	t.Run("happy path", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "tfy-creds", Namespace: testNamespace},
			Data:       map[string][]byte{testSecretKey: []byte("pat-1234")},
		}
		kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
		p := &Provider{}
		c, err := p.NewClient(context.Background(), makeSecretStore(testBaseURL, testTenant, validRef()), kube, testNamespace)
		require.NoError(t, err)
		require.NotNil(t, c)

		tfy, ok := c.(*Client)
		require.True(t, ok, "expected *Client")
		require.Equal(t, testBaseURL, tfy.baseURL)
		require.Equal(t, testTenant, tfy.tenant)
		require.Equal(t, "pat-1234", tfy.apiKey)
		require.Equal(t, defaultMaxConcurrent, tfy.maxConcurrent)
	})
}

func TestNewProviderAndSpec(t *testing.T) {
	require.NotNil(t, NewProvider())
	require.NotNil(t, ProviderSpec().TrueFoundry)
	require.Equal(t, esv1.MaintenanceStatusMaintained, MaintenanceStatus())
	require.Equal(t, esv1.SecretStoreReadOnly, (&Provider{}).Capabilities())
}
