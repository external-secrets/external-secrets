// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// WorkflowRunValidator validates WorkflowRun resources.
type WorkflowRunValidator struct{}

var _ admission.CustomValidator = &WorkflowRunValidator{}

// ValidateCreate implements admission.CustomValidator so a webhook will be registered for the type.
func (v *WorkflowRunValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	workflowRun, ok := obj.(*WorkflowRun)
	if !ok {
		return nil, nil
	}
	return nil, validateWorkflowRunParameters(workflowRun)
}

// ValidateUpdate implements admission.CustomValidator so a webhook will be registered for the type.
func (v *WorkflowRunValidator) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	workflowRun, ok := newObj.(*WorkflowRun)
	if !ok {
		return nil, nil
	}
	return nil, validateWorkflowRunParameters(workflowRun)
}

// ValidateDelete implements admission.CustomValidator; no custom logic needed on delete.
func (v *WorkflowRunValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// SetupWebhookWithManager registers the webhook with the manager.
func (wr *WorkflowRun) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(wr).
		WithValidator(&WorkflowRunValidator{}).
		Complete()
}
