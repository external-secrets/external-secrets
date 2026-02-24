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

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// Ensures GenericStoreValidator implements the admission.Validator interface correctly.
var _ admission.Validator[*esv1.SecretStore] = &GenericStoreValidator{}
var _ admission.Validator[*esv1.ClusterSecretStore] = &GenericClusterStoreValidator{}

const (
	warnStoreUnmaintained = "store %s isn't currently maintained. Please plan and prepare accordingly."
	warnStoreDeprecated   = "store %s is deprecated and will stop working on the next major version. Please plan and prepare accordingly."
)

// GenericStoreValidator implements webhook validation for SecretStore resources.
type GenericStoreValidator struct{}

// GenericClusterStoreValidator implements webhook validation for ClusterSecretStore resources.
type GenericClusterStoreValidator struct{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateCreate(_ context.Context, obj *esv1.SecretStore) (admission.Warnings, error) {
	return validateStore(obj)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateUpdate(_ context.Context, _, newObj *esv1.SecretStore) (admission.Warnings, error) {
	return validateStore(newObj)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateDelete(_ context.Context, _ *esv1.SecretStore) (admission.Warnings, error) {
	return nil, nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericClusterStoreValidator) ValidateCreate(_ context.Context, obj *esv1.ClusterSecretStore) (admission.Warnings, error) {
	return validateStore(obj)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericClusterStoreValidator) ValidateUpdate(_ context.Context, _, newObj *esv1.ClusterSecretStore) (admission.Warnings, error) {
	return validateStore(newObj)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericClusterStoreValidator) ValidateDelete(_ context.Context, _ *esv1.ClusterSecretStore) (admission.Warnings, error) {
	return nil, nil
}

func validateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if err := validateConditions(store); err != nil {
		return nil, err
	}

	provider, err := esv1.GetProvider(store)
	if err != nil {
		return nil, err
	}
	status, err := esv1.GetMaintenanceStatus(store)
	if err != nil {
		return nil, err
	}
	warns, err := provider.ValidateStore(store)
	switch status {
	case esv1.MaintenanceStatusNotMaintained:
		warns = append(warns, fmt.Sprintf(warnStoreUnmaintained, store.GetName()))
	case esv1.MaintenanceStatusDeprecated:
		warns = append(warns, fmt.Sprintf(warnStoreDeprecated, store.GetName()))
	case esv1.MaintenanceStatusMaintained:
	default:
		// no warnings
	}
	return warns, err
}

func validateConditions(store esv1.GenericStore) error {
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
