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
package util

import (
	"fmt"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errNilStore         = "found nil store"
	errMissingStoreSpec = "store is missing spec"
	errMissingProvider  = "storeSpec is missing provider"
	errInvalidProvider  = "invalid provider spec. Missing AWS field in store %s"
)

// GetAWSProvider does the necessary nil checks on the generic store
// it returns the aws provider or an error.
func GetAWSProvider(store esv1beta1.GenericStore) (*esv1beta1.AWSProvider, error) {
	if store == nil {
		return nil, fmt.Errorf(errNilStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, fmt.Errorf(errMissingStoreSpec)
	}
	if spc.Provider == nil {
		return nil, fmt.Errorf(errMissingProvider)
	}
	prov := spc.Provider.AWS
	if prov == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}
	return prov, nil
}
