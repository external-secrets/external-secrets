/*
Copyright © 2026 ESO Maintainer Team

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

package protonpass

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/cache"
)

func TestValidateStore(t *testing.T) {
	tests := []struct {
		name      string
		store     esv1.GenericStore
		wantErr   bool
		errSubstr string
	}{
		{
			name: "nil provider",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: nil,
				},
			},
			wantErr:   true,
			errSubstr: errProtonPassStoreNilSpecProvider,
		},
		{
			name: "nil ProtonPass",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{},
				},
			},
			wantErr:   true,
			errSubstr: errProtonPassStoreNilSpecProviderProtonPass,
		},
		{
			name: "nil auth",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						ProtonPass: &esv1.ProtonPassProvider{
							Username: "user@example.com",
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: errProtonPassStoreNilAuth,
		},
		{
			name: "missing username",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						ProtonPass: &esv1.ProtonPassProvider{
							Auth: &esv1.ProtonPassAuth{
								SecretRef: esv1.ProtonPassAuthSecretRef{
									Password: esmeta.SecretKeySelector{
										Name: "secret",
										Key:  "password",
									},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: errProtonPassStoreMissingUsername,
		},
		{
			name: "missing vault",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						ProtonPass: &esv1.ProtonPassProvider{
							Username: "user@example.com",
							Auth: &esv1.ProtonPassAuth{
								SecretRef: esv1.ProtonPassAuthSecretRef{
									Password: esmeta.SecretKeySelector{
										Name: "secret",
										Key:  "password",
									},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: errProtonPassStoreMissingVault,
		},
		{
			name: "missing password ref name",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						ProtonPass: &esv1.ProtonPassProvider{
							Username: "user@example.com",
							Vault:    "my-vault",
							Auth: &esv1.ProtonPassAuth{
								SecretRef: esv1.ProtonPassAuthSecretRef{
									Password: esmeta.SecretKeySelector{
										Key: "password",
									},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: errProtonPassStoreMissingPasswordRefName,
		},
		{
			name: "missing password ref key",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						ProtonPass: &esv1.ProtonPassProvider{
							Username: "user@example.com",
							Vault:    "my-vault",
							Auth: &esv1.ProtonPassAuth{
								SecretRef: esv1.ProtonPassAuthSecretRef{
									Password: esmeta.SecretKeySelector{
										Name: "secret",
									},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: errProtonPassStoreMissingPasswordRefKey,
		},
		{
			name: "valid minimal config",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						ProtonPass: &esv1.ProtonPassProvider{
							Username: "user@example.com",
							Vault:    "my-vault",
							Auth: &esv1.ProtonPassAuth{
								SecretRef: esv1.ProtonPassAuthSecretRef{
									Password: esmeta.SecretKeySelector{
										Name: "secret",
										Key:  "password",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid full config",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						ProtonPass: &esv1.ProtonPassProvider{
							Username: "user@example.com",
							Vault:    "my-vault",
							Auth: &esv1.ProtonPassAuth{
								SecretRef: esv1.ProtonPassAuthSecretRef{
									Password: esmeta.SecretKeySelector{
										Name: "secret",
										Key:  "password",
									},
									TOTP: &esmeta.SecretKeySelector{
										Name: "secret",
										Key:  "totp",
									},
									ExtraPassword: &esmeta.SecretKeySelector{
										Name: "secret",
										Key:  "extra-password",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &provider{}
			_, err := p.ValidateStore(tc.store)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSessionHomeDir(t *testing.T) {
	key1 := cache.Key{Name: "store-a", Namespace: "ns-a", Kind: "SecretStore"}
	key2 := cache.Key{Name: "store-b", Namespace: "ns-b", Kind: "SecretStore"}

	dir1a := sessionHomeDir(key1)
	dir1b := sessionHomeDir(key1)
	dir2 := sessionHomeDir(key2)

	// Deterministic: same input produces same output.
	assert.Equal(t, dir1a, dir1b)

	// Different inputs produce different paths.
	assert.NotEqual(t, dir1a, dir2)

	// Output is under the expected base directory.
	assert.Contains(t, dir1a, "/tmp/protonpass-sessions/")
	assert.Contains(t, dir2, "/tmp/protonpass-sessions/")
}

func TestCapabilities(t *testing.T) {
	p := &provider{}
	assert.Equal(t, esv1.SecretStoreReadOnly, p.Capabilities())
}
