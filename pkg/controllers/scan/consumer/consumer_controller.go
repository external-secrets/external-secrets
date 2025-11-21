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

// Package consumer implements the Consumer controller.
package consumer

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	targetv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	"github.com/go-logr/logr"
)

// Controller reconciles Consumer resources.
type Controller struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Health represents the health status of a consumer.
type Health struct {
	Healthy bool
	Reason  scanv1alpha1.ConsumerConditionType
	Message string
}

// Reconcile reconciles a Consumer resource.
func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	consumer := &scanv1alpha1.Consumer{}
	if err := c.Get(ctx, req.NamespacedName, consumer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	gvk := schema.GroupVersionKind{Group: targetv1alpha1.Group, Version: targetv1alpha1.Version, Kind: consumer.Spec.Type}
	obj, err := c.Scheme.New(gvk)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create object %v: %w", gvk, err)
	}
	genericTarget, ok := obj.(targetv1alpha1.GenericTarget)
	if !ok {
		return ctrl.Result{}, fmt.Errorf("invalid object: %T", obj)
	}

	key := types.NamespacedName{
		Namespace: consumer.Spec.Target.Namespace,
		Name:      consumer.Spec.Target.Name,
	}
	if err := c.Get(ctx, key, genericTarget); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	status := genericTarget.GetTargetStatus()

	return c.CheckConsumerStatus(ctx, consumer, status.PushIndex)
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (c *Controller) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&scanv1alpha1.Consumer{}).
		Complete(c)
}

// CheckConsumerStatus checks the status of a consumer.
func (c *Controller) CheckConsumerStatus(ctx context.Context, consumer *scanv1alpha1.Consumer, pushSecretIndex map[string][]scanv1alpha1.SecretUpdateRecord) (ctrl.Result, error) {
	consumerStatusCondition := metav1.ConditionTrue
	consumerStatusReason := scanv1alpha1.ConsumerLocationsUpToDate
	consumerStatusMessage := "All observed locations are up to date"

	locationsOutOfDate := make([]string, 0)
	locationsOutOfDateMessages := make([]string, 0)

	for observedIndexKey, observedIndex := range consumer.Status.ObservedIndex {
		secretUpdateRecords, ok := pushSecretIndex[observedIndexKey]
		if !ok || len(secretUpdateRecords) == 0 {
			continue
		}

		latestRecord := secretUpdateRecords[len(secretUpdateRecords)-1]
		if latestRecord.SecretHash != observedIndex.SecretHash {
			locationsOutOfDate = append(locationsOutOfDate, observedIndexKey)
			locationsOutOfDateMessages = append(locationsOutOfDateMessages, fmt.Sprintf("Location %s last updated at %v. Current version updated at %v", observedIndexKey, observedIndex.Timestamp.Time, latestRecord.Timestamp.Time))
			continue
		}

		if observedIndex.Timestamp.Before(&latestRecord.Timestamp) {
			consumerStatusCondition = metav1.ConditionFalse
			consumerStatusReason = scanv1alpha1.ConsumerNotReady
			consumerStatusMessage = fmt.Sprintf("Consumer not ready. Last update at: %v", observedIndex.Timestamp.Time)
			break
		}
	}

	if len(locationsOutOfDate) > 0 {
		consumerStatusCondition = metav1.ConditionFalse
		consumerStatusReason = scanv1alpha1.ConsumerLocationsOutOfDate
		consumerStatusMessage = fmt.Sprint("Observed locations out of date: ", strings.Join(locationsOutOfDateMessages, "; "))
	}

	if consumer.Spec.Attributes.K8sWorkload != nil {
		health, err := CheckWorkloadHealth(ctx, c.Client, consumer.Spec.Attributes.K8sWorkload)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error checking workload health: %w", err)
		}

		if !health.Healthy {
			consumerStatusCondition = metav1.ConditionFalse
			consumerStatusReason = health.Reason
			consumerStatusMessage = health.Message
		}
	}

	changed := meta.SetStatusCondition(&consumer.Status.Conditions, metav1.Condition{
		Type:    string(scanv1alpha1.ConsumerLatestVersion),
		Status:  consumerStatusCondition,
		Reason:  string(consumerStatusReason),
		Message: consumerStatusMessage,
	})
	if changed {
		err := c.Status().Update(ctx, consumer)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update consumer status: %w", err)
		}
	}

	if consumerStatusCondition == metav1.ConditionFalse {
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// CheckWorkloadHealth checks the health of a Kubernetes workload.
func CheckWorkloadHealth(ctx context.Context, client client.Client, wl *scanv1alpha1.K8sWorkloadSpec) (Health, error) {
	switch wl.WorkloadKind {
	case "Deployment":
		var obj appsv1.Deployment
		if err := client.Get(ctx, namespacedName(wl), &obj); err != nil {
			return Health{}, err
		}
		return deploymentHealth(&obj), nil
	case "ReplicaSet":
		var obj appsv1.ReplicaSet
		if err := client.Get(ctx, namespacedName(wl), &obj); err != nil {
			return Health{}, err
		}
		return replicasetHealth(&obj), nil
	case "StatefulSet":
		var obj appsv1.StatefulSet
		if err := client.Get(ctx, namespacedName(wl), &obj); err != nil {
			return Health{}, err
		}
		return statefulsetHealth(&obj), nil
	case "DaemonSet":
		var obj appsv1.DaemonSet
		if err := client.Get(ctx, namespacedName(wl), &obj); err != nil {
			return Health{}, err
		}
		return daemonsetHealth(&obj), nil
	case "Job":
		var obj batchv1.Job
		if err := client.Get(ctx, namespacedName(wl), &obj); err != nil {
			return Health{}, err
		}
		return jobHealth(&obj), nil
	default:
		return Health{}, fmt.Errorf("unsupported kind %q (group=%s version=%s)", wl.WorkloadKind, wl.WorkloadGroup, wl.WorkloadVersion)
	}
}

func namespacedName(w *scanv1alpha1.K8sWorkloadSpec) types.NamespacedName {
	return types.NamespacedName{Namespace: w.Namespace, Name: w.WorkloadName}
}

func deploymentHealth(d *appsv1.Deployment) Health {
	genOK := d.Status.ObservedGeneration >= d.Generation
	avail := getCond(d.Status.Conditions, appsv1.DeploymentAvailable)
	prog := getCond(d.Status.Conditions, appsv1.DeploymentProgressing)

	ready := d.Status.ReadyReplicas
	replicas := d.Status.Replicas
	updated := d.Status.UpdatedReplicas

	missingReady := int32(0)
	if replicas > ready {
		missingReady = replicas - ready
	}
	outdated := int32(0)
	if replicas > updated {
		outdated = replicas - updated
	}

	healthy := genOK && ready == replicas && updated == replicas && avail == corev1.ConditionTrue && prog == corev1.ConditionTrue

	reason := scanv1alpha1.ConsumerWorkloadReady
	msg := "Deployment healthy"
	if !healthy {
		reason, msg = scanv1alpha1.ConsumerWorkloadNotReady,
			fmt.Sprintf(
				"Deployment %s/%s: generation(observed=%d desired=%d): %t, ready=%d/%d (missing=%d), updated=%d/%d (outdated=%d), Available=%s, Progressing=%s",
				d.Namespace, d.Name, d.Status.ObservedGeneration, d.Generation,
				genOK, d.Status.ReadyReplicas, d.Status.Replicas, missingReady,
				updated, replicas, outdated, avail, prog,
			)
	}
	return Health{Healthy: healthy, Reason: reason, Message: msg}
}

func replicasetHealth(rs *appsv1.ReplicaSet) Health {
	readyEq := rs.Status.ReadyReplicas == rs.Status.Replicas
	healthy := readyEq
	reason := scanv1alpha1.ConsumerWorkloadReady
	msg := "Replicaset healthy"
	if !healthy {
		reason, msg = scanv1alpha1.ConsumerWorkloadNotReady,
			fmt.Sprintf(
				"Replicaset %s/%s: ready=%d replicas=%d",
				rs.Namespace, rs.Name, rs.Status.ReadyReplicas, rs.Status.Replicas,
			)
	}
	return Health{Healthy: healthy, Reason: reason, Message: msg}
}

func statefulsetHealth(sts *appsv1.StatefulSet) Health {
	genOK := sts.Status.ObservedGeneration >= sts.Generation
	readyEq := sts.Status.ReadyReplicas == sts.Status.Replicas
	healthy := genOK && readyEq
	reason := scanv1alpha1.ConsumerWorkloadReady
	msg := "Statefulset healthy"
	if !healthy {
		reason, msg = scanv1alpha1.ConsumerWorkloadNotReady,
			fmt.Sprintf(
				"Statefulset %s/%s: generation(observed=%d desired=%d)=%t ready=%d/%d",
				sts.Namespace, sts.Name, sts.Status.ObservedGeneration, sts.Generation,
				genOK, sts.Status.ReadyReplicas, sts.Status.Replicas,
			)
	}
	return Health{Healthy: healthy, Reason: reason, Message: msg}
}

func daemonsetHealth(ds *appsv1.DaemonSet) Health {
	genOK := ds.Status.ObservedGeneration >= ds.Generation
	readyEq := ds.Status.NumberReady == ds.Status.DesiredNumberScheduled
	updatedEq := ds.Status.UpdatedNumberScheduled == ds.Status.DesiredNumberScheduled
	healthy := genOK && readyEq && updatedEq
	reason := scanv1alpha1.ConsumerWorkloadReady
	msg := "Daemonset healthy"
	if !healthy {
		reason, msg = scanv1alpha1.ConsumerWorkloadNotReady,
			fmt.Sprintf(
				"Daemonset %s/%s: generation(observed=%d desired=%d)=%t ready=%d/%d updated=%d",
				ds.Namespace, ds.Name, ds.Status.ObservedGeneration, ds.Generation,
				genOK, ds.Status.NumberReady, ds.Status.DesiredNumberScheduled, ds.Status.UpdatedNumberScheduled,
			)
	}
	return Health{Healthy: healthy, Reason: reason, Message: msg}
}

func jobHealth(j *batchv1.Job) Health {
	want := int32(1)
	if j.Spec.Completions != nil {
		want = *j.Spec.Completions
	}
	succeeded := j.Status.Succeeded
	failed := j.Status.Failed
	backoff := int32(6)
	if j.Spec.BackoffLimit != nil {
		backoff = *j.Spec.BackoffLimit
	}
	healthy := succeeded >= want
	reason := scanv1alpha1.ConsumerWorkloadReady
	msg := "Job completed"
	if !healthy {
		if failed > backoff {
			reason, msg = "Failed", fmt.Sprintf(
				"Job %s/%s: failed=%d backoffLimit=%d",
				j.Namespace, j.Name, failed, backoff,
			)
		} else {
			reason, msg = "Running", fmt.Sprintf(
				"Job %s/%s: succeeded=%d/%d active=%d failed=%d",
				j.Namespace, j.Name, succeeded, want, j.Status.Active, failed,
			)
		}
	}
	return Health{Healthy: healthy, Reason: reason, Message: msg}
}

func getCond(conds []appsv1.DeploymentCondition, t appsv1.DeploymentConditionType) corev1.ConditionStatus {
	for _, c := range conds {
		if c.Type == t {
			return c.Status
		}
	}
	return corev1.ConditionUnknown
}
