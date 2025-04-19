/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package onepasswordsdk

import (
	"context"
	"errors"
	"testing"

	"github.com/1password/onepassword-sdk-go"
	"github.com/stretchr/testify/require"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestProviderGetSecret(t *testing.T) {
	tests := []struct {
		name        string
		ref         v1.ExternalSecretDataRemoteRef
		want        []byte
		assertError func(t *testing.T, err error)
		client      func() *onepassword.Client
	}{
		{
			name: "get secret successfully",
			client: func() *onepassword.Client {
				fc := &fakeClient{
					resolveResult: "secret",
				}
				return &onepassword.Client{
					SecretsAPI: fc,
					VaultsAPI:  fc,
				}
			},
			assertError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			ref: v1.ExternalSecretDataRemoteRef{
				Key: "secret",
			},
			want: []byte("secret"),
		},
		{
			name: "get secret with error",
			client: func() *onepassword.Client {
				fc := &fakeClient{
					resolveError: errors.New("fobar"),
				}
				return &onepassword.Client{
					SecretsAPI: fc,
					VaultsAPI:  fc,
				}
			},
			assertError: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "fobar")
			},
			ref: v1.ExternalSecretDataRemoteRef{
				Key: "secret",
			},
		},
		{
			name: "get secret version not implemented",
			client: func() *onepassword.Client {
				fc := &fakeClient{
					resolveResult: "secret",
				}
				return &onepassword.Client{
					SecretsAPI: fc,
					VaultsAPI:  fc,
				}
			},
			ref: v1.ExternalSecretDataRemoteRef{
				Key:     "secret",
				Version: "1",
			},
			assertError: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "is not implemented in the 1Password SDK provider")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				client:      tt.client(),
				vaultPrefix: "op://vault/",
			}
			got, err := p.GetSecret(context.Background(), tt.ref)
			tt.assertError(t, err)
			require.Equal(t, string(got), string(tt.want))
		})
	}
}

func TestProviderGetSecretMap(t *testing.T) {
	tests := []struct {
		name        string
		ref         v1.ExternalSecretDataRemoteRef
		want        map[string][]byte
		assertError func(t *testing.T, err error)
		client      func() *onepassword.Client
	}{
		{
			name: "get secret successfully",
			client: func() *onepassword.Client {
				fc := &fakeClient{
					resolveResult: `{"key": "value"}`,
				}
				return &onepassword.Client{
					SecretsAPI: fc,
					VaultsAPI:  fc,
				}
			},
			assertError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			ref: v1.ExternalSecretDataRemoteRef{
				Key: "secret",
			},
			want: map[string][]byte{
				"key": []byte("value"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				client:      tt.client(),
				vaultPrefix: "op://vault/",
			}
			got, err := p.GetSecretMap(context.Background(), tt.ref)
			tt.assertError(t, err)
			require.Equal(t, got, tt.want)
		})
	}
}

func TestProviderValidate(t *testing.T) {
	tests := []struct {
		name        string
		want        v1.ValidationResult
		assertError func(t *testing.T, err error)
		client      func() *onepassword.Client
		vaultPrefix string
	}{
		{
			name: "validate successfully",
			client: func() *onepassword.Client {
				fc := &fakeClient{
					listAllResult: onepassword.NewIterator[onepassword.VaultOverview](
						[]onepassword.VaultOverview{
							{
								ID:    "test",
								Title: "test",
							},
						},
					),
				}

				return &onepassword.Client{
					SecretsAPI: fc,
					VaultsAPI:  fc,
				}
			},
			want: v1.ValidationResultReady,
			assertError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			vaultPrefix: "op://vault/",
		},
		{
			name: "validate error",
			client: func() *onepassword.Client {
				fc := &fakeClient{
					listAllResult: onepassword.NewIterator[onepassword.VaultOverview](
						[]onepassword.VaultOverview{},
					),
				}

				return &onepassword.Client{
					SecretsAPI: fc,
					VaultsAPI:  fc,
				}
			},
			want: v1.ValidationResultError,
			assertError: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "no vaults found when listing")
			},
			vaultPrefix: "op://vault/",
		},
		{
			name: "validate error missing vault prefix",
			client: func() *onepassword.Client {
				fc := &fakeClient{
					listAllResult: onepassword.NewIterator[onepassword.VaultOverview](
						[]onepassword.VaultOverview{},
					),
				}

				return &onepassword.Client{
					SecretsAPI: fc,
					VaultsAPI:  fc,
				}
			},
			want: v1.ValidationResultError,
			assertError: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "no vaults found when listing")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				client:      tt.client(),
				vaultPrefix: tt.vaultPrefix,
			}
			got, err := p.Validate()
			tt.assertError(t, err)
			require.Equal(t, got, tt.want)
		})
	}
}

type fakeClient struct {
	resolveResult   string
	resolveError    error
	resolveAll      onepassword.ResolveAllResponse
	resolveAllError error
	listAllResult   *onepassword.Iterator[onepassword.VaultOverview]
	listAllError    error
}

func (f *fakeClient) ListAll(ctx context.Context) (*onepassword.Iterator[onepassword.VaultOverview], error) {
	return f.listAllResult, f.listAllError
}

func (f *fakeClient) Resolve(ctx context.Context, secretReference string) (string, error) {
	return f.resolveResult, f.resolveError
}

func (f *fakeClient) ResolveAll(ctx context.Context, secretReferences []string) (onepassword.ResolveAllResponse, error) {
	return f.resolveAll, f.resolveAllError
}

var _ onepassword.SecretsAPI = &fakeClient{}
var _ onepassword.VaultsAPI = &fakeClient{}
