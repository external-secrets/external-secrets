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
package alibaba

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errNilStore        = "found nil store"
	errMissingProvider = "storeSpec is missing provider"
	errInvalidProvider = "invalid provider spec. Missing Alibaba field in store %s"
	errRegionNotFound  = "region not found"
)

func (kms *KeyManagementService) ValidateStore(store esv1beta1.GenericStore) error {
	prov, err := kms.GetAlibabaProvider(store)
	if err != nil {
		return err
	}
	err = validateRegion(prov)
	if err != nil {
		return err
	}
	return nil

}
func (kms *KeyManagementService) validateStoreAuth(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	switch {
	case alibabaSpec.Auth.RRSAAuth != nil:
		return kms.validateStoreRRSAAuth(store)
	case alibabaSpec.Auth.SecretRef != nil:
		return kms.validateStoreAccessKeyAuth(store)
	default:
		return fmt.Errorf("missing alibaba auth provider")
	}
}

func validateRegion(prov *esv1beta1.AlibabaProvider) error {
	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()
	found := false
	for _, p := range partitions {
		for id := range p.Regions() {
			if id == prov.RegionID {
				found = true
			}
		}
	}
	if !found {
		return fmt.Errorf(errRegionNotFound)
	}
	return nil
}
func (kms *KeyManagementService) GetAlibabaProvider(store esv1beta1.GenericStore) (*esv1beta1.AlibabaProvider, error) {
	if store == nil {
		return nil, fmt.Errorf(errNilStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, fmt.Errorf(errNilStore)
	}
	if spc.Provider == nil {
		return nil, fmt.Errorf(errMissingProvider)
	}
	provider := spc.Provider.Alibaba
	if provider == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}
	return provider, nil
}
