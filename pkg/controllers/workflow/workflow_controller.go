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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/common"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/jobs"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/templates"
)

// Value represents a variable value that can be of different types.
type Value struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// Reconciler watches Workflow resources and manages their lifecycle.
type Reconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Manager  secretstore.ManagerInterface
}

//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflowruns,verbs=get;list;watch
//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflowruns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is the main entrypoint for reconciliation.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("workflow", req.NamespacedName)

	// Fetch the Workflow instance.
	wf := &workflows.Workflow{}
	if err := r.Get(ctx, req.NamespacedName, wf); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Merge sensitive values from secrets
	if err := r.mergeSensitiveValues(ctx, wf); err != nil {
		log.Error(err, "failed to merge sensitive values")
		// Continue with reconciliation even if merging fails
	}

	// Run the validation function.
	if err := validateWorkflowSpec(wf); err != nil {
		log.Error(err, "workflow spec validation failed")
		return r.markWorkflowFailed(ctx, wf, "ValidationError", err.Error())
	}

	// If the workflow is already in a terminal phase (Succeeded or Failed),
	// there is nothing to do.
	if wf.Status.Phase == workflows.PhaseSucceeded || wf.Status.Phase == workflows.PhaseFailed {
		return ctrl.Result{}, nil
	}

	// If the workflow has not been initialized yet, initialize it.
	if wf.Status.Phase == "" {
		return r.initializeWorkflow(ctx, wf, log)
	}

	// If the execution order is not yet calculated, do so now.
	if wf.Status.ExecutionOrder == nil {
		order, err := r.calculateExecutionOrder(wf.Spec.Jobs)
		if err != nil {
			return r.markWorkflowFailed(ctx, wf, "DependencyCycle", err.Error())
		}
		wf.Status.ExecutionOrder = order
		return r.updateStatusWithEvent(ctx, wf,
			ctrl.Result{}, ctrl.Result{RequeueAfter: 5 * time.Second},
			"Normal", "WorkflowInitialized", fmt.Sprintf("Workflow %s execution order calculated", wf.Name))
	}

	// Process the jobs as per the workflow logic.
	allJobsCompleted, procResult, err := r.processJobs(ctx, wf)
	if err != nil {
		return procResult, err
	}

	if allJobsCompleted {
		// All jobs finished. Now mark the workflow as completed.
		return r.markWorkflowCompleted(ctx, wf)
	}

	// If not all jobs have completed, continue processing.
	return r.updateStatusWithEvent(ctx, wf,
		ctrl.Result{Requeue: true}, ctrl.Result{RequeueAfter: 5 * time.Second},
		"Normal", "WorkflowRunning", fmt.Sprintf("Workflow %s is running", wf.Name))
}

// processJobs iterates over the jobs in execution order, starting pending jobs
// when their dependencies are met and executing running jobs.
func (r *Reconciler) processJobs(ctx context.Context, wf *workflows.Workflow) (bool, ctrl.Result, error) {
	allJobsCompleted := true // Assume all jobs are completed initially.

	for _, jobName := range wf.Status.ExecutionOrder {
		jobStatus := wf.Status.JobStatuses[jobName]
		jobSpec := wf.Spec.Jobs[jobName]

		// If the job is not succeeded or failed yet, it means jobs are not complete.
		if jobStatus.Phase != workflows.JobPhaseSucceeded && jobStatus.Phase != workflows.JobPhaseFailed {
			allJobsCompleted = false
		}

		switch jobStatus.Phase {
		case workflows.JobPhasePending:
			// Check if dependencies are met before starting the job.
			if r.jobDependenciesMet(jobSpec, wf.Status.JobStatuses) {
				jobStatus.Phase = workflows.JobPhaseRunning
				now := metav1.Now()
				jobStatus.StartTime = &now
				wf.Status.JobStatuses[jobName] = jobStatus

				// Mark the workflow as running.
				if wf.Status.Phase != workflows.PhaseRunning {
					wf.Status.Phase = workflows.PhaseRunning
					r.Recorder.Eventf(wf, "Normal", "WorkflowRunning", "Workflow %s is now running", wf.Name)
				}
				r.Recorder.Eventf(wf, "Normal", "JobStarted", "Job %s started", jobName)
			}
			// Dependents not met; leave job pending and continue.
		case workflows.JobPhaseRunning:
			// Execute the job and handle errors.
			if err := r.executeJob(ctx, wf, jobName, &jobStatus); err != nil {
				res, markErr := r.markJobFailed(ctx, wf, jobName, err)
				return false, res, markErr
			}
			wf.Status.JobStatuses[jobName] = jobStatus
		case workflows.JobPhaseSucceeded, workflows.JobPhaseFailed:
			// No action needed for already completed jobs.
			continue
		default:
			// Handle any unexpected states (if applicable).
			continue
		}
	}

	// Return whether all jobs are completed after processing.
	return allJobsCompleted, ctrl.Result{}, nil
}

// initializeWorkflow creates initial status values for the workflow and its jobs.
func (r *Reconciler) initializeWorkflow(ctx context.Context, wf *workflows.Workflow, log logr.Logger) (ctrl.Result, error) {
	log.Info("Initializing workflow")
	wf.Status = workflows.WorkflowStatus{
		Phase:       workflows.PhasePending,
		StartTime:   &metav1.Time{Time: time.Now()},
		JobStatuses: make(map[string]workflows.JobStatus),
	}

	for jobName := range wf.Spec.Jobs {
		wf.Status.JobStatuses[jobName] = workflows.JobStatus{
			Phase:        workflows.JobPhasePending,
			StepStatuses: make(map[string]workflows.StepStatus),
		}
	}

	return r.updateStatusWithEvent(ctx, wf,
		ctrl.Result{}, ctrl.Result{Requeue: true},
		"Normal", "WorkflowInitialized", fmt.Sprintf("Workflow %s initialized", wf.Name))
}

// executeJob processes a job using the appropriate job executor based on its type.
func (r *Reconciler) executeJob(ctx context.Context, wf *workflows.Workflow, jobName string, jobStatus *workflows.JobStatus) error {
	r.Log.Info("Executing job", "job", jobName)
	jobSpec := wf.Spec.Jobs[jobName]

	// Create the appropriate job executor based on job type
	jobExecutor, err := jobs.CreateJobExecutor(jobSpec, r.Scheme, r.Log, r.Manager)
	if err != nil {
		return fmt.Errorf("failed to create job executor: %w", err)
	}

	// Execute the job
	if err := jobExecutor.Execute(ctx, r.Client, wf, jobName, jobStatus); err != nil {
		return err
	}

	r.Recorder.Eventf(wf, "Normal", "JobCompleted", "Job %s completed", jobName)
	return nil
}

// calculateExecutionOrder determines a valid order for running jobs based on dependencies.
func (r *Reconciler) calculateExecutionOrder(jobs map[string]workflows.Job) ([]string, error) {
	inDegree := make(map[string]int)
	adj := make(map[string][]string)

	// Initialize in-degrees and validate dependencies.
	for jobName, job := range jobs {
		if _, exists := inDegree[jobName]; !exists {
			inDegree[jobName] = 0
		}
		for _, dep := range job.DependsOn {
			if _, exists := jobs[dep]; !exists {
				return nil, fmt.Errorf("job %s depends on non-existent job %s", jobName, dep)
			}
			adj[dep] = append(adj[dep], jobName)
			inDegree[jobName]++
		}
	}

	// Enqueue jobs with zero in-degree.
	var queue []string
	for jobName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, jobName)
		}
	}

	// Process the graph.
	var order []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)
		for _, dependent := range adj[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Detect cycles.
	if len(order) != len(jobs) {
		var missing []string
		for jobName := range jobs {
			found := false
			for _, orderedJob := range order {
				if orderedJob == jobName {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, jobName)
			}
		}
		return nil, fmt.Errorf("cyclic dependencies detected involving jobs: %v", missing)
	}

	return order, nil
}

// jobDependenciesMet returns true if all dependencies for a job have succeeded.
func (r *Reconciler) jobDependenciesMet(job workflows.Job, statuses map[string]workflows.JobStatus) bool {
	for _, dep := range job.DependsOn {
		if statuses[dep].Phase != workflows.JobPhaseSucceeded {
			return false
		}
	}
	return true
}

// resolveJobVariables resolves any templates in the job's variables using a data context.
func (r *Reconciler) resolveJobVariables(job *workflows.Job, wf *workflows.Workflow) (map[string]interface{}, error) {
	toParseWfVariables, err := json.Marshal(wf.Spec.Variables)
	if err != nil {
		return nil, fmt.Errorf("error marshaling variables from workflow %s: %w", wf.Name, err)
	}
	var parsedWfVariables map[string]interface{}
	err = json.Unmarshal(toParseWfVariables, &parsedWfVariables)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling variables from workflow %s: %w", wf.Name, err)
	}

	variables := make(map[string]interface{}, len(job.Variables))

	data := map[string]interface{}{
		"global": map[string]interface{}{
			"variables": parsedWfVariables,
			"jobs":      wf.Status.JobStatuses,
		},
	}
	for name, value := range job.Variables {
		// Try to unmarshal as WorkflowValue first
		var wfValue Value
		if err := json.Unmarshal([]byte(value), &wfValue); err == nil {
			// Handle specific types
			switch wfValue.Type {
			case "null":
				variables[name] = nil
			case "boolean":
				if b, ok := wfValue.Value.(bool); ok {
					variables[name] = b
				} else {
					return nil, fmt.Errorf("invalid boolean value for variable %s", name)
				}
			case "number":
				variables[name] = wfValue.Value
			case "date":
				if dateStr, ok := wfValue.Value.(string); ok {
					date, err := time.Parse(time.RFC3339, dateStr)
					if err != nil {
						return nil, fmt.Errorf("invalid date format for variable %s: %w", name, err)
					}
					variables[name] = date
				} else {
					return nil, fmt.Errorf("invalid date value for variable %s", name)
				}
			case "json":
				if jsonStr, ok := wfValue.Value.(string); ok {
					var jsonValue interface{}
					if err := json.Unmarshal([]byte(jsonStr), &jsonValue); err != nil {
						return nil, fmt.Errorf("invalid JSON format for variable %s: %w", name, err)
					}
					variables[name] = jsonValue
				} else {
					return nil, fmt.Errorf("invalid JSON string for variable %s", name)
				}
			default:
				// For string type or unknown types, treat as template string
				resolved, err := templates.ResolveTemplate(fmt.Sprint(wfValue.Value), data)
				if err != nil {
					return nil, fmt.Errorf("variable %s: %w", name, err)
				}
				variables[name] = resolved
			}
		} else {
			// Fallback to treating as template string
			resolved, err := templates.ResolveTemplate(value, data)
			if err != nil {
				return nil, fmt.Errorf("variable %s: %w", name, err)
			}
			variables[name] = resolved
		}
	}
	return variables, nil
}

// storeSensitiveValues creates or updates a secret containing sensitive values from workflow steps.
func (r *Reconciler) storeSensitiveValues(ctx context.Context, wf *workflows.Workflow, sensitiveValues map[string]map[string]map[string]string) error {
	// Find the owner WorkflowRun
	ownerRef := metav1.GetControllerOf(wf)
	if ownerRef == nil || ownerRef.Kind != "WorkflowRun" {
		return fmt.Errorf("workflow %s/%s has no WorkflowRun owner", wf.Namespace, wf.Name)
	}

	// Get the WorkflowRun
	workflowRun := &workflows.WorkflowRun{}
	if err := r.Get(ctx, types.NamespacedName{Name: ownerRef.Name, Namespace: wf.Namespace}, workflowRun); err != nil {
		return err
	}

	// Create or update the secret
	secretName := fmt.Sprintf("workflow-sensitive-values-%s-%s", workflowRun.Name, computeHash(sensitiveValues))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: wf.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":              "external-secrets-workflow-controller",
				"workflows.external-secrets.io/workflowrun": workflowRun.Name,
			},
		},
		Data: make(map[string][]byte),
	}

	// Set owner reference to the WorkflowRun
	if err := controllerutil.SetControllerReference(workflowRun, secret, r.Scheme); err != nil {
		return err
	}

	// Populate secret data
	for jobName, jobValues := range sensitiveValues {
		for stepName, stepValues := range jobValues {
			for key, value := range stepValues {
				secretKey := fmt.Sprintf("%s.%s.%s", jobName, stepName, key)
				secret.Data[secretKey] = []byte(value)
			}
		}
	}

	// Create or update the secret
	if err := r.Create(ctx, secret); err != nil {
		if errors.IsAlreadyExists(err) {
			// Secret already exists, update it
			existingSecret := &corev1.Secret{}
			if err := r.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, existingSecret); err != nil {
				return err
			}
			existingSecret.Data = secret.Data
			if err := r.Update(ctx, existingSecret); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Update WorkflowRun status to reference the secret
	if workflowRun.Status.SensitiveValuesSecrets == nil {
		workflowRun.Status.SensitiveValuesSecrets = []string{}
	}

	// Check if the secret is already referenced
	secretExists := false
	for _, s := range workflowRun.Status.SensitiveValuesSecrets {
		if s == secret.Name {
			secretExists = true
			break
		}
	}

	if !secretExists {
		workflowRun.Status.SensitiveValuesSecrets = append(workflowRun.Status.SensitiveValuesSecrets, secret.Name)
		if err := r.Status().Update(ctx, workflowRun); err != nil {
			return err
		}
	}

	return nil
}

// mergeSensitiveValues reads sensitive values from secrets and merges them with the workflow state.
func (r *Reconciler) mergeSensitiveValues(ctx context.Context, wf *workflows.Workflow) error {
	// Find the owner WorkflowRun
	ownerRef := metav1.GetControllerOf(wf)
	if ownerRef == nil || ownerRef.Kind != "WorkflowRun" {
		return nil // No WorkflowRun owner, nothing to merge
	}

	// Get the WorkflowRun
	workflowRun := &workflows.WorkflowRun{}
	if err := r.Get(ctx, types.NamespacedName{Name: ownerRef.Name, Namespace: wf.Namespace}, workflowRun); err != nil {
		return client.IgnoreNotFound(err)
	}

	// No secrets to read
	if len(workflowRun.Status.SensitiveValuesSecrets) == 0 {
		return nil
	}

	// Read sensitive values from secrets
	sensitiveValues := make(map[string]map[string]map[string]string)

	for _, secretName := range workflowRun.Status.SensitiveValuesSecrets {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: wf.Namespace}, secret); err != nil {
			if errors.IsNotFound(err) {
				continue // Skip if secret not found
			}
			return err
		}

		// Process secret data
		for key, value := range secret.Data {
			// Parse key format: {job-name}.{step-name}.{output-key}
			parts := strings.Split(key, ".")
			if len(parts) != 3 {
				continue // Invalid key format
			}

			jobName, stepName, outputKey := parts[0], parts[1], parts[2]

			// Initialize maps if needed
			if sensitiveValues[jobName] == nil {
				sensitiveValues[jobName] = make(map[string]map[string]string)
			}
			if sensitiveValues[jobName][stepName] == nil {
				sensitiveValues[jobName][stepName] = make(map[string]string)
			}

			// Store sensitive value
			sensitiveValues[jobName][stepName][outputKey] = string(value)
		}
	}

	// Merge sensitive values with workflow state
	for jobName, jobValues := range sensitiveValues {
		jobStatus, exists := wf.Status.JobStatuses[jobName]
		if !exists {
			continue
		}

		for stepName, stepValues := range jobValues {
			stepStatus, exists := jobStatus.StepStatuses[stepName]
			if !exists {
				continue
			}

			// Merge sensitive values with step outputs
			for key, value := range stepValues {
				if stepStatus.Outputs[key] == common.MaskValue {
					// Replace masked value with actual sensitive value
					stepStatus.Outputs[key] = value
				}
			}

			jobStatus.StepStatuses[stepName] = stepStatus
		}

		wf.Status.JobStatuses[jobName] = jobStatus
	}

	return nil
}

// markWorkflowCompleted marks the workflow as succeeded.
func (r *Reconciler) markWorkflowCompleted(ctx context.Context, wf *workflows.Workflow) (ctrl.Result, error) {
	wf.Status.Phase = workflows.PhaseSucceeded
	now := metav1.Now()
	wf.Status.CompletionTime = &now
	wf.Status.ExecutionTimeNanos = ptr.Int64(now.Time.Sub(wf.Status.StartTime.Time).Nanoseconds())
	meta.SetStatusCondition(&wf.Status.Conditions, metav1.Condition{
		Type:    "Completed",
		Status:  metav1.ConditionTrue,
		Reason:  "WorkflowCompleted",
		Message: "All jobs completed successfully",
	})
	return r.updateStatusWithEvent(ctx, wf,
		ctrl.Result{}, ctrl.Result{},
		"Normal", "WorkflowCompleted", fmt.Sprintf("Workflow %s completed", wf.Name))
}

// markJobFailed marks a job as failed and records an event.
func (r *Reconciler) markJobFailed(ctx context.Context, wf *workflows.Workflow, jobName string, err error) (ctrl.Result, error) {
	jobStatus := wf.Status.JobStatuses[jobName]
	jobStatus.Phase = workflows.JobPhaseFailed
	now := metav1.Now()
	jobStatus.CompletionTime = &now
	if jobStatus.StartTime != nil {
		jobStatus.ExecutionTimeNanos = ptr.Int64(now.Time.Sub(jobStatus.StartTime.Time).Nanoseconds())
	}
	wf.Status.JobStatuses[jobName] = jobStatus

	// Add these lines to set the workflow phase to Failed
	wf.Status.Phase = workflows.PhaseFailed
	wf.Status.CompletionTime = &now
	if wf.Status.StartTime != nil {
		wf.Status.ExecutionTimeNanos = ptr.Int64(now.Time.Sub(wf.Status.StartTime.Time).Nanoseconds())
	}

	meta.SetStatusCondition(&wf.Status.Conditions, metav1.Condition{
		Type:    "Failed",
		Status:  metav1.ConditionTrue,
		Reason:  "JobFailed",
		Message: fmt.Sprintf("Job %s failed: %v", jobName, err),
	})

	return r.updateStatusWithEvent(ctx, wf,
		ctrl.Result{}, ctrl.Result{},
		"Warning", "JobFailed", fmt.Sprintf("Job %s failed: %v", jobName, err))
}

// markWorkflowFailed marks the entire workflow as failed.
func (r *Reconciler) markWorkflowFailed(ctx context.Context, wf *workflows.Workflow, reason, message string) (ctrl.Result, error) {
	wf.Status.Phase = workflows.PhaseFailed
	now := metav1.Now()
	wf.Status.CompletionTime = &now
	if wf.Status.StartTime != nil {
		wf.Status.ExecutionTimeNanos = ptr.Int64(now.Time.Sub(wf.Status.StartTime.Time).Nanoseconds())
	}
	meta.SetStatusCondition(&wf.Status.Conditions, metav1.Condition{
		Type:    "Failed",
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})

	return r.updateStatusWithEvent(ctx, wf,
		ctrl.Result{}, ctrl.Result{},
		"Warning", "WorkflowFailed", fmt.Sprintf("Workflow failed: %s", message))
}

// updateStatusWithEvent updates the workflow status and records an event.
// It sanitizes sensitive step information before persisting and stores sensitive values in a secret.
func (r *Reconciler) updateStatusWithEvent(ctx context.Context, wf *workflows.Workflow, errorResult, successResult ctrl.Result, eventType, reason, message string) (ctrl.Result, error) {
	// Collect sensitive values
	sensitiveValues := make(map[string]map[string]map[string]string)

	// Sanitize JobStatuses by masking sensitive outputs
	for jobName, jobStatus := range wf.Status.JobStatuses {
		sensitiveValues[jobName] = make(map[string]map[string]string)

		for stepName, stepStatus := range jobStatus.StepStatuses {
			sensitiveValues[jobName][stepName] = make(map[string]string)

			// Get the step definition to check for explicitly marked sensitive outputs
			stepDef := findStepDefinition(wf, stepName)

			// Convert string map to interface map
			interfaceMap := make(map[string]interface{})
			for k, v := range stepStatus.Outputs {
				interfaceMap[k] = v
			}

			// Use modified ProcessOutputs to get both masked outputs and sensitive values
			if stepDef != nil {
				maskedOutputs, stepSensitiveValues, _ := common.ProcessOutputs(interfaceMap, *stepDef)
				stepStatus.Outputs = maskedOutputs

				// Store sensitive values
				for k, v := range stepSensitiveValues {
					sensitiveValues[jobName][stepName][k] = v
				}
			}
			jobStatus.StepStatuses[stepName] = stepStatus
		}
		wf.Status.JobStatuses[jobName] = jobStatus
	}

	// Store sensitive values in a secret if there are any
	hasSensitiveValues := false
	for _, jobValues := range sensitiveValues {
		for _, stepValues := range jobValues {
			if len(stepValues) > 0 {
				hasSensitiveValues = true
				break
			}
		}
		if hasSensitiveValues {
			break
		}
	}

	if hasSensitiveValues {
		if err := r.storeSensitiveValues(ctx, wf, sensitiveValues); err != nil {
			r.Log.Error(err, "failed to store sensitive values in secret")
			// Continue with the update even if storing sensitive values fails
		}
	}

	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		latest := &workflows.Workflow{}
		if e := r.Get(ctx, types.NamespacedName{Name: wf.Name, Namespace: wf.Namespace}, latest); e != nil {
			return e
		}
		latest.Status = wf.Status
		return r.Status().Update(ctx, latest)
	}); err != nil {
		return errorResult, err
	}
	r.Recorder.Eventf(wf, eventType, reason, message)
	return successResult, nil
}

// findStepDefinition finds the step definition for a given step name in the workflow.
func findStepDefinition(wf *workflows.Workflow, stepName string) *workflows.Step {
	for _, job := range wf.Spec.Jobs {
		if standardJob := job.Standard; standardJob != nil {
			for i := range standardJob.Steps {
				if standardJob.Steps[i].Name == stepName {
					return &standardJob.Steps[i]
				}
			}
		}
		if loopJob := job.Loop; loopJob != nil {
			for i := range loopJob.Steps {
				if loopJob.Steps[i].Name == stepName {
					return &loopJob.Steps[i]
				}
			}
		}
		if switchJob := job.Switch; switchJob != nil {
			for _, switchCase := range switchJob.Cases {
				for i := range switchCase.Steps {
					if switchCase.Steps[i].Name == stepName {
						return &switchCase.Steps[i]
					}
				}
			}
		}
	}
	return nil
}

// computeHash generates a hash for a map of sensitive values.
func computeHash(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		// If marshaling fails, use a timestamp-based hash
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("workflow")

	// Initialize the SecretStore Manager
	// We create the Manager with an empty control class and floodgate disabled
	// These parameters can be exposed as controller options if needed
	r.Manager = secretstore.NewManager(mgr.GetClient(), "", false)

	return ctrl.NewControllerManagedBy(mgr).
		For(&workflows.Workflow{}).
		Complete(r)
}
