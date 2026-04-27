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

package client_test

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/sacloud/secretmanager-api-go/apis/v1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/sakura/client"
	"github.com/external-secrets/external-secrets/providers/v1/sakura/client/fake"
	esfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

func TestPushSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		secret    *corev1.Secret
		data      esv1.PushSecretData
		mockSetup func(t *testing.T, mc *fake.MockSecretAPIClient)
		wantErr   bool
	}{
		{
			name: "secret key not found in secret",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte("data-1"),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-2",
				RemoteKey: "test-secret-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {},
			wantErr:   true,
		},
		{
			name: "push secret only with remote key",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte("data-1"),
				},
			},
			data: esfake.PushSecretData{
				RemoteKey: "test-secret-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, `{"k8s-secret-key-1":"data-1"}`, params.Value)
					return &v1.Secret{Name: "test-secret-1"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "push secret in a common way",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte("data-1"),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, "data-1", params.Value)
					return &v1.Secret{Name: "test-secret-1"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "push secret with property",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte(`"value-1"`),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-2":"value-2"}`}, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.JSONEq(t, `{"property-1":"value-1","property-2":"value-2"}`, params.Value)
					return &v1.Secret{Name: "test-secret-1"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "push secret with property to new secret",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte(`"value-1"`),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{}, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.JSONEq(t, `{"property-1":"value-1"}`, params.Value)
					return &v1.Secret{Name: "test-secret-1"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "push secret with property and JSON object value",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte(`{"property-2":"value-2"}`),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{}, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.JSONEq(t, `{"property-1":{"property-2":"value-2"}}`, params.Value)
					return &v1.Secret{Name: "test-secret-1"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "push secret with property and empty value",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte(""),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{}, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.JSONEq(t, `{"property-1":""}`, params.Value)
					return &v1.Secret{Name: "test-secret-1"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "push secret with property and invalid UTF-8 value",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte("value-\xff"),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{}, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.JSONEq(t, `{"property-1":"value-\ufffd"}`, params.Value)
					return &v1.Secret{Name: "test-secret-1"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "push secret with property and overwrite existing property",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte(`"new-value-1"`),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-1":"value-1","property-2":"value-2"}`}, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.JSONEq(t, `{"property-1":"new-value-1","property-2":"value-2"}`, params.Value)
					return &v1.Secret{Name: "test-secret-1"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "invalid existing secret JSON",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte(`"value-1"`),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: "data-1"}, nil
				})
			},
			wantErr: true,
		},
		{
			name: "secret exists check error",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte(`"value-1"`),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return nil, errors.New("API error")
				})
			},
			wantErr: true,
		},
		{
			name: "unveil API error",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte(`"value-1"`),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return nil, errors.New("API error")
				})
			},
			wantErr: true,
		},
		{
			name: "create API error",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte("data-1"),
				},
			},
			data: esfake.PushSecretData{
				RemoteKey: "test-secret-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, `{"k8s-secret-key-1":"data-1"}`, params.Value)
					return nil, errors.New("API error")
				})
			},
			wantErr: true,
		},
		{
			name: "create API error when updating existing secret",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"k8s-secret-key-1": []byte(`"value-1"`),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-1",
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-2":"value-2"}`}, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.JSONEq(t, `{"property-1":"value-1","property-2":"value-2"}`, params.Value)
					return nil, errors.New("API error")
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient(t)
			tt.mockSetup(t, mockClient)
			client := client.NewClient(mockClient)

			err := client.PushSecret(context.Background(), tt.secret, tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
