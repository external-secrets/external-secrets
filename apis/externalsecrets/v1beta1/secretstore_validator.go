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

package v1beta1

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.Validator[*SecretStore] = &GenericStoreValidator{}
var _ admission.Validator[*ClusterSecretStore] = &GenericClusterStoreValidator{}

// GenericStoreValidator provides validation for SecretStore resources.
type GenericStoreValidator struct{}

// GenericClusterStoreValidator provides validation for ClusterSecretStore resources.
type GenericClusterStoreValidator struct{}

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

func validateStore(store GenericStore) (admission.Warnings, error) {
	if err := validateConditions(store); err != nil {
		return nil, err
	}
	if err := validateProviderMode(store); err != nil {
		return nil, err
	}
	if err := validateRuntimeRef(store); err != nil {
		return nil, err
	}
	if store.GetSpec() != nil && store.GetSpec().ProviderRef != nil {
		return nil, nil
	}

	provider, err := GetProvider(store)
	if err != nil {
		return nil, err
	}

	return provider.ValidateStore(store)
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

func validateProviderMode(store GenericStore) error {
	if store == nil || store.GetSpec() == nil {
		return nil
	}
	spec := store.GetSpec()
	hasProvider := spec.Provider != nil
	hasProviderRef := spec.ProviderRef != nil
	if hasProvider == hasProviderRef {
		return fmt.Errorf("exactly one of spec.provider or spec.providerRef must be set")
	}
	if hasProvider {
		if spec.RuntimeRef != nil {
			return fmt.Errorf("spec.runtimeRef must be empty when spec.provider is set")
		}
		return nil
	}
	if spec.RuntimeRef == nil {
		return fmt.Errorf("spec.runtimeRef is required when spec.providerRef is set")
	}
	if store.GetKind() == SecretStoreKind {
		namespace := spec.ProviderRef.Namespace
		if namespace != "" && namespace != store.GetObjectMeta().Namespace {
			return fmt.Errorf("spec.providerRef.namespace must be empty or match metadata.namespace")
		}
	}
	return nil
}

func validateRuntimeRef(store GenericStore) error {
	if store == nil || store.GetSpec() == nil {
		return nil
	}
	runtimeRef := store.GetSpec().RuntimeRef
	if runtimeRef == nil {
		return nil
	}
	if store.GetKind() == ClusterSecretStoreKind && runtimeRef.Kind == StoreRuntimeRefKindProviderClass {
		return fmt.Errorf("%s runtimeRef.kind must not be %q", store.GetKind(), runtimeRef.Kind)
	}
	return nil
}
