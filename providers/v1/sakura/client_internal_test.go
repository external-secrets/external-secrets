/*
Copyright © 2025 ESO Maintainer Team

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
	"github.com/stretchr/testify/assert"

	"github.com/external-secrets/external-secrets/providers/v1/sakura/fake"
)

func TestUnveilSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		secretName string
		version    string
		mockSetup  func(*fake.MockSecretAPIClient)
		wantData   []byte
		wantErr    bool
	}{
		{
			name:       "without version",
			secretName: "test-secret",
			version:    "",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue("secret-value")
				mc.WithUnveil(response, nil)
			},
			wantData: []byte("secret-value"),
			wantErr:  false,
		},
		{
			name:       "with version",
			secretName: "test-secret",
			version:    "2",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				response := &v1.Unveil{}
				response.SetValue("secret-value-v2")
				mc.WithUnveil(response, nil)
			},
			wantData: []byte("secret-value-v2"),
			wantErr:  false,
		},
		{
			name:       "invalid version format",
			secretName: "test-secret",
			version:    "invalid",
			mockSetup:  func(mc *fake.MockSecretAPIClient) {},
			wantData:   nil,
			wantErr:    true,
		},
		{
			name:       "unveil API error",
			secretName: "test-secret",
			version:    "",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithUnveil(nil, errors.New("API error"))
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
			client := &Client{
				api: mockClient,
			}

			data, err := client.unveilSecret(context.Background(), tt.secretName, tt.version)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantData, data)
			}
		})
	}
}

func TestSecretExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		secretName string
		mockSetup  func(*fake.MockSecretAPIClient)
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "secret exists",
			secretName: "existing-secret",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{
					{Name: "other-secret"},
					{Name: "existing-secret"},
					{Name: "another-secret"},
				}
				mc.WithList(secrets, nil)
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "secret does not exist",
			secretName: "non-existing-secret",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				secrets := []v1.Secret{
					{Name: "other-secret"},
					{Name: "another-secret"},
				}
				mc.WithList(secrets, nil)
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name:       "empty list",
			secretName: "any-secret",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithList([]v1.Secret{}, nil)
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name:       "list API error",
			secretName: "test-secret",
			mockSetup: func(mc *fake.MockSecretAPIClient) {
				mc.WithList(nil, errors.New("API error"))
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
			client := &Client{
				api: mockClient,
			}

			exists, err := client.secretExists(context.Background(), tt.secretName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)
			}
		})
	}
}
