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

// Package providerstore implements controllers for ProviderStore and ClusterProviderStore resources.
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
	providermetrics "github.com/external-secrets/external-secrets/pkg/controllers/provider"
)

// StoreReconciler reconciles ProviderStore resources.
type StoreReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	RequeueInterval time.Duration
}

func (r *StoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	start := time.Now()
	log := r.Log.WithValues("ProviderStore", req.NamespacedName)

	var store esv2alpha1.ProviderStore
	if err := r.Get(ctx, req.NamespacedName, &store); err != nil {
		if apierrors.IsNotFound(err) {
			providermetrics.RemoveMetrics(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := validateStore(ctx, r.Client, &store, store.Namespace); err != nil {
		setReadyCondition(&store, corev1.ConditionFalse, "ValidationFailed", err.Error())
		recordProviderStoreCompatibilityMetrics(&store, time.Since(start))
		return updateStatus(ctx, r.Status(), &store, r.RequeueInterval, log)
	}

	setReadyCondition(&store, corev1.ConditionTrue, "StoreValid", "store validated")
	recordProviderStoreCompatibilityMetrics(&store, time.Since(start))
	return updateStatus(ctx, r.Status(), &store, r.RequeueInterval, log)
}

func (r *StoreReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esv2alpha1.ProviderStore{}).
		Watches(
			&esv1alpha1.ClusterProviderClass{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
				runtimeClass, ok := obj.(*esv1alpha1.ClusterProviderClass)
				if !ok {
					return nil
				}

				return findProviderStoresForRuntimeClass(ctx, r.Client, runtimeClass)
			}),
		).
		Complete(r)
}
