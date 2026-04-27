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

func TestGetSecretMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ref       esv1.ExternalSecretDataRemoteRef
		mockSetup func(t *testing.T, mc *fake.MockSecretAPIClient)
		wantMap   map[string][]byte
		wantErr   bool
	}{
		{
			name: "get secret map with string values",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			tt.mockSetup(t, mockClient)
			client := client.NewClient(mockClient)

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
