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
	"github.com/stretchr/testify/assert"

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
			secretName: "test-secret",
			version:    "",
			property:   "",
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
			name:       "with version, without property",
			secretName: "test-secret",
			version:    "2",
			property:   "",
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
			name:       "without version, with property",
			secretName: "test-secret",
			version:    "",
			property:   "password",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue(`{"username":"user1","password":"pass1"}`)
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantData: []byte("pass1"),
			wantErr:  false,
		},
		{
			name:       "with version and property",
			secretName: "test-secret",
			version:    "3",
			property:   "password",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue(`{"username":"user1","password":"pass1"}`)
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantData: []byte("pass1"),
			wantErr:  false,
		},
		{
			name:       "invalid version format",
			secretName: "test-secret",
			version:    "invalid",
			property:   "",
			mockSetup:  func(mc *fake.MockSecretAPIClient) {},
			wantData:   nil,
			wantErr:    true,
		},
		{
			name:       "unable to parse secret as JSON",
			secretName: "test-secret",
			version:    "",
			property:   "password",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue("not-a-json")
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantData: nil,
			wantErr:  true,
		},
		{
			name:       "property not found",
			secretName: "test-secret",
			version:    "",
			property:   "nonexistent",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue(`{"username":"user1","password":"pass1"}`)
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return response, nil
				})
			},
			wantData: nil,
			wantErr:  true,
		},
		{
			name:       "unveil API error",
			secretName: "test-secret",
			version:    "",
			property:   "",
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
			client := NewClient(mockClient)

			data, err := client.unveilSecret(context.Background(), tt.secretName, tt.version, tt.property)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantData, data)
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
		mergeFunc  func(property string, value []byte, kv map[string]json.RawMessage)
		wantErr    bool
	}{
		{
			name:       "create secret without property",
			secretName: "test-secret",
			property:   "",
			value:      []byte("plain-value"),
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					assert.Equal(t, "test-secret", params.Name)
					assert.Equal(t, "plain-value", params.Value)
					return &v1.Secret{Name: "test-secret"}, nil
				})
			},
			mergeFunc: func(property string, value []byte, kv map[string]json.RawMessage) {
			},
			wantErr: false,
		},
		{
			name:       "merge property into existing secret",
			secretName: "test-secret",
			property:   "password",
			value:      []byte(`"pass1"`),
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					response := &v1.Unveil{}
					response.SetValue(`{"username":"user1"}`)
					return response, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					assert.Equal(t, "test-secret", params.Name)
					assert.JSONEq(t, `{"username":"user1","password":"pass1"}`, params.Value)
					return &v1.Secret{Name: "test-secret"}, nil
				})
			},
			mergeFunc: func(property string, value []byte, kv map[string]json.RawMessage) {
				kv[property] = json.RawMessage(value)
			},
			wantErr: false,
		},
		{
			name:       "delete property from existing secret",
			secretName: "test-secret",
			property:   "password",
			value:      nil,
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					response := &v1.Unveil{}
					response.SetValue(`{"username":"user1","password":"pass1"}`)
					return response, nil
				})
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					assert.Equal(t, "test-secret", params.Name)
					assert.JSONEq(t, `{"username":"user1"}`, params.Value)
					return &v1.Secret{Name: "test-secret"}, nil
				})
			},
			mergeFunc: func(property string, value []byte, kv map[string]json.RawMessage) {
				delete(kv, property)
			},
			wantErr: false,
		},
		{
			name:       "invalid existing secret JSON",
			secretName: "test-secret",
			property:   "password",
			value:      []byte("pass1"),
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					response := &v1.Unveil{}
					response.SetValue("not-a-json")
					return response, nil
				})
			},
			mergeFunc: func(property string, value []byte, kv map[string]json.RawMessage) {
				kv[property] = json.RawMessage(value)
			},
			wantErr: true,
		},
		{
			name:       "unveil API error during merge",
			secretName: "test-secret",
			property:   "password",
			value:      []byte(`"pass1"`),
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveilFunc(func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
					return nil, errors.New("API error")
				})
			},
			mergeFunc: func(property string, value []byte, kv map[string]json.RawMessage) {
				kv[property] = json.RawMessage(value)
			},
			wantErr: true,
		},
		{
			name:       "create API error",
			secretName: "test-secret",
			property:   "",
			value:      []byte("plain-value"),
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithCreateFunc(func(_ context.Context, params v1.CreateSecret) (*v1.Secret, error) {
					assert.Equal(t, "test-secret", params.Name)
					assert.Equal(t, "plain-value", params.Value)
					return nil, errors.New("API error")
				})
			},
			mergeFunc: func(property string, value []byte, kv map[string]json.RawMessage) {
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fake.NewMockSecretAPIClient()
			tt.mockSetup(mockClient)
			client := NewClient(mockClient)

			err := client.upsertSecret(context.Background(), tt.secretName, tt.property, tt.value, tt.mergeFunc)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
