/*
Copyright © 2026 SSH Communications

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

package privx

import (
	"context"
	"encoding/json"
	"errors"
	"regexp/syntax"
	"testing"

	"github.com/SSHcom/privx-sdk-go/v2/api/filters"
	"github.com/SSHcom/privx-sdk-go/v2/api/response"
	"github.com/SSHcom/privx-sdk-go/v2/api/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestSecretsClientGetSecret(t *testing.T) {
	tests := map[string]struct {
		ref     esv1.ExternalSecretDataRemoteRef
		fake    *fakeVaultClient
		want    []byte
		wantErr error
	}{
		"returns selected property": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "example",
				Property: "username",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]any{
						"username": "alice",
						"password": "secret",
					}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			want:    []byte("alice"),
			wantErr: nil,
		},
		"returns property not found": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "example",
				Property: "missing",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]any{
						"username": "alice",
					}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			want:    nil,
			wantErr: ErrPropertyNotFound,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &SecretsClient{
				vault: tc.fake,
			}

			got, err := c.GetSecret(context.Background(), tc.ref)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSecretsClientGetSecret_ReturnsWholeObject(t *testing.T) {
	c := &SecretsClient{
		vault: &fakeVaultClient{
			getSecretFn: func(secretName string) (*vault.Secret, error) {
				data := map[string]any{
					"username": "alice",
					"enabled":  true,
				}
				return &vault.Secret{
					SecretRequest: vault.SecretRequest{
						Data: &data,
					},
				}, nil
			},
		},
	}

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: "example",
	})

	require.NoError(t, err)
	assert.JSONEq(t, `{"enabled":true,"username":"alice"}`, string(got))
}

func TestSecretsClientGetSecret_DataMissing(t *testing.T) {
	c := &SecretsClient{
		vault: &fakeVaultClient{
			getSecretFn: func(secretName string) (*vault.Secret, error) {
				return &vault.Secret{}, nil
			},
		},
	}

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: "example",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSecretDataMissing)
	assert.Nil(t, got)
}

type fakePushSecretData struct {
	secretKey string
	remoteKey string
	property  string
}

func (f *fakePushSecretData) GetMetadata() *apiextensionsv1.JSON {
	return nil
}

func (f *fakePushSecretData) GetSecretKey() string {
	return f.secretKey
}

func (f *fakePushSecretData) GetRemoteKey() string {
	return f.remoteKey
}

func (f *fakePushSecretData) GetProperty() string {
	return f.property
}

func TestSecretsClientPushSecret(t *testing.T) {
	tests := map[string]struct {
		secret  *corev1.Secret
		data    esv1.PushSecretData
		fake    *fakeVaultClient
		wantErr error
	}{
		"missing name": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: ""},
				Data:       map[string][]byte{"key": []byte("value")},
			},
			data: &fakePushSecretData{
				secretKey: "key",
				remoteKey: "",
			},
			wantErr: ErrNoName,
		},
		"success with remote key": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "local"},
				Data:       map[string][]byte{"key": []byte("value")},
			},
			data: &fakePushSecretData{
				secretKey: "key",
				remoteKey: "remote-name",
			},
			fake: &fakeVaultClient{
				createSecretFn: func(req *vault.SecretRequest) (vault.SecretCreate, error) {
					return vault.SecretCreate{}, nil
				},
			},
		},
		"error from vault": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "local"},
				Data:       map[string][]byte{"key": []byte("value")},
			},
			data: &fakePushSecretData{
				secretKey: "key",
				remoteKey: "remote-name",
			},
			fake: &fakeVaultClient{
				createSecretFn: func(req *vault.SecretRequest) (vault.SecretCreate, error) {
					return vault.SecretCreate{}, ErrNotImplemented
				},
			},
			wantErr: ErrNotImplemented,
		},
		// Validates: Requirements 1.1, 1.2, 1.3, 2.1, 2.2, 2.3, 2.4
		"value is plaintext not base64": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "local"},
				Data:       map[string][]byte{"password": []byte("my-password")},
			},
			data: &fakePushSecretData{
				secretKey: "password",
				remoteKey: "remote-name",
			},
			fake: &fakeVaultClient{
				createSecretFn: func(req *vault.SecretRequest) (vault.SecretCreate, error) {
					data := *req.Data
					val := data["password"]

					// Must be string type, not []byte
					strVal, ok := val.(string)
					assert.True(t, ok, "expected string, got %T", val)

					// Must be plaintext
					assert.Equal(t, "my-password", strVal)

					// Must NOT be base64
					assert.NotEqual(t, "bXktcGFzc3dvcmQ=", strVal)

					return vault.SecretCreate{}, nil
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.fake == nil {
				tc.fake = &fakeVaultClient{}
			}

			c := &SecretsClient{
				vault: tc.fake,
			}

			err := c.PushSecret(context.Background(), tc.secret, tc.data)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

type fakePushSecretRemoteRef struct {
	remoteKey string
	property  string
}

func (f *fakePushSecretRemoteRef) GetRemoteKey() string {
	return f.remoteKey
}

func (f *fakePushSecretRemoteRef) GetProperty() string {
	return f.property
}

func TestSecretsClientDeleteSecret(t *testing.T) {
	tests := map[string]struct {
		ctx      context.Context
		ref      esv1.PushSecretRemoteRef
		fake     *fakeVaultClient
		wantErr  error
		errCheck func(t *testing.T, err error)
	}{
		"ok": {
			ctx: context.Background(),
			ref: &fakePushSecretRemoteRef{
				remoteKey: "example-secret",
			},
			fake: &fakeVaultClient{
				deleteSecretFn: func(secretName string) error {
					assert.Equal(t, "example-secret", secretName)
					return nil
				},
			},
		},
		"not found is ignored": {
			ctx: context.Background(),
			ref: &fakePushSecretRemoteRef{
				remoteKey: "missing-secret",
			},
			fake: &fakeVaultClient{
				deleteSecretFn: func(secretName string) error {
					assert.Equal(t, "missing-secret", secretName)
					return errors.New("Secret not found")
				},
			},
		},
		"other error is returned": {
			ctx: context.Background(),
			ref: &fakePushSecretRemoteRef{
				remoteKey: "broken-secret",
			},
			fake: &fakeVaultClient{
				deleteSecretFn: func(secretName string) error {
					return ErrNotImplemented
				},
			},
			wantErr: ErrNotImplemented,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &SecretsClient{
				vault: tc.fake,
			}

			err := c.DeleteSecret(tc.ctx, tc.ref)

			if tc.errCheck != nil {
				tc.errCheck(t, err)
				return
			}

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestSecretsClientGetAllSecrets(t *testing.T) {
	tests := map[string]struct {
		ctx      context.Context
		ref      esv1.ExternalSecretFind
		fake     *fakeVaultClient
		want     map[string][]byte
		wantErr  error
		errCheck func(t *testing.T, err error)
	}{
		"context cancelled": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			ref:     esv1.ExternalSecretFind{},
			fake:    &fakeVaultClient{},
			want:    nil,
			wantErr: context.Canceled,
		},
		"path not implemented": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretFind{
				Path: ptrTo("some/path"),
			},
			fake:    &fakeVaultClient{},
			want:    map[string][]byte{},
			wantErr: ErrNotImplemented,
		},
		"tags not implemented": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretFind{
				Tags: map[string]string{"env": "dev"},
			},
			fake:    &fakeVaultClient{},
			want:    map[string][]byte{},
			wantErr: ErrNotImplemented,
		},
		"conversion strategy not implemented": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretFind{
				ConversionStrategy: esv1.ExternalSecretConversionUnicode,
			},
			fake:    &fakeVaultClient{},
			want:    map[string][]byte{},
			wantErr: ErrNotImplemented,
		},
		"invalid regexp": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "[",
				},
			},
			fake: &fakeVaultClient{},
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid regex")

				var synErr *syntax.Error
				require.ErrorAs(t, err, &synErr)
			},
		},
		"get secrets returns error": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretFind{},
			fake: &fakeVaultClient{
				getSecretsFn: func(opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
					return nil, ErrNotImplemented
				},
			},
			want:    map[string][]byte{},
			wantErr: ErrNotImplemented,
		},
		"get secret returns error": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretFind{},
			fake: &fakeVaultClient{
				getSecretsFn: func(opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
					return &response.ResultSet[vault.Secret]{
						Count: 1,
						Items: []vault.Secret{
							{
								SecretRequest: vault.SecretRequest{
									Name: "alpha",
								},
							},
						},
					}, nil
				},
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					return nil, ErrNotImplemented
				},
			},
			want:    map[string][]byte{},
			wantErr: ErrNotImplemented,
		},
		"secret data missing": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretFind{},
			fake: &fakeVaultClient{
				getSecretsFn: func(opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
					return &response.ResultSet[vault.Secret]{
						Count: 1,
						Items: []vault.Secret{
							{
								SecretRequest: vault.SecretRequest{
									Name: "alpha",
								},
							},
						},
					}, nil
				},
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					return &vault.Secret{}, nil
				},
			},
			want:    map[string][]byte{},
			wantErr: ErrSecretDataMissing,
		},
		"returns matching secrets as json": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "^app-",
				},
			},
			fake: &fakeVaultClient{
				getSecretsFn: func(opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
					return &response.ResultSet[vault.Secret]{
						Count: 2,
						Items: []vault.Secret{
							{
								SecretRequest: vault.SecretRequest{
									Name: "app-one",
								},
							},
							{
								SecretRequest: vault.SecretRequest{
									Name: "db-one",
								},
							},
						},
					}, nil
				},
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					switch secretName {
					case "app-one":
						data := map[string]interface{}{
							"username": "alice",
							"enabled":  true,
						}
						return &vault.Secret{
							SecretRequest: vault.SecretRequest{
								Data: &data,
							},
						}, nil
					default:
						t.Fatalf("unexpected secret name: %s", secretName)
						return nil, nil
					}
				},
			},
			want: map[string][]byte{
				"app-one": []byte(`{"enabled":true,"username":"alice"}`),
			},
			wantErr: nil,
		},
		"empty regexp matches all": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretFind{},
			fake: &fakeVaultClient{
				getSecretsFn: func(opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
					return &response.ResultSet[vault.Secret]{
						Count: 1,
						Items: []vault.Secret{
							{
								SecretRequest: vault.SecretRequest{
									Name: "alpha",
								},
							},
						},
					}, nil
				},
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]interface{}{
						"key": "value",
					}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			want: map[string][]byte{
				"alpha": []byte(`{"key":"value"}`),
			},
			wantErr: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &SecretsClient{
				vault: tc.fake,
			}

			got, err := c.GetAllSecrets(tc.ctx, tc.ref)

			if tc.errCheck != nil {
				tc.errCheck(t, err)
				if tc.want != nil {
					assert.Equal(t, tc.want, got)
				}
				return
			}

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				if tc.want != nil {
					assert.Equal(t, tc.want, got)
				}
				return
			}

			require.NoError(t, err)
			require.Len(t, got, len(tc.want))

			for key, wantJSON := range tc.want {
				gotJSON, ok := got[key]
				require.True(t, ok, "missing key %q", key)
				assert.JSONEq(t, string(wantJSON), string(gotJSON))
			}
		})
	}
}

func ptrTo[T any](v T) *T {
	return &v
}

func TestSecretsClientGetSecretMap(t *testing.T) {
	tests := map[string]struct {
		ctx      context.Context
		ref      esv1.ExternalSecretDataRemoteRef
		fake     *fakeVaultClient
		want     map[string][]byte
		wantErr  error
		errCheck func(t *testing.T, err error)
	}{
		"context cancelled": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			ref:     esv1.ExternalSecretDataRemoteRef{Key: "example"},
			fake:    &fakeVaultClient{},
			wantErr: context.Canceled,
		},
		"get secret returns error": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "example",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					return nil, ErrNotImplemented
				},
			},
			wantErr: ErrNotImplemented,
		},
		"secret data missing": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "example",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					return &vault.Secret{}, nil
				},
			},
			wantErr: ErrSecretDataMissing,
		},
		"returns all top-level keys when property is empty": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "example",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]interface{}{
						"username": "alice",
						"enabled":  true,
						"count":    float64(3),
					}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			want: map[string][]byte{
				"username": []byte("alice"),
				"enabled":  []byte("true"),
				"count":    []byte("3"),
			},
		},
		"returns nested object fields when property points to object": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "example",
				Property: "db",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]interface{}{
						"db": map[string]interface{}{
							"user": "alice",
							"port": float64(5432),
						},
					}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			want: map[string][]byte{
				"user": []byte("alice"),
				"port": []byte("5432"),
			},
		},
		"returns single key value when property points to scalar": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "example",
				Property: "username",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]interface{}{
						"username": "alice",
						"enabled":  true,
					}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			want: map[string][]byte{
				"username": []byte("alice"),
			},
		},
		"property not found": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "example",
				Property: "missing",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]interface{}{
						"username": "alice",
					}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			wantErr: ErrPropertyNotFound,
		},
		"property is nil": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "example",
				Property: "username",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]interface{}{
						"username": nil,
					}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			wantErr: ErrPropertyNotFound,
		},
		"top-level value conversion error": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "example",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]interface{}{
						"broken": make(chan int),
					}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported type")
			},
		},
		"nested value conversion error": {
			ctx: context.Background(),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "example",
				Property: "db",
			},
			fake: &fakeVaultClient{
				getSecretFn: func(secretName string) (*vault.Secret, error) {
					data := map[string]interface{}{
						"db": map[string]interface{}{
							"broken": make(chan int),
						},
					}
					return &vault.Secret{
						SecretRequest: vault.SecretRequest{
							Data: &data,
						},
					}, nil
				},
			},
			errCheck: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported type")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &SecretsClient{
				vault: tc.fake,
			}

			got, err := c.GetSecretMap(tc.ctx, tc.ref)

			if tc.errCheck != nil {
				tc.errCheck(t, err)
				return
			}

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Len(t, got, len(tc.want))
			assert.Equal(t, tc.want, got)
		})
	}
}

// =============================================================================
// Bug Condition Exploration Tests: ClusterSecretStore Namespace Bypass
// These tests encode the EXPECTED behavior after the fix is applied.
// On UNFIXED code, they are EXPECTED TO FAIL — failure confirms the bug exists.
// Validates: Requirements 1.3, 1.4
// =============================================================================

// TestNamespaceBypass_GetSecret_NamespaceList tests that GetSecret rejects
// cross-namespace access when a ClusterSecretStore has namespace conditions
// restricting access to "prod" only, but the requesting namespace is "dev".
func TestNamespaceBypass_GetSecret_NamespaceList(t *testing.T) {
	// ClusterSecretStore with conditions: [{namespaces: ["prod"]}]
	store := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "privx-cluster-store",
		},
		Spec: esv1.SecretStoreSpec{
			Conditions: []esv1.ClusterSecretStoreCondition{
				{
					Namespaces: []string{"prod"},
				},
			},
		},
	}

	// Client requesting from namespace "dev" (NOT in the allowed list)
	c := &SecretsClient{
		vault: &fakeVaultClient{
			getSecretFn: func(secretName string) (*vault.Secret, error) {
				data := map[string]any{"password": "super-secret-value"}
				return &vault.Secret{
					SecretRequest: vault.SecretRequest{
						Data: &data,
					},
				}, nil
			},
		},
		store:     store,
		namespace: "dev",
	}

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "db-password",
		Property: "password",
	})

	// Expected: error containing "cross-namespace access denied"
	// On UNFIXED code: this will FAIL because GetSecret returns the secret without error
	require.Error(t, err, "GetSecret should reject cross-namespace access from namespace 'dev' when only 'prod' is allowed")
	assert.Contains(t, err.Error(), "cross-namespace access denied",
		"error should indicate cross-namespace access is denied")
	assert.Nil(t, got, "no secret data should be returned when namespace is not permitted")
}

// TestNamespaceBypass_GetSecret_NamespaceRegex tests that GetSecret rejects
// cross-namespace access when a ClusterSecretStore has namespace regex conditions
// restricting access to "^prod-.*" only, but the requesting namespace is "dev-team".
func TestNamespaceBypass_GetSecret_NamespaceRegex(t *testing.T) {
	// ClusterSecretStore with conditions: [{namespaceRegexes: ["^prod-.*"]}]
	store := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "privx-cluster-store-regex",
		},
		Spec: esv1.SecretStoreSpec{
			Conditions: []esv1.ClusterSecretStoreCondition{
				{
					NamespaceRegexes: []string{"^prod-.*"},
				},
			},
		},
	}

	// Client requesting from namespace "dev-team" (does NOT match "^prod-.*")
	c := &SecretsClient{
		vault: &fakeVaultClient{
			getSecretFn: func(secretName string) (*vault.Secret, error) {
				data := map[string]any{"api-key": "secret-api-key-123"}
				return &vault.Secret{
					SecretRequest: vault.SecretRequest{
						Data: &data,
					},
				}, nil
			},
		},
		store:     store,
		namespace: "dev-team",
	}

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "api-credentials",
		Property: "api-key",
	})

	// Expected: error containing "cross-namespace access denied"
	// On UNFIXED code: this will FAIL because GetSecret returns the secret without error
	require.Error(t, err, "GetSecret should reject cross-namespace access from namespace 'dev-team' when only '^prod-.*' regex is allowed")
	assert.Contains(t, err.Error(), "cross-namespace access denied",
		"error should indicate cross-namespace access is denied")
	assert.Nil(t, got, "no secret data should be returned when namespace does not match regex")
}

// TestNamespaceBypass_GetAllSecrets_NamespaceList tests that GetAllSecrets rejects
// cross-namespace access when a ClusterSecretStore has namespace conditions
// restricting access to "prod" only, but the requesting namespace is "dev".
func TestNamespaceBypass_GetAllSecrets_NamespaceList(t *testing.T) {
	// ClusterSecretStore with conditions: [{namespaces: ["prod"]}]
	store := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "privx-cluster-store",
		},
		Spec: esv1.SecretStoreSpec{
			Conditions: []esv1.ClusterSecretStoreCondition{
				{
					Namespaces: []string{"prod"},
				},
			},
		},
	}

	// Client requesting from namespace "dev" (NOT in the allowed list)
	c := &SecretsClient{
		vault: &fakeVaultClient{
			getSecretsFn: func(opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
				return &response.ResultSet[vault.Secret]{
					Count: 2,
					Items: []vault.Secret{
						{SecretRequest: vault.SecretRequest{Name: "secret-one"}},
						{SecretRequest: vault.SecretRequest{Name: "secret-two"}},
					},
				}, nil
			},
			getSecretFn: func(secretName string) (*vault.Secret, error) {
				data := map[string]any{"key": "leaked-value"}
				return &vault.Secret{
					SecretRequest: vault.SecretRequest{
						Data: &data,
					},
				}, nil
			},
		},
		store:     store,
		namespace: "dev",
	}

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})

	// Expected: error containing "cross-namespace access denied"
	// On UNFIXED code: this will FAIL because GetAllSecrets returns all secrets without error
	require.Error(t, err, "GetAllSecrets should reject cross-namespace access from namespace 'dev' when only 'prod' is allowed")
	assert.Contains(t, err.Error(), "cross-namespace access denied",
		"error should indicate cross-namespace access is denied")
	assert.Nil(t, got, "no secrets should be returned when namespace is not permitted")
}

// TestNamespaceBypass_GetAllSecrets_NamespaceRegex tests that GetAllSecrets rejects
// cross-namespace access when a ClusterSecretStore has namespace regex conditions
// restricting access to "^prod-.*" only, but the requesting namespace is "dev-team".
func TestNamespaceBypass_GetAllSecrets_NamespaceRegex(t *testing.T) {
	// ClusterSecretStore with conditions: [{namespaceRegexes: ["^prod-.*"]}]
	store := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "privx-cluster-store-regex",
		},
		Spec: esv1.SecretStoreSpec{
			Conditions: []esv1.ClusterSecretStoreCondition{
				{
					NamespaceRegexes: []string{"^prod-.*"},
				},
			},
		},
	}

	// Client requesting from namespace "dev-team" (does NOT match "^prod-.*")
	c := &SecretsClient{
		vault: &fakeVaultClient{
			getSecretsFn: func(opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
				return &response.ResultSet[vault.Secret]{
					Count: 1,
					Items: []vault.Secret{
						{SecretRequest: vault.SecretRequest{Name: "prod-secret"}},
					},
				}, nil
			},
			getSecretFn: func(secretName string) (*vault.Secret, error) {
				data := map[string]any{"token": "leaked-token"}
				return &vault.Secret{
					SecretRequest: vault.SecretRequest{
						Data: &data,
					},
				}, nil
			},
		},
		store:     store,
		namespace: "dev-team",
	}

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})

	// Expected: error containing "cross-namespace access denied"
	// On UNFIXED code: this will FAIL because GetAllSecrets returns secrets without error
	require.Error(t, err, "GetAllSecrets should reject cross-namespace access from namespace 'dev-team' when only '^prod-.*' regex is allowed")
	assert.Contains(t, err.Error(), "cross-namespace access denied",
		"error should indicate cross-namespace access is denied")
	assert.Nil(t, got, "no secrets should be returned when namespace does not match regex")
}

// TestNamespaceBypass_GetSecretMap_NamespaceList tests that GetSecretMap rejects
// cross-namespace access when a ClusterSecretStore has namespace conditions
// restricting access to "prod" only, but the requesting namespace is "dev".
func TestNamespaceBypass_GetSecretMap_NamespaceList(t *testing.T) {
	store := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "privx-cluster-store",
		},
		Spec: esv1.SecretStoreSpec{
			Conditions: []esv1.ClusterSecretStoreCondition{
				{
					Namespaces: []string{"prod"},
				},
			},
		},
	}

	c := &SecretsClient{
		vault: &fakeVaultClient{
			getSecretFn: func(secretName string) (*vault.Secret, error) {
				data := map[string]any{"password": "super-secret-value"}
				return &vault.Secret{
					SecretRequest: vault.SecretRequest{
						Data: &data,
					},
				}, nil
			},
		},
		store:     store,
		namespace: "dev",
	}

	got, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: "db-password",
	})

	require.Error(t, err, "GetSecretMap should reject cross-namespace access from namespace 'dev' when only 'prod' is allowed")
	assert.Contains(t, err.Error(), "cross-namespace access denied",
		"error should indicate cross-namespace access is denied")
	assert.Nil(t, got, "no secret data should be returned when namespace is not permitted")
}

// TestNamespaceBypass_GetSecretMap_NamespaceRegex tests that GetSecretMap rejects
// cross-namespace access when a ClusterSecretStore has namespace regex conditions
// restricting access to "^prod-.*" only, but the requesting namespace is "dev-team".
func TestNamespaceBypass_GetSecretMap_NamespaceRegex(t *testing.T) {
	store := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "privx-cluster-store-regex",
		},
		Spec: esv1.SecretStoreSpec{
			Conditions: []esv1.ClusterSecretStoreCondition{
				{
					NamespaceRegexes: []string{"^prod-.*"},
				},
			},
		},
	}

	c := &SecretsClient{
		vault: &fakeVaultClient{
			getSecretFn: func(secretName string) (*vault.Secret, error) {
				data := map[string]any{"api-key": "secret-api-key-123"}
				return &vault.Secret{
					SecretRequest: vault.SecretRequest{
						Data: &data,
					},
				}, nil
			},
		},
		store:     store,
		namespace: "dev-team",
	}

	got, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: "api-credentials",
	})

	require.Error(t, err, "GetSecretMap should reject cross-namespace access from namespace 'dev-team' when only '^prod-.*' regex is allowed")
	assert.Contains(t, err.Error(), "cross-namespace access denied",
		"error should indicate cross-namespace access is denied")
	assert.Nil(t, got, "no secret data should be returned when namespace does not match regex")
}

func TestAnyToBytes(t *testing.T) {
	tests := map[string]struct {
		input   any
		want    []byte
		wantErr bool
	}{
		"bytes": {
			input: []byte("hello"),
			want:  []byte("hello"),
		},
		"string": {
			input: "hello",
			want:  []byte("hello"),
		},
		"bool true": {
			input: true,
			want:  []byte("true"),
		},
		"bool false": {
			input: false,
			want:  []byte("false"),
		},
		"float64 integer value": {
			input: 42.0,
			want:  []byte("42"),
		},
		"float64 decimal value": {
			input: 3.14,
			want:  []byte("3.14"),
		},
		"json number integer": {
			input: json.Number("123"),
			want:  []byte("123"),
		},
		"json number decimal": {
			input: json.Number("45.67"),
			want:  []byte("45.67"),
		},
		"map marshaled as json": {
			input: map[string]any{
				"user": "alice",
				"ok":   true,
			},
			want: []byte(`{"ok":true,"user":"alice"}`),
		},
		"slice marshaled as json": {
			input: []any{"a", 1.0, true},
			want:  []byte(`["a",1,true]`),
		},
		"nil marshaled as json null": {
			input: nil,
			want:  []byte("null"),
		},
		"unsupported marshal type returns error": {
			input:   make(chan int),
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := anyToBytes(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if len(tc.want) > 0 && (tc.want[0] == '{' || tc.want[0] == '[') {
				assert.JSONEq(t, string(tc.want), string(got))
				return
			}

			assert.Equal(t, tc.want, got)
		})
	}
}
