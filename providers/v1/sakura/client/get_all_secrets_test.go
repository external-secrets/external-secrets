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

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/sakura/client"
	"github.com/external-secrets/external-secrets/providers/v1/sakura/client/fake"
)

func TestGetAllSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ref       esv1.ExternalSecretFind
		mockSetup func(t *testing.T, mc *fake.MockSecretAPIClient)
		wantMap   map[string][]byte
		wantErr   bool
	}{
		{
			name: "get all secrets without filter",
			ref:  esv1.ExternalSecretFind{},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
				Path: new(string),
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {},
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {},
			wantMap:   nil,
			wantErr:   true,
		},
		{
			name: "list API error",
			ref:  esv1.ExternalSecretFind{},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			tt.mockSetup(t, mockClient)
			client := client.NewClient(mockClient)

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
