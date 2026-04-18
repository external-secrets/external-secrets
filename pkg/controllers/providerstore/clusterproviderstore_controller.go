/*
Copyright © The ESO Authors

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

package providerstore

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv2alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v2alpha1"
	clusterprovidermetrics "github.com/external-secrets/external-secrets/pkg/controllers/clusterprovider"
)

// ClusterStoreReconciler reconciles ClusterProviderStore resources.
type ClusterStoreReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	RequeueInterval time.Duration
}

func (r *ClusterStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	start := time.Now()
	log := r.Log.WithValues("ClusterProviderStore", req.NamespacedName)

	var store esv2alpha1.ClusterProviderStore
	if err := r.Get(ctx, req.NamespacedName, &store); err != nil {
		if apierrors.IsNotFound(err) {
			clusterprovidermetrics.RemoveMetrics(req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if store.Spec.BackendRef.Namespace == "" {
		if err := assertRuntimeClassReady(ctx, r.Client, store.Spec.RuntimeRef); err != nil {
			setReadyCondition(&store, corev1.ConditionFalse, "RuntimeNotReady", err.Error())
		} else {
			setReadyCondition(&store, corev1.ConditionTrue, "RuntimeReady", "backend namespace resolved per caller namespace")
		}
		recordClusterProviderStoreCompatibilityMetrics(&store, time.Since(start))
		return updateStatus(ctx, r.Status(), &store, r.RequeueInterval, log)
	}

	if err := validateStore(ctx, r.Client, &store, store.Spec.BackendRef.Namespace); err != nil {
		setReadyCondition(&store, corev1.ConditionFalse, "ValidationFailed", err.Error())
		recordClusterProviderStoreCompatibilityMetrics(&store, time.Since(start))
		return updateStatus(ctx, r.Status(), &store, r.RequeueInterval, log)
	}

	setReadyCondition(&store, corev1.ConditionTrue, "StoreValid", "store validated")
	recordClusterProviderStoreCompatibilityMetrics(&store, time.Since(start))
	return updateStatus(ctx, r.Status(), &store, r.RequeueInterval, log)
}

func (r *ClusterStoreReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esv2alpha1.ClusterProviderStore{}).
		Watches(
			&esv1alpha1.ClusterProviderClass{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
				runtimeClass, ok := obj.(*esv1alpha1.ClusterProviderClass)
				if !ok {
					return nil
				}

				return findClusterProviderStoresForRuntimeClass(ctx, r.Client, runtimeClass)
			}),
		).
		Complete(r)
}
