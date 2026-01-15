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

package onepasswordsdk

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
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
			got, err := p.GetSecret(t.Context(), tt.ref)
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
			name: "get secret successfully for files",
			client: func() *onepassword.Client {
				fc := &fakeClient{}
				fl := &fakeLister{
					listAllResult: []onepassword.ItemOverview{
						{
							ID:       "test-item-id",
							Title:    "key",
							Category: "login",
							VaultID:  "vault-id",
						},
					},
					getResult: onepassword.Item{
						ID:       "test-item-id",
						Title:    "key",
						Category: "login",
						VaultID:  "vault-id",
						Files: []onepassword.ItemFile{
							{
								Attributes: onepassword.FileAttributes{
									Name: "name",
									ID:   "id",
								},
								FieldID: "field-id",
							},
						},
					},
					fileLister: &fakeFileLister{
						readContent: []byte("content"),
					},
				}
				return &onepassword.Client{
					SecretsAPI: fc,
					ItemsAPI:   fl,
					VaultsAPI:  fc,
				}
			},
			assertError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			ref: v1.ExternalSecretDataRemoteRef{
				Key:      "key",
				Property: "file/name",
			},
			want: map[string][]byte{
				"name": []byte("content"),
			},
		},
		{
			name: "get secret successfully for fields",
			client: func() *onepassword.Client {
				fc := &fakeClient{}
				fl := &fakeLister{
					listAllResult: []onepassword.ItemOverview{
						{
							ID:       "test-item-id",
							Title:    "key",
							Category: "login",
							VaultID:  "vault-id",
						},
					},
					getResult: onepassword.Item{
						ID:       "test-item-id",
						Title:    "key",
						Category: "login",
						VaultID:  "vault-id",
						Fields: []onepassword.ItemField{
							{
								ID:        "field-id",
								Title:     "name",
								FieldType: onepassword.ItemFieldTypeConcealed,
								Value:     "value",
							},
						},
					},
					fileLister: &fakeFileLister{
						readContent: []byte("content"),
					},
				}
				return &onepassword.Client{
					SecretsAPI: fc,
					ItemsAPI:   fl,
					VaultsAPI:  fc,
				}
			},
			assertError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			ref: v1.ExternalSecretDataRemoteRef{
				Key:      "key",
				Property: "field/name",
			},
			want: map[string][]byte{
				"name": []byte("value"),
			},
		},
		{
			name: "get secret fails with fields with same title",
			client: func() *onepassword.Client {
				fc := &fakeClient{}
				fl := &fakeLister{
					listAllResult: []onepassword.ItemOverview{
						{
							ID:       "test-item-id",
							Title:    "key",
							Category: "login",
							VaultID:  "vault-id",
						},
					},
					getResult: onepassword.Item{
						ID:       "test-item-id",
						Title:    "key",
						Category: "login",
						VaultID:  "vault-id",
						Fields: []onepassword.ItemField{
							{
								ID:        "field-id",
								Title:     "name",
								FieldType: onepassword.ItemFieldTypeConcealed,
								Value:     "value",
							},
							{
								ID:        "field-id",
								Title:     "name",
								FieldType: onepassword.ItemFieldTypeConcealed,
								Value:     "value",
							},
						},
					},
					fileLister: &fakeFileLister{
						readContent: []byte("content"),
					},
				}
				return &onepassword.Client{
					SecretsAPI: fc,
					ItemsAPI:   fl,
					VaultsAPI:  fc,
				}
			},
			assertError: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "found more than 1 fields with title 'name' in 'key', got 2")
			},
			ref: v1.ExternalSecretDataRemoteRef{
				Key:      "key",
				Property: "field/name",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				client:      tt.client(),
				vaultPrefix: "op://vault/",
			}
			got, err := p.GetSecretMap(t.Context(), tt.ref)
			tt.assertError(t, err)
			require.Equal(t, tt.want, got)
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
					listAllResult: []onepassword.VaultOverview{
						{
							ID:    "test",
							Title: "test",
						},
					},
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

func TestPushSecret(t *testing.T) {
	fc := &fakeClient{
		listAllResult: []onepassword.VaultOverview{
			{
				ID:    "test",
				Title: "test",
			},
		},
	}

	tests := []struct {
		name         string
		ref          v1alpha1.PushSecretData
		secret       *corev1.Secret
		assertError  func(t *testing.T, err error)
		lister       func() *fakeLister
		assertLister func(t *testing.T, lister *fakeLister)
	}{
		{
			name: "create is called",
			lister: func() *fakeLister {
				return &fakeLister{
					listAllResult: []onepassword.ItemOverview{},
				}
			},
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"foo": []byte("bar"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
			},
			ref: v1alpha1.PushSecretData{
				Match: v1alpha1.PushSecretMatch{
					SecretKey: "foo",
					RemoteRef: v1alpha1.PushSecretRemoteRef{
						RemoteKey: "key",
					},
				},
			},
			assertError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			assertLister: func(t *testing.T, lister *fakeLister) {
				assert.True(t, lister.createCalled)
			},
		},
		{
			name: "update is called",
			lister: func() *fakeLister {
				return &fakeLister{
					listAllResult: []onepassword.ItemOverview{
						{
							ID:       "test-item-id",
							Title:    "key",
							Category: "login",
							VaultID:  "vault-id",
						},
					},
				}
			},
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"foo": []byte("bar"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "default",
				},
			},
			ref: v1alpha1.PushSecretData{
				Match: v1alpha1.PushSecretMatch{
					SecretKey: "foo",
					RemoteRef: v1alpha1.PushSecretRemoteRef{
						RemoteKey: "key",
					},
				},
			},
			assertError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			assertLister: func(t *testing.T, lister *fakeLister) {
				assert.True(t, lister.putCalled)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			lister := tt.lister()
			p := &Provider{
				client: &onepassword.Client{
					SecretsAPI: fc,
					VaultsAPI:  fc,
					ItemsAPI:   lister,
				},
			}

			err := p.PushSecret(ctx, tt.secret, tt.ref)
			tt.assertError(t, err)
			tt.assertLister(t, lister)
		})
	}
}

func TestDeleteItemField(t *testing.T) {
	fc := &fakeClient{
		listAllResult: []onepassword.VaultOverview{
			{
				ID:    "test",
				Title: "test",
			},
		},
	}

	testCases := []struct {
		name         string
		lister       func() *fakeLister
		ref          *v1alpha1.PushSecretRemoteRef
		assertError  func(t *testing.T, err error)
		assertLister func(t *testing.T, lister *fakeLister)
	}{
		{
			name: "update is called",
			ref: &v1alpha1.PushSecretRemoteRef{
				RemoteKey: "key",
				Property:  "password",
			},
			assertLister: func(t *testing.T, lister *fakeLister) {
				require.True(t, lister.putCalled)
			},
			lister: func() *fakeLister {
				fl := &fakeLister{
					listAllResult: []onepassword.ItemOverview{
						{
							ID:       "test-item-id",
							Title:    "key",
							Category: "login",
							VaultID:  "vault-id",
						},
					},
					getResult: onepassword.Item{
						ID:       "test-item-id",
						Title:    "key",
						Category: "login",
						VaultID:  "vault-id",
						Fields: []onepassword.ItemField{
							{
								ID:        "field-1",
								Title:     "password",
								FieldType: onepassword.ItemFieldTypeConcealed,
								Value:     "password",
							},
							{
								ID:        "field-2",
								Title:     "other-field",
								FieldType: onepassword.ItemFieldTypeConcealed,
								Value:     "username",
							},
						},
					},
				}

				return fl
			},
			assertError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "delete is called",
			ref: &v1alpha1.PushSecretRemoteRef{
				RemoteKey: "key",
				Property:  "password",
			},
			assertLister: func(t *testing.T, lister *fakeLister) {
				require.True(t, lister.deleteCalled, "delete should have been called as the item should have existed")
			},
			lister: func() *fakeLister {
				fl := &fakeLister{
					listAllResult: []onepassword.ItemOverview{
						{
							ID:       "test-item-id",
							Title:    "key",
							Category: "login",
							VaultID:  "vault-id",
						},
					},
					getResult: onepassword.Item{
						ID:       "test-item-id",
						Title:    "key",
						Category: "login",
						VaultID:  "vault-id",
						Fields: []onepassword.ItemField{
							{
								ID:        "field-1",
								Title:     "password",
								FieldType: onepassword.ItemFieldTypeConcealed,
								Value:     "password",
							},
						},
					},
				}

				return fl
			},
			assertError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := t.Context()
			lister := testCase.lister()
			p := &Provider{
				client: &onepassword.Client{
					SecretsAPI: fc,
					VaultsAPI:  fc,
					ItemsAPI:   lister,
				},
			}

			testCase.assertError(t, p.DeleteSecret(ctx, testCase.ref))
			testCase.assertLister(t, lister)
		})
	}
}

func TestGetVault(t *testing.T) {
	fc := &fakeClient{
		listAllResult: []onepassword.VaultOverview{
			{
				ID:    "vault-id",
				Title: "vault-title",
			},
		},
	}

	p := &Provider{
		client: &onepassword.Client{
			VaultsAPI: fc,
		},
	}

	titleOrUuids := []string{"vault-title", "vault-id"}

	for _, titleOrUuid := range titleOrUuids {
		t.Run(titleOrUuid, func(t *testing.T) {
			vaultID, err := p.GetVault(t.Context(), titleOrUuid)
			require.NoError(t, err)
			require.Equal(t, fc.listAllResult[0].ID, vaultID)
		})
	}
}

type fakeLister struct {
	listAllResult []onepassword.ItemOverview
	createCalled  bool
	putCalled     bool
	deleteCalled  bool
	getResult     onepassword.Item
	fileLister    onepassword.ItemsFilesAPI
}

func (f *fakeLister) Create(ctx context.Context, params onepassword.ItemCreateParams) (onepassword.Item, error) {
	f.createCalled = true
	return onepassword.Item{}, nil
}

func (f *fakeLister) Get(ctx context.Context, vaultID, itemID string) (onepassword.Item, error) {
	return f.getResult, nil
}

func (f *fakeLister) Put(ctx context.Context, item onepassword.Item) (onepassword.Item, error) {
	f.putCalled = true
	return onepassword.Item{}, nil
}

func (f *fakeLister) Delete(ctx context.Context, vaultID, itemID string) error {
	f.deleteCalled = true
	return nil
}

func (f *fakeLister) Archive(ctx context.Context, vaultID, itemID string) error {
	return nil
}

func (f *fakeLister) List(ctx context.Context, vaultID string, opts ...onepassword.ItemListFilter) ([]onepassword.ItemOverview, error) {
	return f.listAllResult, nil
}

func (f *fakeLister) Shares() onepassword.ItemsSharesAPI {
	return nil
}

func (f *fakeLister) Files() onepassword.ItemsFilesAPI {
	return f.fileLister
}

type fakeFileLister struct {
	readContent []byte
}

func (f *fakeFileLister) Attach(ctx context.Context, item onepassword.Item, fileParams onepassword.FileCreateParams) (onepassword.Item, error) {
	return onepassword.Item{}, nil
}

func (f *fakeFileLister) Read(ctx context.Context, vaultID, itemID string, attr onepassword.FileAttributes) ([]byte, error) {
	return f.readContent, nil
}

func (f *fakeFileLister) Delete(ctx context.Context, item onepassword.Item, sectionID, fieldID string) (onepassword.Item, error) {
	return onepassword.Item{}, nil
}

func (f *fakeFileLister) ReplaceDocument(ctx context.Context, item onepassword.Item, docParams onepassword.DocumentCreateParams) (onepassword.Item, error) {
	return onepassword.Item{}, nil
}

var _ onepassword.ItemsFilesAPI = (*fakeFileLister)(nil)

type fakeClient struct {
	resolveResult   string
	resolveError    error
	resolveAll      onepassword.ResolveAllResponse
	resolveAllError error
	listAllResult   []onepassword.VaultOverview
	listAllError    error
}

func (f *fakeClient) List(ctx context.Context) ([]onepassword.VaultOverview, error) {
	return f.listAllResult, f.listAllError
}

func (f *fakeClient) Resolve(ctx context.Context, secretReference string) (string, error) {
	return f.resolveResult, f.resolveError
}

func (f *fakeClient) ResolveAll(ctx context.Context, secretReferences []string) (onepassword.ResolveAllResponse, error) {
	return f.resolveAll, f.resolveAllError
}

func TestCachingGetSecret(t *testing.T) {
	t.Run("cache hit returns cached value", func(t *testing.T) {
		fcWithCounter := &fakeClientWithCounter{
			fakeClient: &fakeClient{
				resolveResult: "secret-value",
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fcWithCounter,
				VaultsAPI:  fcWithCounter.fakeClient,
			},
			vaultPrefix: "op://vault/",
		}

		// Initialize cache
		p.cache = expirable.NewLRU[string, []byte](100, nil, time.Minute)

		ref := v1.ExternalSecretDataRemoteRef{Key: "item/field"}

		// First call - cache miss
		val1, err := p.GetSecret(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, []byte("secret-value"), val1)
		assert.Equal(t, 1, fcWithCounter.resolveCallCount)

		// Second call - cache hit, should not call API
		val2, err := p.GetSecret(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, []byte("secret-value"), val2)
		assert.Equal(t, 1, fcWithCounter.resolveCallCount, "API should not be called on cache hit")
	})

	t.Run("cache disabled works normally", func(t *testing.T) {
		fcWithCounter := &fakeClientWithCounter{
			fakeClient: &fakeClient{
				resolveResult: "secret-value",
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fcWithCounter,
				VaultsAPI:  fcWithCounter.fakeClient,
			},
			vaultPrefix: "op://vault/",
			cache:       nil, // Cache disabled
		}

		ref := v1.ExternalSecretDataRemoteRef{Key: "item/field"}

		// Multiple calls should always hit API
		_, err := p.GetSecret(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, 1, fcWithCounter.resolveCallCount)

		_, err = p.GetSecret(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, 2, fcWithCounter.resolveCallCount)
	})
}

func TestCachingGetSecretMap(t *testing.T) {
	t.Run("cache hit returns cached map", func(t *testing.T) {
		fc := &fakeClient{}
		flWithCounter := &fakeListerWithCounter{
			fakeLister: &fakeLister{
				listAllResult: []onepassword.ItemOverview{
					{
						ID:       "item-id",
						Title:    "item",
						Category: "login",
						VaultID:  "vault-id",
					},
				},
				getResult: onepassword.Item{
					ID:       "item-id",
					Title:    "item",
					Category: "login",
					VaultID:  "vault-id",
					Fields: []onepassword.ItemField{
						{Title: "username", Value: "user1"},
						{Title: "password", Value: "pass1"},
					},
				},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fc,
				VaultsAPI:  fc,
				ItemsAPI:   flWithCounter,
			},
			vaultPrefix: "op://vault/",
			vaultID:     "vault-id",
			cache:       expirable.NewLRU[string, []byte](100, nil, time.Minute),
		}

		ref := v1.ExternalSecretDataRemoteRef{Key: "item"}

		// First call - cache miss
		val1, err := p.GetSecretMap(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, map[string][]byte{
			"username": []byte("user1"),
			"password": []byte("pass1"),
		}, val1)
		assert.Equal(t, 1, flWithCounter.getCallCount)

		// Second call - cache hit
		val2, err := p.GetSecretMap(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, val1, val2)
		assert.Equal(t, 1, flWithCounter.getCallCount, "API should not be called on cache hit")
	})
}

func TestCacheInvalidationPushSecret(t *testing.T) {
	t.Run("push secret invalidates cache", func(t *testing.T) {
		fcWithCounter := &fakeClientWithCounter{
			fakeClient: &fakeClient{
				resolveResult: "secret-value",
			},
		}

		fl := &fakeLister{
			listAllResult: []onepassword.ItemOverview{
				{ID: "item-id", Title: "item", VaultID: "vault-id"},
			},
			getResult: onepassword.Item{
				ID:      "item-id",
				Title:   "item",
				VaultID: "vault-id",
				Fields:  []onepassword.ItemField{{Title: "password", Value: "old"}},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fcWithCounter,
				VaultsAPI:  fcWithCounter.fakeClient,
				ItemsAPI:   fl,
			},
			vaultPrefix: "op://vault/",
			vaultID:     "vault-id",
			cache:       expirable.NewLRU[string, []byte](100, nil, time.Minute),
		}

		ref := v1.ExternalSecretDataRemoteRef{Key: "item/password"}

		// Populate cache
		val1, err := p.GetSecret(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, []byte("secret-value"), val1)
		assert.Equal(t, 1, fcWithCounter.resolveCallCount)

		// Push new value (should invalidate cache)
		pushRef := v1alpha1.PushSecretData{
			Match: v1alpha1.PushSecretMatch{
				SecretKey: "key",
				RemoteRef: v1alpha1.PushSecretRemoteRef{
					RemoteKey: "item",
					Property:  "password",
				},
			},
		}
		secret := &corev1.Secret{
			Data: map[string][]byte{"key": []byte("new-value")},
		}
		err = p.PushSecret(t.Context(), secret, pushRef)
		require.NoError(t, err)

		// Next GetSecret should fetch fresh value (cache was invalidated)
		val2, err := p.GetSecret(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, []byte("secret-value"), val2)
		assert.Equal(t, 2, fcWithCounter.resolveCallCount, "Cache should have been invalidated")
	})
}

func TestCacheInvalidationDeleteSecret(t *testing.T) {
	t.Run("delete secret invalidates cache", func(t *testing.T) {
		fcWithCounter := &fakeClientWithCounter{
			fakeClient: &fakeClient{
				resolveResult: "cached-value",
			},
		}

		fl := &fakeLister{
			listAllResult: []onepassword.ItemOverview{
				{ID: "item-id", Title: "item", VaultID: "vault-id"},
			},
			getResult: onepassword.Item{
				ID:      "item-id",
				Title:   "item",
				VaultID: "vault-id",
				Fields: []onepassword.ItemField{
					{Title: "field1", Value: "val1"},
					{Title: "field2", Value: "val2"},
				},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fcWithCounter,
				VaultsAPI:  fcWithCounter.fakeClient,
				ItemsAPI:   fl,
			},
			vaultPrefix: "op://vault/",
			vaultID:     "vault-id",
			cache:       expirable.NewLRU[string, []byte](100, nil, time.Minute),
		}

		ref := v1.ExternalSecretDataRemoteRef{Key: "item/field1"}

		// Populate cache
		_, err := p.GetSecret(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, 1, fcWithCounter.resolveCallCount)

		// Delete field (should invalidate cache)
		deleteRef := v1alpha1.PushSecretRemoteRef{
			RemoteKey: "item",
			Property:  "field1",
		}
		err = p.DeleteSecret(t.Context(), deleteRef)
		require.NoError(t, err)

		// Next GetSecret should miss cache
		_, err = p.GetSecret(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, 2, fcWithCounter.resolveCallCount, "Cache should have been invalidated")
	})
}

func TestInvalidateCacheByPrefix(t *testing.T) {
	t.Run("invalidates all entries with prefix", func(t *testing.T) {
		p := &Provider{
			vaultPrefix: "op://vault/",
			cache:       expirable.NewLRU[string, []byte](100, nil, time.Minute),
		}

		// Add multiple cache entries
		p.cache.Add("op://vault/item1/field1", []byte("val1"))
		p.cache.Add("op://vault/item1/field2", []byte("val2"))
		p.cache.Add("op://vault/item2/field1", []byte("val3"))

		// Invalidate item1 entries
		p.invalidateCacheByPrefix("op://vault/item1")

		// item1 entries should be gone
		_, ok1 := p.cache.Get("op://vault/item1/field1")
		assert.False(t, ok1)
		_, ok2 := p.cache.Get("op://vault/item1/field2")
		assert.False(t, ok2)

		// item2 entry should still exist
		val3, ok3 := p.cache.Get("op://vault/item2/field1")
		assert.True(t, ok3)
		assert.Equal(t, []byte("val3"), val3)
	})

	t.Run("handles nil cache gracefully", func(t *testing.T) {
		p := &Provider{
			vaultPrefix: "op://vault/",
			cache:       nil,
		}

		// Should not panic
		p.invalidateCacheByPrefix("op://vault/item1")
	})

	t.Run("does not invalidate entries with similar prefixes", func(t *testing.T) {
		p := &Provider{
			vaultPrefix: "op://vault/",
			cache:       expirable.NewLRU[string, []byte](100, nil, time.Minute),
		}

		p.cache.Add("op://vault/item/field1", []byte("val1"))
		p.cache.Add("op://vault/item/field2", []byte("val2"))
		p.cache.Add("op://vault/item|property", []byte("val3"))
		p.cache.Add("op://vault/item-backup/field1", []byte("val4"))
		p.cache.Add("op://vault/prod-db/secret", []byte("val5"))
		p.cache.Add("op://vault/prod-db-replica/secret", []byte("val6"))
		p.cache.Add("op://vault/prod-db-replica/secret|property", []byte("val7"))

		p.invalidateCacheByPrefix("op://vault/item")

		_, ok1 := p.cache.Get("op://vault/item/field1")
		assert.False(t, ok1)
		_, ok2 := p.cache.Get("op://vault/item/field2")
		assert.False(t, ok2)
		_, ok3 := p.cache.Get("op://vault/item|property")
		assert.False(t, ok3)

		val4, ok4 := p.cache.Get("op://vault/item-backup/field1")
		assert.True(t, ok4, "item-backup should not be invalidated")
		assert.Equal(t, []byte("val4"), val4)

		p.invalidateCacheByPrefix("op://vault/prod-db")
		_, ok5 := p.cache.Get("op://vault/prod-db/secret")
		assert.False(t, ok5)

		val6, ok6 := p.cache.Get("op://vault/prod-db-replica/secret")
		assert.True(t, ok6, "prod-db-replica/secret should not be invalidated")
		assert.Equal(t, []byte("val6"), val6)

		val7, ok7 := p.cache.Get("op://vault/prod-db-replica/secret|property")
		assert.True(t, ok7, "prod-db-replica/secret|property should not be invalidated")
		assert.Equal(t, []byte("val7"), val7)
	})
}

// fakeClientWithCounter wraps fakeClient and tracks Resolve call count.
type fakeClientWithCounter struct {
	*fakeClient
	resolveCallCount int
}

func (f *fakeClientWithCounter) Resolve(ctx context.Context, secretReference string) (string, error) {
	f.resolveCallCount++
	return f.fakeClient.Resolve(ctx, secretReference)
}

// fakeListerWithCounter wraps fakeLister and tracks Get call count.
type fakeListerWithCounter struct {
	*fakeLister
	getCallCount int
}

func (f *fakeListerWithCounter) Get(ctx context.Context, vaultID, itemID string) (onepassword.Item, error) {
	f.getCallCount++
	return f.fakeLister.Get(ctx, vaultID, itemID)
}

func (f *fakeListerWithCounter) Put(ctx context.Context, item onepassword.Item) (onepassword.Item, error) {
	return f.fakeLister.Put(ctx, item)
}

func (f *fakeListerWithCounter) Delete(ctx context.Context, vaultID, itemID string) error {
	return f.fakeLister.Delete(ctx, vaultID, itemID)
}

func (f *fakeListerWithCounter) Archive(ctx context.Context, vaultID, itemID string) error {
	return f.fakeLister.Archive(ctx, vaultID, itemID)
}

func (f *fakeListerWithCounter) List(ctx context.Context, vaultID string, opts ...onepassword.ItemListFilter) ([]onepassword.ItemOverview, error) {
	return f.fakeLister.List(ctx, vaultID, opts...)
}

func (f *fakeListerWithCounter) Shares() onepassword.ItemsSharesAPI {
	return f.fakeLister.Shares()
}

func (f *fakeListerWithCounter) Files() onepassword.ItemsFilesAPI {
	return f.fakeLister.Files()
}

func (f *fakeListerWithCounter) Create(ctx context.Context, item onepassword.ItemCreateParams) (onepassword.Item, error) {
	return f.fakeLister.Create(ctx, item)
}

var _ onepassword.SecretsAPI = &fakeClient{}
var _ onepassword.VaultsAPI = &fakeClient{}
var _ onepassword.ItemsAPI = &fakeLister{}
var _ onepassword.SecretsAPI = &fakeClientWithCounter{}
var _ onepassword.ItemsAPI = &fakeListerWithCounter{}
