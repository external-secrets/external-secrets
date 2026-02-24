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

package v1beta1

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var _ admission.Validator[*esv1beta1.SecretStore] = &GenericStoreValidator{}
var _ admission.Validator[*esv1beta1.ClusterSecretStore] = &GenericClusterStoreValidator{}

// GenericStoreValidator provides validation for SecretStore resources.
type GenericStoreValidator struct{}

// GenericClusterStoreValidator provides validation for ClusterSecretStore resources.
type GenericClusterStoreValidator struct{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericClusterStoreValidator) ValidateCreate(_ context.Context, obj *esv1beta1.ClusterSecretStore) (admission.Warnings, error) {
	return validateStore(obj)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericClusterStoreValidator) ValidateUpdate(_ context.Context, _, newObj *esv1beta1.ClusterSecretStore) (admission.Warnings, error) {
	return validateStore(newObj)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericClusterStoreValidator) ValidateDelete(_ context.Context, _ *esv1beta1.ClusterSecretStore) (admission.Warnings, error) {
	return nil, nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateCreate(_ context.Context, obj *esv1beta1.SecretStore) (admission.Warnings, error) {
	return validateStore(obj)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateUpdate(_ context.Context, _, newObj *esv1beta1.SecretStore) (admission.Warnings, error) {
	return validateStore(newObj)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *GenericStoreValidator) ValidateDelete(_ context.Context, _ *esv1beta1.SecretStore) (admission.Warnings, error) {
	return nil, nil
}

func validateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	if err := validateConditions(store); err != nil {
		return nil, err
	}

	provider, err := esv1beta1.GetProvider(store)
	if err != nil {
		return nil, err
	}

	return provider.ValidateStore(store)
}

func validateConditions(store esv1beta1.GenericStore) error {
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
