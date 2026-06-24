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

// Package providerclass implements readiness reconciliation for ProviderClass runtimes.
package providerclass

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	grpccommon "github.com/external-secrets/external-secrets/providers/v2/common/grpc"
)

// Reconciler reconciles ProviderClass resources by checking runtime health.
type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	RequeueInterval time.Duration
	CheckHealth     func(context.Context, string, *grpccommon.TLSConfig) error
}

// Reconcile updates the Ready condition based on the runtime health check result.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ProviderClass", req.NamespacedName)

	var obj esv1alpha1.ProviderClass
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	tlsSecretNamespace := grpccommon.ResolveTLSSecretNamespace(obj.Spec.Address, "", "", "")
	tlsConfig, err := grpccommon.LoadClientTLSConfig(ctx, r.Client, obj.Spec.Address, tlsSecretNamespace)
	if err != nil {
		log.Error(err, "failed to load runtime TLS config")
		return r.updateStatus(ctx, &obj, metav1.ConditionFalse, "TLSConfigFailed", err.Error())
	}

	if err := r.reconcileHealth(ctx, &obj, tlsConfig); err != nil {
		log.Error(err, "runtime health check failed")
		return r.updateStatus(ctx, &obj, metav1.ConditionFalse, "HealthCheckFailed", err.Error())
	}

	return r.updateStatus(ctx, &obj, metav1.ConditionTrue, "Healthy", "runtime is serving")
}

func (r *Reconciler) reconcileHealth(ctx context.Context, obj *esv1alpha1.ProviderClass, tlsConfig *grpccommon.TLSConfig) error {
	checker := r.CheckHealth
	if checker == nil {
		checker = grpccommon.CheckHealth
	}

	return checker(ctx, obj.Spec.Address, tlsConfig)
}

func (r *Reconciler) updateStatus(ctx context.Context, obj *esv1alpha1.ProviderClass, status metav1.ConditionStatus, reason, message string) (ctrl.Result, error) {
	meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: obj.GetGeneration(),
	})

	if err := r.Status().Update(ctx, obj); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update ProviderClass status: %w", err)
	}

	return ctrl.Result{RequeueAfter: r.RequeueInterval}, nil
}

// SetupWithManager wires the controller into controller-runtime.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esv1alpha1.ProviderClass{}).
		Complete(r)
}
