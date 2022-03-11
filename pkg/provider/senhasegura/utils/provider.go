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

package utils

import (
	"fmt"

	v1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errNilStore         = "nil store found"
	errMissingStoreSpec = "store is missing spec"
	errMissingProvider  = "storeSpec is missing provider"
	errInvalidProvider  = "invalid provider spec. Missing senhasegura field in store %s"
)

/*
	GetSenhaseguraProvider checks an generic store, returns SenhaseguraProvider and error
*/
func GetSenhaseguraProvider(store v1beta1.GenericStore) (*v1beta1.SenhaseguraProvider, error) {
	if store == nil {
		return nil, fmt.Errorf(errNilStore)
	}

	spec := store.GetSpec()
	if spec == nil {
		return nil, fmt.Errorf(errMissingStoreSpec)
	}

	if spec.Provider == nil {
		return nil, fmt.Errorf(errMissingProvider)
	}

	provider := spec.Provider.Senhasegura
	if provider == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}

	return provider, nil
}
