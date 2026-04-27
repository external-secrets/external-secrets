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

func TestGetSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ref       esv1.ExternalSecretDataRemoteRef
		mockSetup func(t *testing.T, mc *fake.MockSecretAPIClient)
		wantData  []byte
		wantErr   bool
	}{
		{
			name: "unveilSecret succeeds",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "test-secret-1",
			},
			mockSetup: func(t *testing.T, mc *fake.MockSecretAPIClient) {
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
			client := client.NewClient(mockClient)

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
