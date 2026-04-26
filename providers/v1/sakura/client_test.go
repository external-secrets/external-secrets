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
	"github.com/stretchr/testify/require"
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
			name: "unveilSecret succeeds",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: "data-1"}, nil
				})
			},
			wantData: []byte("data-1"),
			wantErr:  false,
		},
		{
			name: "unveilSecret fails with API error",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
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

			mockClient := fake.NewMockSecretAPIClient(t)
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			data, err := client.GetSecret(context.Background(), tt.ref)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantData, data)
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
					"k8s-secret-key-1": []byte("data-1"),
				},
			},
			data: esfake.PushSecretData{
				SecretKey: "k8s-secret-key-2",
				RemoteKey: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {},
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			// Note: This case will fail when we upgrade encoding/json to v2
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, `{"k8s-secret-key-1":"data-1"}`, params.Value)
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
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			err := client.PushSecret(context.Background(), tt.secret, tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
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
				RemoteKey: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithDeleteFunc(func(_ context.Context, params v1.DeleteSecret) error {
					require.Equal(t, "test-secret-1", params.Name)
					return nil
				})
			},
			wantErr: false,
		},
		{
			name: "secret does not exist",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "delete property from secret",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-1":"value-1","property-2":"value-2"}`}, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.JSONEq(t, `{"property-2":"value-2"}`, params.Value)
					return &v1.Secret{Name: "test-secret-1"}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "delete the only property from secret",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-1":"value-1"}`}, nil
				})
				mc.WithDeleteFunc(func(_ context.Context, params v1.DeleteSecret) error {
					require.Equal(t, "test-secret-1", params.Name)
					return nil
				})
			},
			wantErr: false,
		},
		{
			name: "invalid existing secret JSON",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			name: "property doesn't exist in existing secret",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-2":"value-2"}`}, nil
				})
			},
			wantErr: false,
		},
		{
			name: "secret exists check error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return nil, errors.New("API error")
				})
			},
			wantErr: true,
		},
		{
			name: "unveil API error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
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
			name: "delete API error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithDeleteFunc(func(_ context.Context, params v1.DeleteSecret) error {
					require.Equal(t, "test-secret-1", params.Name)
					return errors.New("API error")
				})
			},
			wantErr: true,
		},
		{
			name: "delete property upsert API error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-1":"value-1","property-2":"value-2"}`}, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.JSONEq(t, `{"property-2":"value-2"}`, params.Value)
					return nil, errors.New("API error")
				})
			},
			wantErr: true,
		},
		{
			name: "delete property delete API error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-1":"value-1"}`}, nil
				})
				mc.WithDeleteFunc(func(_ context.Context, params v1.DeleteSecret) error {
					require.Equal(t, "test-secret-1", params.Name)
					return errors.New("API error")
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient(t)
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			err := client.DeleteSecret(context.Background(), tt.remoteRef)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
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
				RemoteKey: "test-secret-4",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-4"}}, nil
				})
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "secret exists with property",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-1":"value-1"}`}, nil
				})
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "property not found",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-2":"value-2"}`}, nil
				})
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "invalid JSON value",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: "data-1"}, nil
				})
			},
			wantExists: false,
			wantErr:    true,
		},
		{
			name: "unveil API error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
				Property:  "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return nil, errors.New("API error")
				})
			},
			wantExists: false,
			wantErr:    true,
		},
		{
			name: "secret does not exist",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{}, nil
				})
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "list API error",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return nil, errors.New("API error")
				})
			},
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient(t)
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			exists, err := client.SecretExists(context.Background(), tt.remoteRef)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantExists, exists)
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

			mockClient := fake.NewMockSecretAPIClient(t)
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			result, err := client.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantResult, result)
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
				Key: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-1":"value-1","property-2":"value-2"}`}, nil
				})
			},
			wantMap: map[string][]byte{
				"property-1": []byte("value-1"),
				"property-2": []byte("value-2"),
			},
			wantErr: false,
		},
		{
			name: "get secret map with mixed types",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-1":"value-1","property-2":42,"property-3":true}`}, nil
				})
			},
			wantMap: map[string][]byte{
				"property-1": []byte("value-1"),
				"property-2": []byte("42"),
				"property-3": []byte("true"),
			},
			wantErr: false,
		},
		{
			name: "get secret map with property",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "test-secret-1",
				Property: "property-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-1":{"property-2":"value-2","property-3":"value-3"}}`}, nil
				})
			},
			wantMap: map[string][]byte{
				"property-2": []byte("value-2"),
				"property-3": []byte("value-3"),
			},
			wantErr: false,
		},
		{
			name: "invalid JSON format",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: "data-1"}, nil
				})
			},
			wantMap: nil,
			wantErr: true,
		},
		{
			name: "unveil API error",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret-1",
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
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

			mockClient := fake.NewMockSecretAPIClient(t)
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			secretMap, err := client.GetSecretMap(context.Background(), tt.ref)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantMap, secretMap)
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
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}, {Name: "test-secret-2"}}, nil
				})
				mc.WithUnveilFunc(func(ctx context.Context, params v1.Unveil) (*v1.Unveil, error) {
					switch params.Name {
					case "test-secret-1":
						return &v1.Unveil{Value: "data-1"}, nil
					case "test-secret-2":
						return &v1.Unveil{Value: "data-2"}, nil
					}
					require.Fail(t, "unexpected secret name in Unveil call: "+params.Name)
					return &v1.Unveil{}, nil
				})
			},
			wantMap: map[string][]byte{
				"test-secret-1": []byte("data-1"),
				"test-secret-2": []byte("data-2"),
			},
			wantErr: false,
		},
		{
			name: "get all secrets with name filter",
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "^(test-secret-[12])$",
				},
			},
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}, {Name: "test-secret-2"}, {Name: "test-secret-3"}}, nil
				})
				mc.WithUnveilFunc(func(ctx context.Context, params v1.Unveil) (*v1.Unveil, error) {
					switch params.Name {
					case "test-secret-1":
						return &v1.Unveil{Value: "data-1"}, nil
					case "test-secret-2":
						return &v1.Unveil{Value: "data-2"}, nil
					}
					require.Fail(t, "unexpected Unveil call for secret: "+params.Name)
					return &v1.Unveil{}, nil
				})
			},
			wantMap: map[string][]byte{
				"test-secret-1": []byte("data-1"),
				"test-secret-2": []byte("data-2"),
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
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return nil, errors.New("API error")
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
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
			},
			wantMap: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient(t)
			tt.mockSetup(mockClient)
			client := sakura.NewClient(mockClient)

			secretMap, err := client.GetAllSecrets(context.Background(), tt.ref)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantMap, secretMap)
			}
		})
	}
}
