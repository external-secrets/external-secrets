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

package clusterpushsecret

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/clusterpushsecret/cpsmetrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/pushsecret"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// Reconciler reconciles a ClusterPushSecret object.
type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	RequeueInterval time.Duration
	Recorder        record.EventRecorder
}

const (
	errPatchStatus          = "error merging"
	errGetCES               = "could not get ClusterPushSecret"
	errConvertLabelSelector = "unable to convert label selector"
	errGetExistingPS        = "could not get existing PushSecret"
	errNamespacesFailed     = "one or more namespaces failed"
)

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ClusterPushSecret", req.NamespacedName)

	var cps v1alpha1.ClusterPushSecret
	err := r.Get(ctx, req.NamespacedName, &cps)
	if err != nil {
		if apierrors.IsNotFound(err) {
			cpsmetrics.RemoveMetrics(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}

		log.Error(err, errGetCES)
		return ctrl.Result{}, err
	}

	// skip reconciliation if deletion timestamp is set on cluster external secret
	if cps.DeletionTimestamp != nil {
		log.Info("skipping as it is in deletion")
		return ctrl.Result{}, nil
	}

	p := client.MergeFrom(cps.DeepCopy())
	defer r.deferPatch(ctx, log, &cps, p)

	refreshInt := r.RequeueInterval
	if cps.Spec.RefreshInterval != nil {
		refreshInt = cps.Spec.RefreshInterval.Duration
	}

	esName := cps.Spec.PushSecretName
	if esName == "" {
		esName = cps.ObjectMeta.Name
	}

	if err := r.deleteOldPushSecrets(ctx, &cps, esName, log); err != nil {
		return ctrl.Result{}, err
	}

	cps.Status.PushSecretName = esName

	namespaces, err := utils.GetTargetNamespaces(ctx, r.Client, nil, cps.Spec.NamespaceSelectors)
	if err != nil {
		log.Error(err, "failed to get target Namespaces")
		r.markAsFailed("failed to get target Namespaces", &cps)
		return ctrl.Result{}, err
	}

	failedNamespaces := r.deleteOutdatedPushSecrets(ctx, namespaces, esName, cps.Name, cps.Status.ProvisionedNamespaces)
	provisionedNamespaces := r.updateProvisionedNamespaces(ctx, namespaces, esName, log, failedNamespaces, &cps)

	condition := NewClusterPushSecretCondition(failedNamespaces)
	SetClusterPushSecretCondition(&cps, *condition)

	cps.Status.FailedNamespaces = toNamespaceFailures(failedNamespaces)
	sort.Strings(provisionedNamespaces)
	cps.Status.ProvisionedNamespaces = provisionedNamespaces

	return ctrl.Result{RequeueAfter: refreshInt}, nil
}

func (r *Reconciler) updateProvisionedNamespaces(
	ctx context.Context,
	namespaces []v1.Namespace,
	esName string,
	log logr.Logger,
	failedNamespaces map[string]error,
	cps *v1alpha1.ClusterPushSecret,
) []string {
	var provisionedNamespaces []string //nolint:prealloc // I have no idea what the size will be.
	for _, namespace := range namespaces {
		var pushSecret v1alpha1.PushSecret
		err := r.Get(ctx, types.NamespacedName{
			Name:      esName,
			Namespace: namespace.Name,
		}, &pushSecret)
		if err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, errGetExistingPS)
			failedNamespaces[namespace.Name] = err
			continue
		}

		if err == nil && !isPushSecretOwnedBy(&pushSecret, cps.Name) {
			failedNamespaces[namespace.Name] = errors.New("push secret already exists in namespace")
			continue
		}

		if err := r.createOrUpdatePushSecret(ctx, cps, namespace, esName, cps.Spec.PushSecretMetadata); err != nil {
			log.Error(err, "failed to create or update push secret")
			failedNamespaces[namespace.Name] = err
			continue
		}

		provisionedNamespaces = append(provisionedNamespaces, namespace.Name)
	}

	return provisionedNamespaces
}

func (r *Reconciler) deleteOldPushSecrets(ctx context.Context, cps *v1alpha1.ClusterPushSecret, esName string, log logr.Logger) error {
	var lastErr error
	if prevName := cps.Status.PushSecretName; prevName != esName {
		// PushSecretName has changed, so remove the old ones
		failedNamespaces := map[string]error{}
		for _, ns := range cps.Status.ProvisionedNamespaces {
			if err := r.deletePushSecret(ctx, prevName, cps.Name, ns); err != nil {
				log.Error(err, "could not delete PushSecret")
				failedNamespaces[ns] = err
				lastErr = err
			}
		}

		if len(failedNamespaces) > 0 {
			r.markAsFailed("failed to delete push secret", cps)
			cps.Status.FailedNamespaces = toNamespaceFailures(failedNamespaces)
			return lastErr
		}
	}

	return nil
}

func (r *Reconciler) markAsFailed(msg string, ps *v1alpha1.ClusterPushSecret) {
	cond := pushsecret.NewPushSecretCondition(v1alpha1.PushSecretReady, v1.ConditionFalse, v1alpha1.ReasonErrored, msg)
	setClusterPushSecretCondition(ps, *cond)
	r.Recorder.Event(ps, v1.EventTypeWarning, v1alpha1.ReasonErrored, msg)
}

func setClusterPushSecretCondition(ps *v1alpha1.ClusterPushSecret, condition v1alpha1.PushSecretStatusCondition) {
	currentCond := pushsecret.GetPushSecretCondition(ps.Status.Conditions, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status &&
		currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	ps.Status.Conditions = append(pushsecret.FilterOutCondition(ps.Status.Conditions, condition.Type), condition)
}

func (r *Reconciler) createOrUpdatePushSecret(ctx context.Context, csp *v1alpha1.ClusterPushSecret, namespace v1.Namespace, esName string, esMetadata v1alpha1.PushSecretMetadata) error {
	pushSecret := &v1alpha1.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace.Name,
			Name:      esName,
		},
	}

	mutateFunc := func() error {
		pushSecret.Labels = esMetadata.Labels
		pushSecret.Annotations = esMetadata.Annotations
		pushSecret.Spec = csp.Spec.PushSecretSpec

		if err := controllerutil.SetControllerReference(csp, pushSecret, r.Scheme); err != nil {
			return fmt.Errorf("could not set the controller owner reference %w", err)
		}

		return nil
	}

	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, pushSecret, mutateFunc); err != nil {
		return fmt.Errorf("could not create or update push secret: %w", err)
	}

	return nil
}

func (r *Reconciler) deletePushSecret(ctx context.Context, esName, cesName, namespace string) error {
	var existingPs v1alpha1.PushSecret
	err := r.Get(ctx, types.NamespacedName{
		Name:      esName,
		Namespace: namespace,
	}, &existingPs)
	if err != nil {
		// If we can't find it then just leave
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if !isPushSecretOwnedBy(&existingPs, cesName) {
		return nil
	}

	err = r.Delete(ctx, &existingPs, &client.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("external secret in non matching namespace could not be deleted: %w", err)
	}

	return nil
}

func (r *Reconciler) deferPatch(ctx context.Context, log logr.Logger, cps *v1alpha1.ClusterPushSecret, p client.Patch) {
	if err := r.Status().Patch(ctx, cps, p); err != nil {
		log.Error(err, errPatchStatus)
	}
}

func (r *Reconciler) deleteOutdatedPushSecrets(ctx context.Context, namespaces []v1.Namespace, esName, cesName string, provisionedNamespaces []string) map[string]error {
	failedNamespaces := map[string]error{}
	// Loop through existing namespaces first to make sure they still have our labels
	for _, namespace := range getRemovedNamespaces(namespaces, provisionedNamespaces) {
		err := r.deletePushSecret(ctx, esName, cesName, namespace)
		if err != nil {
			r.Log.Error(err, "unable to delete external secret")
			failedNamespaces[namespace] = err
		}
	}

	return failedNamespaces
}

func isPushSecretOwnedBy(ps *v1alpha1.PushSecret, cesName string) bool {
	owner := metav1.GetControllerOf(ps)
	return owner != nil && owner.APIVersion == v1alpha1.SchemeGroupVersion.String() && owner.Kind == "ClusterPushSecret" && owner.Name == cesName
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

func toNamespaceFailures(failedNamespaces map[string]error) []v1alpha1.ClusterPushSecretNamespaceFailure {
	namespaceFailures := make([]v1alpha1.ClusterPushSecretNamespaceFailure, len(failedNamespaces))

	i := 0
	for namespace, err := range failedNamespaces {
		namespaceFailures[i] = v1alpha1.ClusterPushSecretNamespaceFailure{
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
		For(&v1alpha1.ClusterPushSecret{}).
		Owns(&v1alpha1.PushSecret{}).
		Watches(
			&v1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForNamespace),
			builder.WithPredicates(utils.NamespacePredicate()),
		).
		Complete(r)
}

func (r *Reconciler) findObjectsForNamespace(ctx context.Context, namespace client.Object) []reconcile.Request {
	var cpsl v1alpha1.ClusterPushSecretList
	if err := r.List(ctx, &cpsl); err != nil {
		r.Log.Error(err, errGetCES)
		return []reconcile.Request{}
	}

	var requests []reconcile.Request
	for i := range cpsl.Items {
		cps := &cpsl.Items[i]
		for _, selector := range cps.Spec.NamespaceSelectors {
			labelSelector, err := metav1.LabelSelectorAsSelector(selector)
			if err != nil {
				r.Log.Error(err, errConvertLabelSelector)
				continue
			}

			if labelSelector.Matches(labels.Set(namespace.GetLabels())) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      cps.GetName(),
						Namespace: cps.GetNamespace(),
					},
				})
				break
			}
		}
	}

	return requests
}
