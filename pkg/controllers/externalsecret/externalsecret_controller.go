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

package externalsecret

import (
	"context"

	// nolint
	"crypto/md5"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"

	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
	schema "github.com/external-secrets/external-secrets/pkg/provider/schema"
	"github.com/external-secrets/external-secrets/pkg/template"
	utils "github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	requeueAfter = time.Second * 30

	errGetES                 = "could not get ExternalSecret"
	errReconcileES           = "could not reconcile ExternalSecret"
	errPatchStatus           = "unable to patch status"
	errGetSecretStore        = "could not get SecretStore %q, %w"
	errGetClusterSecretStore = "could not get ClusterSecretStore %q, %w"
	errStoreRef              = "could not get store reference"
	errStoreProvider         = "could not get store provider"
	errStoreClient           = "could not get provider client"
	errCloseStoreClient      = "could not close provider client"
	errSetCtrlReference      = "could not set ExternalSecret controller reference: %w"
	errFetchTplFrom          = "error fetching templateFrom data: %w"
	errGetSecretData         = "could not get secret data from provider: %w"
	errExecTpl               = "could not execute template: %w"
	errPolicyMergeNotFound   = "the desired secret %s was not found. With creationPolicy=Merge the secret won't be created"
	errPolicyMergeGetSecret  = "unable to get secret %s: %w"
	errPolicyMergeMutate     = "unable to mutate secret %s: %w"
	errPolicyMergePatch      = "unable to patch secret %s: %w"
	errGetSecretKey          = "key %q from ExternalSecret %q: %w"
	errTplCMMissingKey       = "error in configmap %s: missing key %s"
	errTplSecMissingKey      = "error in secret %s: missing key %s"
)

// Reconciler reconciles a ExternalSecret object.
type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	ControllerClass string
	RequeueInterval time.Duration
}

// Reconcile implements the main reconciliation loop
// for watched objects (ExternalSecret, ClusterSecretStore and SecretStore),
// and updates/creates a Kubernetes secret based on them.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ExternalSecret", req.NamespacedName)

	syncCallsMetricLabels := prometheus.Labels{"name": req.Name, "namespace": req.Namespace}

	var externalSecret esv1alpha1.ExternalSecret

	err := r.Get(ctx, req.NamespacedName, &externalSecret)
	if apierrors.IsNotFound(err) {
		syncCallsTotal.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, errGetES)
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{}, nil
	}

	// patch status when done processing
	p := client.MergeFrom(externalSecret.DeepCopy())
	defer func() {
		err = r.Status().Patch(ctx, &externalSecret, p)
		if err != nil {
			log.Error(err, errPatchStatus)
		}
	}()

	store, err := r.getStore(ctx, &externalSecret)
	if err != nil {
		log.Error(err, errStoreRef)
		conditionSynced := NewExternalSecretCondition(esv1alpha1.ExternalSecretReady, v1.ConditionFalse, esv1alpha1.ConditionReasonSecretSyncedError, err.Error())
		SetExternalSecretCondition(&externalSecret, *conditionSynced)
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	log = log.WithValues("SecretStore", store.GetNamespacedName())

	// check if store should be handled by this controller instance
	if !shouldProcessStore(store, r.ControllerClass) {
		log.Info("skippig unmanaged store")
		return ctrl.Result{}, nil
	}

	storeProvider, err := schema.GetProvider(store)
	if err != nil {
		log.Error(err, errStoreProvider)
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	secretClient, err := storeProvider.NewClient(ctx, store, r.Client, req.Namespace)
	if err != nil {
		log.Error(err, errStoreClient)
		conditionSynced := NewExternalSecretCondition(esv1alpha1.ExternalSecretReady, v1.ConditionFalse, esv1alpha1.ConditionReasonSecretSyncedError, err.Error())
		SetExternalSecretCondition(&externalSecret, *conditionSynced)
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	defer func() {
		err = secretClient.Close()
		if err != nil {
			log.Error(err, errCloseStoreClient)
		}
	}()

	refreshInt := r.RequeueInterval
	if externalSecret.Spec.RefreshInterval != nil {
		refreshInt = externalSecret.Spec.RefreshInterval.Duration
	}

	// refresh should be skipped if
	// 1. resource generation hasn't changed
	// 2. refresh interval is 0
	// 3. if we're still within refresh-interval
	if !shouldRefresh(externalSecret) {
		log.V(1).Info("skipping refresh", "rv", getResourceVersion(externalSecret))
		return ctrl.Result{RequeueAfter: refreshInt}, nil
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalSecret.Spec.Target.Name,
			Namespace: externalSecret.Namespace,
		},
		Data: make(map[string][]byte),
	}

	mutationFunc := func() error {
		if externalSecret.Spec.Target.CreationPolicy == esv1alpha1.Owner {
			err = controllerutil.SetControllerReference(&externalSecret, &secret.ObjectMeta, r.Scheme)
			if err != nil {
				return fmt.Errorf(errSetCtrlReference, err)
			}
		}
		mergeMetadata(secret, externalSecret)
		var tplMap map[string][]byte
		var dataMap map[string][]byte

		// get data
		dataMap, err = r.getProviderSecretData(ctx, secretClient, &externalSecret)
		if err != nil {
			return fmt.Errorf(errGetSecretData, err)
		}

		// no template: copy data and return
		if externalSecret.Spec.Target.Template == nil {
			for k, v := range dataMap {
				secret.Data[k] = v
			}
			return nil
		}

		// template: fetch & execute templates
		tplMap, err = r.getTemplateData(ctx, &externalSecret)
		if err != nil {
			return fmt.Errorf(errFetchTplFrom, err)
		}
		// override templateFrom data with template data
		for k, v := range externalSecret.Spec.Target.Template.Data {
			tplMap[k] = []byte(v)
		}

		log.V(1).Info("found template data", "tpl_data", tplMap)
		err = template.Execute(tplMap, dataMap, secret)
		if err != nil {
			return fmt.Errorf(errExecTpl, err)
		}
		return nil
	}

	//nolint
	switch externalSecret.Spec.Target.CreationPolicy {
	case esv1alpha1.Merge:
		err = patchSecret(ctx, r.Client, r.Scheme, secret, mutationFunc)
	case esv1alpha1.None:
		log.V(1).Info("secret creation skipped due to creationPolicy=None")
		err = nil
	default:
		_, err = ctrl.CreateOrUpdate(ctx, r.Client, secret, mutationFunc)
	}

	if err != nil {
		log.Error(err, errReconcileES)
		conditionSynced := NewExternalSecretCondition(esv1alpha1.ExternalSecretReady, v1.ConditionFalse, esv1alpha1.ConditionReasonSecretSyncedError, err.Error())
		SetExternalSecretCondition(&externalSecret, *conditionSynced)
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	conditionSynced := NewExternalSecretCondition(esv1alpha1.ExternalSecretReady, v1.ConditionTrue, esv1alpha1.ConditionReasonSecretSynced, "Secret was synced")
	SetExternalSecretCondition(&externalSecret, *conditionSynced)
	externalSecret.Status.RefreshTime = metav1.NewTime(time.Now())
	externalSecret.Status.SyncedResourceVersion = getResourceVersion(externalSecret)
	syncCallsTotal.With(syncCallsMetricLabels).Inc()
	log.V(1).Info("reconciled secret")

	return ctrl.Result{
		RequeueAfter: refreshInt,
	}, nil
}

func patchSecret(ctx context.Context, c client.Client, scheme *runtime.Scheme, secret *v1.Secret, mutationFunc func() error) error {
	err := c.Get(ctx, client.ObjectKeyFromObject(secret), secret.DeepCopy())
	if apierrors.IsNotFound(err) {
		return fmt.Errorf(errPolicyMergeNotFound, secret.Name)
	}
	if err != nil {
		return fmt.Errorf(errPolicyMergeGetSecret, secret.Name, err)
	}
	err = mutationFunc()
	if err != nil {
		return fmt.Errorf(errPolicyMergeMutate, secret.Name, err)
	}
	// GVK is missing in the Secret, see:
	// https://github.com/kubernetes-sigs/controller-runtime/issues/526
	// https://github.com/kubernetes-sigs/controller-runtime/issues/1517
	// https://github.com/kubernetes/kubernetes/issues/80609
	// we need to manually set it befor doing a Patch() as it depends on the GVK
	gvks, unversioned, err := scheme.ObjectKinds(secret)
	if err != nil {
		return err
	}
	if !unversioned && len(gvks) == 1 {
		secret.SetGroupVersionKind(gvks[0])
	}
	// we might get into a conflict here if we are not the manager of that particular field
	// we do not resolve the conflict and return an error instead
	// see: https://kubernetes.io/docs/reference/using-api/server-side-apply/#conflicts
	err = c.Patch(ctx, secret, client.Apply, client.FieldOwner("external-secrets"))
	if err != nil {
		return fmt.Errorf(errPolicyMergePatch, secret.Name, err)
	}
	return nil
}

// shouldProcessStore returns true if the store should be processed.
func shouldProcessStore(store esv1alpha1.GenericStore, class string) bool {
	if store.GetSpec().Controller == "" || store.GetSpec().Controller == class {
		return true
	}
	return false
}

func getResourceVersion(es esv1alpha1.ExternalSecret) string {
	return fmt.Sprintf("%d-%s", es.ObjectMeta.GetGeneration(), hashMeta(es.ObjectMeta))
}

func hashMeta(m metav1.ObjectMeta) string {
	type meta struct {
		annotations map[string]string
		labels      map[string]string
	}
	h := md5.New() //nolint
	_, _ = h.Write([]byte(fmt.Sprintf("%v", meta{
		annotations: m.Annotations,
		labels:      m.Labels,
	})))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func shouldRefresh(es esv1alpha1.ExternalSecret) bool {
	// refresh if resource version changed
	if es.Status.SyncedResourceVersion != getResourceVersion(es) {
		return true
	}
	// skip refresh if refresh interval is 0
	if es.Spec.RefreshInterval.Duration == 0 && es.Status.SyncedResourceVersion != "" {
		return false
	}
	if es.Status.RefreshTime.IsZero() {
		return true
	}
	return !es.Status.RefreshTime.Add(es.Spec.RefreshInterval.Duration).After(time.Now())
}

// we do not want to force-override the label/annotations
// and only copy the necessary key/value pairs.
func mergeMetadata(secret *v1.Secret, externalSecret esv1alpha1.ExternalSecret) {
	if secret.ObjectMeta.Labels == nil {
		secret.ObjectMeta.Labels = make(map[string]string)
	}
	if secret.ObjectMeta.Annotations == nil {
		secret.ObjectMeta.Annotations = make(map[string]string)
	}
	if externalSecret.Spec.Target.Template == nil {
		utils.MergeStringMap(secret.ObjectMeta.Labels, externalSecret.ObjectMeta.Labels)
		utils.MergeStringMap(secret.ObjectMeta.Annotations, externalSecret.ObjectMeta.Annotations)
		return
	}
	// if template is defined: use those labels/annotations
	secret.Type = externalSecret.Spec.Target.Template.Type
	utils.MergeStringMap(secret.ObjectMeta.Labels, externalSecret.Spec.Target.Template.Metadata.Labels)
	utils.MergeStringMap(secret.ObjectMeta.Annotations, externalSecret.Spec.Target.Template.Metadata.Annotations)
}

// getStore returns the store with the provided ExternalSecret.
func (r *Reconciler) getStore(ctx context.Context, externalSecret *esv1alpha1.ExternalSecret) (esv1alpha1.GenericStore, error) {
	ref := types.NamespacedName{
		Name: externalSecret.Spec.SecretStoreRef.Name,
	}

	if externalSecret.Spec.SecretStoreRef.Kind == esv1alpha1.ClusterSecretStoreKind {
		var store esv1alpha1.ClusterSecretStore
		err := r.Get(ctx, ref, &store)
		if err != nil {
			return nil, fmt.Errorf(errGetClusterSecretStore, ref.Name, err)
		}

		return &store, nil
	}

	ref.Namespace = externalSecret.Namespace

	var store esv1alpha1.SecretStore
	err := r.Get(ctx, ref, &store)
	if err != nil {
		return nil, fmt.Errorf(errGetSecretStore, ref.Name, err)
	}
	return &store, nil
}

// getProviderSecretData returns the provider's secret data with the provided ExternalSecret.
func (r *Reconciler) getProviderSecretData(ctx context.Context, providerClient provider.SecretsClient, externalSecret *esv1alpha1.ExternalSecret) (map[string][]byte, error) {
	providerData := make(map[string][]byte)

	for _, remoteRef := range externalSecret.Spec.DataFrom {
		secretMap, err := providerClient.GetSecretMap(ctx, remoteRef)
		if err != nil {
			return nil, fmt.Errorf(errGetSecretKey, remoteRef.Key, externalSecret.Name, err)
		}

		providerData = utils.MergeByteMap(providerData, secretMap)
	}

	for _, secretRef := range externalSecret.Spec.Data {
		secretData, err := providerClient.GetSecret(ctx, secretRef.RemoteRef)
		if err != nil {
			return nil, fmt.Errorf(errGetSecretKey, secretRef.RemoteRef.Key, externalSecret.Name, err)
		}

		providerData[secretRef.SecretKey] = secretData
	}

	return providerData, nil
}

func (r *Reconciler) getTemplateData(ctx context.Context, externalSecret *esv1alpha1.ExternalSecret) (map[string][]byte, error) {
	out := make(map[string][]byte)
	if externalSecret.Spec.Target.Template == nil {
		return out, nil
	}
	for _, tpl := range externalSecret.Spec.Target.Template.TemplateFrom {
		if tpl.ConfigMap != nil {
			var cm v1.ConfigMap
			err := r.Client.Get(ctx, types.NamespacedName{
				Name:      tpl.ConfigMap.Name,
				Namespace: externalSecret.Namespace,
			}, &cm)
			if err != nil {
				return nil, err
			}
			for _, k := range tpl.ConfigMap.Items {
				val, ok := cm.Data[k.Key]
				if !ok {
					return nil, fmt.Errorf(errTplCMMissingKey, tpl.ConfigMap.Name, k.Key)
				}
				out[k.Key] = []byte(val)
			}
		}
		if tpl.Secret != nil {
			var sec v1.Secret
			err := r.Client.Get(ctx, types.NamespacedName{
				Name:      tpl.Secret.Name,
				Namespace: externalSecret.Namespace,
			}, &sec)
			if err != nil {
				return nil, err
			}
			for _, k := range tpl.Secret.Items {
				val, ok := sec.Data[k.Key]
				if !ok {
					return nil, fmt.Errorf(errTplSecMissingKey, tpl.Secret.Name, k.Key)
				}
				out[k.Key] = val
			}
		}
	}
	return out, nil
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&esv1alpha1.ExternalSecret{}).
		Owns(&v1.Secret{}).
		Complete(r)
}
