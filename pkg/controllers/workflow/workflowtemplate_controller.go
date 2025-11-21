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

// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

// Package workflow implements workflow controllers.
package workflow

import (
	"context"
	"fmt"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TemplateReconciler reconciles a WorkflowTemplate object.
type TemplateReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// Reconcile handles WorkflowTemplate resources.
func (r *TemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("workflowtemplate", req.NamespacedName)
	log.Info("reconciling WorkflowTemplate")

	// Fetch the WorkflowTemplate instance
	template := &workflows.WorkflowTemplate{}
	if err := r.Get(ctx, req.NamespacedName, template); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Validate the template
	if err := r.validateTemplate(template); err != nil {
		log.Error(err, "invalid template")
		r.Recorder.Event(template, "Warning", "ValidationFailed", fmt.Sprintf("Template validation failed: %v", err))
		return ctrl.Result{}, err
	}

	// Template is valid, nothing else to do
	r.Recorder.Event(template, "Normal", "Validated", "Template validation succeeded")
	return ctrl.Result{}, nil
}

// validateTemplate validates a WorkflowTemplate.
func (r *TemplateReconciler) validateTemplate(template *workflows.WorkflowTemplate) error {
	// TODO: The same validation logic for the workflows needs to be applied here too - jobs, dependencies, etc.

	if err := r.validateBasicTemplateFields(template); err != nil {
		return err
	}

	if err := r.validateParameterGroups(template.Spec.ParameterGroups); err != nil {
		return err
	}

	return nil
}

// validateBasicTemplateFields validates the basic required fields of a template.
func (r *TemplateReconciler) validateBasicTemplateFields(template *workflows.WorkflowTemplate) error {
	if template.Spec.Name == "" {
		return fmt.Errorf("template must have a name")
	}

	if template.Spec.Version == "" {
		return fmt.Errorf("template must have a version")
	}

	if len(template.Spec.Jobs) == 0 {
		return fmt.Errorf("template must have at least one job")
	}

	return nil
}

// validateParameterGroups validates all parameter groups and their parameters.
func (r *TemplateReconciler) validateParameterGroups(groups []workflows.ParameterGroup) error {
	paramNames := make(map[string]bool)
	groupNames := make(map[string]bool)

	for _, group := range groups {
		if err := r.validateParameterGroup(group, groupNames, paramNames); err != nil {
			return err
		}
	}

	return nil
}

// validateParameterGroup validates a single parameter group.
func (r *TemplateReconciler) validateParameterGroup(group workflows.ParameterGroup, groupNames, paramNames map[string]bool) error {
	if group.Name == "" {
		return fmt.Errorf("parameter group must have a name")
	}

	if groupNames[group.Name] {
		return fmt.Errorf("duplicate parameter group name: %s", group.Name)
	}
	groupNames[group.Name] = true

	for _, param := range group.Parameters {
		if err := r.validateParameter(param, paramNames); err != nil {
			return err
		}
	}

	return nil
}

// validateParameter validates a single parameter.
func (r *TemplateReconciler) validateParameter(param workflows.Parameter, paramNames map[string]bool) error {
	if param.Name == "" {
		return fmt.Errorf("parameter must have a name")
	}

	if paramNames[param.Name] {
		return fmt.Errorf("duplicate parameter name: %s", param.Name)
	}
	paramNames[param.Name] = true

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workflows.WorkflowTemplate{}).
		Complete(r)
}
