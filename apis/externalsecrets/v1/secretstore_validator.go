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
)

// Ensures ExternalSecretValidator implements the admission.CustomValidator interface correctly.
var _ admission.Validator[*SecretStore] = &GenericStoreValidator{}
var _ admission.Validator[*ClusterSecretStore] = &GenericClusterStoreValidator{}

const (
	warnStoreUnmaintained = "store %s isn't currently maintained. Please plan and prepare accordingly."
	warnStoreDeprecated   = "store %s is deprecated and will stop working on the next major version. Please plan and prepare accordingly."
)

// GenericStoreValidator implements webhook validation for SecretStore resources.
type GenericStoreValidator struct{}

// GenericClusterStoreValidator implements webhook validation for ClusterSecretStore resources.
type GenericClusterStoreValidator struct{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateCreate(_ context.Context, obj *SecretStore) (admission.Warnings, error) {
	return validateStore(obj)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateUpdate(_ context.Context, _, newObj *SecretStore) (admission.Warnings, error) {
	return validateStore(newObj)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateDelete(_ context.Context, _ *SecretStore) (admission.Warnings, error) {
	return nil, nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericClusterStoreValidator) ValidateCreate(_ context.Context, obj *ClusterSecretStore) (admission.Warnings, error) {
	return validateStore(obj)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericClusterStoreValidator) ValidateUpdate(_ context.Context, _, newObj *ClusterSecretStore) (admission.Warnings, error) {
	return validateStore(newObj)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericClusterStoreValidator) ValidateDelete(_ context.Context, _ *ClusterSecretStore) (admission.Warnings, error) {
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
	status, err := GetMaintenanceStatus(store)
	if err != nil {
		return nil, err
	}
	warns, err := provider.ValidateStore(store)
	switch status {
	case MaintenanceStatusNotMaintained:
		warns = append(warns, fmt.Sprintf(warnStoreUnmaintained, store.GetName()))
	case MaintenanceStatusDeprecated:
		warns = append(warns, fmt.Sprintf(warnStoreDeprecated, store.GetName()))
	case MaintenanceStatusMaintained:
	default:
		// no warnings
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
