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
	"maps"
	"slices"
	"strconv"
	"sync"
	"time"

	cronV3 "github.com/robfig/cron/v3"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	ctrlutil "github.com/external-secrets/external-secrets/pkg/controllers/util"
)

var mu = sync.Mutex{}

// RunTemplateReconciler reconciles a WorkflowRunTemplate object.
type RunTemplateReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// Reconcile handles WorkflowRun resources.
func (r *RunTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("workflowruntemplate", req.NamespacedName)
	log.Info("reconciling WorkflowRunTemplate")

	// Fetch the WorkflowRun instance
	run := &workflows.WorkflowRunTemplate{}
	if err := r.Get(ctx, req.NamespacedName, run); err != nil {
		// We'll ignore not-found errors, since they can't be fixed by an immediate requeue
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// While we are getting Children to calculate revision, we should not allow for any other ops.
	// if we should not reconcile, just skip and leave it as is
	if !r.shouldReconcile(run) {
		// If workflowRunTemplate already created, check it's children
		workflowRuns, err := r.getChildrenFor(ctx, run)
		if err != nil {
			return ctrl.Result{}, err
		}
		// Updating Run Status otherwise this is a bit pointless lol :)
		if r.needsStatusUpdate(run, workflowRuns) {
			defer func() {
				runs, err := r.getChildrenFor(ctx, run)
				if err != nil {
					r.Log.Error(err, "Failed to Get Children for WorkflowRunTemplate")
				}
				if err := r.updateWorkflowRunTemplate(ctx, run, runs, run.Status.LastRunTime.Time); err != nil {
					r.Log.Error(err, "Failed to update WorkflowRunTemplate", "namespace", run.Namespace, "name", run.Name)
				}
			}()
		}
		return r.requeueAfter(run)
	}

	// From this moment on we need to sync multiple controllers
	// as the deletion of Runs and updates of RunTemplates will
	// make the `generateRevision` outputs to change
	mu.Lock()
	defer mu.Unlock()

	// If workflowRunTemplate already created, check it's children
	workflowRuns, err := r.getChildrenFor(ctx, run)
	if err != nil {
		return ctrl.Result{}, err
	}
	revision, err := r.generateRevision(workflowRuns)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Check if this reconcile is a fake one (due to this being a change on the owned WorkflowRun being processed)
	for _, workflowRun := range workflowRuns {
		if workflowRun.Status.WorkflowRef == nil {
			// False alarm
			return r.requeueAfter(run)
		}
		if workflowRun.Status.Conditions == nil {
			// False alarm
			return r.requeueAfter(run)
		}
	}

	// Revision History Limit Logic
	if len(workflowRuns) >= run.Spec.RevisionHistoryLimit {
		workflowRuns, err = r.cleanup(ctx, workflowRuns, run.Spec.RevisionHistoryLimit)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	// Create new WorkflowRun
	newRun, err := r.createWorkflowRun(ctx, run, revision)
	if err != nil {
		return ctrl.Result{}, err
	}
	workflowRuns = append(workflowRuns, *newRun)
	defer func() {
		if err := r.updateWorkflowRunTemplate(ctx, run, workflowRuns, time.Now()); err != nil {
			r.Log.Error(err, "Failed to update WorkflowRunTemplate", "namespace", run.Namespace, "name", run.Name)
		}
	}()
	// Get
	return r.requeueAfter(run)
}

func (r *RunTemplateReconciler) needsStatusUpdate(run *workflows.WorkflowRunTemplate, workflowRuns []workflows.WorkflowRun) bool {
	newStatus := []workflows.WorkflowRunStatus{}
	for _, run := range workflowRuns {
		newStatus = append(newStatus, run.Status)
	}
	return !slices.EqualFunc(newStatus, run.Status.RunStatuses, func(a workflows.WorkflowRunStatus, b workflows.NamedWorkflowRunStatus) bool {
		return slices.Equal(a.Conditions, b.WorkflowRunStatus.Conditions)
	})
}

func (r *RunTemplateReconciler) getChildrenFor(ctx context.Context, run *workflows.WorkflowRunTemplate) ([]workflows.WorkflowRun, error) {
	workflowRunList := &workflows.WorkflowRunList{}
	if err := r.List(ctx, workflowRunList, client.InNamespace(run.Namespace), client.MatchingLabels{"workflowruntemplate.external-secrets.io/owner": run.Name}); err != nil {
		return nil, err
	}
	return workflowRunList.Items, nil
}

func (r *RunTemplateReconciler) cleanup(ctx context.Context, workflowRuns []workflows.WorkflowRun, limit int) ([]workflows.WorkflowRun, error) {
	if len(workflowRuns) < limit {
		return workflowRuns, nil
	}
	totalToDelete := len(workflowRuns) - limit + 1
	remain, toDelete, err := sortWorkflowsByRevision(workflowRuns, totalToDelete)
	if err != nil {
		return nil, err
	}
	for _, d := range toDelete {
		err := r.Delete(ctx, &d)
		if err != nil {
			return nil, err
		}
	}
	return remain, nil
}

func sortWorkflowsByRevision(workflowRuns []workflows.WorkflowRun, totalToDelete int) ([]workflows.WorkflowRun, []workflows.WorkflowRun, error) {
	remaining := []workflows.WorkflowRun{}
	del := []workflows.WorkflowRun{}
	helperMap := map[int]workflows.WorkflowRun{}
	for _, run := range workflowRuns {
		revString, ok := run.Annotations["workflowruntemplate.external-secrets.io/revision"]
		if !ok {
			return nil, nil, fmt.Errorf("workflow run %s does not have a revision", run.Name)
		}
		revision, err := strconv.Atoi(revString)
		if err != nil {
			return nil, nil, fmt.Errorf("workflow run %s has an invalid revision %s: %w", run.Name, revString, err)
		}
		helperMap[revision] = run
	}
	deleted := 0
	for deleted < totalToDelete {
		mapKeys := slices.Collect(maps.Keys(helperMap))
		keyToDelete := slices.Min(mapKeys)
		del = append(del, helperMap[keyToDelete])
		delete(helperMap, keyToDelete)
		deleted++
	}
	for k := range helperMap {
		remaining = append(remaining, helperMap[k])
	}
	return remaining, del, nil
}

func (r *RunTemplateReconciler) generateRevision(workflowRuns []workflows.WorkflowRun) (string, error) {
	currentRevision := 0
	for _, run := range workflowRuns {
		revString, ok := run.Annotations["workflowruntemplate.external-secrets.io/revision"]
		if !ok {
			return "", fmt.Errorf("workflow run %s does not have a revision", run.Name)
		}
		workflowRevision, err := strconv.Atoi(revString)
		if err != nil {
			return "", fmt.Errorf("workflow run %s has an invalid revision %s: %w", run.Name, revString, err)
		}
		if workflowRevision > currentRevision {
			currentRevision = workflowRevision
		}
	}
	return strconv.Itoa(currentRevision + 1), nil
}

func (r *RunTemplateReconciler) createWorkflowRun(ctx context.Context, run *workflows.WorkflowRunTemplate, revision string) (*workflows.WorkflowRun, error) {
	workflowRun := &workflows.WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: run.Name + "-",
			Namespace:    run.Namespace,
			Annotations: map[string]string{
				"workflowruntemplate.external-secrets.io/revision": revision,
			},
			Labels: map[string]string{
				"workflowruntemplate.external-secrets.io/owner": run.Name,
			},
		},
		Spec: run.Spec.RunSpec,
	}
	err := ctrl.SetControllerReference(run, workflowRun, r.Scheme)
	if err != nil {
		return nil, fmt.Errorf("could not set controller reference: %w", err)
	}
	if err := r.Create(ctx, workflowRun); err != nil {
		return nil, fmt.Errorf("could not create workflow run: %w", err)
	}
	return workflowRun, nil
}

func (r *RunTemplateReconciler) updateWorkflowRunTemplate(ctx context.Context, run *workflows.WorkflowRunTemplate, workflowRuns []workflows.WorkflowRun, runTime time.Time) error {
	stats := []workflows.NamedWorkflowRunStatus{}
	for _, run := range workflowRuns {
		namedStat := workflows.NamedWorkflowRunStatus{
			RunName:           run.Name,
			WorkflowRunStatus: run.Status,
		}
		stats = append(stats, namedStat)
	}
	run.Status.RunStatuses = stats
	run.Status.SyncedResourceVersion = ctrlutil.GetResourceVersion(run.ObjectMeta)
	run.Status.LastRunTime = &metav1.Time{Time: runTime}
	return r.Status().Update(ctx, run)
}

func (r *RunTemplateReconciler) requeueAfter(run *workflows.WorkflowRunTemplate) (ctrl.Result, error) {
	if run.Spec.RunPolicy.Once != nil || run.Spec.RunPolicy.OnChange != nil {
		// Never Requeue these
		return ctrl.Result{}, nil
	}
	if run.Spec.RunPolicy.Scheduled != nil {
		lastRun := run.Status.LastRunTime
		if lastRun == nil {
			// First run of this workflow, we can safely do this
			// as we will get a requeue anyways for the update
			return ctrl.Result{}, nil
		}
		if run.Spec.RunPolicy.Scheduled.Every != nil {
			if time.Now().After(lastRun.Time.Add(run.Spec.RunPolicy.Scheduled.Every.Duration)) {
				// Requeue Immediately
				r.Log.Info("Requeuing WorkflowRunTemplate immediately", "lastRun", lastRun.Time.String(), "now", time.Now().String())
				return ctrl.Result{Requeue: true}, nil
			}
			remaining := lastRun.Time.Add(run.Spec.RunPolicy.Scheduled.Every.Duration).Sub(time.Now())
			if remaining < 0 {
				r.Log.Info("Requeuing Immediately", "nextRun", lastRun.Time.Add(run.Spec.RunPolicy.Scheduled.Every.Duration).String(), "now", time.Now().String())
				return ctrl.Result{Requeue: true}, nil
			}
			r.Log.Info("Requeuing After next period", "nextRun", lastRun.Time.Add(run.Spec.RunPolicy.Scheduled.Every.Duration).String(), "now", time.Now().String())
			return ctrl.Result{RequeueAfter: remaining}, nil
		}
		if run.Spec.RunPolicy.Scheduled.Cron != nil && *run.Spec.RunPolicy.Scheduled.Cron != "" {
			cronExpression := *run.Spec.RunPolicy.Scheduled.Cron
			schedule, err := cronV3.ParseStandard(cronExpression)
			if err != nil {
				r.Log.Error(err, "Failed to parse cron expression, requeuing after 1 minute for WorkflowRunTemplate", "cronExpression", cronExpression, "namespace", run.Namespace, "name", run.Name)
				return ctrl.Result{}, err
			}

			now := time.Now()
			nextRunTime := schedule.Next(now)
			requeueAfter := nextRunTime.Sub(now)

			if requeueAfter <= 0 {
				r.Log.Info("Cron schedule is due now or has passed, requeuing immediately for WorkflowRunTemplate", "cronExpression", cronExpression, "calculatedNextRun", nextRunTime.Format(time.RFC3339), "currentTime", now.Format(time.RFC3339), "namespace", run.Namespace, "name", run.Name)
				return ctrl.Result{Requeue: true}, nil // Requeue immediately
			}

			r.Log.Info("Requeuing WorkflowRunTemplate based on cron schedule", "cronExpression", cronExpression, "nextRun", nextRunTime.Format(time.RFC3339), "requeueAfter", requeueAfter.String(), "namespace", run.Namespace, "name", run.Name)
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}
	}
	// In theory we should never reach this
	return ctrl.Result{}, nil
}

func (r *RunTemplateReconciler) shouldReconcile(run *workflows.WorkflowRunTemplate) bool {
	if run.Spec.RunPolicy.Once != nil {
		return run.Status.SyncedResourceVersion == ""
	}
	if run.Spec.RunPolicy.OnChange != nil {
		return run.Status.SyncedResourceVersion == "" || run.Status.SyncedResourceVersion != ctrlutil.GetResourceVersion(run.ObjectMeta)
	}
	if run.Spec.RunPolicy.Scheduled != nil {
		// Change on object should force a reconcile here Regardless of time interval.
		if run.Status.SyncedResourceVersion == "" || run.Status.SyncedResourceVersion != ctrlutil.GetResourceVersion(run.ObjectMeta) {
			return true
		}
		if run.Status.LastRunTime == nil {
			// No info on run time - should reconcile
			return true
		}
		if run.Spec.RunPolicy.Scheduled.Every != nil {
			return time.Now().After(run.Status.LastRunTime.Time.Add(run.Spec.RunPolicy.Scheduled.Every.Duration))
		}
		if run.Spec.RunPolicy.Scheduled.Cron != nil {
			cronExpression := *run.Spec.RunPolicy.Scheduled.Cron
			schedule, err := cronV3.ParseStandard(cronExpression)
			if err != nil {
				r.Log.Error(err, "Failed to parse cron expression, requeuing after 1 minute for WorkflowRunTemplate", "cronExpression", cronExpression, "namespace", run.Namespace, "name", run.Name)
				return false
			}
			nextRun := schedule.Next(run.Status.LastRunTime.Time)
			return nextRun.Before(time.Now())
		}
	}
	// Should never reach this
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workflows.WorkflowRunTemplate{}).
		Owns(&workflows.WorkflowRun{}).
		Complete(r)
}
