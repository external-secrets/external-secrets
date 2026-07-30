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

package v1

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Ensures ExternalSecretValidator implements the admission.CustomValidator interface correctly.
var _ admission.Validator[*ExternalSecret] = &ExternalSecretValidator{}

// ExternalSecretValidator implements a validating webhook for ExternalSecrets.
type ExternalSecretValidator struct{}

// ValidateCreate validates the creation of an external secret object.
func (in *ExternalSecretValidator) ValidateCreate(_ context.Context, obj *ExternalSecret) (warnings admission.Warnings, err error) {
	return validateExternalSecret(obj)
}

// ValidateUpdate validates the update of an external secret object.
func (in *ExternalSecretValidator) ValidateUpdate(_ context.Context, _, newObj *ExternalSecret) (warnings admission.Warnings, err error) {
	return validateExternalSecret(newObj)
}

// ValidateDelete validates the deletion of an external secret object.
func (in *ExternalSecretValidator) ValidateDelete(_ context.Context, _ *ExternalSecret) (warnings admission.Warnings, err error) {
	return nil, nil
}

func validateExternalSecret(es *ExternalSecret) (admission.Warnings, error) {
	if es == nil {
		return nil, errors.New("external secret cannot be nil during validation")
	}

	var errs error
	if err := validatePolicies(es); err != nil {
		errs = errors.Join(errs, err)
	}

	if len(es.Spec.Data) == 0 && len(es.Spec.DataFrom) == 0 {
		errs = errors.Join(errs, errors.New("either data or dataFrom should be specified"))
	}

	if err := validatePrivilegedTemplate(es.Spec.Target.Template); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := validateTemplateFromTarget(es); err != nil {
		errs = errors.Join(errs, err)
	}

	for _, ref := range es.Spec.DataFrom {
		if err := validateExtractFindGenerator(ref); err != nil {
			errs = errors.Join(errs, err)
		}

		if err := validateFindExtractSourceRef(ref); err != nil {
			errs = errors.Join(errs, err)
		}

		if err := validateSourceRef(ref); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	errs = validateDuplicateKeys(es, errs)
	return nil, errs
}

func validateSourceRef(ref ExternalSecretDataFromRemoteRef) error {
	if ref.SourceRef != nil && ref.SourceRef.GeneratorRef == nil && ref.SourceRef.SecretStoreRef == nil {
		return errors.New("generatorRef or storeRef must be set when using sourceRef in dataFrom")
	}

	return nil
}

func validateFindExtractSourceRef(ref ExternalSecretDataFromRemoteRef) error {
	if ref.Find == nil && ref.Extract == nil && ref.SourceRef == nil {
		return errors.New("either extract, find, or sourceRef must be set to dataFrom")
	}

	return nil
}

func validateExtractFindGenerator(ref ExternalSecretDataFromRemoteRef) error {
	generatorRef := ref.SourceRef != nil && ref.SourceRef.GeneratorRef != nil
	if (ref.Find != nil && (ref.Extract != nil || generatorRef)) || (ref.Extract != nil && (ref.Find != nil || generatorRef)) || (generatorRef && (ref.Find != nil || ref.Extract != nil)) {
		return errors.New("extract, find, or generatorRef cannot be set at the same time")
	}

	return nil
}

func validatePolicies(es *ExternalSecret) error {
	var errs error
	if es.Spec.Target.DeletionPolicy == DeletionPolicyDelete &&
		(es.Spec.Target.CreationPolicy == CreatePolicyMerge ||
			es.Spec.Target.CreationPolicy == CreatePolicyNone ||
			es.Spec.Target.CreationPolicy == CreatePolicyCreateOrMerge) {
		errs = errors.Join(errs, errors.New("deletionPolicy=Delete must not be used when the controller doesn't own the secret. Please set creationPolicy=Owner"))
	}

	if es.Spec.Target.DeletionPolicy == DeletionPolicyMerge && es.Spec.Target.CreationPolicy == CreatePolicyNone {
		errs = errors.Join(errs, errors.New("deletionPolicy=Merge must not be used with creationPolicy=None. There is no Secret to merge with"))
	}

	return errs
}

// validatePrivilegedTemplate rejects templates with specific types and annotations combinations
// to prevent users from creating long-lived tokens beyond the scope of the defined RBAC.
func validatePrivilegedTemplate(tpl *ExternalSecretTemplate) error {
	if tpl == nil {
		return nil
	}
	//nolint:exhaustive // don't need exhaustive
	switch tpl.Type {
	case corev1.SecretTypeServiceAccountToken:
		if _, ok := tpl.Metadata.Annotations[corev1.ServiceAccountNameKey]; ok {
			return fmt.Errorf("template.type=%q with annotation %q is not allowed", corev1.SecretTypeServiceAccountToken, corev1.ServiceAccountNameKey)
		}
		for _, tf := range tpl.TemplateFrom {
			if strings.EqualFold(tf.Target, TemplateTargetAnnotations) {
				return fmt.Errorf("template.type=%q with templateFrom target=%q is not allowed", corev1.SecretTypeServiceAccountToken, TemplateTargetAnnotations)
			}
		}
	case corev1.SecretTypeBootstrapToken:
		return fmt.Errorf("template.type=%q is not allowed", corev1.SecretTypeBootstrapToken)
	}
	return nil
}

// ValidateSecretTemplate applies every template restriction that must hold when an
// ExternalSecret renders into a Secret. The admission webhook reaches these rules through
// validateExternalSecret; the controller calls this so the same set is enforced when no
// webhook sits in front of it.
func ValidateSecretTemplate(tpl *ExternalSecretTemplate) error {
	return errors.Join(
		validatePrivilegedTemplate(tpl),
		ValidateSecretTemplateFromTargets(tpl),
	)
}

// isManifestSecretTarget reports whether the ExternalSecret renders into a core/v1 Secret.
// That is the default target, and it is also reachable through an explicit manifest
// reference naming a Secret.
func isManifestSecretTarget(es *ExternalSecret) bool {
	manifest := es.Spec.Target.Manifest
	if manifest == nil {
		return true
	}
	return manifest.APIVersion == "v1" && manifest.Kind == "Secret"
}

// validateTemplateFromTarget restricts templateFrom targets whenever the ExternalSecret
// renders into a Secret.
func validateTemplateFromTarget(es *ExternalSecret) error {
	if !isManifestSecretTarget(es) {
		return nil
	}

	return ValidateSecretTemplateFromTargets(es.Spec.Target.Template)
}

// ValidateSecretTemplateFromTargets restricts templateFrom targets to the well-known Secret
// fields. The templating engine treats any other value as a dotted path into the rendered
// object, which would let a user write privileged top-level fields such as type, immutable
// or metadata.ownerReferences and so sidestep validatePrivilegedTemplate. Nested paths
// remain available for custom resource targets.
func ValidateSecretTemplateFromTargets(tpl *ExternalSecretTemplate) error {
	if tpl == nil {
		return nil
	}

	var errs error
	for _, tf := range tpl.TemplateFrom {
		switch {
		case tf.Target == "",
			strings.EqualFold(tf.Target, TemplateTargetData),
			strings.EqualFold(tf.Target, TemplateTargetAnnotations),
			strings.EqualFold(tf.Target, TemplateTargetLabels):
			continue
		}

		errs = errors.Join(errs, fmt.Errorf(
			"templateFrom target=%q is not allowed when targeting a Secret, must be one of %q, %q or %q",
			tf.Target, TemplateTargetData, TemplateTargetAnnotations, TemplateTargetLabels))
	}

	return errs
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
