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

package cerberus

import (
	"context"
	"testing"

	cerberussdkapi "github.com/Nike-Inc/cerberus-go-client/v3/api"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	fake "github.com/external-secrets/external-secrets/pkg/provider/cerberus/fake"
)

const (
	testSDB = "test-sdb/"
)

var (
	nestedDir = "nested"
)

func newFakeSecretStore() map[string]*vaultapi.Secret {
	return map[string]*vaultapi.Secret{
		"test-sdb/test1": {Data: map[string]interface{}{
			"id": "abc123",
		}},
		"test-sdb/test2": {Data: map[string]interface{}{
			"id":   "def456",
			"test": "me",
		}},
		"test-sdb/different-name": {Data: map[string]interface{}{
			"id":   "def456",
			"test": "me",
		}},
		"test-sdb/nested/test3": {Data: map[string]interface{}{
			"id": "xyz",
		}},
		"test-sdb/nested/test4": {Data: map[string]interface{}{
			"id": "123",
		}},
		"test-sdb-2/test2": {Data: map[string]interface{}{
			"id": "def456",
		}},
	}
}

func newTestClient(fss map[string]*vaultapi.Secret) *cerberus {
	return &cerberus{
		client: &fake.MockCerberusClient{
			FakeSecretStore: fss,
		},
		sdb: &cerberussdkapi.SafeDepositBox{
			Path: testSDB,
		},
	}
}

func TestClientGetSecret(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(newFakeSecretStore())

	testCases := map[string]struct {
		ref  esv1beta1.ExternalSecretDataRemoteRef
		want []byte
		err  error
	}{
		"get valid secret": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "test1",
			},
			want: []byte(`{"id":"abc123"}`),
		},
		"get valid secret with property": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "test1",
				Property: "id",
			},
			want: []byte("abc123"),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := client.GetSecret(ctx, tc.ref)
			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, string(tc.want), string(got))
			} else {
				assert.Nil(t, got)
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.err, err)
			}
		})
	}
}

func TestClientGetSecretMap(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(newFakeSecretStore())

	testCases := map[string]struct {
		ref  esv1beta1.ExternalSecretDataRemoteRef
		want map[string][]byte
		err  error
	}{
		"get valid secret": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "test1",
			},
			want: map[string][]byte{
				"id": []byte(`abc123`),
			},
		},
		"get another valid secret": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "test2",
			},
			want: map[string][]byte{
				"id":   []byte(`def456`),
				"test": []byte(`me`),
			}},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := client.GetSecretMap(ctx, tc.ref)
			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			} else {
				assert.Nil(t, got)
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.err, err)
			}
		})
	}
}

func TestClientGetAllSecrets(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(newFakeSecretStore())

	testCases := map[string]struct {
		ref  esv1beta1.ExternalSecretFind
		want map[string][]byte
		err  error
	}{
		"returns all secrets in SDB": {
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{RegExp: ".*"},
			},
			want: map[string][]byte{
				"test1":          []byte(`{"id":"abc123"}`),
				"test2":          []byte(`{"id":"def456","test":"me"}`),
				"different-name": []byte(`{"id":"def456","test":"me"}`),
				"nested_test3":   []byte(`{"id":"xyz"}`),
				"nested_test4":   []byte(`{"id":"123"}`),
			},
		},
		"returns all secrets in SDB with path": {
			ref: esv1beta1.ExternalSecretFind{
				Path: &nestedDir,
				Name: &esv1beta1.FindName{RegExp: ".*"},
			},
			want: map[string][]byte{
				"test3": []byte(`{"id":"xyz"}`),
				"test4": []byte(`{"id":"123"}`),
			},
		},
		"works well with regex search": {
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{RegExp: "^diff"},
			},
			want: map[string][]byte{
				"different-name": []byte(`{"id":"def456","test":"me"}`),
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := client.GetAllSecrets(ctx, tc.ref)
			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			} else {
				assert.Nil(t, got)
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.err, err)
			}
		})
	}
}
