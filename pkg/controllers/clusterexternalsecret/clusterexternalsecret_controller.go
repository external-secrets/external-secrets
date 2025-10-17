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

// Package clusterexternalsecret implements a controller for managing ClusterExternalSecret resources,
// which allow creating ExternalSecrets across multiple namespaces.
package clusterexternalsecret

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/controllers/clusterexternalsecret/cesmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/esutils"
)

// Reconciler reconciles a ClusterExternalSecret object.
type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	RequeueInterval time.Duration
}

const (
	errGetCES               = "could not get ClusterExternalSecret"
	errPatchStatus          = "unable to patch status"
	errConvertLabelSelector = "unable to convert labelselector"
	errGetExistingES        = "could not get existing ExternalSecret"
	errNamespacesFailed     = "one or more namespaces failed"

	// ClusterExternalSecretFinalizer is the finalizer for ClusterExternalSecret resources.
	// This finalizer ensures that all ExternalSecrets created by the ClusterExternalSecret
	// are properly cleaned up before the ClusterExternalSecret is deleted, preventing orphaned resources.
	ClusterExternalSecretFinalizer = "externalsecrets.external-secrets.io/clusterexternalsecret-cleanup"
)

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ClusterExternalSecret", req.NamespacedName)

	resourceLabels := ctrlmetrics.RefineNonConditionMetricLabels(map[string]string{"name": req.Name, "namespace": req.Namespace})
	start := time.Now()

	externalSecretReconcileDuration := cesmetrics.GetGaugeVec(cesmetrics.ClusterExternalSecretReconcileDurationKey)
	defer func() { externalSecretReconcileDuration.With(resourceLabels).Set(float64(time.Since(start))) }()

	var clusterExternalSecret esv1.ClusterExternalSecret
	err := r.Get(ctx, req.NamespacedName, &clusterExternalSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			cesmetrics.RemoveMetrics(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}

		log.Error(err, errGetCES)
		return ctrl.Result{}, err
	}

	// Handle deletion with finalizer
	// When a ClusterExternalSecret is being deleted, we need to ensure all created ExternalSecrets
	// and namespace finalizers are cleaned up to prevent resource leaks and namespace deletion blocking.
	if clusterExternalSecret.DeletionTimestamp != nil {
		// Always attempt cleanup to handle edge case where finalizer might be removed externally
		if err := r.cleanupExternalSecrets(ctx, log, &clusterExternalSecret); err != nil {
			log.Error(err, "failed to cleanup ExternalSecrets")
			return ctrl.Result{}, err
		}

		// Remove finalizer from ClusterExternalSecret if it exists
		if updated := controllerutil.RemoveFinalizer(&clusterExternalSecret, ClusterExternalSecretFinalizer); updated {
			if err := r.Update(ctx, &clusterExternalSecret); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if it doesn't exist
	// This ensures the ClusterExternalSecret cannot be deleted until we've cleaned up all
	// ExternalSecrets it created and removed our finalizers from namespaces.
	if updated := controllerutil.AddFinalizer(&clusterExternalSecret, ClusterExternalSecretFinalizer); updated {
		if err := r.Update(ctx, &clusterExternalSecret); err != nil {
			return ctrl.Result{}, err
		}
		// Return immediately after update to let the change propagate
		return ctrl.Result{}, nil
	}

	p := client.MergeFrom(clusterExternalSecret.DeepCopy())
	defer r.deferPatch(ctx, log, &clusterExternalSecret, p)

	return r.reconcile(ctx, log, &clusterExternalSecret)
}

func (r *Reconciler) reconcile(ctx context.Context, log logr.Logger, clusterExternalSecret *esv1.ClusterExternalSecret) (ctrl.Result, error) {
	refreshInt := r.RequeueInterval
	if clusterExternalSecret.Spec.RefreshInterval != nil {
		refreshInt = clusterExternalSecret.Spec.RefreshInterval.Duration
	}

	esName := clusterExternalSecret.Spec.ExternalSecretName
	if esName == "" {
		esName = clusterExternalSecret.ObjectMeta.Name
	}
	if prevName := clusterExternalSecret.Status.ExternalSecretName; prevName != "" && prevName != esName {
		// ExternalSecretName has changed, so remove the old ones
		if err := r.removeOldSecrets(ctx, log, clusterExternalSecret, prevName); err != nil {
			return ctrl.Result{}, err
		}
	}
	clusterExternalSecret.Status.ExternalSecretName = esName

	selectors := []*metav1.LabelSelector{}
	if s := clusterExternalSecret.Spec.NamespaceSelector; s != nil {
		selectors = append(selectors, s)
	}
	selectors = append(selectors, clusterExternalSecret.Spec.NamespaceSelectors...)

	namespaces, err := esutils.GetTargetNamespaces(ctx, r.Client, clusterExternalSecret.Spec.Namespaces, selectors)
	if err != nil {
		log.Error(err, "failed to get target Namespaces")
		failedNamespaces := map[string]error{
			"unknown": err,
		}
		condition := NewClusterExternalSecretCondition(failedNamespaces)
		SetClusterExternalSecretCondition(clusterExternalSecret, *condition)

		clusterExternalSecret.Status.FailedNamespaces = toNamespaceFailures(failedNamespaces)

		return ctrl.Result{}, err
	}

	failedNamespaces := r.deleteOutdatedExternalSecrets(ctx, namespaces, esName, clusterExternalSecret.Name, clusterExternalSecret.Status.ProvisionedNamespaces)

	provisionedNamespaces := r.gatherProvisionedNamespaces(ctx, log, clusterExternalSecret, namespaces, esName, failedNamespaces)

	condition := NewClusterExternalSecretCondition(failedNamespaces)
	SetClusterExternalSecretCondition(clusterExternalSecret, *condition)

	clusterExternalSecret.Status.FailedNamespaces = toNamespaceFailures(failedNamespaces)
	sort.Strings(provisionedNamespaces)
	clusterExternalSecret.Status.ProvisionedNamespaces = provisionedNamespaces

	// Check if any failures are due to conflicts - if so, requeue immediately
	for _, err := range failedNamespaces {
		if apierrors.IsConflict(err) {
			log.V(1).Info("conflict detected, requeuing immediately")
			return ctrl.Result{}, fmt.Errorf("conflict detected, will retry: %w", err)
		}
	}

	return ctrl.Result{RequeueAfter: refreshInt}, nil
}

func (r *Reconciler) gatherProvisionedNamespaces(
	ctx context.Context,
	log logr.Logger,
	clusterExternalSecret *esv1.ClusterExternalSecret,
	namespaces []v1.Namespace,
	esName string,
	failedNamespaces map[string]error,
) []string {
	var provisionedNamespaces []string //nolint:prealloc // we don't know the size
	for _, namespace := range namespaces {
		// If namespace is being deleted, remove our finalizer to allow deletion to proceed
		if namespace.DeletionTimestamp != nil {
			log.Info("namespace is being deleted, removing finalizer", "namespace", namespace.Name)
			if err := r.removeNamespaceFinalizer(ctx, log, &namespace, clusterExternalSecret.Name); err != nil {
				log.Error(err, "failed to remove finalizer from terminating namespace", "namespace", namespace.Name)
				// Don't add to failedNamespaces - this is cleanup, not provisioning
			}
			continue
		}
		var existingES esv1.ExternalSecret
		err := r.Get(ctx, types.NamespacedName{
			Name:      esName,
			Namespace: namespace.Name,
		}, &existingES)
		if err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, errGetExistingES)
			failedNamespaces[namespace.Name] = err
			continue
		}

		if err == nil && !isExternalSecretOwnedBy(&existingES, clusterExternalSecret.Name) {
			failedNamespaces[namespace.Name] = errors.New("external secret already exists in namespace")
			continue
		}

		if err := r.createOrUpdateExternalSecret(ctx, clusterExternalSecret, namespace, esName, clusterExternalSecret.Spec.ExternalSecretMetadata); err != nil {
			// If conflict, don't log as error - just add to failed namespaces for retry
			if apierrors.IsConflict(err) {
				log.V(1).Info("conflict while updating namespace, will retry", "namespace", namespace.Name)
				failedNamespaces[namespace.Name] = err
				continue
			}
			log.Error(err, "failed to create or update external secret")
			failedNamespaces[namespace.Name] = err
			continue
		}

		provisionedNamespaces = append(provisionedNamespaces, namespace.Name)
	}
	return provisionedNamespaces
}

func (r *Reconciler) removeOldSecrets(ctx context.Context, log logr.Logger, clusterExternalSecret *esv1.ClusterExternalSecret, prevName string) error {
	var (
		failedNamespaces = map[string]error{}
		lastErr          error
	)
	for _, ns := range clusterExternalSecret.Status.ProvisionedNamespaces {
		if err := r.deleteExternalSecret(ctx, prevName, clusterExternalSecret.Name, ns); err != nil {
			log.Error(err, "could not delete ExternalSecret")
			failedNamespaces[ns] = err
			lastErr = err
		}
	}
	if len(failedNamespaces) > 0 {
		condition := NewClusterExternalSecretCondition(failedNamespaces)
		SetClusterExternalSecretCondition(clusterExternalSecret, *condition)
		clusterExternalSecret.Status.FailedNamespaces = toNamespaceFailures(failedNamespaces)
		return lastErr
	}

	return nil
}

// cleanupExternalSecrets removes all ExternalSecrets created by this ClusterExternalSecret
// and removes the namespace finalizers we added. This uses a dual-finalizer strategy:
// 1. ClusterExternalSecret has a finalizer to ensure cleanup happens
// 2. Each namespace gets a CES-specific finalizer to prevent deletion race conditions
// The namespace finalizer is named with the CES name to allow multiple CES resources
// to provision into the same namespace independently.
func (r *Reconciler) cleanupExternalSecrets(ctx context.Context, log logr.Logger, clusterExternalSecret *esv1.ClusterExternalSecret) error {
	esName := r.getExternalSecretName(clusterExternalSecret)

	var err error
	for _, ns := range clusterExternalSecret.Status.ProvisionedNamespaces {
		if cleanupErr := r.cleanupNamespaceResources(ctx, log, clusterExternalSecret, ns, esName); cleanupErr != nil {
			err = errors.Join(err, cleanupErr)
		}
	}

	return err
}

// getExternalSecretName returns the name to use for ExternalSecrets.
func (r *Reconciler) getExternalSecretName(clusterExternalSecret *esv1.ClusterExternalSecret) string {
	if clusterExternalSecret.Status.ExternalSecretName != "" {
		return clusterExternalSecret.Status.ExternalSecretName
	}
	return clusterExternalSecret.ObjectMeta.Name
}

// cleanupNamespaceResources handles cleanup for a single namespace.
func (r *Reconciler) cleanupNamespaceResources(ctx context.Context, log logr.Logger, clusterExternalSecret *esv1.ClusterExternalSecret, namespaceName, esName string) error {
	// Get the namespace
	namespace, err := r.getNamespace(ctx, namespaceName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Namespace already deleted, that's ok
			return nil
		}
		return fmt.Errorf("failed to get namespace %s: %w", namespaceName, err)
	}

	// Remove namespace finalizer
	if err := r.removeNamespaceFinalizer(ctx, log, namespace, clusterExternalSecret.Name); err != nil {
		return err
	}

	// Delete ExternalSecret
	if err := r.deleteExternalSecret(ctx, esName, clusterExternalSecret.Name, namespaceName); err != nil {
		return fmt.Errorf("failed to delete ExternalSecret in namespace %s: %w", namespaceName, err)
	}

	return nil
}

// getNamespace fetches a namespace by name.
func (r *Reconciler) getNamespace(ctx context.Context, name string) (*v1.Namespace, error) {
	var namespace v1.Namespace
	err := r.Get(ctx, types.NamespacedName{Name: name}, &namespace)
	return &namespace, err
}

// removeNamespaceFinalizer removes the CES-specific finalizer from a namespace.
func (r *Reconciler) removeNamespaceFinalizer(ctx context.Context, log logr.Logger, namespace *v1.Namespace, cesName string) error {
	finalizer := r.buildCESFinalizer(cesName)

	if !controllerutil.ContainsFinalizer(namespace, finalizer) {
		return nil // Finalizer doesn't exist, nothing to do
	}

	return r.updateNamespaceRemoveFinalizer(ctx, log, namespace.Name, finalizer)
}

// buildCESFinalizer creates the finalizer name for a CES.
func (r *Reconciler) buildCESFinalizer(cesName string) string {
	return "externalsecrets.external-secrets.io/ces-" + cesName
}

// updateNamespaceRemoveFinalizer removes a finalizer from a namespace with conflict handling.
func (r *Reconciler) updateNamespaceRemoveFinalizer(ctx context.Context, log logr.Logger, namespaceName, finalizer string) error {
	// Fetch the latest namespace to avoid conflicts
	namespace, err := r.getNamespace(ctx, namespaceName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil // Namespace deleted, that's OK
		}
		return fmt.Errorf("failed to get latest namespace %s: %w", namespaceName, err)
	}

	// Only update if the finalizer was actually removed
	if updated := controllerutil.RemoveFinalizer(namespace, finalizer); updated {
		if err := r.Update(ctx, namespace); err != nil {
			// Ignore NotFound (namespace deleted)
			if apierrors.IsNotFound(err) {
				log.V(1).Info("ignoring expected error during finalizer removal",
					"namespace", namespaceName,
					"error", err.Error())
				return nil
			}

			return fmt.Errorf("failed to remove finalizer from namespace %s: %w", namespaceName, err)
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdateExternalSecret(ctx context.Context, clusterExternalSecret *esv1.ClusterExternalSecret, namespace v1.Namespace, esName string, esMetadata esv1.ExternalSecretMetadata) error {
	// Add namespace finalizer first to prevent deletion race conditions
	if err := r.ensureNamespaceFinalizer(ctx, &namespace, clusterExternalSecret.Name); err != nil {
		return err
	}

	// Create or update the ExternalSecret
	externalSecret := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace.Name,
			Name:      esName,
		},
	}

	mutateFunc := func() error {
		externalSecret.Labels = esMetadata.Labels
		externalSecret.Annotations = esMetadata.Annotations
		if value, ok := clusterExternalSecret.Annotations[esv1.AnnotationForceSync]; ok {
			if externalSecret.Annotations == nil {
				externalSecret.Annotations = map[string]string{}
			}
			externalSecret.Annotations[esv1.AnnotationForceSync] = value
		} else {
			delete(externalSecret.Annotations, esv1.AnnotationForceSync)
		}

		externalSecret.Spec = clusterExternalSecret.Spec.ExternalSecretSpec

		if err := controllerutil.SetControllerReference(clusterExternalSecret, externalSecret, r.Scheme); err != nil {
			return fmt.Errorf("could not set the controller owner reference %w", err)
		}

		return nil
	}

	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, externalSecret, mutateFunc); err != nil {
		return fmt.Errorf("could not create or update ExternalSecret: %w", err)
	}

	return nil
}

// ensureNamespaceFinalizer adds a CES-specific finalizer to the namespace if it doesn't exist.
// This prevents the namespace from being deleted while we're managing ExternalSecrets in it.
func (r *Reconciler) ensureNamespaceFinalizer(ctx context.Context, namespace *v1.Namespace, cesName string) error {
	finalizer := r.buildCESFinalizer(cesName)

	if controllerutil.ContainsFinalizer(namespace, finalizer) {
		return nil // Already has finalizer
	}

	return r.addNamespaceFinalizer(ctx, namespace.Name, finalizer)
}

// addNamespaceFinalizer adds a finalizer to a namespace with conflict handling.
func (r *Reconciler) addNamespaceFinalizer(ctx context.Context, namespaceName, finalizer string) error {
	// Fetch the latest namespace to avoid conflicts
	namespace, err := r.getNamespace(ctx, namespaceName)
	if err != nil {
		return fmt.Errorf("could not get latest namespace: %w", err)
	}

	// Only update if the finalizer was actually added
	if updated := controllerutil.AddFinalizer(namespace, finalizer); updated {
		if err := r.Update(ctx, namespace); err != nil {
			// If conflict, return error to trigger requeue
			if apierrors.IsConflict(err) {
				return err
			}
			return fmt.Errorf("could not add namespace finalizer: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) deleteExternalSecret(ctx context.Context, esName, cesName, namespace string) error {
	var existingES esv1.ExternalSecret
	err := r.Get(ctx, types.NamespacedName{
		Name:      esName,
		Namespace: namespace,
	}, &existingES)
	if err != nil {
		// If we can't find it then just leave
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if !isExternalSecretOwnedBy(&existingES, cesName) {
		return nil
	}

	err = r.Delete(ctx, &existingES, &client.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("external secret in non matching namespace could not be deleted: %w", err)
	}

	return nil
}

func (r *Reconciler) deferPatch(ctx context.Context, log logr.Logger, clusterExternalSecret *esv1.ClusterExternalSecret, p client.Patch) {
	if err := r.Status().Patch(ctx, clusterExternalSecret, p); err != nil {
		log.Error(err, errPatchStatus)
	}
}

func (r *Reconciler) deleteOutdatedExternalSecrets(ctx context.Context, namespaces []v1.Namespace, esName, cesName string, provisionedNamespaces []string) map[string]error {
	failedNamespaces := map[string]error{}
	// Loop through existing namespaces first to make sure they still have our labels
	for _, namespace := range getRemovedNamespaces(namespaces, provisionedNamespaces) {
		err := r.deleteExternalSecret(ctx, esName, cesName, namespace)
		if err != nil {
			r.Log.Error(err, "unable to delete external secret")
			failedNamespaces[namespace] = err
		}
	}

	return failedNamespaces
}

func isExternalSecretOwnedBy(es *esv1.ExternalSecret, cesName string) bool {
	owner := metav1.GetControllerOf(es)
	return owner != nil && schema.FromAPIVersionAndKind(owner.APIVersion, owner.Kind).GroupKind().String() == esv1.ClusterExtSecretGroupKind && owner.Name == cesName
}

func getRemovedNamespaces(currentNSs []v1.Namespace, provisionedNSs []string) []string {
	currentNSSet := map[string]struct{}{}
	for _, currentNs := range currentNSs {
		currentNSSet[currentNs.Name] = struct{}{}
	}

	var removedNSs []string
	for _, ns := range provisionedNSs {
		if _, ok := currentNSSet[ns]; !ok {
			removedNSs = append(removedNSs, ns)
		}
	}

	return removedNSs
}

func toNamespaceFailures(failedNamespaces map[string]error) []esv1.ClusterExternalSecretNamespaceFailure {
	namespaceFailures := make([]esv1.ClusterExternalSecretNamespaceFailure, len(failedNamespaces))

	i := 0
	for namespace, err := range failedNamespaces {
		namespaceFailures[i] = esv1.ClusterExternalSecretNamespaceFailure{
			Namespace: namespace,
			Reason:    err.Error(),
		}
		i++
	}
	sort.Slice(namespaceFailures, func(i, j int) bool { return namespaceFailures[i].Namespace < namespaceFailures[j].Namespace })
	return namespaceFailures
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esv1.ClusterExternalSecret{}).
		Owns(&esv1.ExternalSecret{}).
		Watches(
			&v1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForNamespace),
			builder.WithPredicates(esutils.NamespacePredicate()),
		).
		Complete(r)
}

func (r *Reconciler) findObjectsForNamespace(ctx context.Context, namespace client.Object) []reconcile.Request {
	var clusterExternalSecrets esv1.ClusterExternalSecretList
	if err := r.List(ctx, &clusterExternalSecrets); err != nil {
		r.Log.Error(err, errGetCES)
		return []reconcile.Request{}
	}

	return r.queueRequestsForItem(&clusterExternalSecrets, namespace)
}

func (r *Reconciler) queueRequestsForItem(clusterExternalSecrets *esv1.ClusterExternalSecretList, namespace client.Object) []reconcile.Request {
	var requests []reconcile.Request
	for i := range clusterExternalSecrets.Items {
		clusterExternalSecret := clusterExternalSecrets.Items[i]
		var selectors []*metav1.LabelSelector
		if s := clusterExternalSecret.Spec.NamespaceSelector; s != nil {
			selectors = append(selectors, s)
		}
		selectors = append(selectors, clusterExternalSecret.Spec.NamespaceSelectors...)

		var selected bool
		for _, selector := range selectors {
			labelSelector, err := metav1.LabelSelectorAsSelector(selector)
			if err != nil {
				r.Log.Error(err, errConvertLabelSelector)
				continue
			}

			if labelSelector.Matches(labels.Set(namespace.GetLabels())) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      clusterExternalSecret.GetName(),
						Namespace: clusterExternalSecret.GetNamespace(),
					},
				})
				selected = true
				break
			}
		}

		// Prevent the object from being added twice if it happens to be listed
		// by Namespaces selector as well.
		if selected {
			continue
		}

		if slices.Contains(clusterExternalSecret.Spec.Namespaces, namespace.GetName()) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      clusterExternalSecret.GetName(),
					Namespace: clusterExternalSecret.GetNamespace(),
				},
			})
		}
	}

	return requests
}
