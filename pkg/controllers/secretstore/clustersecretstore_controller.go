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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrlreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/cssmetrics"

	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

// ClusterStoreReconciler reconciles a SecretStore object.
type ClusterStoreReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	ControllerClass string
	RequeueInterval time.Duration
	recorder        record.EventRecorder
}

func (r *ClusterStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("clustersecretstore", req.NamespacedName)

	resourceLabels := ctrlmetrics.RefineNonConditionMetricLabels(map[string]string{"name": req.Name, "namespace": req.Namespace})
	start := time.Now()

	clusterSecretStoreReconcileDuration := cssmetrics.GetGaugeVec(cssmetrics.ClusterSecretStoreReconcileDurationKey)
	defer func() { clusterSecretStoreReconcileDuration.With(resourceLabels).Set(float64(time.Since(start))) }()

	var css esapi.ClusterSecretStore
	err := r.Get(ctx, req.NamespacedName, &css)
	if apierrors.IsNotFound(err) {
		cssmetrics.RemoveMetrics(req.Namespace, req.Name)
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "unable to get ClusterSecretStore")
		return ctrl.Result{}, err
	}

	return reconcile(ctx, req, &css, r.Client, log, Opts{
		ControllerClass: r.ControllerClass,
		GaugeVecGetter:  cssmetrics.GetGaugeVec,
		Recorder:        r.recorder,
		RequeueInterval: r.RequeueInterval,
	})
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *ClusterStoreReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("cluster-secret-store")

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esapi.ClusterSecretStore{}).
		Watches(
			&esv1alpha1.PushSecret{},
			handler.EnqueueRequestsFromMapFunc(r.findClusterSecretStoresForPushSecret),
		).
		Complete(r)
}

// findClusterSecretStoresForPushSecret finds ClusterSecretStores that should be reconciled when a PushSecret changes.
func (r *ClusterStoreReconciler) findClusterSecretStoresForPushSecret(ctx context.Context, obj client.Object) []ctrlreconcile.Request {
	ps := obj.(*esv1alpha1.PushSecret)
	var requests []ctrlreconcile.Request

	// Get all ClusterSecretStores
	var clusterSecretStoreList esapi.ClusterSecretStoreList
	if err := r.List(ctx, &clusterSecretStoreList); err != nil {
		return requests
	}

	// Check which ClusterSecretStores this PushSecret references
	for _, store := range clusterSecretStoreList.Items {
		if shouldReconcileClusterSecretStoreForPushSecret(&store, ps) {
			requests = append(requests, ctrlreconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: store.Name,
				},
			})
		}
	}

	return requests
}

// shouldReconcileClusterSecretStoreForPushSecret determines if a ClusterSecretStore should be reconciled
// when a PushSecret changes, based on whether the PushSecret references this store.
func shouldReconcileClusterSecretStoreForPushSecret(store *esapi.ClusterSecretStore, ps *esv1alpha1.PushSecret) bool {
	// Check if this PushSecret has pushed to this store
	storeKey := fmt.Sprintf("%s/%s", esapi.ClusterSecretStoreKind, store.Name)
	_, hasPushed := ps.Status.SyncedPushSecrets[storeKey]

	// Also check if the PushSecret references this store in its spec
	for _, storeRef := range ps.Spec.SecretStoreRefs {
		if storeRef.Name == store.Name && storeRef.Kind == esapi.ClusterSecretStoreKind {
			return true
		}
		// Check labelSelector match
		if storeRef.LabelSelector != nil && storeRef.Kind == esapi.ClusterSecretStoreKind {
			selector, err := metav1.LabelSelectorAsSelector(storeRef.LabelSelector)
			if err == nil && selector.Matches(labels.Set(store.Labels)) {
				return true
			}
		}
	}

	return hasPushed
}
