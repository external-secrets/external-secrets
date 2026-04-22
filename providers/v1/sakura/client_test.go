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

package sakura_test

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/sacloud/secretmanager-api-go/apis/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/providers/v1/sakura"
	"github.com/external-secrets/external-secrets/providers/v1/sakura/fake"
	esfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

func TestGetSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ref       esv1.ExternalSecretDataRemoteRef
		mockSetup func(*fake.MockSecretAPIClient)
		wantData  []byte
		wantErr   bool
	}{
		{
			name: "get secret without version",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue("secret-value")
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantData: []byte("secret-value"),
			wantErr:  false,
		},
		{
			name: "get secret with version",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:     "test-secret",
				Version: "2",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue("secret-value-v2")
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantData: []byte("secret-value-v2"),
			wantErr:  false,
		},
		{
			name: "get secret with property",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "test-secret",
				Property: "password",
				Version:  "1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue(`{"username":"admin","password":"secret123"}`)
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantData: []byte("secret123"),
			wantErr:  false,
		},
		{
			name: "unveil API error",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return nil, errors.New("API error")
				})
			},
			wantData: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient()
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			data, err := client.GetSecret(context.Background(), tt.ref)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantData, data)
			}
		})
	}
}

func TestPushSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		secret    *corev1.Secret
		data      esv1.PushSecretData
		mockSetup func(*fake.MockSecretAPIClient)
		wantErr   bool
	}{
		{
			name: "secret key not found in secret",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("admin"),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "password",
				RemoteKey: "remote-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {},
			wantErr:   true,
		},
		{
			name: "push secret with RemoteRef",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
			data: esfake.PushSecretData{
				RemoteKey: "remote-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					assert.Equal(t, "remote-secret", params.Name)
					assert.Equal(t, `{"key":"value"}`, params.Value)
					return &v1.Secret{Name: "remote-secret"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "push secret with SecretKey",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("admin"),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "username",
				RemoteKey: "remote-username",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					assert.Equal(t, "remote-username", params.Name)
					assert.Equal(t, "admin", params.Value)
					return &v1.Secret{Name: "remote-username"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "push secret with property",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"password": []byte(`"secret123"`),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "password",
				RemoteKey: "remote-secret",
				Property:  "password",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					response := &v1.Unveil{}
					response.SetValue(`{"existing":"value"}`)
					return response, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					assert.Equal(t, "remote-secret", params.Name)
					assert.JSONEq(t, `{"existing":"value","password":"secret123"}`, params.Value)
					return &v1.Secret{Name: "remote-secret"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "create API error",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
			data: esfake.PushSecretData{
				RemoteKey: "remote-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					assert.Equal(t, "remote-secret", params.Name)
					assert.Equal(t, `{"key":"value"}`, params.Value)
					return nil, errors.New("API error")
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient()
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			err := client.PushSecret(context.Background(), tt.secret, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		remoteRef esv1.PushSecretRemoteRef
		mockSetup func(*fake.MockSecretAPIClient)
		wantErr   bool
	}{
		{
			name: "delete secret successfully",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithDeleteFunc(func(_ context.Context, _ v1.DeleteSecret) error {
					return nil
				})
				mc.WithCreateFunc(func(_ context.Context, _ v1.CreateSecret) (*v1.Secret, error) {
					t.Errorf("Create must not be called when deleting an entire secret")
					return &v1.Secret{}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "delete API error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithDeleteFunc(func(_ context.Context, _ v1.DeleteSecret) error {
					return errors.New("API error")
				})
			},
			wantErr: true,
		},
		{
			name: "delete property from secret",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret",
				Property:  "password",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					response := &v1.Unveil{}
					response.SetValue(`{"username":"admin","password":"secret123"}`)
					return response, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					assert.Equal(t, "test-secret", params.Name)
					assert.JSONEq(t, `{"username":"admin"}`, params.Value)
					return &v1.Secret{Name: "test-secret"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "delete property upsert error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret",
				Property:  "password",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					response := &v1.Unveil{}
					response.SetValue(`{"username":"admin","password":"secret123"}`)
					return response, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					assert.Equal(t, "test-secret", params.Name)
					assert.JSONEq(t, `{"username":"admin"}`, params.Value)
					return nil, errors.New("API error")
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient()
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			err := client.DeleteSecret(context.Background(), tt.remoteRef)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecretExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		remoteRef  esv1.PushSecretRemoteRef
		mockSetup  func(*fake.MockSecretAPIClient)
		wantExists bool
		wantErr    bool
	}{
		{
			name: "secret exists without property",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "existing-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{
					{Name: "other-secret"},
					{Name: "existing-secret"},
				}
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return secrets, nil
				})
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "secret does not exist",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "non-existing-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{{Name: "other-secret"}}
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return secrets, nil
				})
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "secret exists with property",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "existing-secret",
				Property:  "password",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{
					{Name: "existing-secret"},
				}
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return secrets, nil
				})
				response := &v1.Unveil{}
				response.SetValue(`{"password":"secret"}`)
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "property not found",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "existing-secret",
				Property:  "missing",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{{Name: "existing-secret"}}
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return secrets, nil
				})
				response := &v1.Unveil{}
				response.SetValue(`{"password":"secret"}`)
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "invalid JSON value",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "existing-secret",
				Property:  "password",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{{Name: "existing-secret"}}
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return secrets, nil
				})
				response := &v1.Unveil{}
				response.SetValue("not-a-json")
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantExists: false,
			wantErr:    true,
		},
		{
			name: "list API error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return nil, errors.New("API error")
				})
			},
			wantExists: false,
			wantErr:    true,
		},
		{
			name: "unveil API error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "existing-secret",
				Property:  "password",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{{Name: "existing-secret"}}
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return secrets, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return nil, errors.New("unveil error")
				})
			},
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient()
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			exists, err := client.SecretExists(context.Background(), tt.remoteRef)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockSetup  func(*fake.MockSecretAPIClient)
		wantResult esv1.ValidationResult
		wantErr    bool
	}{
		{
			name: "validation success",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{}, nil
				})
			},
			wantResult: esv1.ValidationResultReady,
			wantErr:    false,
		},
		{
			name: "validation failure",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return nil, errors.New("API error")
				})
			},
			wantResult: esv1.ValidationResultError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient()
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			result, err := client.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ref       esv1.ExternalSecretDataRemoteRef
		mockSetup func(*fake.MockSecretAPIClient)
		wantMap   map[string][]byte
		wantErr   bool
	}{
		{
			name: "get secret map with string values",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue(`{"username":"admin","password":"secret123"}`)
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantMap: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret123"),
			},
			wantErr: false,
		},
		{
			name: "get secret map with mixed types",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue(`{"name":"test","count":42,"enabled":true}`)
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantMap: map[string][]byte{
				"name":    []byte("test"),
				"count":   []byte("42"),
				"enabled": []byte("true"),
			},
			wantErr: false,
		},
		{
			name: "get secret map with property",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "test-secret",
				Property: "config",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue(`{"operation":"add","config":{"username":"admin","password":"secret123"}}`)
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantMap: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret123"),
			},
			wantErr: false,
		},
		{
			name: "invalid JSON format",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue("not-a-json")
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantMap: nil,
			wantErr: true,
		},
		{
			name: "unveil API error",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return nil, errors.New("API error")
				})
			},
			wantMap: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient()
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			secretMap, err := client.GetSecretMap(context.Background(), tt.ref)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMap, secretMap)
			}
		})
	}
}

func TestGetAllSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ref       esv1.ExternalSecretFind
		mockSetup func(*fake.MockSecretAPIClient)
		wantMap   map[string][]byte
		wantErr   bool
	}{
		{
			name: "get all secrets without filter",
			ref:  esv1.ExternalSecretFind{},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{
					{Name: "secret1"},
					{Name: "secret2"},
				}
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return secrets, nil
				})
				mc.WithUnveilFunc(func(ctx context.Context, params v1.Unveil) (*v1.Unveil, error) {
					response := &v1.Unveil{}
					switch params.Name {
					case "secret1":
						response.SetValue("value1")
					case "secret2":
						response.SetValue("value2")
					}
					return response, nil
				})
			},
			wantMap: map[string][]byte{
				"secret1": []byte("value1"),
				"secret2": []byte("value2"),
			},
			wantErr: false,
		},
		{
			name: "get all secrets with name filter",
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "^test-.*",
				},
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{
					{Name: "test-secret1"},
					{Name: "other-secret"},
					{Name: "test-secret2"},
				}
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return secrets, nil
				})
				mc.WithUnveilFunc(func(ctx context.Context, params v1.Unveil) (*v1.Unveil, error) {
					response := &v1.Unveil{}
					switch params.Name {
					case "test-secret1":
						response.SetValue("value1")
					case "test-secret2":
						response.SetValue("value2")
					}
					return response, nil
				})
			},
			wantMap: map[string][]byte{
				"test-secret1": []byte("value1"),
				"test-secret2": []byte("value2"),
			},
			wantErr: false,
		},
		{
			name: "try to use path filter (not supported)",
			ref: esv1.ExternalSecretFind{
				Path: new("some/path"),
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {},
			wantMap:   nil,
			wantErr:   true,
		},
		{
			name: "try to use tags filter (not supported)",
			ref: esv1.ExternalSecretFind{
				Tags: map[string]string{
					"env": "prod",
				},
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {},
			wantMap:   nil,
			wantErr:   true,
		},
		{
			name: "list API error",
			ref:  esv1.ExternalSecretFind{},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return nil, errors.New("API error")
				})
			},
			wantMap: nil,
			wantErr: true,
		},
		{
			name: "unveil API error",
			ref:  esv1.ExternalSecretFind{},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{
					{Name: "secret1"},
				}
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return secrets, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return nil, errors.New("unveil error")
				})
			},
			wantMap: nil,
			wantErr: true,
		},
		{
			name: "invalid regex pattern",
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "[invalid",
				},
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{}, nil
				})
			},
			wantMap: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient()
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			secretMap, err := client.GetAllSecrets(context.Background(), tt.ref)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMap, secretMap)
			}
		})
	}
}
