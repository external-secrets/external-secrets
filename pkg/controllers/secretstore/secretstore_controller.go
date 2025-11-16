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

package secretstore

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrlreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/ssmetrics"

	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/register"
)

// StoreReconciler reconciles a SecretStore object.
type StoreReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *runtime.Scheme
	recorder          record.EventRecorder
	RequeueInterval   time.Duration
	ControllerClass   string
	PushSecretEnabled bool
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *StoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("secretstore", req.NamespacedName)

	resourceLabels := ctrlmetrics.RefineNonConditionMetricLabels(map[string]string{"name": req.Name, "namespace": req.Namespace})
	start := time.Now()

	secretStoreReconcileDuration := ssmetrics.GetGaugeVec(ssmetrics.SecretStoreReconcileDurationKey)
	defer func() { secretStoreReconcileDuration.With(resourceLabels).Set(float64(time.Since(start))) }()

	var ss esapi.SecretStore
	err := r.Get(ctx, req.NamespacedName, &ss)
	if apierrors.IsNotFound(err) {
		ssmetrics.RemoveMetrics(req.Namespace, req.Name)
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "unable to get SecretStore")
		return ctrl.Result{}, err
	}

	return reconcile(ctx, req, &ss, r.Client, r.PushSecretEnabled, log, Opts{
		ControllerClass: r.ControllerClass,
		GaugeVecGetter:  ssmetrics.GetGaugeVec,
		Recorder:        r.recorder,
		RequeueInterval: r.RequeueInterval,
	})
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *StoreReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("secret-store")

	builder := ctrl.NewControllerManagedBy(mgr)

	if r.PushSecretEnabled {
		return builder.WithOptions(opts).
			For(&esapi.SecretStore{}).
			Watches(
				&esv1alpha1.PushSecret{},
				handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrlreconcile.Request {
					return findStoresForPushSecret(ctx, r.Client, obj, &esapi.SecretStoreList{})
				}),
			).
			Complete(r)
	}

	return builder.WithOptions(opts).
		For(&esapi.SecretStore{}).
		Complete(r)
}
