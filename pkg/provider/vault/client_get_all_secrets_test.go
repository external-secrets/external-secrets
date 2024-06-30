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
package vault

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	vault "github.com/hashicorp/vault/api"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/util"
)

func TestGetAllSecrets(t *testing.T) {
	secret1Bytes := []byte("{\"access_key\":\"access_key\",\"access_secret\":\"access_secret\"}")
	secret2Bytes := []byte("{\"access_key\":\"access_key2\",\"access_secret\":\"access_secret2\"}")
	path1Bytes := []byte("{\"access_key\":\"path1\",\"access_secret\":\"path1\"}")
	path2Bytes := []byte("{\"access_key\":\"path2\",\"access_secret\":\"path2\"}")
	tagBytes := []byte("{\"access_key\":\"unfetched\",\"access_secret\":\"unfetched\"}")
	path := "path"
	secret := map[string]any{
		"secret1": map[string]any{
			"metadata": map[string]any{
				"custom_metadata": map[string]any{
					"foo": "bar",
				},
			},
			"data": map[string]any{
				"access_key":    "access_key",
				"access_secret": "access_secret",
			},
		},
		"secret2": map[string]any{
			"metadata": map[string]any{
				"custom_metadata": map[string]any{
					"foo": "baz",
				},
			},
			"data": map[string]any{
				"access_key":    "access_key2",
				"access_secret": "access_secret2",
			},
		},
		"secret3": map[string]any{
			"metadata": map[string]any{
				"custom_metadata": map[string]any{
					"foo": "baz",
				},
			},
			"data": nil,
		},
		"tag": map[string]any{
			"metadata": map[string]any{
				"custom_metadata": map[string]any{
					"foo": "baz",
				},
			},
			"data": map[string]any{
				"access_key":    "unfetched",
				"access_secret": "unfetched",
			},
		},
		"path/1": map[string]any{
			"metadata": map[string]any{
				"custom_metadata": map[string]any{
					"foo": "path",
				},
			},
			"data": map[string]any{
				"access_key":    "path1",
				"access_secret": "path1",
			},
		},
		"path/2": map[string]any{
			"metadata": map[string]any{
				"custom_metadata": map[string]any{
					"foo": "path",
				},
			},
			"data": map[string]any{
				"access_key":    "path2",
				"access_secret": "path2",
			},
		},
		"default": map[string]any{
			"data": map[string]any{
				"empty": "true",
			},
			"metadata": map[string]any{
				"keys": []any{"secret1", "secret2", "secret3", "tag", "path/"},
			},
		},
		"path/": map[string]any{
			"data": map[string]any{
				"empty": "true",
			},
			"metadata": map[string]any{
				"keys": []any{"1", "2"},
			},
		},
	}
	type args struct {
		store    *esv1beta1.VaultProvider
		kube     kclient.Client
		vLogical util.Logical
		ns       string
		data     esv1beta1.ExternalSecretFind
	}

	type want struct {
		err error
		val map[string][]byte
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"FindByName": {
			reason: "should map multiple secrets matching name",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ListWithContextFn:         newListWithContextFn(secret),
					ReadWithDataWithContextFn: newReadtWithContextFn(secret),
				},
				data: esv1beta1.ExternalSecretFind{
					Name: &esv1beta1.FindName{
						RegExp: "secret.*",
					},
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"secret1": secret1Bytes,
					"secret2": secret2Bytes,
				},
			},
		},
		"FindByTag": {
			reason: "should map multiple secrets matching tags",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ListWithContextFn:         newListWithContextFn(secret),
					ReadWithDataWithContextFn: newReadtWithContextFn(secret),
				},
				data: esv1beta1.ExternalSecretFind{
					Tags: map[string]string{
						"foo": "baz",
					},
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"tag":     tagBytes,
					"secret2": secret2Bytes,
				},
			},
		},
		"FilterByPath": {
			reason: "should filter secrets based on path",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ListWithContextFn:         newListWithContextFn(secret),
					ReadWithDataWithContextFn: newReadtWithContextFn(secret),
				},
				data: esv1beta1.ExternalSecretFind{
					Path: &path,
					Tags: map[string]string{
						"foo": "path",
					},
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"path/1": path1Bytes,
					"path/2": path2Bytes,
				},
			},
		},
		"FailIfKv1": {
			reason: "should not work if using kv1 store",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ListWithContextFn:         newListWithContextFn(secret),
					ReadWithDataWithContextFn: newReadtWithContextFn(secret),
				},
				data: esv1beta1.ExternalSecretFind{
					Tags: map[string]string{
						"foo": "baz",
					},
				},
			},
			want: want{
				err: errors.New(errUnsupportedKvVersion),
			},
		},
		"MetadataNotFound": {
			reason: "metadata secret not found",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ListWithContextFn: newListWithContextFn(secret),
					ReadWithDataWithContextFn: func(ctx context.Context, path string, d map[string][]string) (*vault.Secret, error) {
						return nil, nil
					},
				},
				data: esv1beta1.ExternalSecretFind{
					Tags: map[string]string{
						"foo": "baz",
					},
				},
			},
			want: want{
				err: errors.New(errNotFound),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vStore := &client{
				kube:      tc.args.kube,
				logical:   tc.args.vLogical,
				store:     tc.args.store,
				namespace: tc.args.ns,
			}
			val, err := vStore.GetAllSecrets(context.Background(), tc.args.data)
			if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
				t.Errorf("\n%s\nvault.GetSecretMap(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.val, val); diff != "" {
				t.Errorf("\n%s\nvault.GetSecretMap(...): -want val, +got val:\n%s", tc.reason, diff)
			}
		})
	}
}

func newListWithContextFn(secrets map[string]any) func(ctx context.Context, path string) (*vault.Secret, error) {
	return func(ctx context.Context, path string) (*vault.Secret, error) {
		path = strings.TrimPrefix(path, "secret/metadata/")
		if path == "" {
			path = "default"
		}
		data, ok := secrets[path]
		if !ok {
			return nil, errors.New("Secret not found")
		}
		meta := data.(map[string]any)
		ans := meta["metadata"].(map[string]any)
		secret := &vault.Secret{
			Data: map[string]any{
				"keys": ans["keys"],
			},
		}
		return secret, nil
	}
}

func newReadtWithContextFn(secrets map[string]any) func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
	return func(ctx context.Context, path string, d map[string][]string) (*vault.Secret, error) {
		path = strings.TrimPrefix(path, "secret/data/")
		path = strings.TrimPrefix(path, "secret/metadata/")
		if path == "" {
			path = "default"
		}
		data, ok := secrets[path]
		if !ok {
			return nil, errors.New("Secret not found")
		}
		meta := data.(map[string]any)
		metadata := meta["metadata"].(map[string]any)
		content := map[string]any{
			"data":            meta["data"],
			"custom_metadata": metadata["custom_metadata"],
		}
		secret := &vault.Secret{
			Data: content,
		}
		return secret, nil
	}
}
