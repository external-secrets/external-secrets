/*
Copyright © 2025 ESO Maintainer Team

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

// Package generatorstate implements controllers for managing GeneratorState resources
package generatorstate

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

// Reconciler reconciles a GeneratorState object, managing its lifecycle and cleanup.
type Reconciler struct {
	client.Client

	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	recorder   record.EventRecorder
}

const generatorStateFinalizer = "generatorstate.externalsecrets.io/finalizer"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	generatorState := &genv1alpha1.GeneratorState{}
	err = r.Get(ctx, req.NamespacedName, generatorState)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	defer func() {
		err := r.Status().Update(ctx, generatorState)
		if err != nil {
			r.Log.Error(err, "could not update generator state status")
		}
	}()
	gen, err := r.getGenerator(generatorState.Spec.Resource.Raw)
	if err != nil {
		r.markAsFailed(genv1alpha1.GeneratorStatePendingDeletion, "Could not get generator", err, generatorState)
		return ctrl.Result{}, fmt.Errorf("could not get generator: %w", err)
	}

	requeue, err := r.handleFinalizer(ctx, generatorState, gen)
	if err != nil {
		return ctrl.Result{}, err
	}
	if requeue {
		return ctrl.Result{RequeueAfter: time.Millisecond}, nil
	}

	if generatorState.Spec.GarbageCollectionDeadline != nil {
		cleanupPolicy, err := gen.GetCleanupPolicy(generatorState.Spec.Resource)
		if err != nil {
			r.markAsFailed(genv1alpha1.GeneratorStatePendingDeletion, "Could not get generator cleanup policy", err, generatorState)
			return ctrl.Result{}, fmt.Errorf("could not get cleanup policy: %w", err)
		}

		gcDeadlineReached := generatorState.Spec.GarbageCollectionDeadline.Time.Before(time.Now())
		if cleanupPolicy != nil && cleanupPolicy.Type == genv1alpha1.IdleCleanupPolicy && gcDeadlineReached {
			if generatorState.DeletionTimestamp != nil {
				return ctrl.Result{}, nil
			}
			shouldCleanup, err := r.isIdleTimeoutExpired(ctx, *cleanupPolicy, gen, generatorState)
			if err != nil {
				r.markAsFailed(genv1alpha1.GeneratorStatePendingDeletion, "Could not check idle timeout", err, generatorState)
				return ctrl.Result{}, fmt.Errorf("could not check idle timeout: %w", err)
			}
			if shouldCleanup {
				if err := r.Client.Delete(ctx, generatorState, &client.DeleteOptions{}); err != nil {
					r.markAsFailed(genv1alpha1.GeneratorStateReady, "Could not delete GeneratorState", err, generatorState)
					return ctrl.Result{}, fmt.Errorf("could not delete GeneratorState: %w", err)
				}
				r.markSuccess(genv1alpha1.GeneratorStateTerminating, genv1alpha1.ConditionReasonDeadlineReached, "Reached idle timout", generatorState)
				return ctrl.Result{}, nil
			}
			r.markSuccess(
				genv1alpha1.GeneratorStateDeletionScheduled,
				genv1alpha1.ConditionReasonStillActive,
				fmt.Sprintf("State still active. Next check in %s", cleanupPolicy.IdleTimeout.Duration.String()),
				generatorState,
			)
			return ctrl.Result{RequeueAfter: cleanupPolicy.IdleTimeout.Duration}, nil
		}

		if gcDeadlineReached {
			if generatorState.DeletionTimestamp != nil {
				return ctrl.Result{}, nil
			}

			if err := r.Client.Delete(ctx, generatorState, &client.DeleteOptions{}); err != nil {
				r.markAsFailed(genv1alpha1.GeneratorStateReady, "Could not delete GeneratorState", err, generatorState)
				return ctrl.Result{}, fmt.Errorf("could not delete GeneratorState: %w", err)
			}
			r.markSuccess(genv1alpha1.GeneratorStateTerminating, genv1alpha1.ConditionReasonDeadlineReached, "Reached garbage collection deadline", generatorState)
			return ctrl.Result{}, nil
		}
		r.markSuccess(
			genv1alpha1.GeneratorStateDeletionScheduled,
			genv1alpha1.ConditionReasonGarbageCollectionSetted,
			fmt.Sprintf("Deletion scheduled to: %s", generatorState.Spec.GarbageCollectionDeadline.Time.String()),
			generatorState,
		)
		return ctrl.Result{
			RequeueAfter: time.Until(generatorState.Spec.GarbageCollectionDeadline.Time),
		}, nil
	}

	// Add active status here
	r.markSuccess(genv1alpha1.GeneratorStateReady, genv1alpha1.ConditionReasonCreated, "GeneratorState created", generatorState)
	return ctrl.Result{}, nil
}

func (r *Reconciler) handleFinalizer(ctx context.Context, generatorState *genv1alpha1.GeneratorState, gen genv1alpha1.Generator) (bool, error) {
	if generatorState.ObjectMeta.DeletionTimestamp.IsZero() {
		if added := controllerutil.AddFinalizer(generatorState, generatorStateFinalizer); added {
			if err := r.Client.Update(ctx, generatorState, &client.UpdateOptions{}); err != nil {
				return false, fmt.Errorf("could not update finalizers: %w", err)
			}
			return true, nil
		}
	} else if controllerutil.ContainsFinalizer(generatorState, generatorStateFinalizer) {
		if err := gen.Cleanup(ctx, generatorState.Spec.Resource, generatorState.Spec.State, r.Client, generatorState.Namespace); err != nil {
			r.markAsFailed(genv1alpha1.GeneratorStatePendingDeletion, "Could not cleanup generator state", err, generatorState)
			return false, fmt.Errorf("could not cleanup generator state: %w", err)
		}

		controllerutil.RemoveFinalizer(generatorState, generatorStateFinalizer)
		if err := r.Client.Update(ctx, generatorState, &client.UpdateOptions{}); err != nil {
			return false, fmt.Errorf("could not update finalizers: %w", err)
		}
	}
	return false, nil
}

func (r *Reconciler) getGenerator(resource []byte) (genv1alpha1.Generator, error) {
	us := &unstructured.Unstructured{}
	if err := us.UnmarshalJSON(resource); err != nil {
		return nil, fmt.Errorf("unable to unmarshal resource: %w", err)
	}
	gen, ok := genv1alpha1.GetGeneratorByKind(us.GroupVersionKind().Kind)
	if !ok {
		return nil, fmt.Errorf("generator not found")
	}
	return gen, nil
}

func (r *Reconciler) isIdleTimeoutExpired(ctx context.Context, policy genv1alpha1.CleanupPolicy, gen genv1alpha1.Generator, gs *genv1alpha1.GeneratorState) (bool, error) {
	// Determine last activity timestamp
	lastActivity, found, err := gen.LastActivityTime(ctx, gs.Spec.Resource, gs.Spec.State, r.Client, gs.Namespace)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	// If still within idle timeout, schedule next check
	if time.Since(lastActivity) < policy.IdleTimeout.Duration {
		return false, nil
	}

	// Idle timeout expired; proceed with cleanup
	return true, nil
}

func (r *Reconciler) markAsFailed(conditionType genv1alpha1.GeneratorStateConditionType, msg string, err error, gs *genv1alpha1.GeneratorState) {
	conditionSynced := NewGeneratorStateCondition(conditionType, v1.ConditionFalse, genv1alpha1.ConditionReasonError, fmt.Sprintf("%s: %v", msg, err))
	SetGeneratorStateCondition(gs, *conditionSynced)
	SetLastGeneratorStateCondition(gs, *conditionSynced)
}

func (r *Reconciler) markSuccess(conditionType genv1alpha1.GeneratorStateConditionType, conditionReason, msg string, gs *genv1alpha1.GeneratorState) {
	conditionSynced := NewGeneratorStateCondition(conditionType, v1.ConditionTrue, conditionReason, msg)
	SetGeneratorStateCondition(gs, *conditionSynced)
	SetLastGeneratorStateCondition(gs, *conditionSynced)
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("external-secrets")
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&genv1alpha1.GeneratorState{}).
		Complete(r)
}
