//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.CustomValidator = &GenericStoreValidator{}

const (
	errInvalidStore = "invalid store"
)

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
