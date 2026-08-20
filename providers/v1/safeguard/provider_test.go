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

package safeguard

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestNormalizeAppliance(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    string
		wantErr bool
	}{
		"adds https scheme": {
			input: "safeguard.example.com",
			want:  "https://safeguard.example.com",
		},
		"accepts https url": {
			input: "https://safeguard.example.com",
			want:  "https://safeguard.example.com",
		},
		"rejects http": {
			input:   "http://safeguard.example.com",
			wantErr: true,
		},
		"rejects empty": {
			input:   "  ",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := normalizeAppliance(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestValidateStore(t *testing.T) {
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Safeguard: &esv1.SafeguardProvider{
					Appliance: "safeguard.example.com",
					Auth: esv1.SafeguardAuth{
						A2A: &esv1.SafeguardA2AAuth{
							Certificate: esv1.SafeguardProviderSecretRef{
								SecretRef: &esmeta.SecretKeySelector{
									Name: "cert",
									Key:  "tls.crt",
								},
							},
						},
					},
				},
			},
		},
	}

	provider := &Provider{}
	_, err := provider.ValidateStore(store)
	require.NoError(t, err)
}

func TestValidateStoreMissingCertificate(t *testing.T) {
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Safeguard: &esv1.SafeguardProvider{
					Appliance: "safeguard.example.com",
					Auth: esv1.SafeguardAuth{
						A2A: &esv1.SafeguardA2AAuth{},
					},
				},
			},
		},
	}

	provider := &Provider{}
	_, err := provider.ValidateStore(store)
	require.Error(t, err)
}

func TestConfigDependsOnNamespace(t *testing.T) {
	cfg := &esv1.SafeguardProvider{
		Auth: esv1.SafeguardAuth{
			A2A: &esv1.SafeguardA2AAuth{
				Certificate: esv1.SafeguardProviderSecretRef{
					SecretRef: &esmeta.SecretKeySelector{Name: "cert", Key: "tls.crt"},
				},
			},
		},
	}
	assert.True(t, configDependsOnNamespace(cfg))
}

func TestGetSecretUsesAPIKeyDirectly(t *testing.T) {
	fake := &fakeA2A{
		passwords: map[string]string{
			"api-key-1": "secret-password",
		},
	}
	client := &secretsClient{a2a: fake}

	value, err := client.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: "api-key-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "secret-password", string(value))
}

func TestGetSecretAccountLookup(t *testing.T) {
	fake := &fakeA2A{
		accounts: []accountEntry{
			{accountName: "dbadmin", assetName: "database", apiKey: "lookup-key", password: "lookup-password"},
		},
	}
	client := &secretsClient{a2a: fake}

	value, err := client.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key: "dbadmin/database",
	})
	require.NoError(t, err)
	assert.Equal(t, "lookup-password", string(value))
}
