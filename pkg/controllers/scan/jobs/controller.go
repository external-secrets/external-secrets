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

// Copyright External Secrets Inc. 2025
// All Rights Reserved

// Package jobs implements the Job controller.
package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"sort"
	"sync"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	targetv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	utils "github.com/external-secrets/external-secrets/pkg/scan/jobs"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// JobController reconciles Job resources.
type JobController struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	running sync.Map
}

// Reconcile reconciles a Job resource.
func (c *JobController) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	jobSpec := &v1alpha1.Job{}
	if err := c.Get(ctx, req.NamespacedName, jobSpec); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if jobSpec.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}
	// Check if we should already run this job
	// If there is no status, it means this is the first run and should be ran anyways
	if jobSpec.Status.RunStatus != "" && jobSpec.Status.RunStatus != v1alpha1.JobRunStatusRunning {
		// Ignore new Runs
		if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyOnce {
			return ctrl.Result{}, nil
		}
		// TODO: add correct On Change condition
		if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyOnChange {
			return ctrl.Result{}, nil
		}

		if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyPull {
			// Check if a dependency has changed by comparing digests
			stores := &esv1.SecretStoreList{}
			if err := c.Client.List(ctx, stores, client.InNamespace(jobSpec.Namespace)); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to list secret stores for digest calculation: %w", err)
			}
			currentSecretStoresDigest := calculateSecretStoresDigest(stores.Items)

			targets, err := collectTargets(ctx, c.Client, jobSpec.Namespace)
			if err != nil {
				return ctrl.Result{}, err
			}
			currentTargetsDigest := calculateTargetsDigest(targets)

			// If digests are different, a SecretStore has changed, so run immediately.
			if currentSecretStoresDigest != jobSpec.Status.ObservedSecretStoresDigest {
				c.Log.V(1).Info("secretstore digest changed, running job immediately", "job", jobSpec.GetName())
			} else if currentTargetsDigest != jobSpec.Status.ObservedTargetsDigest {
				c.Log.V(1).Info("target digest changed, running job immediately", "job", jobSpec.GetName())
			} else {
				// Otherwise, respect the polling interval
				timeToReconcile := time.Since(jobSpec.Status.LastRunTime.Time)
				if timeToReconcile < jobSpec.Spec.Interval.Duration {
					return ctrl.Result{RequeueAfter: jobSpec.Spec.Interval.Duration - timeToReconcile}, nil
				}
			}
		}
	}

	timeout := jobSpec.Spec.JobTimeout.Duration

	if timeout > 0 && jobSpec.Status.RunStatus == v1alpha1.JobRunStatusRunning {
		runningTime := time.Since(jobSpec.Status.LastRunTime.Time)
		if runningTime > timeout {
			c.stopJob(req)

			jobSpec.Status.RunStatus = v1alpha1.JobRunStatusFailed
			condition := metav1.Condition{
				Type:               string(v1alpha1.JobRunStatusFailed),
				Status:             metav1.ConditionFalse,
				Reason:             "TimedOut",
				Message:            fmt.Sprintf("timed out after %s", timeout),
				LastTransitionTime: metav1.Now(),
			}
			jobSpec.Status.Conditions = append(jobSpec.Status.Conditions, condition)
			jobSpec.Status.LastRunTime = metav1.Now()

			if err := c.Status().Update(ctx, jobSpec); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: time.Second}, nil
		}

		// still running, requeue exactly when timeout would occur
		remaining := timeout - runningTime
		if remaining < time.Second {
			remaining = time.Second
		}
		return ctrl.Result{RequeueAfter: remaining}, nil
	}

	// Synchronize
	j := utils.NewRunner(c.Client, c.Log, jobSpec.Namespace, jobSpec.Spec.Constraints)

	jobSpec.Status = v1alpha1.JobStatus{
		LastRunTime: metav1.Now(),
		RunStatus:   v1alpha1.JobRunStatusRunning,
	}
	if err := c.Status().Update(ctx, jobSpec); err != nil {
		return ctrl.Result{}, err
	}

	// Start async job with cancel support
	runCtx, cancel := context.WithCancel(context.Background())
	c.running.Store(keyFor(req), cancel)

	// Run the Job applying constraints after leaving the reconcile loop
	defer func() {
		go func() {
			c.Log.V(1).Info("Starting async job", "job", jobSpec.GetName())
			defer c.running.Delete(keyFor(req))
			defer func() {
				_ = j.Close(context.Background())
			}()

			err := c.runJob(runCtx, jobSpec, j)
			if err != nil {
				c.Log.Error(err, "failed to run job")
			}
		}()
	}()

	if jobSpec.Spec.RunPolicy != v1alpha1.JobRunPolicyPull {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: jobSpec.Spec.Interval.Duration}, nil
}

func findingNeedsToUpdate(existing, finding *v1alpha1.Finding) bool {
	if existing == nil {
		return true
	}
	if finding == nil {
		return true
	}
	loc1 := existing.Status.Locations
	loc2 := finding.Status.Locations

	return !(slices.EqualFunc(loc1, loc2, utils.EqualLocations) && finding.Spec.Hash == existing.Spec.Hash)
}

func consumerNeedsToUpdate(existing, consumer *v1alpha1.Consumer) bool {
	if existing == nil {
		return true
	}
	if consumer == nil {
		return true
	}

	equalLocations := slices.EqualFunc(existing.Status.Locations, consumer.Status.Locations, utils.EqualLocations)
	equalObservedIndex := maps.EqualFunc(existing.Status.ObservedIndex, consumer.Status.ObservedIndex, utils.EqualSecretUpdateRecord)

	return !(equalLocations && equalObservedIndex)
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (c *JobController) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	controller := ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&v1alpha1.Job{}).
		Watches(
			&esv1.SecretStore{},
			handler.EnqueueRequestsFromMapFunc(c.mapSecretStoreToJobs),
		)

	targets := targetv1alpha1.GetAllTargets()
	for _, target := range targets {
		controller = controller.Watches(
			target,
			handler.EnqueueRequestsFromMapFunc(c.mapTargetToJobs),
		)
	}

	return controller.Complete(c)
}

func (c *JobController) mapSecretStoreToJobs(ctx context.Context, obj client.Object) []reconcile.Request {
	c.Log.V(1).Info("reconciling all jobs due to SecretStore change", "secretstore", obj.GetName())

	jobList := &v1alpha1.JobList{}
	if err := c.List(ctx, jobList, client.InNamespace(obj.GetNamespace())); err != nil {
		c.Log.Error(err, "failed to list jobs for secretstore change")
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(jobList.Items))
	for i, job := range jobList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			},
		}
	}
	return requests
}

func (c *JobController) mapTargetToJobs(ctx context.Context, obj client.Object) []reconcile.Request {
	// TODO - as soon as a PushSecret is done, this is updaated, causing the job reconcile to trigger way before what we want.
	// We actually want it to trigger only once after all reconciles are done (or sort of),
	c.Log.V(1).Info("reconciling all jobs due to Target change", "target", obj.GetName())

	jobList := &v1alpha1.JobList{}
	if err := c.List(ctx, jobList, client.InNamespace(obj.GetNamespace())); err != nil {
		c.Log.Error(err, "failed to list jobs for target change")
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(jobList.Items))
	for i, job := range jobList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			},
		}
	}
	return requests
}

// calculateSecretStoresDigest computes a sha256 digest from the resourceVersions of the provided SecretStores.
func calculateSecretStoresDigest(stores []esv1.SecretStore) string {
	if len(stores) == 0 {
		return ""
	}
	// Sort by name to ensure consistent digest
	sort.Slice(stores, func(i, j int) bool {
		return stores[i].Name < stores[j].Name
	})
	hash := sha256.New()
	for _, store := range stores {
		hash.Write([]byte(store.ResourceVersion))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// collectTargets lists all Target CRDs in a namespace and returns them as GenericTarget.
func collectTargets(ctx context.Context, c client.Client, ns string) ([]targetv1alpha1.GenericTarget, error) {
	var (
		out       []targetv1alpha1.GenericTarget
		mu        sync.Mutex
		eg, egCtx = errgroup.WithContext(ctx)
	)

	add := func(f func() error) { eg.Go(func() error { return f() }) }

	add(func() error {
		l := &targetv1alpha1.GithubRepositoryList{}
		if err := c.List(egCtx, l, client.InNamespace(ns)); err != nil {
			return fmt.Errorf("list github targets: %w", err)
		}
		mu.Lock()
		for i := range l.Items {
			out = append(out, &l.Items[i]) // IMPORTANT: take address by index
		}
		mu.Unlock()
		return nil
	})

	add(func() error {
		l := &targetv1alpha1.KubernetesClusterList{}
		if err := c.List(egCtx, l, client.InNamespace(ns)); err != nil {
			return fmt.Errorf("list kubernetes targets: %w", err)
		}
		mu.Lock()
		for i := range l.Items {
			out = append(out, &l.Items[i])
		}
		mu.Unlock()
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// calculateTargetsDigest computes a sha256 digest from the resourceVersions of the provided Targets.
func calculateTargetsDigest(targets []targetv1alpha1.GenericTarget) string {
	if len(targets) == 0 {
		return ""
	}
	// Sort by name to ensure consistent digest
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].GetName() < targets[j].GetName()
	})
	hash := sha256.New()
	for _, target := range targets {
		hash.Write([]byte(target.GetResourceVersion()))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (c *JobController) runJob(ctx context.Context, jobSpec *v1alpha1.Job, j *utils.Runner) error {
	defer func() {
		err := j.Close(context.Background())
		if err != nil {
			c.Log.Error(err, "failed to close job runner")
		}
	}()
	var jobTime metav1.Time
	var jobStatus v1alpha1.JobRunStatus
	var observedSecretStoresDigest string
	var observedTargetsDigest string
	defer func() {
		jobSpec.Status = v1alpha1.JobStatus{
			LastRunTime:                jobTime,
			RunStatus:                  jobStatus,
			ObservedSecretStoresDigest: observedSecretStoresDigest,
			ObservedTargetsDigest:      observedTargetsDigest,
		}
		c.Log.V(1).Info("Updating Job Status", "RunStatus", jobStatus)
		if err := c.Status().Update(ctx, jobSpec); err != nil {
			c.Log.Error(err, "failed to update job status")
		}
	}()
	c.Log.V(1).Info("Running Job", "job", jobSpec.GetName())
	findings, consumers, usedStores, usedTargets, err := j.Run(ctx)
	if err != nil {
		jobStatus = v1alpha1.JobRunStatusFailed
		jobTime = metav1.Now()
		return err
	}

	jobStatus, jobTime, err = c.UpdateFindings(ctx, findings, jobSpec.Namespace)
	if err != nil {
		return err
	}

	jobStatus, jobTime, err = c.UpdateConsumers(ctx, consumers, jobSpec.Namespace)
	if err != nil {
		return err
	}

	jobStatus = v1alpha1.JobRunStatusSucceeded
	jobTime = metav1.Now()
	observedSecretStoresDigest = calculateSecretStoresDigest(usedStores)
	observedTargetsDigest = calculateTargetsDigest(usedTargets)
	return nil
}

// UpdateFindings updates the findings for a job.
func (c *JobController) UpdateFindings(ctx context.Context, findings []v1alpha1.Finding, namespace string) (v1alpha1.JobRunStatus, metav1.Time, error) {
	c.Log.V(1).Info("Found findings for job", "total findings", len(findings))
	// for each finding, see if it already exists and update it if it does;
	currentFindings := &v1alpha1.FindingList{}
	c.Log.V(1).Info("Listing Current findings")
	if err := c.List(ctx, currentFindings, client.InNamespace(namespace)); err != nil {
		return v1alpha1.JobRunStatusFailed, metav1.Now(), err
	}
	c.Log.V(1).Info("Found Current findings", "total findings", len(currentFindings.Items))

	currentFindingsByID := map[string]*v1alpha1.Finding{}
	for i := range currentFindings.Items {
		f := &currentFindings.Items[i]
		id := f.Spec.ID
		if id == "" {
			continue
		} // legacy; can be handled separately
		currentFindingsByID[id] = f
	}

	newFindingsByHash := map[string]*v1alpha1.Finding{}
	for i := range findings {
		f := &findings[i]
		newFindingsByHash[f.Spec.Hash] = f
	}
	// 2 out of 3 should be a match. This gives a jaccard index of 2/3
	// Thus we set the mininum to be a little bit lower than that.
	params := utils.JaccardParams{MinJaccard: 0.6, MinIntersection: 2}
	assigned := utils.AssignIDs(currentFindings.Items, findings, params)
	seenIDs := make(map[string]struct{}, len(assigned))

	for i, assignedFinding := range assigned {
		newFinding := newFindingsByHash[findings[i].Spec.Hash]
		newFinding.Spec.ID = assignedFinding.Spec.ID
		seenIDs[assignedFinding.Spec.ID] = struct{}{}

		if currentFinding, ok := currentFindingsByID[assignedFinding.Spec.ID]; ok {
			if !findingNeedsToUpdate(currentFinding, newFinding) {
				continue
			}
			// Update Finding
			currentFinding.Status.Locations = newFinding.Status.Locations
			c.Log.V(1).Info("Updating finding", "finding", currentFinding.Spec.ID)
			if err := c.Status().Update(ctx, currentFinding); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}

			currentFinding.Spec.Hash = newFinding.Spec.Hash
			if err := c.Update(ctx, currentFinding); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
		} else {
			// create new CR with stable name
			create := newFinding.DeepCopy()
			create.SetNamespace(namespace)
			c.Log.V(1).Info("Creating finding", "finding", create.GetName())
			if err := c.Create(ctx, create); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
			create.Status.Locations = newFinding.Status.Locations
			c.Log.V(1).Info("Updating finding status", "finding", create.GetName())
			if err := c.Status().Update(ctx, create); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
		}
	}

	// Delete Findings that are no longer found
	for id, currentFinding := range currentFindingsByID {
		if _, ok := seenIDs[id]; !ok {
			c.Log.V(1).Info("Deleting stale finding (not observed this run)", "id", id, "name", currentFinding.GetName())
			// TODO - Fix this nasty concurrency bug
			// if err := c.Delete(ctx, currentFinding); err != nil {
			// 	return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			// }
		}
	}

	return v1alpha1.JobRunStatusRunning, metav1.Now(), nil
}

// UpdateConsumers updates the consumers for a job.
func (c *JobController) UpdateConsumers(
	ctx context.Context,
	consumers []v1alpha1.Consumer,
	namespace string,
) (v1alpha1.JobRunStatus, metav1.Time, error) {
	c.Log.V(1).Info("Found consumers for job", "total", len(consumers))

	seenIDs := make(map[string]struct{}, len(consumers))
	for i := range consumers {
		if consumers[i].Name == "" {
			consumers[i].Name = consumers[i].Spec.ID
		}
		consumers[i].Namespace = namespace
		seenIDs[consumers[i].Spec.ID] = struct{}{}
	}

	// Upsert each desired Consumer with retry-on-conflict.
	for i := range consumers {
		newCons := consumers[i]
		namespacedName := types.NamespacedName{Namespace: newCons.Namespace, Name: newCons.Name}

		if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			var cur v1alpha1.Consumer
			err := c.Get(ctx, namespacedName, &cur)
			if apierrors.IsNotFound(err) {
				cur = v1alpha1.Consumer{
					ObjectMeta: metav1.ObjectMeta{
						Name:        newCons.Name,
						Namespace:   newCons.Namespace,
						Labels:      newCons.Labels,
						Annotations: newCons.Annotations,
					},
					Spec: newCons.Spec,
				}
				c.Log.V(1).Info("Creating consumer", "name", cur.Name)
				if err := c.Create(ctx, &cur); err != nil {
					if !apierrors.IsAlreadyExists(err) {
						return err
					}
				}
				err := retry.OnError(retry.DefaultBackoff, apierrors.IsNotFound, func() error {
					return c.Get(ctx, namespacedName, &cur)
				})
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			if !consumerNeedsToUpdate(&cur, &newCons) {
				return nil
			}
			base := cur.DeepCopy()
			cur.Status = newCons.Status
			c.Log.V(1).Info("Patching consumer status", "name", cur.Name)

			return c.Status().Patch(ctx, &cur, ctrlclient.MergeFrom(base))
		}); err != nil {
			return v1alpha1.JobRunStatusFailed, metav1.Now(), fmt.Errorf("upsert consumer %s/%s: %w", namespace, newCons.Name, err)
		}
	}

	// Delete Consumers that were NOT seen this run.
	var current v1alpha1.ConsumerList
	c.Log.V(1).Info("Listing current consumers before deletion")
	if err := c.List(ctx, &current, ctrlclient.InNamespace(namespace)); err != nil {
		return v1alpha1.JobRunStatusFailed, metav1.Now(), err
	}

	for i := range current.Items {
		cur := current.Items[i]
		if _, ok := seenIDs[cur.Spec.ID]; ok {
			continue
		}
		c.Log.V(1).Info("Deleting stale consumer (not observed this run)", "id", cur.Spec.ID, "name", cur.Name)
		if err := c.Delete(ctx, &cur); err != nil && !apierrors.IsNotFound(err) {
			return v1alpha1.JobRunStatusFailed, metav1.Now(), err
		}
	}

	return v1alpha1.JobRunStatusRunning, metav1.Now(), nil
}

func (c *JobController) stopJob(req ctrl.Request) {
	key := keyFor(req)
	if v, ok := c.running.LoadAndDelete(key); ok {
		v.(context.CancelFunc)()
	}
}

func keyFor(req ctrl.Request) string { return req.NamespacedName.String() }
