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
	"fmt"
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

type statefulFakeLister struct {
	listAllResult []onepassword.ItemOverview
	items         map[string]onepassword.Item
	deletedItems  map[string]bool
	createCalled  bool
	putCalled     bool
	deleteCalled  bool
	fileLister    onepassword.ItemsFilesAPI
}

func (f *statefulFakeLister) Create(ctx context.Context, params onepassword.ItemCreateParams) (onepassword.Item, error) {
	f.createCalled = true
	return onepassword.Item{}, nil
}

func (f *statefulFakeLister) Get(ctx context.Context, vaultID, itemID string) (onepassword.Item, error) {
	if f.deletedItems != nil && f.deletedItems[itemID] {
		return onepassword.Item{}, fmt.Errorf("item not found")
	}
	if item, ok := f.items[itemID]; ok {
		return item, nil
	}
	return onepassword.Item{}, fmt.Errorf("item not found")
}

func (f *statefulFakeLister) Put(ctx context.Context, item onepassword.Item) (onepassword.Item, error) {
	f.putCalled = true
	if f.items == nil {
		f.items = make(map[string]onepassword.Item)
	}
	f.items[item.ID] = item
	return item, nil
}

func (f *statefulFakeLister) Delete(ctx context.Context, vaultID, itemID string) error {
	f.deleteCalled = true
	if f.deletedItems == nil {
		f.deletedItems = make(map[string]bool)
	}
	f.deletedItems[itemID] = true
	delete(f.items, itemID)
	f.listAllResult = nil
	return nil
}

func (f *statefulFakeLister) Archive(ctx context.Context, vaultID, itemID string) error {
	return nil
}

func (f *statefulFakeLister) List(ctx context.Context, vaultID string, opts ...onepassword.ItemListFilter) ([]onepassword.ItemOverview, error) {
	return f.listAllResult, nil
}

func (f *statefulFakeLister) Shares() onepassword.ItemsSharesAPI {
	return nil
}

func (f *statefulFakeLister) Files() onepassword.ItemsFilesAPI {
	return f.fileLister
}

var _ onepassword.ItemsAPI = (*statefulFakeLister)(nil)

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

func TestDeleteMultipleFieldsFromSameItem(t *testing.T) {
	fc := &fakeClient{
		listAllResult: []onepassword.VaultOverview{
			{
				ID:    "test",
				Title: "test",
			},
		},
	}

	t.Run("deleting second field after item was deleted should not error", func(t *testing.T) {
		fl := &statefulFakeLister{
			listAllResult: []onepassword.ItemOverview{
				{
					ID:       "test-item-id",
					Title:    "key",
					Category: "login",
					VaultID:  "vault-id",
				},
			},
			items: map[string]onepassword.Item{
				"test-item-id": {
					ID:       "test-item-id",
					Title:    "key",
					Category: "login",
					VaultID:  "vault-id",
					Fields: []onepassword.ItemField{
						{
							ID:        "field-1",
							Title:     "username",
							FieldType: onepassword.ItemFieldTypeConcealed,
							Value:     "testuser",
						},
						{
							ID:        "field-2",
							Title:     "password",
							FieldType: onepassword.ItemFieldTypeConcealed,
							Value:     "testpass",
						},
					},
				},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fc,
				VaultsAPI:  fc,
				ItemsAPI:   fl,
			},
		}

		ctx := t.Context()

		err := p.DeleteSecret(ctx, &v1alpha1.PushSecretRemoteRef{
			RemoteKey: "key",
			Property:  "username",
		})
		require.NoError(t, err, "first field deletion should succeed")
		assert.True(t, fl.putCalled, "Put should have been called to update the item")
		assert.False(t, fl.deleteCalled, "Delete should not have been called yet")

		fl.putCalled = false

		err = p.DeleteSecret(ctx, &v1alpha1.PushSecretRemoteRef{
			RemoteKey: "key",
			Property:  "password",
		})
		require.NoError(t, err, "second field deletion should succeed")
		assert.True(t, fl.deleteCalled, "Delete should have been called to remove the item")

		fl.listAllResult = nil

		err = p.DeleteSecret(ctx, &v1alpha1.PushSecretRemoteRef{
			RemoteKey: "key",
			Property:  "some-other-field",
		})
		require.NoError(t, err, "deleting a field from an already-deleted item should not error (this is the bug!)")
	})
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

// fakeListerWithCounter wraps fakeLister and tracks Get and List call counts.
type fakeListerWithCounter struct {
	*fakeLister
	getCallCount  int
	listCallCount int
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
	f.listCallCount++
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

func TestItemListCaching(t *testing.T) {
	t.Run("multiple findItem calls reuse cached list", func(t *testing.T) {
		fc := &fakeClient{}
		fl := &fakeListerWithCounter{
			fakeLister: &fakeLister{
				listAllResult: []onepassword.ItemOverview{
					{ID: "id-1", Title: "item-a", Category: "login", VaultID: "vault-id"},
					{ID: "id-2", Title: "item-b", Category: "login", VaultID: "vault-id"},
				},
				getResult: onepassword.Item{
					ID: "id-1", Title: "item-a", Category: "login", VaultID: "vault-id",
					Fields: []onepassword.ItemField{{Title: "f", Value: "v"}},
				},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fc,
				VaultsAPI:  fc,
				ItemsAPI:   fl,
			},
			vaultPrefix:   "op://vault/",
			vaultID:       "vault-id",
			cache:         expirable.NewLRU[string, []byte](100, nil, time.Minute),
			itemListCache: expirable.NewLRU[string, []onepassword.ItemOverview](100, nil, time.Minute),
			itemCache:     expirable.NewLRU[string, onepassword.Item](100, nil, time.Minute),
		}

		ctx := t.Context()

		// First call — cache miss, calls List + Get
		_, err := p.findItem(ctx, "item-a")
		require.NoError(t, err)
		assert.Equal(t, 1, fl.listCallCount)
		assert.Equal(t, 1, fl.getCallCount)

		// Second call same name — list cached, item cached
		_, err = p.findItem(ctx, "item-a")
		require.NoError(t, err)
		assert.Equal(t, 1, fl.listCallCount, "List should not be called again")
		assert.Equal(t, 1, fl.getCallCount, "Get should not be called again")

		// Third call different name — list cached, but needs Get for new item
		fl.fakeLister.getResult = onepassword.Item{
			ID: "id-2", Title: "item-b", Category: "login", VaultID: "vault-id",
			Fields: []onepassword.ItemField{{Title: "f2", Value: "v2"}},
		}
		_, err = p.findItem(ctx, "item-b")
		require.NoError(t, err)
		assert.Equal(t, 1, fl.listCallCount, "List should still not be called again")
		assert.Equal(t, 2, fl.getCallCount, "Get should be called for new item")
	})
}

func TestItemCachingByUUID(t *testing.T) {
	t.Run("UUID lookups cache Items.Get results", func(t *testing.T) {
		fc := &fakeClient{}
		fl := &fakeListerWithCounter{
			fakeLister: &fakeLister{
				getResult: onepassword.Item{
					ID: "550e8400-e29b-41d4-a716-446655440000", Title: "my-item",
					Category: "login", VaultID: "vault-id",
					Fields: []onepassword.ItemField{{Title: "pw", Value: "secret"}},
				},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fc,
				VaultsAPI:  fc,
				ItemsAPI:   fl,
			},
			vaultPrefix:   "op://vault/",
			vaultID:       "vault-id",
			cache:         expirable.NewLRU[string, []byte](100, nil, time.Minute),
			itemListCache: expirable.NewLRU[string, []onepassword.ItemOverview](100, nil, time.Minute),
			itemCache:     expirable.NewLRU[string, onepassword.Item](100, nil, time.Minute),
		}

		ctx := t.Context()
		uuid := "550e8400-e29b-41d4-a716-446655440000"

		item1, err := p.findItem(ctx, uuid)
		require.NoError(t, err)
		assert.Equal(t, "my-item", item1.Title)
		assert.Equal(t, 1, fl.getCallCount)

		item2, err := p.findItem(ctx, uuid)
		require.NoError(t, err)
		assert.Equal(t, "my-item", item2.Title)
		assert.Equal(t, 1, fl.getCallCount, "Get should not be called again for cached UUID")
		assert.Equal(t, 0, fl.listCallCount, "List should never be called for UUID lookups")
	})
}

func TestItemListCacheSurvivesCreate(t *testing.T) {
	t.Run("after PushSecret creates an item, cached list is updated, next findItem does NOT re-list", func(t *testing.T) {
		fc := &fakeClient{}
		fl := &fakeListerWithCounter{
			fakeLister: &fakeLister{
				listAllResult: []onepassword.ItemOverview{
					{ID: "id-existing", Title: "existing-item", Category: "login", VaultID: "vault-id"},
				},
				getResult: onepassword.Item{
					ID: "id-existing", Title: "existing-item", Category: "login", VaultID: "vault-id",
					Fields: []onepassword.ItemField{{Title: "password", Value: "old"}},
				},
			},
		}

		// Wrap with a Create that returns a proper Item
		cl := &createReturningLister{
			fakeListerWithCounter: fl,
			createResult: onepassword.Item{
				ID: "id-new", Title: "new-item", Category: onepassword.ItemCategoryServer, VaultID: "vault-id",
				Fields: []onepassword.ItemField{{Title: "password", Value: "new-value"}},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fc,
				VaultsAPI:  fc,
				ItemsAPI:   cl,
			},
			vaultPrefix:   "op://vault/",
			vaultID:       "vault-id",
			cache:         expirable.NewLRU[string, []byte](100, nil, time.Minute),
			itemListCache: expirable.NewLRU[string, []onepassword.ItemOverview](100, nil, time.Minute),
			itemCache:     expirable.NewLRU[string, onepassword.Item](100, nil, time.Minute),
		}

		ctx := t.Context()

		// Populate the list cache by finding an existing item
		_, err := p.findItem(ctx, "existing-item")
		require.NoError(t, err)
		assert.Equal(t, 1, fl.listCallCount)

		// PushSecret for a new item — findItem uses cached list, returns ErrKeyNotFound, calls createItem
		secret := &corev1.Secret{
			Data: map[string][]byte{"key": []byte("new-value")},
		}
		pushRef := v1alpha1.PushSecretData{
			Match: v1alpha1.PushSecretMatch{
				SecretKey: "key",
				RemoteRef: v1alpha1.PushSecretRemoteRef{
					RemoteKey: "new-item",
				},
			},
		}
		err = p.PushSecret(ctx, secret, pushRef)
		require.NoError(t, err)
		assert.True(t, cl.createCalled)
		assert.Equal(t, 1, fl.listCallCount, "List should not be called again — list was already cached")

		// Now find the newly created item — it should be in the cached list from the surgical add
		fl.fakeLister.getResult = onepassword.Item{
			ID: "id-new", Title: "new-item", Category: onepassword.ItemCategoryServer, VaultID: "vault-id",
			Fields: []onepassword.ItemField{{Title: "password", Value: "new-value"}},
		}
		item, err := p.findItem(ctx, "new-item")
		require.NoError(t, err)
		assert.Equal(t, "new-item", item.Title)
		assert.Equal(t, 1, fl.listCallCount, "List should still not be called — list cache was surgically updated")
	})
}

// createReturningLister wraps fakeListerWithCounter to return a configured Item from Create.
type createReturningLister struct {
	*fakeListerWithCounter
	createResult onepassword.Item
	createCalled bool
}

func (c *createReturningLister) Create(_ context.Context, _ onepassword.ItemCreateParams) (onepassword.Item, error) {
	c.createCalled = true
	return c.createResult, nil
}

func TestItemListCacheSurvivesDelete(t *testing.T) {
	t.Run("after DeleteSecret, item is removed from cached list, next findItem does NOT re-list", func(t *testing.T) {
		fc := &fakeClient{}
		fl := &fakeListerWithCounter{
			fakeLister: &fakeLister{
				listAllResult: []onepassword.ItemOverview{
					{ID: "id-1", Title: "item-a", Category: "login", VaultID: "vault-id"},
					{ID: "id-2", Title: "item-b", Category: "login", VaultID: "vault-id"},
				},
				getResult: onepassword.Item{
					ID: "id-1", Title: "item-a", Category: "login", VaultID: "vault-id",
					Fields: []onepassword.ItemField{{Title: "password", Value: "secret"}},
				},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fc,
				VaultsAPI:  fc,
				ItemsAPI:   fl,
			},
			vaultPrefix:   "op://vault/",
			vaultID:       "vault-id",
			cache:         expirable.NewLRU[string, []byte](100, nil, time.Minute),
			itemListCache: expirable.NewLRU[string, []onepassword.ItemOverview](100, nil, time.Minute),
			itemCache:     expirable.NewLRU[string, onepassword.Item](100, nil, time.Minute),
		}

		ctx := t.Context()

		// Populate list cache
		_, err := p.findItem(ctx, "item-a")
		require.NoError(t, err)
		assert.Equal(t, 1, fl.listCallCount)

		// Delete item-a (only one field, will trigger full delete)
		err = p.DeleteSecret(ctx, &v1alpha1.PushSecretRemoteRef{
			RemoteKey: "item-a",
			Property:  "password",
		})
		require.NoError(t, err)
		assert.True(t, fl.fakeLister.deleteCalled)

		// The list cache should have item-a removed, item-b still present
		// findItem for item-b should NOT call List again
		fl.fakeLister.getResult = onepassword.Item{
			ID: "id-2", Title: "item-b", Category: "login", VaultID: "vault-id",
			Fields: []onepassword.ItemField{{Title: "pw", Value: "val"}},
		}
		_, err = p.findItem(ctx, "item-b")
		require.NoError(t, err)
		assert.Equal(t, 1, fl.listCallCount, "List should not be called again — cache survived the delete")

		// findItem for deleted item-a should return ErrKeyNotFound (it was removed from list cache)
		_, err = p.findItem(ctx, "item-a")
		assert.ErrorIs(t, err, ErrKeyNotFound)
		assert.Equal(t, 1, fl.listCallCount, "List should still not be called — item was removed from cached list")
	})
}

func TestItemCacheInvalidationOnPut(t *testing.T) {
	t.Run("Items.Put invalidates item cache so next findItem re-fetches the full item", func(t *testing.T) {
		fc := &fakeClient{}
		fl := &fakeListerWithCounter{
			fakeLister: &fakeLister{
				listAllResult: []onepassword.ItemOverview{
					{ID: "id-1", Title: "item", Category: "login", VaultID: "vault-id"},
				},
				getResult: onepassword.Item{
					ID: "id-1", Title: "item", Category: "login", VaultID: "vault-id",
					Fields: []onepassword.ItemField{{Title: "password", Value: "old-value"}},
				},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fc,
				VaultsAPI:  fc,
				ItemsAPI:   fl,
			},
			vaultPrefix:   "op://vault/",
			vaultID:       "vault-id",
			cache:         expirable.NewLRU[string, []byte](100, nil, time.Minute),
			itemListCache: expirable.NewLRU[string, []onepassword.ItemOverview](100, nil, time.Minute),
			itemCache:     expirable.NewLRU[string, onepassword.Item](100, nil, time.Minute),
		}

		ctx := t.Context()

		// First findItem populates both list and item caches
		item, err := p.findItem(ctx, "item")
		require.NoError(t, err)
		assert.Equal(t, "old-value", item.Fields[0].Value)
		assert.Equal(t, 1, fl.getCallCount)

		// PushSecret updates the item (calls Put)
		secret := &corev1.Secret{
			Data: map[string][]byte{"key": []byte("new-value")},
		}
		pushRef := v1alpha1.PushSecretData{
			Match: v1alpha1.PushSecretMatch{
				SecretKey: "key",
				RemoteRef: v1alpha1.PushSecretRemoteRef{
					RemoteKey: "item",
					Property:  "password",
				},
			},
		}
		err = p.PushSecret(ctx, secret, pushRef)
		require.NoError(t, err)
		// findItem inside PushSecret used cached item (no additional Get)
		assert.Equal(t, 1, fl.getCallCount, "findItem within PushSecret should use item cache")

		// Now update the fake to return new value
		fl.fakeLister.getResult = onepassword.Item{
			ID: "id-1", Title: "item", Category: "login", VaultID: "vault-id",
			Fields: []onepassword.ItemField{{Title: "password", Value: "new-value"}},
		}

		// Next findItem should call Get again because item cache was invalidated by Put
		item, err = p.findItem(ctx, "item")
		require.NoError(t, err)
		assert.Equal(t, "new-value", item.Fields[0].Value)
		assert.Equal(t, 2, fl.getCallCount, "Get should be called again after item cache invalidation")
		assert.Equal(t, 1, fl.listCallCount, "List should still not be called — list cache survived")
	})
}

func TestCacheDisabledWorks(t *testing.T) {
	t.Run("nil caches means every call hits API", func(t *testing.T) {
		fc := &fakeClient{}
		fl := &fakeListerWithCounter{
			fakeLister: &fakeLister{
				listAllResult: []onepassword.ItemOverview{
					{ID: "id-1", Title: "item", Category: "login", VaultID: "vault-id"},
				},
				getResult: onepassword.Item{
					ID: "id-1", Title: "item", Category: "login", VaultID: "vault-id",
					Fields: []onepassword.ItemField{{Title: "f", Value: "v"}},
				},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fc,
				VaultsAPI:  fc,
				ItemsAPI:   fl,
			},
			vaultPrefix: "op://vault/",
			vaultID:     "vault-id",
			// All caches nil
		}

		ctx := t.Context()

		_, err := p.findItem(ctx, "item")
		require.NoError(t, err)
		assert.Equal(t, 1, fl.listCallCount)
		assert.Equal(t, 1, fl.getCallCount)

		_, err = p.findItem(ctx, "item")
		require.NoError(t, err)
		assert.Equal(t, 2, fl.listCallCount, "List should be called again with no cache")
		assert.Equal(t, 2, fl.getCallCount, "Get should be called again with no cache")
	})
}

func TestItemListCacheNotCachedOnNotFound(t *testing.T) {
	t.Run("ErrKeyNotFound does NOT cache the item list", func(t *testing.T) {
		fc := &fakeClient{}
		fl := &fakeListerWithCounter{
			fakeLister: &fakeLister{
				listAllResult: []onepassword.ItemOverview{
					{ID: "id-1", Title: "existing-item", Category: "login", VaultID: "vault-id"},
				},
			},
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fc,
				VaultsAPI:  fc,
				ItemsAPI:   fl,
			},
			vaultPrefix:   "op://vault/",
			vaultID:       "vault-id",
			cache:         expirable.NewLRU[string, []byte](100, nil, time.Minute),
			itemListCache: expirable.NewLRU[string, []onepassword.ItemOverview](100, nil, time.Minute),
			itemCache:     expirable.NewLRU[string, onepassword.Item](100, nil, time.Minute),
		}

		ctx := t.Context()

		// Search for non-existent item
		_, err := p.findItem(ctx, "no-such-item")
		assert.ErrorIs(t, err, ErrKeyNotFound)
		assert.Equal(t, 1, fl.listCallCount)

		// Search again — list should NOT be cached, so List is called again
		_, err = p.findItem(ctx, "no-such-item")
		assert.ErrorIs(t, err, ErrKeyNotFound)
		assert.Equal(t, 2, fl.listCallCount, "List should be called again because not-found does not cache the list")
	})
}

func TestItemCacheNotCachedOnGetError(t *testing.T) {
	t.Run("Items.Get error does NOT cache, next call retries", func(t *testing.T) {
		fc := &fakeClient{}
		getErr := errors.New("temporary API error")
		callCount := 0
		fl := &fakeListerWithCounter{
			fakeLister: &fakeLister{
				listAllResult: []onepassword.ItemOverview{
					{ID: "id-1", Title: "item", Category: "login", VaultID: "vault-id"},
				},
			},
		}

		// We need to override Get to return an error on first call and succeed on second.
		// Use a custom wrapper.
		errorOnFirstGet := &errorOnFirstGetLister{
			fakeListerWithCounter: fl,
			getErr:                getErr,
			callCount:             &callCount,
		}

		p := &Provider{
			client: &onepassword.Client{
				SecretsAPI: fc,
				VaultsAPI:  fc,
				ItemsAPI:   errorOnFirstGet,
			},
			vaultPrefix:   "op://vault/",
			vaultID:       "vault-id",
			cache:         expirable.NewLRU[string, []byte](100, nil, time.Minute),
			itemListCache: expirable.NewLRU[string, []onepassword.ItemOverview](100, nil, time.Minute),
			itemCache:     expirable.NewLRU[string, onepassword.Item](100, nil, time.Minute),
		}

		ctx := t.Context()

		// First call — Get fails
		_, err := p.findItem(ctx, "item")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "temporary API error")
		assert.Equal(t, 1, callCount)

		// Second call — Get succeeds, proves nothing was cached from the error
		item, err := p.findItem(ctx, "item")
		require.NoError(t, err)
		assert.Equal(t, "item", item.Title)
		assert.Equal(t, 2, callCount)
	})
}

// errorOnFirstGetLister wraps fakeListerWithCounter to return an error on the first Get call.
type errorOnFirstGetLister struct {
	*fakeListerWithCounter
	getErr    error
	callCount *int
}

func (e *errorOnFirstGetLister) Get(ctx context.Context, vaultID, itemID string) (onepassword.Item, error) {
	*e.callCount++
	if *e.callCount == 1 {
		return onepassword.Item{}, e.getErr
	}
	return onepassword.Item{
		ID: itemID, Title: "item", Category: "login", VaultID: vaultID,
		Fields: []onepassword.ItemField{{Title: "f", Value: "v"}},
	}, nil
}

var _ onepassword.SecretsAPI = &fakeClient{}
var _ onepassword.VaultsAPI = &fakeClient{}
var _ onepassword.ItemsAPI = &fakeLister{}
var _ onepassword.SecretsAPI = &fakeClientWithCounter{}
var _ onepassword.ItemsAPI = &fakeListerWithCounter{}
