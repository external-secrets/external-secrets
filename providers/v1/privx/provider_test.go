/*
Copyright © 2026 SSH Communications

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

package privx

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var errFakeNewClient = errors.New("fake new client error")

func TestProviderNewClient(t *testing.T) {
	tests := map[string]struct {
		namespace string
		store     esv1.SecretStore
		wantErr   error
	}{
		"ok": {
			namespace: "default",
			store: esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						PrivX: &esv1.PrivxProvider{
							Host: "https://privx.example.com",
						},
					},
				},
			},
			wantErr: nil,
		},
		"factory returns error": {
			namespace: "default",
			store: esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						PrivX: &esv1.PrivxProvider{
							Host: "https://privx.example.com",
						},
					},
				},
			},
			wantErr: errFakeNewClient,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var gotNamespace string
			var gotStore esv1.GenericStore

			p := &provider{
				newClient: func(
					ctx context.Context,
					store esv1.GenericStore,
					kube kclient.Client,
					namespace string,
				) (esv1.SecretsClient, error) {
					gotNamespace = namespace
					gotStore = store

					if name == "factory returns error" {
						return nil, errFakeNewClient
					}

					return &fakeSecretsClient{}, nil
				},
			}

			client, err := p.NewClient(
				context.Background(),
				&tc.store,
				nil,
				tc.namespace,
			)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, client)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
			assert.IsType(t, &fakeSecretsClient{}, client)
			assert.Equal(t, tc.namespace, gotNamespace)
			assert.Equal(t, &tc.store, gotStore)
		})
	}
}

func TestMaintenanceStatus(t *testing.T) {
	got := MaintenanceStatus()
	assert.Equal(t, esv1.MaintenanceStatusMaintained, got)
}

func TestProviderCapabilities(t *testing.T) {
	p := &provider{}

	got := p.Capabilities()

	assert.Equal(t, esv1.SecretStoreReadWrite, got)
}

func TestProviderValidateStore(t *testing.T) {
	tests := map[string]struct {
		store   esv1.SecretStore
		wantErr error
	}{
		"missing provider": {
			store: esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: nil,
				},
			},
			wantErr: errNoStoreAuth{Field: "spec.provider"},
		},
		"missing privx provider": {
			store: esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{},
				},
			},
			wantErr: errNoStoreAuth{Field: "spec.provider.privx"},
		},
		"missing host": {
			store: esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						PrivX: &esv1.PrivxProvider{},
					},
				},
			},
			wantErr: errNoStoreAuth{Field: "spec.provider.privx.host"},
		},
		"valid store": {
			store: esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						PrivX: &esv1.PrivxProvider{
							Host: "https://privx.example.com",
						},
					},
				},
			},
			wantErr: nil,
		},
	}

	p := &provider{}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotWarnings, gotErr := p.ValidateStore(&tc.store)

			assert.Nil(t, gotWarnings)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}
