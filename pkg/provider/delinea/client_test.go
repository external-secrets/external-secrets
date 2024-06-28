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

package delinea

import (
	"context"
	"errors"
	"testing"

	"github.com/DelineaXPM/dsv-sdk-go/v2/vault"
	"github.com/stretchr/testify/assert"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type fakeAPI struct {
	secrets []*vault.Secret
}

// createVaultSecret assembles a vault.Secret.
// vault.Secret has unexported nested types, and is therefore quite
// tricky from outside the vault package. This function facilitates easy setup.
func createVaultSecret(path string, data map[string]any) *vault.Secret {
	s := &vault.Secret{}
	s.Path = path
	s.Data = data
	return s
}

// Secret returns secret matching path.
func (f *fakeAPI) Secret(path string) (*vault.Secret, error) {
	for _, s := range f.secrets {
		if s.Path == path {
			return s, nil
		}
	}
	return nil, errors.New("not found")
}

func newTestClient() esv1beta1.SecretsClient {
	return &client{
		api: &fakeAPI{
			secrets: []*vault.Secret{
				createVaultSecret("a", map[string]any{}),
				createVaultSecret("b", map[string]any{
					"hello": "world",
				}),
				createVaultSecret("c", map[string]any{
					"foo": map[string]string{"bar": "baz"},
				}),
			},
		},
	}
}

func TestGetSecret(t *testing.T) {
	ctx := context.Background()
	c := newTestClient()

	testCases := map[string]struct {
		ref  esv1beta1.ExternalSecretDataRemoteRef
		want []byte
		err  error
	}{
		"querying for the key returns the map": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "b",
			},
			want: []byte(`{"hello":"world"}`),
		},
		"querying for the key and property returns a single value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "b",
				Property: "hello",
			},
			want: []byte(`world`),
		},
		"querying for the key and nested property returns a single value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "c",
				Property: "foo.bar",
			},
			want: []byte(`baz`),
		},
		"querying for existent key and non-existing propery": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "c",
				Property: "foo.bar.x",
			},
			err: esv1beta1.NoSecretErr,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := c.GetSecret(ctx, tc.ref)
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
