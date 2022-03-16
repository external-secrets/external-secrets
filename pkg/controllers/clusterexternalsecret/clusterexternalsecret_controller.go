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

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// ClusterExternalSecretReconciler reconciles a ClusterExternalSecret object.
type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	RequeueInterval time.Duration
}

const (
	errGetCES              = "could not get ClusterExternalSecret"
	errPatchStatus         = "unable to patch status"
	errLabelMap            = "unable to get map from labels"
	errNamespaces          = "could not get namespaces from selector"
	errGetExistingES       = "could not get existing ExternalSecret"
	errCreatingOrUpdating  = "could not create or update ExternalSecret"
	errSetCtrlReference    = "could not set the controller owner reference"
	errSecretAlreadyExists = "external secret already exists in namespace"
	errNamespacesFailed    = "one or more namespaces failed"
	errFailedToDelete      = "external secret in non matching namespace could not be deleted"
)

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ClusterExternalSecret", req.NamespacedName)

	var clusterExternalSecret esv1beta1.ClusterExternalSecret

	err := r.Get(ctx, req.NamespacedName, &clusterExternalSecret)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, errGetCES)
		return ctrl.Result{}, nil
	}

	p := client.MergeFrom(clusterExternalSecret.DeepCopy())
	defer r.deferPatch(ctx, log, &clusterExternalSecret, p)

	refreshInt := r.RequeueInterval
	if clusterExternalSecret.Spec.RefreshInterval != nil {
		refreshInt = clusterExternalSecret.Spec.RefreshInterval.Duration
	}

	labelMap, err := metav1.LabelSelectorAsMap(&clusterExternalSecret.Spec.NamespaceSelector)
	if err != nil {
		log.Error(err, errLabelMap)
		return ctrl.Result{RequeueAfter: refreshInt}, err
	}

	namespaceList := v1.NamespaceList{}

	err = r.List(ctx, &namespaceList, &client.ListOptions{LabelSelector: labels.SelectorFromSet(labelMap)})
	if err != nil {
		log.Error(err, errNamespaces)
		return ctrl.Result{RequeueAfter: refreshInt}, err
	}

	esName := clusterExternalSecret.Spec.ExternalSecretName
	if esName == "" {
		esName = clusterExternalSecret.ObjectMeta.Name
	}

	failedNamespaces := r.removeOldNamespaces(ctx, namespaceList, esName, clusterExternalSecret.Status.ProvisionedNamespaces)
	provisionedNamespaces := []string{}

	for _, namespace := range namespaceList.Items {
		var existingES esv1beta1.ExternalSecret
		err = r.Get(ctx, types.NamespacedName{
			Name:      esName,
			Namespace: namespace.Name,
		}, &existingES)

		if result := checkForError(err, &existingES); result != "" {
			log.Error(err, result)
			failedNamespaces[namespace.Name] = result
			continue
		}

		if result, err := r.resolveExternalSecret(ctx, &clusterExternalSecret, &existingES, namespace, esName); err != nil {
			log.Error(err, result)
			failedNamespaces[namespace.Name] = result
			continue
		}

		provisionedNamespaces = append(provisionedNamespaces, namespace.ObjectMeta.Name)
	}

	conditionType := getCondition(failedNamespaces, &namespaceList)

	condition := NewClusterExternalSecretCondition(conditionType, v1.ConditionTrue)

	if conditionType != esv1beta1.ClusterExternalSecretReady {
		condition.Message = errNamespacesFailed
	}

	SetClusterExternalSecretCondition(&clusterExternalSecret, *condition)
	setFailedNamespaces(&clusterExternalSecret, failedNamespaces)

	if len(provisionedNamespaces) > 0 {
		clusterExternalSecret.Status.ProvisionedNamespaces = provisionedNamespaces
	}

	return ctrl.Result{RequeueAfter: refreshInt}, nil
}

func (r *Reconciler) resolveExternalSecret(ctx context.Context, clusterExternalSecret *esv1beta1.ClusterExternalSecret, existingES *esv1beta1.ExternalSecret, namespace v1.Namespace, esName string) (string, error) {
	// this means the existing ES does not belong to us
	if err := controllerutil.SetControllerReference(clusterExternalSecret, existingES, r.Scheme); err != nil {
		return errSetCtrlReference, err
	}

	externalSecret := esv1beta1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      esName,
			Namespace: namespace.Name,
		},
		Spec: clusterExternalSecret.Spec.ExternalSecretSpec,
	}

	if err := controllerutil.SetControllerReference(clusterExternalSecret, &externalSecret, r.Scheme); err != nil {
		return errSetCtrlReference, err
	}

	mutateFunc := func() error {
		externalSecret.Spec = clusterExternalSecret.Spec.ExternalSecretSpec
		return nil
	}

	// An empty mutate func as nothing needs to happen currently
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, &externalSecret, mutateFunc); err != nil {
		return errCreatingOrUpdating, err
	}

	return "", nil
}

func (r *Reconciler) removeExternalSecret(ctx context.Context, esName, namespace string) (string, error) {
	//
	var existingES esv1beta1.ExternalSecret
	err := r.Get(ctx, types.NamespacedName{
		Name:      esName,
		Namespace: namespace,
	}, &existingES)

	// If we can't find it then just leave
	if err != nil && apierrors.IsNotFound(err) {
		return "", nil
	}

	if result := checkForError(err, &existingES); result != "" {
		return result, err
	}

	err = r.Delete(ctx, &existingES, &client.DeleteOptions{})

	if err != nil {
		return errFailedToDelete, err
	}

	return "", nil
}

func (r *Reconciler) deferPatch(ctx context.Context, log logr.Logger, clusterExternalSecret *esv1beta1.ClusterExternalSecret, p client.Patch) {
	if err := r.Status().Patch(ctx, clusterExternalSecret, p); err != nil {
		log.Error(err, errPatchStatus)
	}
}

func (r *Reconciler) removeOldNamespaces(ctx context.Context, namespaceList v1.NamespaceList, esName string, provisionedNamespaces []string) map[string]string {
	failedNamespaces := map[string]string{}
	// Loop through existing namespaces first to make sure they still have our labels
	for _, namespace := range getRemovedNamespaces(namespaceList, provisionedNamespaces) {
		if result, _ := r.removeExternalSecret(ctx, esName, namespace); result != "" {
			failedNamespaces[namespace] = result
		}
	}

	return failedNamespaces
}

func checkForError(getError error, existingES *esv1beta1.ExternalSecret) string {
	if getError != nil && !apierrors.IsNotFound(getError) {
		return errGetExistingES
	}

	// No one owns this resource so error out
	if !apierrors.IsNotFound(getError) && len(existingES.ObjectMeta.OwnerReferences) == 0 {
		return errSecretAlreadyExists
	}

	return ""
}

func getCondition(namespaces map[string]string, namespaceList *v1.NamespaceList) esv1beta1.ClusterExternalSecretConditionType {
	if len(namespaces) == 0 {
		return esv1beta1.ClusterExternalSecretReady
	}

	if len(namespaces) < len(namespaceList.Items) {
		return esv1beta1.ClusterExternalSecretPartiallyReady
	}

	return esv1beta1.ClusterExternalSecretNotReady
}

func getRemovedNamespaces(nsList v1.NamespaceList, provisionedNs []string) []string {
	result := []string{}

	for _, ns := range provisionedNs {
		if !ContainsNamespace(nsList, ns) {
			result = append(result, ns)
		}
	}

	return result
}

func setFailedNamespaces(ces *esv1beta1.ClusterExternalSecret, failedNamespaces map[string]string) {
	if len(failedNamespaces) == 0 {
		return
	}

	ces.Status.FailedNamespaces = []esv1beta1.ClusterExternalSecretNamespaceFailure{}

	for namespace, message := range failedNamespaces {
		ces.Status.FailedNamespaces = append(ces.Status.FailedNamespaces, esv1beta1.ClusterExternalSecretNamespaceFailure{
			Namespace: namespace,
			Reason:    message,
		})
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esv1beta1.ClusterExternalSecret{}).
		Owns(&esv1beta1.ExternalSecret{}, builder.OnlyMetadata).
		Complete(r)
}
