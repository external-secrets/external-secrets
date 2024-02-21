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

package clusterexternalsecret

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/clusterexternalsecret/cesmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

// ClusterExternalSecretReconciler reconciles a ClusterExternalSecret object.
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
	errNamespaces           = "could not get namespaces from selector"
	errGetExistingES        = "could not get existing ExternalSecret"
	errNamespacesFailed     = "one or more namespaces failed"
)

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ClusterExternalSecret", req.NamespacedName)

	resourceLabels := ctrlmetrics.RefineNonConditionMetricLabels(map[string]string{"name": req.Name, "namespace": req.Namespace})
	start := time.Now()

	externalSecretReconcileDuration := cesmetrics.GetGaugeVec(cesmetrics.ClusterExternalSecretReconcileDurationKey)
	defer func() { externalSecretReconcileDuration.With(resourceLabels).Set(float64(time.Since(start))) }()

	var clusterExternalSecret esv1beta1.ClusterExternalSecret
	err := r.Get(ctx, req.NamespacedName, &clusterExternalSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			cesmetrics.RemoveMetrics(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}

		log.Error(err, errGetCES)
		return ctrl.Result{}, err
	}

	// skip reconciliation if deletion timestamp is set on cluster external secret
	if clusterExternalSecret.DeletionTimestamp != nil {
		log.Info("skipping as it is in deletion")
		return ctrl.Result{}, nil
	}

	p := client.MergeFrom(clusterExternalSecret.DeepCopy())
	defer r.deferPatch(ctx, log, &clusterExternalSecret, p)

	refreshInt := r.RequeueInterval
	if clusterExternalSecret.Spec.RefreshInterval != nil {
		refreshInt = clusterExternalSecret.Spec.RefreshInterval.Duration
	}

	namespaceList := v1.NamespaceList{}

	if clusterExternalSecret.Spec.NamespaceSelector != nil {
		labelSelector, err := metav1.LabelSelectorAsSelector(clusterExternalSecret.Spec.NamespaceSelector)
		if err != nil {
			log.Error(err, errConvertLabelSelector)
			return ctrl.Result{}, err
		}

		err = r.List(ctx, &namespaceList, &client.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			log.Error(err, errNamespaces)
			return ctrl.Result{}, err
		}
	}

	if len(clusterExternalSecret.Spec.Namespaces) > 0 {
		var additionalNamespace []v1.Namespace

		for _, ns := range clusterExternalSecret.Spec.Namespaces {
			namespace := &v1.Namespace{}
			if err = r.Get(ctx, types.NamespacedName{Name: ns}, namespace); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}

				log.Error(err, errNamespaces)
				return ctrl.Result{}, err
			}

			additionalNamespace = append(additionalNamespace, *namespace)
		}

		namespaceList.Items = append(namespaceList.Items, additionalNamespace...)
	}

	esName := clusterExternalSecret.Spec.ExternalSecretName
	if esName == "" {
		esName = clusterExternalSecret.ObjectMeta.Name
	}

	if prevName := clusterExternalSecret.Status.ExternalSecretName; prevName != esName {
		// ExternalSecretName has changed, so remove the old ones
		for _, ns := range clusterExternalSecret.Status.ProvisionedNamespaces {
			if err := r.deleteExternalSecret(ctx, prevName, clusterExternalSecret.Name, ns); err != nil {
				log.Error(err, "could not delete ExternalSecret")
				return ctrl.Result{}, err
			}
		}
	}

	clusterExternalSecret.Status.ExternalSecretName = esName

	failedNamespaces := r.deleteOutdatedExternalSecrets(ctx, namespaceList, esName, clusterExternalSecret.Name, clusterExternalSecret.Status.ProvisionedNamespaces)

	provisionedNamespaces := []string{}
	for _, namespace := range namespaceList.Items {
		var existingES esv1beta1.ExternalSecret
		err = r.Get(ctx, types.NamespacedName{
			Name:      esName,
			Namespace: namespace.Name,
		}, &existingES)
		if err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, errGetExistingES)
			failedNamespaces[namespace.Name] = err
			continue
		}

		if err == nil && !isExternalSecretOwnedBy(&existingES, clusterExternalSecret.Name) {
			failedNamespaces[namespace.Name] = fmt.Errorf("external secret already exists in namespace")
			continue
		}

		if err := r.createOrUpdateExternalSecret(ctx, &clusterExternalSecret, namespace, esName, clusterExternalSecret.Spec.ExternalSecretMetadata); err != nil {
			log.Error(err, "failed to create or update external secret")
			failedNamespaces[namespace.Name] = err
			continue
		}

		provisionedNamespaces = append(provisionedNamespaces, namespace.Name)
	}

	condition := NewClusterExternalSecretCondition(failedNamespaces)
	SetClusterExternalSecretCondition(&clusterExternalSecret, *condition)

	clusterExternalSecret.Status.FailedNamespaces = toNamespaceFailures(failedNamespaces)
	sort.Strings(provisionedNamespaces)
	clusterExternalSecret.Status.ProvisionedNamespaces = provisionedNamespaces

	return ctrl.Result{RequeueAfter: refreshInt}, nil
}

func (r *Reconciler) createOrUpdateExternalSecret(ctx context.Context, clusterExternalSecret *esv1beta1.ClusterExternalSecret, namespace v1.Namespace, esName string, esMetadata esv1beta1.ExternalSecretMetadata) error {
	externalSecret := &esv1beta1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace.Name,
			Name:      esName,
		},
	}

	mutateFunc := func() error {
		externalSecret.Labels = esMetadata.Labels
		externalSecret.Annotations = esMetadata.Annotations
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

func (r *Reconciler) deleteExternalSecret(ctx context.Context, esName, cesName, namespace string) error {
	var existingES esv1beta1.ExternalSecret
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

func (r *Reconciler) deferPatch(ctx context.Context, log logr.Logger, clusterExternalSecret *esv1beta1.ClusterExternalSecret, p client.Patch) {
	if err := r.Status().Patch(ctx, clusterExternalSecret, p); err != nil {
		log.Error(err, errPatchStatus)
	}
}

func (r *Reconciler) deleteOutdatedExternalSecrets(ctx context.Context, namespaceList v1.NamespaceList, esName, cesName string, provisionedNamespaces []string) map[string]error {
	failedNamespaces := map[string]error{}
	// Loop through existing namespaces first to make sure they still have our labels
	for _, namespace := range getRemovedNamespaces(namespaceList, provisionedNamespaces) {
		err := r.deleteExternalSecret(ctx, esName, cesName, namespace)
		if err != nil {
			r.Log.Error(err, "unable to delete external secret")
			failedNamespaces[namespace] = err
		}
	}

	return failedNamespaces
}

func isExternalSecretOwnedBy(es *esv1beta1.ExternalSecret, cesName string) bool {
	owner := metav1.GetControllerOf(es)
	return owner != nil && owner.APIVersion == esv1beta1.SchemeGroupVersion.String() && owner.Kind == esv1beta1.ClusterExtSecretKind && owner.Name == cesName
}

func getRemovedNamespaces(currentNSs v1.NamespaceList, provisionedNSs []string) []string {
	currentNSSet := map[string]struct{}{}
	for i := range currentNSs.Items {
		currentNSSet[currentNSs.Items[i].Name] = struct{}{}
	}

	var removedNSs []string
	for _, ns := range provisionedNSs {
		if _, ok := currentNSSet[ns]; !ok {
			removedNSs = append(removedNSs, ns)
		}
	}

	return removedNSs
}

func toNamespaceFailures(failedNamespaces map[string]error) []esv1beta1.ClusterExternalSecretNamespaceFailure {
	namespaceFailures := make([]esv1beta1.ClusterExternalSecretNamespaceFailure, len(failedNamespaces))

	i := 0
	for namespace, err := range failedNamespaces {
		namespaceFailures[i] = esv1beta1.ClusterExternalSecretNamespaceFailure{
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
		For(&esv1beta1.ClusterExternalSecret{}).
		Owns(&esv1beta1.ExternalSecret{}).
		Watches(
			&v1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForNamespace),
			builder.WithPredicates(namespacePredicate()),
		).
		Complete(r)
}

func (r *Reconciler) findObjectsForNamespace(ctx context.Context, namespace client.Object) []reconcile.Request {
	var clusterExternalSecrets esv1beta1.ClusterExternalSecretList
	if err := r.List(ctx, &clusterExternalSecrets); err != nil {
		r.Log.Error(err, errGetCES)
		return []reconcile.Request{}
	}

	var requests []reconcile.Request
	for i := range clusterExternalSecrets.Items {
		clusterExternalSecret := &clusterExternalSecrets.Items[i]
		if clusterExternalSecret.Spec.NamespaceSelector != nil {
			labelSelector, err := metav1.LabelSelectorAsSelector(clusterExternalSecret.Spec.NamespaceSelector)
			if err != nil {
				r.Log.Error(err, errConvertLabelSelector)
				return []reconcile.Request{}
			}

			if labelSelector.Matches(labels.Set(namespace.GetLabels())) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      clusterExternalSecret.GetName(),
						Namespace: clusterExternalSecret.GetNamespace(),
					},
				})

				// Prevent the object from being added twice if it happens to be listed
				// by Namespaces selector as well.
				continue
			}
		}

		if len(clusterExternalSecret.Spec.Namespaces) > 0 {
			if slices.Contains(clusterExternalSecret.Spec.Namespaces, namespace.GetName()) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      clusterExternalSecret.GetName(),
						Namespace: clusterExternalSecret.GetNamespace(),
					},
				})
			}
		}
	}

	return requests
}

func namespacePredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}
			return !reflect.DeepEqual(e.ObjectOld.GetLabels(), e.ObjectNew.GetLabels())
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return true
		},
	}
}
