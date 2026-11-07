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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

func TestGetProviderSecretDataExternalSecretMetadata(t *testing.T) {
	tests := []struct {
		kind    string
		extract bool
	}{
		{kind: esv1.SecretStoreKind},
		{kind: esv1.SecretStoreKind, extract: true},
		{kind: esv1.ClusterSecretStoreKind},
		{kind: esv1.ClusterSecretStoreKind, extract: true},
	}
	for _, tt := range tests {
		t.Run(tt.kind+map[bool]string{false: "/data", true: "/extract"}[tt.extract], func(t *testing.T) {
			testProviderSecretDataMetadata(t, tt.kind, tt.extract)
		})
	}
}

func testProviderSecretDataMetadata(t *testing.T, kind string, extract bool) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Name") != "payments" || r.Header.Get("X-Namespace") != "apps" ||
			r.Header.Get("X-Application") != "payments" {
			t.Errorf("missing requesting metadata: %v", r.Header)
		}

		_, _ = w.Write([]byte(`{"password":"value"}`))
	}))
	defer server.Close()

	scheme := runtime.NewScheme()
	require.NoError(t, esv1.AddToScheme(scheme))
	spec := esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{
		Webhook: &esv1.WebhookProvider{
			URL: server.URL,
			Headers: map[string]string{
				"X-Name":        `{{ .externalSecret.name }}`,
				"X-Namespace":   `{{ .externalSecret.namespace }}`,
				"X-Application": `{{ index .externalSecret.labels "example.com/application" }}`,
			},
		},
	}}
	var store client.Object = &esv1.SecretStore{
		Name: "webhook", Namespace: "apps", Spec: spec,
	}
	if kind == esv1.ClusterSecretStoreKind {
		store = &esv1.ClusterSecretStore{Name: "webhook", Spec: spec}
	}
	reconciler := &Reconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(store).Build()}
	resource := &esv1.ExternalSecret{
		Name: "payments", Namespace: "apps",
		Labels: map[string]string{"example.com/application": "payments"},
		Spec: esv1.ExternalSecretSpec{
			SecretStoreRef: esv1.SecretStoreRef{Name: "webhook", Kind: kind},
		},
	}
	if extract {
		resource.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{{
			Extract: &esv1.ExternalSecretDataRemoteRef{Key: "key"},
		}}
	} else {
		resource.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "password", RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "key"},
		}}
	}

	got, err := reconciler.GetProviderSecretData(context.Background(), resource)

	require.NoError(t, err)
	if extract {
		require.Equal(t, []byte("value"), got["password"])
	} else {
		require.JSONEq(t, `{"password":"value"}`, string(got["password"]))
	}
}

func TestGetProviderSecretDataWebhookGeneratorMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Name") != "payments" || r.Header.Get("X-Application") != "payments" {
			t.Errorf("missing requesting metadata in generator request: %v", r.Header)
		}
		_, _ = w.Write([]byte(`{"password":"value"}`))
	}))
	defer server.Close()

	scheme := runtime.NewScheme()
	require.NoError(t, genv1alpha1.AddToScheme(scheme))
	generator := &genv1alpha1.Webhook{
		Name: "generator", Namespace: "apps",
		Spec: genv1alpha1.WebhookSpec{
			URL: server.URL,
			Headers: map[string]string{
				"X-Name":        `{{ .externalSecret.name }}`,
				"X-Application": `{{ index .externalSecret.labels "example.com/application" }}`,
			},
		},
	}
	reconciler := &Reconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(generator).Build(),
		Scheme: scheme,
	}
	resource := &esv1.ExternalSecret{
		Name: "payments", Namespace: "apps",
		Labels: map[string]string{"example.com/application": "payments"},
		Spec: esv1.ExternalSecretSpec{
			DataFrom: []esv1.ExternalSecretDataFromRemoteRef{{
				SourceRef: &esv1.StoreGeneratorSourceRef{GeneratorRef: &esv1.GeneratorRef{
					APIVersion: genv1alpha1.SchemeGroupVersion.String(),
					Kind:       "Webhook", Name: "generator",
				}},
			}},
		},
	}
	got, err := reconciler.GetProviderSecretData(context.Background(), resource)
	require.NoError(t, err)
	require.Equal(t, []byte("value"), got["password"])
}
