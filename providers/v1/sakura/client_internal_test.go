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

package sakura

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	v1 "github.com/sacloud/secretmanager-api-go/apis/v1"
	"github.com/stretchr/testify/require"

	"github.com/external-secrets/external-secrets/providers/v1/sakura/fake"
)

func TestUnveilSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		secretName string
		version    string
		property   string
		mockSetup  func(*fake.MockSecretAPIClient)
		wantData   []byte
		wantErr    bool
	}{
		{
			name:       "without version or property",
			secretName: "test-secret-1",
			version:    "",
			property:   "",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, v1.OptNilInt{}, params.Version)
					return &v1.Unveil{Value: "data-1"}, nil
				})
			},
			wantData: []byte("data-1"),
			wantErr:  false,
		},
		{
			name:       "with version, without property",
			secretName: "test-secret-1",
			version:    "2",
			property:   "",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, v1.NewOptNilInt(2), params.Version)
					return &v1.Unveil{Value: "data-1"}, nil
				})
			},
			wantData: []byte("data-1"),
			wantErr:  false,
		},
		{
			name:       "without version, with property",
			secretName: "test-secret-1",
			version:    "",
			property:   "property-1",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, v1.OptNilInt{}, params.Version)
					return &v1.Unveil{Value: `{"property-1":"value-1"}`}, nil
				})
			},
			wantData: []byte("value-1"),
			wantErr:  false,
		},
		{
			name:       "without version, with property (JSON object)",
			secretName: "test-secret-1",
			version:    "",
			property:   "property-1",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, v1.OptNilInt{}, params.Version)
					return &v1.Unveil{Value: `{"property-1":{"property-2":"value-2"}}`}, nil
				})
			},
			wantData: []byte(`{"property-2":"value-2"}`),
			wantErr:  false,
		},
		{
			name:       "with version and property",
			secretName: "test-secret-1",
			version:    "2",
			property:   "property-1",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, v1.NewOptNilInt(2), params.Version)
					return &v1.Unveil{Value: `{"property-1":"value-1"}`}, nil
				})
			},
			wantData: []byte("value-1"),
			wantErr:  false,
		},
		{
			name:       "invalid version format",
			secretName: "test-secret-1",
			version:    "invalid",
			property:   "",
			mockSetup:  func(mc *fake.MockSecretAPIClient) {},
			wantData:   nil,
			wantErr:    true,
		},
		{
			name:       "unable to parse secret as JSON",
			secretName: "test-secret-1",
			version:    "",
			property:   "property-1",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: "data-1"}, nil
				})
			},
			wantData: nil,
			wantErr:  true,
		},
		{
			name:       "property not found",
			secretName: "test-secret-1",
			version:    "",
			property:   "property-1",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: `{"property-2":"value-2ß"}`}, nil
				})
			},
			wantData: nil,
			wantErr:  true,
		},
		{
			name:       "unveil API error",
			secretName: "test-secret-1",
			version:    "",
			property:   "",
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
			client := NewClient(mockClient)

			data, err := client.unveilSecret(context.Background(), tt.secretName, tt.version, tt.property)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantData, data)
			}
		})
	}
}

func TestSecretExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		key        string
		property   string
		mockSetup  func(*fake.MockSecretAPIClient)
		wantExists bool
		wantErr    bool
	}{
		{
			name:     "secret exists without property",
			key:      "test-secret-1",
			property: "",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name:     "secret does not exist",
			key:      "test-secret-1",
			property: "",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-2"}}, nil
				})
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name:     "secret exists with property",
			key:      "test-secret-1",
			property: "property-1",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, v1.OptNilInt{}, params.Version)
					return &v1.Unveil{Value: `{"property-1":"value-1"}`}, nil
				})
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name:     "property not found",
			key:      "test-secret-1",
			property: "property-1",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, v1.OptNilInt{}, params.Version)
					return &v1.Unveil{Value: `{"property-2":"value-2"}`}, nil
				})
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name:     "invalid JSON value",
			key:      "test-secret-1",
			property: "property-1",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, v1.OptNilInt{}, params.Version)
					return &v1.Unveil{Value: "data-1"}, nil
				})
			},
			wantExists: false,
			wantErr:    true,
		},
		{
			name:     "list API error",
			key:      "test-secret-1",
			property: "",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return nil, errors.New("API error")
				})
			},
			wantExists: false,
			wantErr:    true,
		},
		{
			name:     "unveil API error",
			key:      "test-secret-1",
			property: "property-1",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, v1.OptNilInt{}, params.Version)
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
			client := NewClient(mockClient)

			exists, err := client.secretExists(context.Background(), tt.key, tt.property)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantExists, exists)
			}
		})
	}
}

func TestUpsertSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		secretName string
		property   string
		value      []byte
		mockSetup  func(*fake.MockSecretAPIClient)
		mergeFunc  func(property string, value json.RawMessage, kv map[string]json.RawMessage)
		wantErr    bool
	}{
		{
			name:       "create secret without property",
			secretName: "test-secret-1",
			property:   "",
			value:      []byte("data-1"),
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, "data-1", params.Value)
					return &v1.Secret{Name: "test-secret-1"}, nil
				})
			},
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				require.Fail(t, "mergeFunc should not be called")
			},
			wantErr: false,
		},
		{
			name:       "create new secret with property",
			secretName: "test-secret-1",
			property:   "property-1",
			value:      []byte("value-1"),
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
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				kv[property] = value
			},
			wantErr: false,
		},
		{
			name:       "create new secret with property, json value",
			secretName: "test-secret-1",
			property:   "property-1",
			value:      []byte(`{"property-2":"value-2"}`),
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
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				kv[property] = value
			},
			wantErr: false,
		},
		{
			name:       "create new secret with property, empty value",
			secretName: "test-secret-1",
			property:   "property-1",
			value:      []byte(""),
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
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				kv[property] = value
			},
			wantErr: false,
		},
		{
			// TODO: This test case will fail when encoding/json is updated to encoding/json/v2
			name:       "create new secret with property, invalid UTF-8 value",
			secretName: "test-secret-1",
			property:   "property-1",
			value:      []byte("value-\xff"),
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
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				kv[property] = value
			},
			wantErr: false,
		},
		{
			name:       "merge property into existing secret",
			secretName: "test-secret-1",
			property:   "property-1",
			value:      []byte("value-1"),
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
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				kv[property] = value
			},
			wantErr: false,
		},
		{
			name:       "overwrite existing property in existing secret",
			secretName: "test-secret-1",
			property:   "property-1",
			value:      []byte("new-value-1"),
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
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				kv[property] = value
			},
			wantErr: false,
		},
		{
			name:       "invalid existing secret JSON",
			secretName: "test-secret-1",
			property:   "property-1",
			value:      []byte("value-1"),
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return &v1.Unveil{Value: "data-1"}, nil
				})
			},
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				require.Fail(t, "mergeFunc should not be called")
			},
			wantErr: true,
		},
		{
			name:       "secret exists check error",
			secretName: "test-secret-1",
			property:   "property-1",
			value:      []byte("value-1"),
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return nil, errors.New("API error")
				})
			},
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				require.Fail(t, "mergeFunc should not be called")
			},
			wantErr: true,
		},
		{
			name:       "unveil API error during merge",
			secretName: "test-secret-1",
			property:   "property-1",
			value:      []byte("value-1"),
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
				mc.WithUnveilFunc(func(_ context.Context, params v1.Unveil) (*v1.Unveil, error) {
					require.Equal(t, "test-secret-1", params.Name)
					return nil, errors.New("API error")
				})
			},
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				require.Fail(t, "mergeFunc should not be called")
			},
			wantErr: true,
		},
		{
			name:       "create API error",
			secretName: "test-secret-1",
			property:   "",
			value:      []byte("data-1"),
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					require.Equal(t, "test-secret-1", params.Name)
					require.Equal(t, "data-1", params.Value)
					return nil, errors.New("API error")
				})
			},
			mergeFunc: func(property string, value json.RawMessage, kv map[string]json.RawMessage) {
				require.Fail(t, "mergeFunc should not be called")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient(t)
			tt.mockSetup(mockClient)
			client := NewClient(mockClient)

			err := client.upsertSecret(context.Background(), tt.secretName, tt.property, tt.value, tt.mergeFunc)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
