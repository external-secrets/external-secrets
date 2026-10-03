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

package grafana

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

const (
	testNamespace = "default"
	testSecret    = "grafana-admin"
)

func newKube(data map[string][]byte) client.Client {
	return clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecret,
			Namespace: testNamespace,
		},
		Data: data,
	}).Build()
}

func TestResolveURL(t *testing.T) {
	kube := newKube(map[string][]byte{
		"url":    []byte("https://from-secret.example.com"),
		"url-nl": []byte("https://from-secret.example.com\n"),
		"blank":  []byte("   \n"),
	})

	tests := []struct {
		name    string
		gen     *genv1alpha1.Grafana
		want    string
		wantErr string
	}{
		{
			name: "literal url",
			gen:  &genv1alpha1.Grafana{Spec: genv1alpha1.GrafanaSpec{URL: "https://grafana.example.com"}},
			want: "https://grafana.example.com",
		},
		{
			name: "literal url trims space",
			gen:  &genv1alpha1.Grafana{Spec: genv1alpha1.GrafanaSpec{URL: "  https://grafana.example.com\n"}},
			want: "https://grafana.example.com",
		},
		{
			name: "urlFrom",
			gen: &genv1alpha1.Grafana{Spec: genv1alpha1.GrafanaSpec{
				URLFrom: &genv1alpha1.SecretKeySelector{Name: testSecret, Key: "url"},
			}},
			want: "https://from-secret.example.com",
		},
		{
			name: "urlFrom trims trailing newline",
			gen: &genv1alpha1.Grafana{Spec: genv1alpha1.GrafanaSpec{
				URLFrom: &genv1alpha1.SecretKeySelector{Name: testSecret, Key: "url-nl"},
			}},
			want: "https://from-secret.example.com",
		},
		{
			name:    "neither url nor urlFrom",
			gen:     &genv1alpha1.Grafana{},
			wantErr: "exactly one of spec.url or spec.urlFrom must be set",
		},
		{
			name: "both url and urlFrom",
			gen: &genv1alpha1.Grafana{Spec: genv1alpha1.GrafanaSpec{
				URL:     "https://grafana.example.com",
				URLFrom: &genv1alpha1.SecretKeySelector{Name: testSecret, Key: "url"},
			}},
			wantErr: "exactly one of spec.url or spec.urlFrom must be set",
		},
		{
			name: "missing secret",
			gen: &genv1alpha1.Grafana{Spec: genv1alpha1.GrafanaSpec{
				URLFrom: &genv1alpha1.SecretKeySelector{Name: "missing", Key: "url"},
			}},
			wantErr: "cannot get Kubernetes secret",
		},
		{
			name: "missing key",
			gen: &genv1alpha1.Grafana{Spec: genv1alpha1.GrafanaSpec{
				URLFrom: &genv1alpha1.SecretKeySelector{Name: testSecret, Key: "nope"},
			}},
			wantErr: "cannot find secret data for key",
		},
		{
			name: "empty secret value",
			gen: &genv1alpha1.Grafana{Spec: genv1alpha1.GrafanaSpec{
				URLFrom: &genv1alpha1.SecretKeySelector{Name: testSecret, Key: "blank"},
			}},
			wantErr: "is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveURL(context.Background(), tt.gen, kube, testNamespace)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewClientURLFrom(t *testing.T) {
	kube := newKube(map[string][]byte{
		"url":   []byte("https://from-secret.example.com"),
		"token": []byte("glsa-test"),
	})
	gen := &genv1alpha1.Grafana{
		Spec: genv1alpha1.GrafanaSpec{
			URLFrom: &genv1alpha1.SecretKeySelector{Name: testSecret, Key: "url"},
			Auth: genv1alpha1.GrafanaAuth{
				Token: &genv1alpha1.SecretKeySelector{Name: testSecret, Key: "token"},
			},
		},
	}

	cl, err := newClient(context.Background(), gen, kube, testNamespace)
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	if cl == nil {
		t.Fatal("expected grafana client")
	}
}
