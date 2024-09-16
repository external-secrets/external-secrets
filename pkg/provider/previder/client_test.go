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
package previder

import (
	"errors"

	"github.com/previder/vault-cli/pkg"
	"github.com/previder/vault-cli/pkg/model"
)

type PreviderVaultFakeClient struct {
	pkg.PreviderVaultClient
}

var (
	secrets = map[string]string{"secret1": "secret1content", "secret2": "secret2content"}
)

func (v *PreviderVaultFakeClient) DecryptSecret(id string) (*model.SecretDecrypt, error) {
	for k, v := range secrets {
		if k == id {
			return &model.SecretDecrypt{Secret: v}, nil
		}
	}
	return nil, errors.New("404 not found")
}

func (v *PreviderVaultFakeClient) GetSecrets() ([]model.Secret, error) {
	secretList := make([]model.Secret, 0)
	for k := range secrets {
		secretList = append(secretList, model.Secret{Description: k})
	}
	return secretList, nil
}
