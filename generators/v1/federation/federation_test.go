// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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
package federation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestGenerator_Generate(t *testing.T) {
	// Create a test server to mock federation server responses
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path == "/generators/test-namespace/test-kind/test-name" {
			// Check authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Return a successful response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"key1": "value1",
				"key2": "value2",
			})
			return
		}

		if r.URL.Path == "/generators/test-namespace/error-kind/error-name" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal server error"))
			return
		}

		// Default: not found
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	// Create a mock Kubernetes client with test secrets
	kube := fake.NewClientBuilder().WithObjects(
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "token-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"token": []byte("test-token"),
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ca-cert-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"ca.crt": []byte("test-ca-cert"),
			},
		},
	).Build()

	// Create test cases
	tests := []struct {
		name       string
		jsonSpec   *apiextensions.JSON
		wantErr    bool
		errMessage string
		wantData   map[string][]byte
	}{
		{
			name:       "nil spec",
			jsonSpec:   nil,
			wantErr:    true,
			errMessage: errNoSpec,
		},
		{
			name: "invalid spec",
			jsonSpec: &apiextensions.JSON{
				Raw: []byte(`invalid json`),
			},
			wantErr:    true,
			errMessage: "unable to parse spec:",
		},
		{
			name: "successful generation",
			jsonSpec: &apiextensions.JSON{
				Raw: mustMarshal(t, &genv1alpha1.Federation{
					Spec: genv1alpha1.FederationSpec{
						Server: genv1alpha1.FederationServer{
							URL: ts.URL,
						},
						Auth: genv1alpha1.FederationAuthKubernetes{
							TokenSecretRef: &esmeta.SecretKeySelector{
								Name: "token-secret",
								Key:  "token",
							},
							CACertSecretRef: &esmeta.SecretKeySelector{
								Name: "ca-cert-secret",
								Key:  "ca.crt",
							},
						},
						Generator: genv1alpha1.FederationGeneratorRef{
							Namespace: "test-namespace",
							Kind:      "test-kind",
							Name:      "test-name",
						},
					},
				}),
			},
			wantErr: false,
			wantData: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
		},
		{
			name: "server error",
			jsonSpec: &apiextensions.JSON{
				Raw: mustMarshal(t, &genv1alpha1.Federation{
					Spec: genv1alpha1.FederationSpec{
						Server: genv1alpha1.FederationServer{
							URL: ts.URL,
						},
						Auth: genv1alpha1.FederationAuthKubernetes{
							TokenSecretRef: &esmeta.SecretKeySelector{
								Name: "token-secret",
								Key:  "token",
							},
						},
						Generator: genv1alpha1.FederationGeneratorRef{
							Namespace: "test-namespace",
							Kind:      "error-kind",
							Name:      "error-name",
						},
					},
				}),
			},
			wantErr:    true,
			errMessage: "federation server returned non-OK status: 500",
		},
		{
			name: "invalid server URL",
			jsonSpec: &apiextensions.JSON{
				Raw: mustMarshal(t, &genv1alpha1.Federation{
					Spec: genv1alpha1.FederationSpec{
						Server: genv1alpha1.FederationServer{
							URL: "http://invalid-url",
						},
						Auth: genv1alpha1.FederationAuthKubernetes{
							TokenSecretRef: &esmeta.SecretKeySelector{
								Name: "token-secret",
								Key:  "token",
							},
						},
						Generator: genv1alpha1.FederationGeneratorRef{
							Namespace: "test-namespace",
							Kind:      "test-kind",
							Name:      "test-name",
						},
					},
				}),
			},
			wantErr:    true,
			errMessage: "failed to call federation server:",
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, _, err := g.Generate(context.Background(), tt.jsonSpec, kube, "default")

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantData, got)
		})
	}
}

func TestGenerator_Cleanup(t *testing.T) {
	// Create a test server to mock federation server responses
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request method
		if r.Method != "DELETE" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Check the request path
		if r.URL.Path == "/generators/test-namespace/test-kind/test-name" {
			// Check authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Return a successful response
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.URL.Path == "/generators/test-namespace/error-kind/error-name" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal server error"))
			return
		}

		// Default: not found
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	// Create a mock Kubernetes client with test secrets
	kube := fake.NewClientBuilder().WithObjects(
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "token-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"token": []byte("test-token"),
			},
		},
	).Build()

	// Create test cases
	tests := []struct {
		name       string
		jsonSpec   *apiextensions.JSON
		wantErr    bool
		errMessage string
	}{
		{
			name:       "nil spec",
			jsonSpec:   nil,
			wantErr:    true,
			errMessage: errNoSpec,
		},
		{
			name: "invalid spec",
			jsonSpec: &apiextensions.JSON{
				Raw: []byte(`invalid json`),
			},
			wantErr:    true,
			errMessage: "unable to parse spec:",
		},
		{
			name: "successful cleanup",
			jsonSpec: &apiextensions.JSON{
				Raw: mustMarshal(t, &genv1alpha1.Federation{
					Spec: genv1alpha1.FederationSpec{
						Server: genv1alpha1.FederationServer{
							URL: ts.URL,
						},
						Auth: genv1alpha1.FederationAuthKubernetes{
							TokenSecretRef: &esmeta.SecretKeySelector{
								Name: "token-secret",
								Key:  "token",
							},
						},
						Generator: genv1alpha1.FederationGeneratorRef{
							Namespace: "test-namespace",
							Kind:      "test-kind",
							Name:      "test-name",
						},
					},
				}),
			},
			wantErr: false,
		},
		{
			name: "server error",
			jsonSpec: &apiextensions.JSON{
				Raw: mustMarshal(t, &genv1alpha1.Federation{
					Spec: genv1alpha1.FederationSpec{
						Server: genv1alpha1.FederationServer{
							URL: ts.URL,
						},
						Auth: genv1alpha1.FederationAuthKubernetes{
							TokenSecretRef: &esmeta.SecretKeySelector{
								Name: "token-secret",
								Key:  "token",
							},
						},
						Generator: genv1alpha1.FederationGeneratorRef{
							Namespace: "test-namespace",
							Kind:      "error-kind",
							Name:      "error-name",
						},
					},
				}),
			},
			wantErr:    true,
			errMessage: "federation server returned non-OK status: 500",
		},
		{
			name: "invalid server URL",
			jsonSpec: &apiextensions.JSON{
				Raw: mustMarshal(t, &genv1alpha1.Federation{
					Spec: genv1alpha1.FederationSpec{
						Server: genv1alpha1.FederationServer{
							URL: "http://invalid-url",
						},
						Auth: genv1alpha1.FederationAuthKubernetes{
							TokenSecretRef: &esmeta.SecretKeySelector{
								Name: "token-secret",
								Key:  "token",
							},
						},
						Generator: genv1alpha1.FederationGeneratorRef{
							Namespace: "test-namespace",
							Kind:      "test-kind",
							Name:      "test-name",
						},
					},
				}),
			},
			wantErr:    true,
			errMessage: "failed to call federation server:",
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			err := g.Cleanup(context.Background(), tt.jsonSpec, nil, kube, "default")

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestParseSpec(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid spec",
			data:    mustMarshal(t, &genv1alpha1.Federation{}),
			wantErr: false,
		},
		{
			name:    "invalid spec",
			data:    []byte(`invalid yaml`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSpec(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetFromSecretRef(t *testing.T) {
	// Create a mock Kubernetes client with test secrets
	kube := fake.NewClientBuilder().WithObjects(
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"test-key": []byte("test-value"),
			},
		},
	).Build()

	tests := []struct {
		name        string
		keySelector *esmeta.SecretKeySelector
		namespace   string
		want        string
		wantErr     bool
	}{
		{
			name: "valid secret ref",
			keySelector: &esmeta.SecretKeySelector{
				Name: "test-secret",
				Key:  "test-key",
			},
			namespace: "default",
			want:      "test-value",
			wantErr:   false,
		},
		{
			name:        "nil secret ref",
			keySelector: nil,
			namespace:   "default",
			want:        "",
			wantErr:     true,
		},
		{
			name: "non-existent secret",
			keySelector: &esmeta.SecretKeySelector{
				Name: "non-existent-secret",
				Key:  "test-key",
			},
			namespace: "default",
			want:      "",
			wantErr:   true,
		},
		{
			name: "non-existent key",
			keySelector: &esmeta.SecretKeySelector{
				Name: "test-secret",
				Key:  "non-existent-key",
			},
			namespace: "default",
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFromSecretRef(context.Background(), tt.keySelector, "", kube, tt.namespace)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Helper function to marshal an object to YAML bytes.
func mustMarshal(t *testing.T, v interface{}) []byte {
	data, err := yaml.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	return data
}
