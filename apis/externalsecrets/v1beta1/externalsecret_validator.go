/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package v1beta1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ExternalSecretValidator struct{}

func (esv *ExternalSecretValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, validateExternalSecret(obj)
}

func (esv *ExternalSecretValidator) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	return nil, validateExternalSecret(newObj)
}

func (esv *ExternalSecretValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateExternalSecret(obj runtime.Object) error {
	es, ok := obj.(*ExternalSecret)
	if !ok {
		return fmt.Errorf("unexpected type")
	}

	if (es.Spec.Target.DeletionPolicy == DeletionPolicyDelete && es.Spec.Target.CreationPolicy == CreatePolicyMerge) ||
		(es.Spec.Target.DeletionPolicy == DeletionPolicyDelete && es.Spec.Target.CreationPolicy == CreatePolicyNone) {
		return fmt.Errorf("deletionPolicy=Delete must not be used when the controller doesn't own the secret. Please set creationPolcy=Owner")
	}

	if es.Spec.Target.DeletionPolicy == DeletionPolicyMerge && es.Spec.Target.CreationPolicy == CreatePolicyNone {
		return fmt.Errorf("deletionPolicy=Merge must not be used with creationPolcy=None. There is no Secret to merge with")
	}

	for _, ref := range es.Spec.DataFrom {
		findOrExtract := ref.Find != nil || ref.Extract != nil
		if findOrExtract && ref.SourceRef != nil && ref.SourceRef.GeneratorRef != nil {
			return fmt.Errorf("generator can not be used with find or extract")
		}
	}

	return nil
}
