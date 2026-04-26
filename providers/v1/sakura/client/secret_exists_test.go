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
	v1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/providers/v1/sakura/client"
	"github.com/external-secrets/external-secrets/providers/v1/sakura/client/fake"
)

func TestSecretExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		remoteRef  esv1.PushSecretRemoteRef
		mockSetup  func(*testing.T, *fake.MockSecretAPIClient)
		wantExists bool
		wantErr    bool
	}{
		{
			name: "secret exists without property",
			remoteRef: v1alpha1.PushSecretRemoteRef{
				RemoteKey: "test-secret-4",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			client := client.NewClient(mockClient)

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
