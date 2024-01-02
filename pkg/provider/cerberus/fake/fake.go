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

package fake

import (
	"errors"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
)

var (
	ErrNotFound = "not found"
)

type MockCerberusClient struct {
	FakeSecretStore map[string]*vaultapi.Secret
}

func (c *MockCerberusClient) ReadSecret(path string, _ map[string][]string) (*vaultapi.Secret, error) {
	secret, ok := c.FakeSecretStore[path]
	if !ok {
		return nil, errors.New(ErrNotFound)
	}

	return secret, nil
}

func (c *MockCerberusClient) WriteSecret(_ string, _ map[string]interface{}) (*vaultapi.Secret, error) {
	return &vaultapi.Secret{}, nil
}

func (c *MockCerberusClient) ListSecret(path string) (*vaultapi.Secret, error) {
	keys := make([]interface{}, 0)

	for k := range c.FakeSecretStore {
		if strings.HasPrefix(k, path) {
			keys = append(keys, strings.Split(k, path)[1])
		}
	}

	return &vaultapi.Secret{
		Data: map[string]interface{}{
			"keys": keys,
		},
	}, nil
}

func (c *MockCerberusClient) DeleteSecret(_ string) (*vaultapi.Secret, error) {
	return &vaultapi.Secret{}, nil
}

func (c *MockCerberusClient) IsAuthenticated() bool {
	return true
}
