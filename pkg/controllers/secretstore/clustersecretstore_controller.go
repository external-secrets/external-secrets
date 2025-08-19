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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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

	if err := r.handleClusterSecretStoreFinalizer(ctx, &css, log); err != nil {
		return ctrl.Result{}, err
	}

	return reconcile(ctx, req, &css, r.Client, log, Opts{
		ControllerClass: r.ControllerClass,
		GaugeVecGetter:  cssmetrics.GetGaugeVec,
		Recorder:        r.recorder,
		RequeueInterval: r.RequeueInterval,
	})
}

// handleClusterSecretStoreFinalizer manages the finalizer for ClusterSecretStores
// It adds a finalizer when there are PushSecrets with DeletionPolicy=Delete that reference this store
// and removes it when there are no such PushSecrets.
func (r *ClusterStoreReconciler) handleClusterSecretStoreFinalizer(ctx context.Context, css *esapi.ClusterSecretStore, log logr.Logger) error {
	// Check if this ClusterSecretStore is referenced by any PushSecrets with DeletionPolicy=Delete
	hasPushSecretsWithDeletePolicy, err := r.hasPushSecretsWithDeletePolicy(ctx, css)
	if err != nil {
		return fmt.Errorf("failed to check PushSecrets: %w", err)
	}

	// If the store is being deleted and has the finalizer, check if we can remove it
	if !css.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(css, secretStoreFinalizer) {
			if hasPushSecretsWithDeletePolicy {
				// Cannot remove finalizer yet, there are still PushSecrets that need this store
				log.Info("cannot remove finalizer, there are still PushSecrets with DeletionPolicy=Delete that reference this store")
				return nil
			}
			// Safe to remove finalizer
			controllerutil.RemoveFinalizer(css, secretStoreFinalizer)
			if err := r.Client.Update(ctx, css, &client.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to remove finalizer: %w", err)
			}
			log.Info("removed finalizer from ClusterSecretStore")
		}
		return nil
	}

	// If not being deleted, manage the finalizer based on whether there are PushSecrets with Delete policy
	if hasPushSecretsWithDeletePolicy {
		if added := controllerutil.AddFinalizer(css, secretStoreFinalizer); added {
			if err := r.Client.Update(ctx, css, &client.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to add finalizer: %w", err)
			}
			log.Info("added finalizer to ClusterSecretStore due to PushSecrets with DeletionPolicy=Delete")
		}
	} else {
		if controllerutil.ContainsFinalizer(css, secretStoreFinalizer) {
			controllerutil.RemoveFinalizer(css, secretStoreFinalizer)
			if err := r.Client.Update(ctx, css, &client.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to remove finalizer: %w", err)
			}
			log.Info("removed finalizer from ClusterSecretStore, no PushSecrets with DeletionPolicy=Delete found")
		}
	}

	return nil
}

// hasPushSecretsWithDeletePolicy checks if there are any PushSecrets with DeletionPolicy=Delete
// that reference this ClusterSecretStore using the controller-runtime index.
func (r *ClusterStoreReconciler) hasPushSecretsWithDeletePolicy(ctx context.Context, css *esapi.ClusterSecretStore) (bool, error) {
	storeName := css.GetName()

	// Use the index to find PushSecrets that have pushed to this store
	var pushSecretList esv1alpha1.PushSecretList
	if err := r.List(ctx, &pushSecretList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.syncedPushSecrets", storeName),
		// No namespace filter for ClusterSecretStore - look in all namespaces
	}); err != nil {
		return false, fmt.Errorf("failed to list PushSecrets by store index: %w", err)
	}

	// Check if any of these PushSecrets have DeletionPolicy=Delete
	for _, ps := range pushSecretList.Items {
		if ps.Spec.DeletionPolicy == esv1alpha1.PushSecretDeletionPolicyDelete {
			// Verify the store reference matches
			storeKey := fmt.Sprintf("%s/%s", esapi.ClusterSecretStoreKind, storeName)
			if _, hasPushed := ps.Status.SyncedPushSecrets[storeKey]; hasPushed {
				return true, nil
			}
		}
	}

	// Also check for PushSecrets that reference this store by name or labelSelector
	// but haven't pushed to it yet (for initial finalizer setup)
	if err := r.List(ctx, &pushSecretList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.deletionPolicy", string(esv1alpha1.PushSecretDeletionPolicyDelete)),
		// No namespace filter for ClusterSecretStore - look in all namespaces
	}); err != nil {
		return false, fmt.Errorf("failed to list PushSecrets by deletionPolicy: %w", err)
	}

	for _, ps := range pushSecretList.Items {
		// Check if this PushSecret references our store
		for _, storeRef := range ps.Spec.SecretStoreRefs {
			if storeRef.Name == storeName && storeRef.Kind == esapi.ClusterSecretStoreKind {
				return true, nil
			}
		}

		// Check labelSelector match
		for _, storeRef := range ps.Spec.SecretStoreRefs {
			if storeRef.LabelSelector != nil && storeRef.Kind == esapi.ClusterSecretStoreKind {
				selector, err := metav1.LabelSelectorAsSelector(storeRef.LabelSelector)
				if err != nil {
					continue
				}
				if selector.Matches(labels.Set(css.GetLabels())) {
					return true, nil
				}
			}
		}
	}

	return false, nil
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
			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
				// Only watch PushSecrets with DeletionPolicy=Delete
				ps := obj.(*esv1alpha1.PushSecret)
				return ps.Spec.DeletionPolicy == esv1alpha1.PushSecretDeletionPolicyDelete
			})),
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
