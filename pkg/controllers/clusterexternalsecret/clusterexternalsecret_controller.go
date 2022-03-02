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
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
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
	errSecretAlreadyExists = "secret already exists in namespace"
)

//+kubebuilder:rbac:groups=external-secrets.io,resources=clusterexternalsecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=external-secrets.io,resources=clusterexternalsecrets/status,verbs=get;update;patch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ClusterExternalSecret", req.NamespacedName)

	var clusterExternalSecret esv1alpha1.ClusterExternalSecret

	err := r.Get(ctx, req.NamespacedName, &clusterExternalSecret)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, errGetCES)
		return ctrl.Result{}, nil
	}

	p := client.MergeFrom(clusterExternalSecret.DeepCopy())
	defer func() {
		err = r.Status().Patch(ctx, &clusterExternalSecret, p)
		if err != nil {
			log.Error(err, errPatchStatus)
		}
	}()

	// Fetch Namespaces to grab ExternalSecrets
	genClient := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())
	namespaces := genClient.CoreV1().Namespaces()

	refreshInt := r.RequeueInterval
	if clusterExternalSecret.Spec.RefreshInterval != nil {
		refreshInt = clusterExternalSecret.Spec.RefreshInterval.Duration
	}

	labelMap, err := metav1.LabelSelectorAsMap(&clusterExternalSecret.Spec.NamespaceSelector)
	if err != nil {
		log.Error(err, errLabelMap)
		return ctrl.Result{RequeueAfter: refreshInt}, err
	}

	namespaceList, err := namespaces.List(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labelMap).String()})
	if err != nil {
		log.Error(err, errNamespaces)
		return ctrl.Result{RequeueAfter: refreshInt}, err
	}

	esName := clusterExternalSecret.Spec.ExternalSecretName
	if esName == "" {
		esName = clusterExternalSecret.ObjectMeta.Name
	}

	failedNamespaces := []string{}

	for _, namespace := range namespaceList.Items {
		var existingES esv1beta1.ExternalSecret
		err = r.Get(ctx, types.NamespacedName{
			Name:      esName,
			Namespace: namespace.Name,
		}, &existingES)

		if err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, errGetExistingES)
		}

		// No one owns this resource so error out
		if !apierrors.IsNotFound(err) && len(existingES.ObjectMeta.OwnerReferences) == 0 {
			log.Error(nil, errSecretAlreadyExists, "namespace", namespace)
			failedNamespaces = append(failedNamespaces, namespace.Name)
			continue
		}

		if err = r.resolveExternalSecret(log, ctx, clusterExternalSecret, existingES, namespace, esName); err != nil {
			failedNamespaces = append(failedNamespaces, namespace.Name)
		}
	}

	if len(failedNamespaces) > 0 {
		conditionType := getConditionType(failedNamespaces, namespaceList)
		conditionFailed := NewClusterExternalSecretCondition(conditionType, v1.ConditionFalse, "one or more namespaces failed")
		SetClusterExternalSecretCondition(&clusterExternalSecret, *conditionFailed)
		clusterExternalSecret.Status.FailedNamespaces = failedNamespaces
		return ctrl.Result{RequeueAfter: refreshInt}, fmt.Errorf("failed to sync to the following namespaces:%v", failedNamespaces)
	}

	return ctrl.Result{RequeueAfter: refreshInt}, nil
}

func (r *Reconciler) resolveExternalSecret(log logr.Logger, ctx context.Context, clusterExternalSecret esv1alpha1.ClusterExternalSecret, existingES esv1beta1.ExternalSecret, namespace v1.Namespace, esName string) error {
	// this means the existing ES does not belong to us
	if err := controllerutil.SetControllerReference(&clusterExternalSecret, &existingES, r.Scheme); err != nil {
		log.Error(err, errSetCtrlReference, "namespace", namespace)
		return err
	}

	externalSecret := esv1alpha1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      esName,
			Namespace: namespace.Name,
		},
		Spec: clusterExternalSecret.Spec.ExternalSecretSpec,
	}

	if err := controllerutil.SetControllerReference(&clusterExternalSecret, &externalSecret, r.Scheme); err != nil {
		log.Error(err, errSetCtrlReference)
		return err
	}

	mutateFunc := func() error {
		externalSecret.Spec = clusterExternalSecret.Spec.ExternalSecretSpec
		return nil
	}

	// An empty mutate func as nothing needs to happen currently
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, &externalSecret, mutateFunc); err != nil {
		log.Error(err, errCreatingOrUpdating)
		return err
	}

	return nil
}

func getConditionType(namespaces []string, namespaceList *v1.NamespaceList) esv1alpha1.ClusterExternalSecretConditionType {
	if len(namespaces) < len(namespaceList.Items) {
		return esv1alpha1.ClusterExternalSecretPartiallyReady
	} else {
		return esv1alpha1.ClusterExternalSecretNotReady
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esv1alpha1.ClusterExternalSecret{}).
		Owns(&esv1alpha1.ExternalSecret{}, builder.OnlyMetadata).
		Complete(r)
}
