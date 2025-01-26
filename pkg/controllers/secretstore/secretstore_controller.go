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

package secretstore

import (
	"context"
	"slices"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/ssmetrics"

	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

const (
	secretStoreFinalizer = "secretstore.externalsecrets.io/finalizer"
)

// StoreReconciler reconciles a SecretStore object.
type StoreReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	RequeueInterval time.Duration
	ControllerClass string
}

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

	if ss.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&ss, secretStoreFinalizer) {
			controllerutil.AddFinalizer(&ss, secretStoreFinalizer)
			if err := r.Update(ctx, &ss); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(&ss, secretStoreFinalizer) {
			var pushSecretList v1alpha1.PushSecretList

			listOptions := &client.ListOptions{
				Namespace: ss.ObjectMeta.Namespace,
			}

			if err := r.List(ctx, &pushSecretList, listOptions); err != nil {
				log.Error(err, "unable to get PushSecretList")
				return ctrl.Result{}, err
			}

			for _, ps := range pushSecretList.Items {
				if ps.Spec.DeletionPolicy == v1alpha1.PushSecretDeletionPolicyDelete {
					hasRef := slices.IndexFunc(ps.Spec.SecretStoreRefs, func(pushSecretStoreRef v1alpha1.PushSecretStoreRef) bool {
						return pushSecretStoreRef.Name == ss.ObjectMeta.Name
					})

					if hasRef != -1 {
						return ctrl.Result{RequeueAfter: 5}, nil
					}
				}
			}
		}
		controllerutil.RemoveFinalizer(&ss, secretStoreFinalizer)
		if err := r.Update(ctx, &ss); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return reconcile(ctx, req, &ss, r.Client, log, Opts{
		ControllerClass: r.ControllerClass,
		GaugeVecGetter:  ssmetrics.GetGaugeVec,
		Recorder:        r.recorder,
		RequeueInterval: r.RequeueInterval,
	})
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *StoreReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("secret-store")

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esapi.SecretStore{}).
		Complete(r)
}
