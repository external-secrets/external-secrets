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

package sakura

import (
	"errors"
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// getSakuraProvider extracts the SakuraProvider from GenericStore.
func getSakuraProvider(store esv1.GenericStore) (*esv1.SakuraProvider, error) {
	if store == nil {
		return nil, errors.New("found nil store")
	}

	spc := store.GetSpec()
	if spc == nil {
		return nil, errors.New("store is missing spec")
	}

	if spc.Provider == nil {
		return nil, errors.New("storeSpec is missing provider")
	}

	prov := spc.Provider.Sakura
	if prov == nil {
		return nil, fmt.Errorf("invalid provider spec: missing Sakura field in store %s", store.GetObjectMeta().String())
	}

	return prov, nil
}
