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

package v2alpha1

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Ensures validators implement the admission.CustomValidator interface correctly.
var _ admission.Validator[*ProviderStore] = &ProviderStoreValidator{}
var _ admission.Validator[*ClusterProviderStore] = &ClusterProviderStoreValidator{}

// ProviderStoreValidator implements webhook validation for ProviderStore resources.
type ProviderStoreValidator struct{}

// ClusterProviderStoreValidator implements webhook validation for ClusterProviderStore resources.
type ClusterProviderStoreValidator struct{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *ProviderStoreValidator) ValidateCreate(_ context.Context, obj *ProviderStore) (admission.Warnings, error) {
	return nil, validateProviderStoreSpec(obj.Namespace, obj.Spec.BackendRef, nil)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *ProviderStoreValidator) ValidateUpdate(_ context.Context, _, newObj *ProviderStore) (admission.Warnings, error) {
	return nil, validateProviderStoreSpec(newObj.Namespace, newObj.Spec.BackendRef, nil)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *ProviderStoreValidator) ValidateDelete(_ context.Context, _ *ProviderStore) (admission.Warnings, error) {
	return nil, nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *ClusterProviderStoreValidator) ValidateCreate(_ context.Context, obj *ClusterProviderStore) (admission.Warnings, error) {
	return nil, validateProviderStoreSpec("", obj.Spec.BackendRef, obj.Spec.Conditions)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *ClusterProviderStoreValidator) ValidateUpdate(_ context.Context, _, newObj *ClusterProviderStore) (admission.Warnings, error) {
	return nil, validateProviderStoreSpec("", newObj.Spec.BackendRef, newObj.Spec.Conditions)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *ClusterProviderStoreValidator) ValidateDelete(_ context.Context, _ *ClusterProviderStore) (admission.Warnings, error) {
	return nil, nil
}

func validateProviderStoreSpec(namespace string, backendRef BackendObjectReference, conditions []StoreNamespaceCondition) error {
	var errs error

	if namespace != "" && backendRef.Namespace != "" && backendRef.Namespace != namespace {
		errs = errors.Join(errs, fmt.Errorf("backendRef.namespace %q must match metadata.namespace %q", backendRef.Namespace, namespace))
	}

	for ci, condition := range conditions {
		for ri, expr := range condition.NamespaceRegexes {
			if _, err := regexp.Compile(expr); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to compile %dth namespace regex in %dth condition: %w", ri, ci, err))
			}
		}
	}

	return errs
}
