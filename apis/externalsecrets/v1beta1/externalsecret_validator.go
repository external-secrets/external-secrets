//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ExternalSecretValidator struct{}

func (esv *ExternalSecretValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return validateExternalSecret(obj)
}

func (esv *ExternalSecretValidator) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	return validateExternalSecret(newObj)
}

func (esv *ExternalSecretValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateExternalSecret(obj runtime.Object) (admission.Warnings, error) {
	es, ok := obj.(*ExternalSecret)
	if !ok {
		return nil, errors.New("unexpected type")
	}

	var errs error
	if (es.Spec.Target.DeletionPolicy == DeletionPolicyDelete && es.Spec.Target.CreationPolicy == CreatePolicyMerge) ||
		(es.Spec.Target.DeletionPolicy == DeletionPolicyDelete && es.Spec.Target.CreationPolicy == CreatePolicyNone) {
		errs = errors.Join(errs, errors.New("deletionPolicy=Delete must not be used when the controller doesn't own the secret. Please set creationPolicy=Owner"))
	}

	if es.Spec.Target.DeletionPolicy == DeletionPolicyMerge && es.Spec.Target.CreationPolicy == CreatePolicyNone {
		errs = errors.Join(errs, errors.New("deletionPolicy=Merge must not be used with creationPolicy=None. There is no Secret to merge with"))
	}

	if len(es.Spec.Data) == 0 && len(es.Spec.DataFrom) == 0 {
		errs = errors.Join(errs, errors.New("either data or dataFrom should be specified"))
	}

	for _, ref := range es.Spec.DataFrom {
		generatorRef := ref.SourceRef != nil && ref.SourceRef.GeneratorRef != nil
		if (ref.Find != nil && (ref.Extract != nil || generatorRef)) || (ref.Extract != nil && (ref.Find != nil || generatorRef)) || (generatorRef && (ref.Find != nil || ref.Extract != nil)) {
			errs = errors.Join(errs, errors.New("extract, find, or generatorRef cannot be set at the same time"))
		}

		if ref.Find == nil && ref.Extract == nil && ref.SourceRef == nil {
			errs = errors.Join(errs, errors.New("either extract, find, or sourceRef must be set to dataFrom"))
		}

		if ref.SourceRef != nil && ref.SourceRef.GeneratorRef == nil && ref.SourceRef.SecretStoreRef == nil {
			errs = errors.Join(errs, errors.New("generatorRef or storeRef must be set when using sourceRef in dataFrom"))
		}
	}

	errs = validateDuplicateKeys(es, errs)
	return nil, errs
}

func validateDuplicateKeys(es *ExternalSecret, errs error) error {
	if es.Spec.Target.DeletionPolicy == DeletionPolicyRetain {
		seenKeys := make(map[string]struct{})
		for _, data := range es.Spec.Data {
			secretKey := data.SecretKey
			if _, exists := seenKeys[secretKey]; exists {
				errs = errors.Join(errs, fmt.Errorf("duplicate secretKey found: %s", secretKey))
			}
			seenKeys[secretKey] = struct{}{}
		}
	}
	return errs
}
