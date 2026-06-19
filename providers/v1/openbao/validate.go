/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package openbao

import (
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const (
	errInvalidStore             = "invalid store"
	errInvalidStoreSpec         = "invalid store spec"
	errInvalidStoreProv         = "invalid store provider"
	errInvalidOpenBaoProv       = "invalid OpenBao provider"
	errInvalidTokenRef          = "invalid Auth.TokenSecretRef: %w"
	errInvalidUserPassSecretRef = "invalid Auth.UserPass.SecretRef: %w"
)

// ValidateStore validates the OpenBao provider configuration in the SecretStore.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, errors.New(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, errors.New(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return nil, errors.New(errInvalidStoreProv)
	}
	baoProvider := spc.Provider.OpenBao
	if baoProvider == nil {
		return nil, errors.New(errInvalidOpenBaoProv)
	}
	if baoProvider.Auth != nil {
		auth := baoProvider.Auth
		if auth.TokenSecretRef != nil {
			if err := esutils.ValidateReferentSecretSelector(store, *auth.TokenSecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidTokenRef, err)
			}
		}
		if auth.UserPass != nil {
			if err := esutils.ValidateReferentSecretSelector(store, auth.UserPass.SecretRef); err != nil {
				return nil, fmt.Errorf(errInvalidUserPassSecretRef, err)
			}
		}
	}

	return nil, nil
}
