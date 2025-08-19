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
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	ctrlreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/ssmetrics"

	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	if err := r.handleSecretStoreFinalizer(ctx, &ss, log); err != nil {
		return ctrl.Result{}, err
	}

	return reconcile(ctx, req, &ss, r.Client, log, Opts{
		ControllerClass: r.ControllerClass,
		GaugeVecGetter:  ssmetrics.GetGaugeVec,
		Recorder:        r.recorder,
		RequeueInterval: r.RequeueInterval,
	})
}

// handleSecretStoreFinalizer manages the finalizer for SecretStores
// It adds a finalizer when there are PushSecrets with DeletionPolicy=Delete that reference this store
// and removes it when there are no such PushSecrets.
func (r *StoreReconciler) handleSecretStoreFinalizer(ctx context.Context, ss *esapi.SecretStore, log logr.Logger) error {
	// Check if this SecretStore is referenced by any PushSecrets with DeletionPolicy=Delete
	hasPushSecretsWithDeletePolicy, err := r.hasPushSecretsWithDeletePolicy(ctx, ss)
	if err != nil {
		return fmt.Errorf("failed to check PushSecrets: %w", err)
	}

	// If the store is being deleted and has the finalizer, check if we can remove it
	if !ss.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(ss, secretStoreFinalizer) {
			if hasPushSecretsWithDeletePolicy {
				// Cannot remove finalizer yet, there are still PushSecrets that need this store
				log.Info("cannot remove finalizer, there are still PushSecrets with DeletionPolicy=Delete that reference this store")
				return nil
			}
			// Safe to remove finalizer
			controllerutil.RemoveFinalizer(ss, secretStoreFinalizer)
			if err := r.Client.Update(ctx, ss, &client.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to remove finalizer: %w", err)
			}
			log.Info("removed finalizer from SecretStore")
		}
		return nil
	}

	// If not being deleted, manage the finalizer based on whether there are PushSecrets with Delete policy
	if hasPushSecretsWithDeletePolicy {
		if added := controllerutil.AddFinalizer(ss, secretStoreFinalizer); added {
			if err := r.Client.Update(ctx, ss, &client.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to add finalizer: %w", err)
			}
			log.Info("added finalizer to SecretStore due to PushSecrets with DeletionPolicy=Delete")
		}
	} else {
		if controllerutil.ContainsFinalizer(ss, secretStoreFinalizer) {
			controllerutil.RemoveFinalizer(ss, secretStoreFinalizer)
			if err := r.Client.Update(ctx, ss, &client.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to remove finalizer: %w", err)
			}
			log.Info("removed finalizer from SecretStore, no PushSecrets with DeletionPolicy=Delete found")
		}
	}

	return nil
}

// hasPushSecretsWithDeletePolicy checks if there are any PushSecrets with DeletionPolicy=Delete
// that reference this SecretStore using the controller-runtime index.
func (r *StoreReconciler) hasPushSecretsWithDeletePolicy(ctx context.Context, ss *esapi.SecretStore) (bool, error) {
	storeName := ss.GetName()
	storeNamespace := ss.GetNamespace()

	// Use the index to find PushSecrets that have pushed to this store
	var pushSecretList esv1alpha1.PushSecretList
	if err := r.List(ctx, &pushSecretList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.syncedPushSecrets", storeName),
		Namespace:     storeNamespace, // Only look in the same namespace for SecretStore
	}); err != nil {
		return false, fmt.Errorf("failed to list PushSecrets by store index: %w", err)
	}

	// Check if any of these PushSecrets have DeletionPolicy=Delete
	for _, ps := range pushSecretList.Items {
		if ps.Spec.DeletionPolicy == esv1alpha1.PushSecretDeletionPolicyDelete {
			// Verify the store reference matches
			storeKey := fmt.Sprintf("%s/%s", esapi.SecretStoreKind, storeName)
			if _, hasPushed := ps.Status.SyncedPushSecrets[storeKey]; hasPushed {
				return true, nil
			}
		}
	}

	// Also check for PushSecrets that reference this store by name or labelSelector
	// but haven't pushed to it yet (for initial finalizer setup)
	if err := r.List(ctx, &pushSecretList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.deletionPolicy", string(esv1alpha1.PushSecretDeletionPolicyDelete)),
		Namespace:     storeNamespace,
	}); err != nil {
		return false, fmt.Errorf("failed to list PushSecrets by deletionPolicy: %w", err)
	}

	for _, ps := range pushSecretList.Items {
		// Check if this PushSecret references our store
		for _, storeRef := range ps.Spec.SecretStoreRefs {
			if storeRef.Name == storeName {
				// Check if kind matches (default is SecretStore)
				if storeRef.Kind == "" || storeRef.Kind == esapi.SecretStoreKind {
					return true, nil
				}
			}
		}

		// Check labelSelector match
		for _, storeRef := range ps.Spec.SecretStoreRefs {
			if storeRef.LabelSelector != nil && (storeRef.Kind == "" || storeRef.Kind == esapi.SecretStoreKind) {
				selector, err := metav1.LabelSelectorAsSelector(storeRef.LabelSelector)
				if err != nil {
					continue
				}
				if selector.Matches(labels.Set(ss.GetLabels())) {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *StoreReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("secret-store")

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esapi.SecretStore{}).
		Watches(
			&esv1alpha1.PushSecret{},
			handler.EnqueueRequestsFromMapFunc(r.findSecretStoresForPushSecret),
			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
				// Only watch PushSecrets with DeletionPolicy=Delete
				ps := obj.(*esv1alpha1.PushSecret)
				return ps.Spec.DeletionPolicy == esv1alpha1.PushSecretDeletionPolicyDelete
			})),
		).
		Complete(r)
}

// findSecretStoresForPushSecret finds SecretStores that should be reconciled when a PushSecret changes.
func (r *StoreReconciler) findSecretStoresForPushSecret(ctx context.Context, obj client.Object) []ctrlreconcile.Request {
	ps := obj.(*esv1alpha1.PushSecret)
	var requests []ctrlreconcile.Request

	// Get all SecretStores in the same namespace as the PushSecret
	var secretStoreList esapi.SecretStoreList
	if err := r.List(ctx, &secretStoreList, &client.ListOptions{Namespace: ps.Namespace}); err != nil {
		return requests
	}

	// Check which SecretStores this PushSecret references
	for _, store := range secretStoreList.Items {
		if shouldReconcileSecretStoreForPushSecret(&store, ps) {
			requests = append(requests, ctrlreconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      store.Name,
					Namespace: store.Namespace,
				},
			})
		}
	}

	return requests
}

// shouldReconcileSecretStoreForPushSecret determines if a SecretStore should be reconciled
// when a PushSecret changes, based on whether the PushSecret references this store.
func shouldReconcileSecretStoreForPushSecret(store *esapi.SecretStore, ps *esv1alpha1.PushSecret) bool {
	// Check if this PushSecret has pushed to this store
	storeKey := fmt.Sprintf("%s/%s", esapi.SecretStoreKind, store.Name)
	_, hasPushed := ps.Status.SyncedPushSecrets[storeKey]

	// Also check if the PushSecret references this store in its spec
	for _, storeRef := range ps.Spec.SecretStoreRefs {
		if storeRef.Name == store.Name && (storeRef.Kind == "" || storeRef.Kind == esapi.SecretStoreKind) {
			return true
		}
		// Check labelSelector match
		if storeRef.LabelSelector != nil && storeRef.Kind == esapi.SecretStoreKind {
			selector, err := metav1.LabelSelectorAsSelector(storeRef.LabelSelector)
			if err == nil && selector.Matches(labels.Set(store.Labels)) {
				return true
			}
		}
	}

	return hasPushed
}
