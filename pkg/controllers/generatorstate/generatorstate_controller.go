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

type Reconciler struct {
	client.Client

	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	recorder   record.EventRecorder
}

const generatorStateFinalizer = "generatorstate.externalsecrets.io/finalizer"

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	generatorState := &genv1alpha1.GeneratorState{}
	err = r.Get(ctx, req.NamespacedName, generatorState)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	requeue, err := r.handleFinalizer(ctx, generatorState)
	if err != nil {
		return ctrl.Result{}, err
	}
	if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	if generatorState.Spec.GarbageCollectionDeadline != nil {
		if generatorState.Spec.GarbageCollectionDeadline.Time.Before(time.Now()) {
			if err := r.Client.Delete(ctx, generatorState, &client.DeleteOptions{}); err != nil {
				r.markAsFailed("could not delete GeneratorState", err, generatorState)
				return ctrl.Result{}, fmt.Errorf("could not delete GeneratorState: %w", err)
			}
			r.markSuccess("Reached gc deadline", generatorState)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{
			RequeueAfter: time.Until(generatorState.Spec.GarbageCollectionDeadline.Time),
		}, nil
	}

	r.markSuccess("GeneratorState created", generatorState)
	return ctrl.Result{}, nil
}

func (r *Reconciler) handleFinalizer(ctx context.Context, generatorState *genv1alpha1.GeneratorState) (bool, error) {
	if generatorState.ObjectMeta.DeletionTimestamp.IsZero() {
		if added := controllerutil.AddFinalizer(generatorState, generatorStateFinalizer); added {
			if err := r.Client.Update(ctx, generatorState, &client.UpdateOptions{}); err != nil {
				return false, fmt.Errorf("could not update finalizers: %w", err)
			}
			return true, nil
		}
	} else if controllerutil.ContainsFinalizer(generatorState, generatorStateFinalizer) {
		gen, err := r.getGenerator(generatorState.Spec.Resource.Raw)
		if err != nil {
			r.markAsFailed("could not get generator", err, generatorState)
			return false, fmt.Errorf("could not get generator: %w", err)
		}

		if err := gen.Cleanup(ctx, generatorState.Spec.Resource, generatorState.Spec.State, r.Client, generatorState.Namespace); err != nil {
			r.markAsFailed("could not cleanup generator state", err, generatorState)
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
	gen, ok := genv1alpha1.GetGeneratorByName(us.GroupVersionKind().Kind)
	if !ok {
		return nil, fmt.Errorf("generator not found")
	}
	return gen, nil
}

func (r *Reconciler) markAsFailed(msg string, err error, gs *genv1alpha1.GeneratorState) {
	conditionSynced := NewGeneratorStateCondition(genv1alpha1.GeneratorStateReady, v1.ConditionFalse, genv1alpha1.ConditionReasonError, fmt.Sprintf("%s: %v", msg, err))
	SetGeneratorStateCondition(gs, *conditionSynced)
}

func (r *Reconciler) markSuccess(msg string, gs *genv1alpha1.GeneratorState) {
	newReadyCondition := NewGeneratorStateCondition(genv1alpha1.GeneratorStateReady, v1.ConditionTrue, genv1alpha1.ConditionReasonCreated, msg)
	SetGeneratorStateCondition(gs, *newReadyCondition)
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("external-secrets")
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&genv1alpha1.GeneratorState{}).
		Complete(r)
}
