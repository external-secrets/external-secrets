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
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// RunReconciler reconciles a WorkflowRun object.
type RunReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflowruns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflowruns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflowtemplates,verbs=get;list;watch
//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles WorkflowRun resources.
func (r *RunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("workflowrun", req.NamespacedName)

	// Fetch the WorkflowRun instance
	run := &workflows.WorkflowRun{}
	if err := r.Get(ctx, req.NamespacedName, run); err != nil {
		// We'll ignore not-found errors, since they can't be fixed by an immediate requeue
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If workflow already created, check its status
	if run.Status.WorkflowRef != nil {
		return r.checkWorkflowStatus(ctx, run)
	}

	// Fetch the template
	template := &workflows.WorkflowTemplate{}
	templateNamespace := run.Spec.TemplateRef.Namespace
	if templateNamespace == "" {
		templateNamespace = run.Namespace
	}

	if err := r.Get(ctx, types.NamespacedName{
		Name:      run.Spec.TemplateRef.Name,
		Namespace: templateNamespace,
	}, template); err != nil {
		if errors.IsNotFound(err) {
			r.Recorder.Event(run, corev1.EventTypeWarning, "TemplateNotFound",
				fmt.Sprintf("Template %s not found in namespace %s", run.Spec.TemplateRef.Name, templateNamespace))
			// Update status with error condition
			run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
				Type:               "TemplateFound",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "TemplateNotFound",
				Message:            fmt.Sprintf("Template %s not found in namespace %s", run.Spec.TemplateRef.Name, templateNamespace),
			})
			if err := r.Status().Update(ctx, run); err != nil {
				log.Error(err, "unable to update WorkflowRun status")
				return ctrl.Result{}, err
			}
			// Requeue after some time in case the workflow template is created later
			return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
		}
		log.Error(err, "unable to fetch WorkflowTemplate")
		return ctrl.Result{}, err
	}

	// Create workflow from template
	workflow, err := r.resolveWorkflowFromTemplate(ctx, template, run)
	if err != nil {
		r.Recorder.Event(run, corev1.EventTypeWarning, "ResolutionFailed",
			fmt.Sprintf("Failed to resolve workflow from template: %v", err))
		// Update status with error condition
		run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
			Type:               "WorkflowResolved",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "ResolutionFailed",
			Message:            fmt.Sprintf("Failed to resolve workflow from template: %v", err),
		})
		if err := r.Status().Update(ctx, run); err != nil {
			log.Error(err, "unable to update WorkflowRun status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Set the WorkflowRun as the owner of the Workflow
	if err := controllerutil.SetControllerReference(run, workflow, r.Scheme); err != nil {
		log.Error(err, "unable to set controller reference on Workflow")
		return ctrl.Result{}, err
	}

	// Create the workflow
	if err := r.Create(ctx, workflow); err != nil {
		log.Error(err, "unable to create Workflow for WorkflowRun")
		return ctrl.Result{}, err
	}

	// Update run status with workflow reference
	run.Status.WorkflowRef = &workflows.WorkflowRef{
		Name:      workflow.Name,
		Namespace: workflow.Namespace,
	}

	// Add success condition
	run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
		Type:               "WorkflowCreated",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "WorkflowCreated",
		Message:            fmt.Sprintf("Created workflow %s from template %s", workflow.Name, template.Name),
	})

	// Update status
	if err := r.Status().Update(ctx, run); err != nil {
		log.Error(err, "unable to update WorkflowRun status")
		return ctrl.Result{}, err
	}

	r.Recorder.Event(run, corev1.EventTypeNormal, "WorkflowCreated",
		fmt.Sprintf("Created workflow %s from template %s", workflow.Name, template.Name))

	return ctrl.Result{}, nil
}

// checkWorkflowStatus checks the status of the created workflow and updates the WorkflowRun status accordingly.
func (r *RunReconciler) checkWorkflowStatus(ctx context.Context, run *workflows.WorkflowRun) (ctrl.Result, error) {
	log := r.Log.WithValues("workflowrun", types.NamespacedName{Name: run.Name, Namespace: run.Namespace})

	// Fetch the workflow
	workflow := &workflows.Workflow{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      run.Status.WorkflowRef.Name,
		Namespace: run.Status.WorkflowRef.Namespace,
	}, workflow); err != nil {
		if errors.IsNotFound(err) {
			// Workflow was deleted, update the status
			run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
				Type:               "WorkflowExists",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "WorkflowDeleted",
				Message:            fmt.Sprintf("Workflow %s was deleted", run.Status.WorkflowRef.Name),
			})
			if err := r.Status().Update(ctx, run); err != nil {
				log.Error(err, "unable to update WorkflowRun status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Workflow")
		return ctrl.Result{}, err
	}

	// Check if the workflow status has changed
	statusChanged := false

	// Copy the workflow phase directly to WorkflowRun
	if run.Status.Phase != workflow.Status.Phase {
		run.Status.Phase = workflow.Status.Phase
		statusChanged = true
	}

	// Copy timing information
	if workflow.Status.StartTime != nil && run.Status.StartTime == nil {
		run.Status.StartTime = workflow.Status.StartTime
		statusChanged = true
	}

	if workflow.Status.CompletionTime != nil && run.Status.CompletionTime == nil {
		run.Status.CompletionTime = workflow.Status.CompletionTime
		run.Status.ExecutionTimeNanos = workflow.Status.ExecutionTimeNanos
		statusChanged = true
	}

	// Copy workflow conditions to WorkflowRun
	for _, cond := range workflow.Status.Conditions {
		// Check if this condition already exists in the WorkflowRun
		exists := false
		for _, runCond := range run.Status.Conditions {
			if runCond.Type == cond.Type && runCond.Status == cond.Status && runCond.Reason == cond.Reason {
				exists = true
				break
			}
		}

		if !exists {
			run.Status.Conditions = append(run.Status.Conditions, cond)
			statusChanged = true
		}
	}

	// Translate workflow phase into WorkflowRun conditions
	switch workflow.Status.Phase {
	case workflows.PhasePending:
		// Check if Pending condition already exists
		pendingExists := false
		for _, runCond := range run.Status.Conditions {
			if runCond.Type == "Pending" && runCond.Status == metav1.ConditionTrue {
				pendingExists = true
				break
			}
		}
		if !pendingExists {
			run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
				Type:               "Pending",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "WorkflowPending",
				Message:            fmt.Sprintf("Workflow %s is pending", workflow.Name),
			})
			statusChanged = true
		}
	case workflows.PhaseFailed:
		// Check if Failed condition already exists
		failedExists := false
		for _, runCond := range run.Status.Conditions {
			if runCond.Type == "Failed" && runCond.Status == metav1.ConditionTrue {
				failedExists = true
				break
			}
		}
		if !failedExists {
			run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
				Type:               "Failed",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "WorkflowFailed",
				Message:            fmt.Sprintf("Workflow %s failed", workflow.Name),
			})
			statusChanged = true
		}
	case workflows.PhaseSucceeded:
		// Check if Succeeded condition already exists
		succeededExists := false
		for _, runCond := range run.Status.Conditions {
			if runCond.Type == "Succeeded" && runCond.Status == metav1.ConditionTrue {
				succeededExists = true
				break
			}
		}
		if !succeededExists {
			run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
				Type:               "Succeeded",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "WorkflowSucceeded",
				Message:            fmt.Sprintf("Workflow %s completed successfully", workflow.Name),
			})
			statusChanged = true
		}
	case workflows.PhaseRunning:
		// Check if Running condition already exists
		runningExists := false
		for _, runCond := range run.Status.Conditions {
			if runCond.Type == "Running" && runCond.Status == metav1.ConditionTrue {
				runningExists = true
				break
			}
		}
		if !runningExists {
			run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
				Type:               "Running",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "WorkflowRunning",
				Message:            fmt.Sprintf("Workflow %s is running", workflow.Name),
			})
			statusChanged = true
		}
	}

	if statusChanged {
		if err := r.Status().Update(ctx, run); err != nil {
			log.Error(err, "unable to update WorkflowRun status")
			return ctrl.Result{}, err
		}
	}

	// Requeue to check for status updates
	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// resolveWorkflowFromTemplate creates a new Workflow from a WorkflowTemplate and WorkflowRun.
func (r *RunReconciler) resolveWorkflowFromTemplate(ctx context.Context, template *workflows.WorkflowTemplate, run *workflows.WorkflowRun) (*workflows.Workflow, error) {
	// Create a new workflow
	workflow := &workflows.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      run.Name,
			Namespace: run.Namespace,
			Labels: map[string]string{
				"workflows.external-secrets.io/template": template.Name,
				"workflows.external-secrets.io/run":      run.Name,
			},
		},
		Spec: workflows.WorkflowSpec{
			Version:   template.Spec.Version,
			Name:      template.Spec.Name,
			Variables: apiextensionsv1.JSON{},
			Jobs:      template.Spec.Jobs,
		},
	}
	runLabels := run.GetLabels()
	if _, ok := runLabels["workflowruntemplate.external-secrets.io/owner"]; ok {
		workflow.Labels["workflows.external-secrets.io/runtemplate"] = runLabels["workflowruntemplate.external-secrets.io/owner"]
	}

	toParseArguments, err := json.Marshal(run.Spec.Arguments)
	if err != nil {
		return nil, fmt.Errorf("error marshaling arguments from run %s: %w", run.Name, err)
	}
	var parsedArguments map[string]interface{}
	err = json.Unmarshal(toParseArguments, &parsedArguments)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling arguments from run %s: %w", run.Name, err)
	}

	// Convert arguments to variables
	variables := make(map[string]any)
	for _, group := range template.Spec.ParameterGroups {
		for _, param := range group.Parameters {
			value, exists := parsedArguments[param.Name]
			if !exists {
				if param.Required && param.Default == "" {
					return nil, fmt.Errorf("required parameter %s not provided", param.Name)
				}
				if param.Default != "" {
					variables[param.Name] = param.Default
				}
			} else {
				if err := param.ValidateValue(value); err != nil {
					return nil, fmt.Errorf("invalid argument value for param %s: %v", param.Name, value)
				}

				parsedValue, err := r.parseCustomTypes(ctx, run.Namespace, param, value)
				if err != nil {
					return nil, fmt.Errorf("error parsing custom types: %w", err)
				}

				variables[param.Name] = parsedValue
			}
		}
	}

	variablesBytes, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("error marshaling default variables")
	}
	workflow.Spec.Variables = apiextensionsv1.JSON{
		Raw: variablesBytes,
	}

	return workflow, nil
}

func (r *RunReconciler) parseCustomTypes(ctx context.Context, namespace string, param workflows.Parameter, value interface{}) (interface{}, error) {
	if param.Type == workflows.ParameterTypeFinding {
		findingValue, err := param.ToFindingParameterType(value)
		if err != nil {
			return nil, fmt.Errorf("error converting value %v to finding type: %w", value, err)
		}

		locations, err := r.getLocationsArrayFromFindingParam(
			ctx, namespace, param, findingValue,
		)
		if err != nil {
			return nil, fmt.Errorf("error getting locations from finding param: %w", err)
		}

		value = locations
	}

	if param.Type == workflows.ParameterTypeFindingArray {
		findingArrayValue, err := param.ToFindingParameterTypeArray(value)
		if err != nil {
			return nil, fmt.Errorf("error converting value %v to finding array type: %w", value, err)
		}

		locations := make([]scanv1alpha1.SecretInStoreRef, 0, len(findingArrayValue))
		for _, findingValue := range findingArrayValue {
			location, err := r.getLocationsArrayFromFindingParam(
				ctx, namespace, param, &findingValue,
			)
			if err != nil {
				return nil, fmt.Errorf("error getting locations from finding param: %w", err)
			}
			locations = append(locations, location...)
		}

		value = locations
	}

	ok, err := param.Type.IsCustomObjectType()
	if err != nil {
		return nil, fmt.Errorf("error checking if param is custom object type: %w", err)
	}

	if ok {
		objectValue, err := param.ParseCustomObject(value)
		if err != nil {
			return nil, fmt.Errorf("error parsing custom object: %w", err)
		}

		customType := param.Type.ExtractCustomObjectType()
		param.Type = customType
		for key, customValue := range objectValue {
			parsedCustomValue, err := r.parseCustomTypes(ctx, namespace, param, customValue)
			if err != nil {
				return nil, fmt.Errorf("error parsing value of key %s from custom object: %w", key, err)
			}
			objectValue[key] = parsedCustomValue
		}
		value = objectValue
	}

	return value, nil
}

func (r *RunReconciler) getLocationsArrayFromFindingParam(ctx context.Context, resourceNamespace string, param workflows.Parameter, findingValue *workflows.FindingParameterType) ([]scanv1alpha1.SecretInStoreRef, error) {
	if param.ResourceConstraints != nil && param.ResourceConstraints.Namespace != "" {
		resourceNamespace = param.ResourceConstraints.Namespace
	}

	finding := &scanv1alpha1.Finding{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: resourceNamespace,
		Name:      findingValue.Name,
	}, finding)

	if err != nil {
		return nil, fmt.Errorf("error getting finding %s: %w", findingValue.Name, err)
	}

	return finding.Status.Locations, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workflows.WorkflowRun{}).
		Owns(&workflows.Workflow{}).
		Complete(r)
}
