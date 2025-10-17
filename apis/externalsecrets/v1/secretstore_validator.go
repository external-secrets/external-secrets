/*
Copyright © 2025 ESO Maintainer Team

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

package v1

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Ensures ExternalSecretValidator implements the admission.CustomValidator interface correctly.
var _ admission.CustomValidator = &GenericStoreValidator{}

const (
	errInvalidStore       = "invalid store"
	warnStoreUnmaintained = "store %s isn't currently maintained. Please plan and prepare accordingly."
)

// GenericStoreValidator implements webhook validation for SecretStore and ClusterSecretStore resources.
type GenericStoreValidator struct{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	st, ok := obj.(GenericStore)
	if !ok {
		return nil, errors.New(errInvalidStore)
	}
	return validateStore(st)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	st, ok := newObj.(GenericStore)
	if !ok {
		return nil, errors.New(errInvalidStore)
	}
	return validateStore(st)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateStore(store GenericStore) (admission.Warnings, error) {
	if err := validateConditions(store); err != nil {
		return nil, err
	}

	provider, err := GetProvider(store)
	if err != nil {
		return nil, err
	}
	isMaintained, err := GetMaintenanceStatus(store)
	if err != nil {
		return nil, err
	}
	warns, err := provider.ValidateStore(store)
	if !isMaintained {
		warns = append(warns, fmt.Sprintf(warnStoreUnmaintained, store.GetName()))
	}
	return warns, err
}

func validateConditions(store GenericStore) error {
	var errs error
	for ci, condition := range store.GetSpec().Conditions {
		for ri, r := range condition.NamespaceRegexes {
			if _, err := regexp.Compile(r); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to compile %dth namespace regex in %dth condition: %w", ri, ci, err))
			}
		}
	}

	return errs
}
