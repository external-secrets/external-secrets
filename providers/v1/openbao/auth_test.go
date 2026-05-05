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

package openbao

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	bao "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/openbao/fake"
)

// Test OpenBao Namespace logic.
func TestSetAuthNamespace(t *testing.T) {
	store := makeValidSecretStore()

	kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openbao-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"key": []byte("token"),
		},
	}).Build()

	store.Spec.Provider.OpenBao.Auth.Kubernetes.ServiceAccountRef = nil
	store.Spec.Provider.OpenBao.Auth.Kubernetes.SecretRef = &esmeta.SecretKeySelector{
		Name:      "openbao-secret",
		Namespace: new("default"),
		Key:       "key",
	}

	adminNS := "admin"
	teamNS := "admin/team-a"

	type result struct {
		Before string
		During string
		After  string
	}

	type args struct {
		store    *esv1.SecretStore
		expected result
	}
	cases := map[string]struct {
		reason string
		args   args
	}{
		"StoreNoNamespace": {
			reason: "no namespace should ever be set",
			args: args{
				store:    store,
				expected: result{Before: "", During: "", After: ""},
			},
		},
		"StoreWithNamespace": {
			reason: "use the team namespace throughout",
			args: args{
				store: func(store *esv1.SecretStore) *esv1.SecretStore {
					s := store.DeepCopy()
					s.Spec.Provider.OpenBao.Namespace = new(teamNS)
					return s
				}(store),
				expected: result{Before: teamNS, During: teamNS, After: teamNS},
			},
		},
		"StoreWithAuthNamespace": {
			reason: "switch to the auth namespace during login then revert",
			args: args{
				store: func(store *esv1.SecretStore) *esv1.SecretStore {
					s := store.DeepCopy()
					s.Spec.Provider.OpenBao.Auth.Namespace = new(adminNS)
					return s
				}(store),
				expected: result{Before: "", During: adminNS, After: ""},
			},
		},
		"StoreWithSameNamespace": {
			reason: "the admin namespace throughout",
			args: args{
				store: func(store *esv1.SecretStore) *esv1.SecretStore {
					s := store.DeepCopy()
					s.Spec.Provider.OpenBao.Namespace = new(adminNS)
					s.Spec.Provider.OpenBao.Auth.Namespace = new(adminNS)
					return s
				}(store),
				expected: result{Before: adminNS, During: adminNS, After: adminNS},
			},
		},
		"StoreWithDistinctNamespace": {
			reason: "switch from team namespace, to admin, then back",
			args: args{
				store: func(store *esv1.SecretStore) *esv1.SecretStore {
					s := store.DeepCopy()
					s.Spec.Provider.OpenBao.Namespace = new(teamNS)
					s.Spec.Provider.OpenBao.Auth.Namespace = new(adminNS)
					return s
				}(store),
				expected: result{Before: teamNS, During: adminNS, After: teamNS},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			prov := &Provider{
				NewOpenBaoClient: fake.ClientWithLoginMock,
			}

			c, cfg, err := prov.prepareConfig(context.Background(), kube, nil, tc.args.store.Spec.Provider.OpenBao, nil, "default", store.GetObjectKind().GroupVersionKind().Kind)
			if err != nil {
				t.Error(err.Error())
			}

			client, err := getOpenBaoClient(prov, tc.args.store, cfg, "default")
			if err != nil {
				t.Errorf("openbao.useAuthNamespace: failed to create client: %s", err.Error())
			}

			_, err = prov.initClient(context.Background(), c, client, cfg, tc.args.store.Spec.Provider.OpenBao)
			if err != nil {
				t.Errorf("openbao.useAuthNamespace: failed to init client: %s", err.Error())
			}

			c.client = client

			// before auth
			actual := result{
				Before: c.client.Namespace(),
			}

			// during authentication (getting a token)
			resetNS := c.useAuthNamespace(context.Background())
			actual.During = c.client.Namespace()
			resetNS()

			// after getting the token
			actual.After = c.client.Namespace()

			if diff := cmp.Diff(tc.args.expected, actual, cmpopts.EquateComparable()); diff != "" {
				t.Errorf("\n%s\nopenbao.useAuthNamepsace(...): -want namespace, +got namespace:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestCheckTokenErrors(t *testing.T) {
	cases := map[string]struct {
		message string
		secret  *bao.Secret
		err     error
	}{
		"SuccessWithNoData": {
			message: "should not cache if token lookup returned no data",
			secret:  &bao.Secret{},
			err:     nil,
		},
		"Error": {
			message: "should not cache if token lookup errored",
			secret:  nil,
			err:     errors.New(""),
		},
		// This happens when a token is expired and the OpenBao server returns:
		// {"errors":["permission denied"]}
		"NoDataNorError": {
			message: "should not cache if token lookup returned no data nor error",
			secret:  nil,
			err:     nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			token := fake.Token{
				LookupSelfWithContextFn: func(_ context.Context) (*bao.Secret, error) {
					return tc.secret, tc.err
				},
			}

			cached, _, _ := checkToken(context.Background(), token)
			if cached {
				t.Errorf("%v", tc.message)
			}
		})
	}
}

func TestCheckTokenTtl(t *testing.T) {
	cases := map[string]struct {
		message string
		secret  *bao.Secret
		cache   bool
	}{
		"LongTTLExpirable": {
			message: "should cache if expirable token expires far into the future",
			secret: &bao.Secret{
				Data: map[string]any{
					"expire_time": "2024-01-01T00:00:00.000000000Z",
					"ttl":         json.Number("3600"),
					"type":        "service",
				},
			},
			cache: true,
		},
		"ShortTTLExpirable": {
			message: "should not cache if expirable token is about to expire",
			secret: &bao.Secret{
				Data: map[string]any{
					"expire_time": "2024-01-01T00:00:00.000000000Z",
					"ttl":         json.Number("5"),
					"type":        "service",
				},
			},
			cache: false,
		},
		"ZeroTTLExpirable": {
			message: "should not cache if expirable token has TTL of 0",
			secret: &bao.Secret{
				Data: map[string]any{
					"expire_time": "2024-01-01T00:00:00.000000000Z",
					"ttl":         json.Number("0"),
					"type":        "service",
				},
			},
			cache: false,
		},
		"NonExpirable": {
			message: "should cache if token is non-expirable",
			secret: &bao.Secret{
				Data: map[string]any{
					"expire_time": nil,
					"ttl":         json.Number("0"),
					"type":        "service",
				},
			},
			cache: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			token := fake.Token{
				LookupSelfWithContextFn: func(_ context.Context) (*bao.Secret, error) {
					return tc.secret, nil
				},
			}

			cached, _, err := checkToken(context.Background(), token)
			if cached != tc.cache || err != nil {
				t.Errorf("%v: err = %v", tc.message, err)
			}
		})
	}
}

// Test GCP authentication detection logic.
func TestGCPAuthDetection(t *testing.T) {
	tests := []struct {
		name            string
		gcpAuth         *esv1.OpenBaoGCPAuth
		expectedHasAuth bool
		expectError     bool
	}{
		{
			name: "GCP auth configured",
			gcpAuth: &esv1.OpenBaoGCPAuth{
				Role: "test-role",
				Path: "gcp",
			},
			expectedHasAuth: true,
			expectError:     true, // Will error because auth client is not initialized in test
		},
		{
			name:            "No GCP auth configured",
			gcpAuth:         nil,
			expectedHasAuth: false,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client
			c := &client{
				store: &esv1.OpenBaoProvider{
					Auth: &esv1.OpenBaoAuth{
						GCP: tt.gcpAuth,
					},
				},
				// auth: nil (not initialized for test)
			}

			// Test detection logic
			hasAuth, err := setGcpAuthToken(context.Background(), c)

			if hasAuth != tt.expectedHasAuth {
				t.Errorf("setGcpAuthToken() returned hasAuth = %v, want %v", hasAuth, tt.expectedHasAuth)
			}

			if tt.expectError && err == nil {
				t.Errorf("setGcpAuthToken() expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("setGcpAuthToken() unexpected error: %v", err)
			}
		})
	}
}

func TestGCPAuthMountPathDefault(t *testing.T) {
	c := &client{}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "default path when empty",
			path:     "",
			expected: "gcp",
		},
		{
			name:     "custom path",
			path:     "custom-gcp",
			expected: "custom-gcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.getGCPAuthMountPathOrDefault(tt.path)
			if result != tt.expected {
				t.Errorf("getGCPAuthMountPathOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}
