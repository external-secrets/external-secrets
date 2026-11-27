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

package webhook

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

func TestGetTemplateDataExternalSecret(t *testing.T) {
	tests := []struct {
		name     string
		resource *esv1.ExternalSecret
		template string
		want     string
		wantErr  bool
	}{
		{
			name: "resource metadata",
			resource: &esv1.ExternalSecret{
				Name: "payments", Namespace: "apps",
				Labels: map[string]string{"example.com/application": "payments"},
			},
			template: `{{ .externalSecret.name }}/{{ .externalSecret.namespace }}/{{ index .externalSecret.labels "example.com/application" }}`,
			want:     "payments/apps/payments",
		},
		{
			name: "missing labels",
			resource: &esv1.ExternalSecret{
				Name: "payments", Namespace: "apps",
			},
			template: `{{ index .externalSecret.labels "missing" }}`,
		},
		{
			name:     "missing resource fails on field access",
			template: `{{ .externalSecret.name }}`,
			wantErr:  true,
		},
		{
			name:     "missing resource can be guarded",
			template: `{{ with .externalSecret }}{{ .name }}{{ else }}none{{ end }}`,
			want:     "none",
		},
		{
			name: "annotations are not exposed",
			resource: &esv1.ExternalSecret{
				Annotations: map[string]string{"internal": "not-exposed"},
			},
			template: `{{ index .externalSecret "annotations" | default "absent" }}`,
			want:     "absent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.resource != nil {
				ctx = esutils.WithExternalSecretMetadata(ctx, tt.resource)
			}
			webhook := &Webhook{}
			data, err := webhook.GetTemplateData(ctx, nil, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ExecuteTemplateString(tt.template, data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected metadata field access to fail without a requesting resource")
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("template result = %q, error = %v, want %q", got, err, tt.want)
			}
		})
	}
}

func TestWebhookExternalSecretMetadataConcurrentRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("name") != r.Header.Get("X-Name") ||
			r.URL.Query().Get("application") != r.Header.Get("X-Application") ||
			r.Header.Get("X-Namespace") != "apps" ||
			string(body) != r.Header.Get("X-Application") {
			t.Errorf("inconsistent request context: query=%v headers=%v body=%q", r.URL.Query(), r.Header, body)
		}
		_, _ = w.Write([]byte(r.Header.Get("X-Name") + "/" + r.Header.Get("X-Application")))
	}))
	defer server.Close()

	webhook := &Webhook{HTTP: server.Client(), Namespace: "apps"}
	provider := &Spec{
		URL:    server.URL + `?name={{ .externalSecret.name }}&application={{ index .externalSecret.labels "example.com/application" }}`,
		Method: http.MethodPost,
		Headers: map[string]string{
			"X-Name":        `{{ .externalSecret.name }}`,
			"X-Namespace":   `{{ .externalSecret.namespace }}`,
			"X-Application": `{{ index .externalSecret.labels "example.com/application" }}`,
		},
		Body: `{{ index .externalSecret.labels "example.com/application" }}`,
	}
	// Parallel subtests share a single client; each operation must retain its own context.
	t.Run("requests", func(t *testing.T) {
		for i := range 20 {
			t.Run(fmt.Sprintf("resource-%d", i), func(t *testing.T) {
				t.Parallel()
				name := fmt.Sprintf("resource-%d", i)
				application := fmt.Sprintf("application %d&other=value", i)
				ctx := esutils.WithExternalSecretMetadata(context.Background(), &esv1.ExternalSecret{
					Name: name, Namespace: "apps",
					Labels: map[string]string{"example.com/application": application},
				})
				got, err := webhook.GetWebhookData(ctx, provider, &esv1.ExternalSecretDataRemoteRef{Key: "key"})
				if err != nil || string(got) != name+"/"+application {
					t.Fatalf("response = %q, error = %v, want %q", got, err, name+"/"+application)
				}
			})
		}
	})
}

func TestWebhookExternalSecretMetadataIsOptIn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		if r.URL.RawQuery != "key=key" || len(body) != 0 {
			t.Errorf("unexpected request URL/body: %s %q", r.URL, body)
		}
		for key, values := range r.Header {
			if strings.Contains(strings.Join(values, " "), "private-context") || strings.HasPrefix(key, "X-") {
				t.Errorf("metadata forwarded without a template: %s=%v", key, values)
			}
		}
		_, _ = w.Write([]byte("value"))
	}))
	defer server.Close()
	webhook := &Webhook{HTTP: server.Client()}
	ctx := esutils.WithExternalSecretMetadata(context.Background(), &esv1.ExternalSecret{
		Name: "private-context", Labels: map[string]string{"application": "private-context"},
	})
	got, err := webhook.GetWebhookData(ctx, &Spec{URL: server.URL + "?key={{ .remoteRef.key }}"}, &esv1.ExternalSecretDataRemoteRef{Key: "key"})
	if err != nil || string(got) != "value" {
		t.Fatalf("response = %q, error = %v, want value", got, err)
	}
}

func TestGetTemplateDataPreservesExternalSecretAlias(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	credentials := &corev1.Secret{
		Name: "credentials", Namespace: "apps",
		Data: map[string][]byte{"name": []byte("legacy-value")},
	}
	webhook := &Webhook{
		Kube:      fake.NewClientBuilder().WithScheme(scheme).WithObjects(credentials).Build(),
		Namespace: "apps",
	}
	ctx := esutils.WithExternalSecretMetadata(context.Background(), &esv1.ExternalSecret{
		Name: "requester",
	})
	data, err := webhook.GetTemplateData(ctx, nil, []Secret{{
		Name: "externalSecret", SecretRef: esmeta.SecretKeySelector{Name: "credentials"},
	}}, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ExecuteTemplateString(`{{ .externalSecret.name }}`, data)
	if err != nil || got != "legacy-value" {
		t.Fatalf("template result = %q, error = %v, want legacy-value", got, err)
	}
}
