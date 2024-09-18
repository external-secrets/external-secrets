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

// Package conjur provides a Conjur provider for External Secrets.
package conjur

import (
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/conjur/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// ValidateStore validates the store.
func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	prov, err := util.GetConjurProvider(store)
	if err != nil {
		return nil, err
	}

	if prov.URL == "" {
		return nil, errors.New("conjur URL cannot be empty")
	}
	if prov.Auth.APIKey != nil {
		err := validateAPIKeyStore(store, *prov.Auth.APIKey)
		if err != nil {
			return nil, err
		}
	}

	if prov.Auth.Jwt != nil {
		err := validateJWTStore(store, *prov.Auth.Jwt)
		if err != nil {
			return nil, err
		}
	}

	// At least one auth must be configured
	if prov.Auth.APIKey == nil && prov.Auth.Jwt == nil {
		return nil, errors.New("missing Auth.* configuration")
	}

	return nil, nil
}

func validateAPIKeyStore(store esv1beta1.GenericStore, auth esv1beta1.ConjurAPIKey) error {
	if auth.Account == "" {
		return errors.New("missing Auth.ApiKey.Account")
	}
	if auth.UserRef == nil {
		return errors.New("missing Auth.Apikey.UserRef")
	}
	if auth.APIKeyRef == nil {
		return errors.New("missing Auth.Apikey.ApiKeyRef")
	}
	if err := utils.ValidateReferentSecretSelector(store, *auth.UserRef); err != nil {
		return fmt.Errorf("invalid Auth.Apikey.UserRef: %w", err)
	}
	if err := utils.ValidateReferentSecretSelector(store, *auth.APIKeyRef); err != nil {
		return fmt.Errorf("invalid Auth.Apikey.ApiKeyRef: %w", err)
	}
	return nil
}

func validateJWTStore(store esv1beta1.GenericStore, auth esv1beta1.ConjurJWT) error {
	if auth.Account == "" {
		return errors.New("missing Auth.Jwt.Account")
	}
	if auth.ServiceID == "" {
		return errors.New("missing Auth.Jwt.ServiceID")
	}
	if auth.ServiceAccountRef == nil && auth.SecretRef == nil {
		return errors.New("must specify Auth.Jwt.SecretRef or Auth.Jwt.ServiceAccountRef")
	}
	if auth.SecretRef != nil {
		if err := utils.ValidateReferentSecretSelector(store, *auth.SecretRef); err != nil {
			return fmt.Errorf("invalid Auth.Jwt.SecretRef: %w", err)
		}
	}
	if auth.ServiceAccountRef != nil {
		if err := utils.ValidateReferentServiceAccountSelector(store, *auth.ServiceAccountRef); err != nil {
			return fmt.Errorf("invalid Auth.Jwt.ServiceAccountRef: %w", err)
		}
	}
	return nil
}
