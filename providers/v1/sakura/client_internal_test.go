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
		mockSetup  func(*testing.T, *fake.MockSecretAPIClient)
		wantData   []byte
		wantErr    bool
	}{
		{
			name:       "without version or property",
			secretName: "test-secret-1",
			version:    "",
			property:   "",
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup:  func(t *testing.T, mc *fake.MockSecretAPIClient) {},
			wantData:   nil,
			wantErr:    true,
		},
		{
			name:       "unable to parse secret as JSON",
			secretName: "test-secret-1",
			version:    "",
			property:   "property-1",
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			tt.mockSetup(t, mockClient)
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

func TestSecretKeyExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		key        string
		mockSetup  func(*testing.T, *fake.MockSecretAPIClient)
		wantExists bool
		wantErr    bool
	}{
		{
			name: "secret exists",
			key:  "test-secret-1",
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-1"}}, nil
				})
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "secret does not exist",
			key:  "test-secret-1",
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
				mc.WithListFunc(func(_ context.Context) ([]v1.Secret, error) {
					return []v1.Secret{{Name: "test-secret-2"}}, nil
				})
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "list API error",
			key:  "test-secret-1",
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			tt.mockSetup(t, mockClient)
			client := NewClient(mockClient)

			exists, err := client.secretKeyExists(context.Background(), tt.key)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantExists, exists)
			}
		})
	}
}
