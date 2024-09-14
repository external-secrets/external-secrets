//Copyright External Secrets Inc. All Rights Reserved

package util

import (
	"errors"
	"fmt"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errNilStore         = "found nil store"
	errMissingStoreSpec = "store is missing spec"
	errMissingProvider  = "storeSpec is missing provider"
	errInvalidProvider  = "invalid provider spec. Missing Conjur field in store %s"
)

// GetConjurProvider does the necessary nil checks on the generic store
// it returns the conjur provider or an error.
func GetConjurProvider(store esv1beta1.GenericStore) (*esv1beta1.ConjurProvider, error) {
	if store == nil {
		return nil, errors.New(errNilStore)
	}
	spec := store.GetSpec()
	if spec == nil {
		return nil, errors.New(errMissingStoreSpec)
	}
	if spec.Provider == nil {
		return nil, errors.New(errMissingProvider)
	}

	if spec.Provider.Conjur == nil {
		return nil, errors.New(errMissingProvider)
	}

	prov := spec.Provider.Conjur
	if prov == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}
	return prov, nil
}
