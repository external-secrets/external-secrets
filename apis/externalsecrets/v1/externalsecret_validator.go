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
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
func (in *ExternalSecretValidator) ValidateUpdate(_ context.Context, oldObj, newObj *ExternalSecret) (warnings admission.Warnings, err error) {
	// An unchanged spec can only reproduce the verdict the stored object already carries, so
	// refusing such an update prevents no invalid state. It only strands objects stored
	// before a rule tightened: the controller attaches its finalizer before it validates the
	// target, so a refusal there loops on the rejected patch instead of reaching the status
	// condition that explains the spec, and an operator cannot edit the object's metadata to
	// work around it either. Soundness rests on validateExternalSecretSpec reading nothing
	// but the spec, which its signature enforces.
	if oldObj != nil && newObj != nil && equality.Semantic.DeepEqual(oldObj.Spec, newObj.Spec) {
		return nil, nil
	}
	return validateExternalSecret(newObj)
}

// ValidateDelete validates the deletion of an external secret object.
func (in *ExternalSecretValidator) ValidateDelete(_ context.Context, _ *ExternalSecret) (warnings admission.Warnings, err error) {
	return nil, nil
}

// validateExternalSecret guards against a nil object and hands the spec to the rules. It
// holds no rules itself so that validateExternalSecretSpec can keep its narrow signature.
func validateExternalSecret(es *ExternalSecret) (admission.Warnings, error) {
	if es == nil {
		return nil, errors.New("external secret cannot be nil during validation")
	}

	return validateExternalSecretSpec(&es.Spec)
}

// validateExternalSecretSpec holds every ExternalSecret validation rule.
//
// Keep this signature. Taking the spec alone is what makes "the verdict depends on the spec
// and nothing else" true by construction, and ValidateUpdate relies on exactly that to admit
// updates that leave the spec untouched. Widening it to *ExternalSecret, to reach a name,
// label, annotation, owner reference or status field, would silently turn every rule in here
// into a create-only rule: on update the spec would still compare equal and ValidateUpdate
// would return before any of this ran. Nothing would fail, the rule would simply stop being
// enforced. So if a rule genuinely needs a field outside the spec, change ValidateUpdate to
// compare whole objects first, and expect that to reopen the stranding problem this shortcut
// exists to solve.
func validateExternalSecretSpec(spec *ExternalSecretSpec) (admission.Warnings, error) {
	var errs error
	if err := validatePolicies(spec); err != nil {
		errs = errors.Join(errs, err)
	}

	if len(spec.Data) == 0 && len(spec.DataFrom) == 0 {
		errs = errors.Join(errs, errors.New("either data or dataFrom should be specified"))
	}

	if err := validatePrivilegedTemplate(spec.Target.Template); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := ValidateManifestTarget(spec); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := validateTemplateFromTarget(spec); err != nil {
		errs = errors.Join(errs, err)
	}

	for _, ref := range spec.DataFrom {
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

	errs = validateDuplicateKeys(spec, errs)
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

func validatePolicies(spec *ExternalSecretSpec) error {
	var errs error
	if spec.Target.DeletionPolicy == DeletionPolicyDelete &&
		(spec.Target.CreationPolicy == CreatePolicyMerge ||
			spec.Target.CreationPolicy == CreatePolicyNone ||
			spec.Target.CreationPolicy == CreatePolicyCreateOrMerge) {
		errs = errors.Join(errs, errors.New("deletionPolicy=Delete must not be used when the controller doesn't own the secret. Please set creationPolicy=Owner"))
	}

	if spec.Target.DeletionPolicy == DeletionPolicyMerge && spec.Target.CreationPolicy == CreatePolicyNone {
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

// ErrManifestSecretTarget rejects a manifest target naming the core v1 Secret kind. A Secret
// is the default target, reached by leaving spec.target.manifest unset, and that path carries
// Secret-specific protections the generic path does not. Allowing the same resource to be
// requested either way would mean maintaining two sets of those protections.
var ErrManifestSecretTarget = errors.New(
	"spec.target.manifest must not target the core v1 Secret kind: remove spec.target.manifest to sync into a Secret")

// ValidateManifestTarget restricts what spec.target.manifest may declare. The admission
// webhook reaches it through validateExternalSecret; the controller calls it so stored
// objects and a bypassed webhook are covered too.
func ValidateManifestTarget(spec *ExternalSecretSpec) error {
	manifest := spec.Target.Manifest
	if manifest == nil {
		return nil
	}

	// Parsed the same way the controller resolves the target GVK, so "v1" and "/v1" both
	// land on the core group and neither can slip past. A custom group's Secret kind is
	// a different resource and stays allowed.
	gv, err := schema.ParseGroupVersion(manifest.APIVersion)
	if err != nil {
		return fmt.Errorf("invalid spec.target.manifest.apiVersion %q: %w", manifest.APIVersion, err)
	}
	if gv.WithKind(manifest.Kind) == corev1.SchemeGroupVersion.WithKind("Secret") {
		return ErrManifestSecretTarget
	}

	return nil
}

// validateTemplateFromTarget restricts templateFrom targets whenever the ExternalSecret
// renders into a Secret. Only the default target does: ValidateManifestTarget rejects a
// manifest naming the core Secret kind, so any manifest target is a non-Secret resource
// and keeps the nested paths custom resources need.
func validateTemplateFromTarget(spec *ExternalSecretSpec) error {
	if spec.Target.Manifest != nil {
		return nil
	}

	return ValidateSecretTemplateFromTargets(spec.Target.Template)
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

func validateDuplicateKeys(spec *ExternalSecretSpec, errs error) error {
	if spec.Target.DeletionPolicy == DeletionPolicyRetain {
		seenKeys := make(map[string]struct{})
		for _, data := range spec.Data {
			secretKey := data.SecretKey
			if _, exists := seenKeys[secretKey]; exists {
				errs = errors.Join(errs, fmt.Errorf("duplicate secretKey found: %s", secretKey))
			}
			seenKeys[secretKey] = struct{}{}
		}
	}
	return errs
}
