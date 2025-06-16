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
			got, err := p.GetSecretMap(context.Background(), tt.ref)
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
		{
			name: "validate error",
			client: func() *onepassword.Client {
				fc := &fakeClient{
					listAllResult: []onepassword.VaultOverview{},
					listAllError:  errors.New("no vaults found when listing"),
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
					listAllResult: []onepassword.VaultOverview{},
					listAllError:  errors.New("no vaults found when listing"),
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
			ctx := context.Background()
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
			ctx := context.Background()
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
			vaultID, err := p.GetVault(context.Background(), titleOrUuid)
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

var _ onepassword.SecretsAPI = &fakeClient{}
var _ onepassword.VaultsAPI = &fakeClient{}
var _ onepassword.ItemsAPI = &fakeLister{}
