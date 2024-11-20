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
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	// Metrics.
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"

	// Loading registered generators.
	_ "github.com/external-secrets/external-secrets/pkg/generator/register"
	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

const (
	fieldOwnerTemplate = "externalsecrets.external-secrets.io/%v"

	// condition messages for "SecretSynced" reason.
	msgSynced       = "secret synced"
	msgSyncedRetain = "secret retained due to DeletionPolicy=Retain"

	// condition messages for "SecretDeleted" reason.
	msgDeleted = "secret deleted due to DeletionPolicy=Delete"

	// condition messages for "SecretMissing" reason.
	msgMissing = "secret will not be created due to CreationPolicy=Merge"

	// condition messages for "SecretSyncedError" reason.
	msgErrorGetSecretData   = "could not get secret data from provider"
	msgErrorDeleteSecret    = "could not delete secret"
	msgErrorDeleteOrphaned  = "could not delete orphaned secrets"
	msgErrorUpdateSecret    = "could not update secret"
	msgErrorUpdateImmutable = "could not update secret, target is immutable"

	// log messages.
	logErrorGetES          = "unable to get ExternalSecret"
	logErrorUpdateESStatus = "unable to update ExternalSecret status"
	logErrorGetSecret      = "unable to get Secret"
	logErrorPatchSecret    = "unable to patch Secret"

	// error formats.
	errConvert            = "could not apply conversion strategy to keys: %v"
	errDecode             = "could not apply decoding strategy to %v[%d]: %v"
	errGenerate           = "could not generate [%d]: %w"
	errRewrite            = "could not rewrite spec.dataFrom[%d]: %v"
	errInvalidKeys        = "secret keys from spec.dataFrom.%v[%d] can only have alphanumeric, '-', '_' or '.' characters. Convert them using rewrite (https://external-secrets.io/latest/guides/datafrom-rewrite/)"
	errSetCtrlReference   = "could not set controller reference: %w"
	errFetchTplFrom       = "error fetching templateFrom data: %w"
	errApplyTemplate      = "could not apply template: %w"
	errExecTpl            = "could not execute template: %w"
	errMutate             = "unable to mutate secret %s: %w"
	errUpdate             = "unable to update secret %s: %w"
	errUpdateNotFound     = "unable to update secret %s: not found"
	errUpdateImmutable    = "unable to update secret %s: immutable"
	errDeleteCreatePolicy = "unable to delete secret %s: creationPolicy=%s is not Owner"
)

const indexESTargetSecretNameField = ".metadata.targetSecretName"

// Reconciler reconciles a ExternalSecret object.
type Reconciler struct {
	client.Client
	SecretClient              client.Client
	Log                       logr.Logger
	Scheme                    *runtime.Scheme
	RestConfig                *rest.Config
	ControllerClass           string
	RequeueInterval           time.Duration
	ClusterSecretStoreEnabled bool
	EnableFloodGate           bool
	recorder                  record.EventRecorder
}

// Reconcile implements the main reconciliation loop
// for watched objects (ExternalSecret, ClusterSecretStore and SecretStore),
// and updates/creates a Kubernetes secret based on them.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := r.Log.WithValues("ExternalSecret", req.NamespacedName)

	resourceLabels := ctrlmetrics.RefineNonConditionMetricLabels(map[string]string{"name": req.Name, "namespace": req.Namespace})
	start := time.Now()

	syncCallsError := esmetrics.GetCounterVec(esmetrics.SyncCallsErrorKey)

	// use closures to dynamically update resourceLabels
	defer func() {
		esmetrics.GetGaugeVec(esmetrics.ExternalSecretReconcileDurationKey).With(resourceLabels).Set(float64(time.Since(start)))
		esmetrics.GetCounterVec(esmetrics.SyncCallsKey).With(resourceLabels).Inc()
	}()

	externalSecret := &esv1beta1.ExternalSecret{}
	err = r.Get(ctx, req.NamespacedName, externalSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// NOTE: this does not actually set the condition on the ExternalSecret, because it does not exist
			//       this is a hack to disable metrics for deleted ExternalSecrets, see:
			//       https://github.com/external-secrets/external-secrets/pull/612
			conditionSynced := NewExternalSecretCondition(esv1beta1.ExternalSecretDeleted, v1.ConditionFalse, esv1beta1.ConditionReasonSecretDeleted, "Secret was deleted")
			SetExternalSecretCondition(&esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      req.Name,
					Namespace: req.Namespace,
				},
			}, *conditionSynced)

			return ctrl.Result{}, nil
		}

		log.Error(err, logErrorGetES)
		syncCallsError.With(resourceLabels).Inc()
		return ctrl.Result{}, err
	}

	// skip reconciliation if deletion timestamp is set on external secret
	if !externalSecret.GetDeletionTimestamp().IsZero() {
		log.V(1).Info("external secret is marked for deletion, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// if extended metrics is enabled, refine the time series vector
	resourceLabels = ctrlmetrics.RefineLabels(resourceLabels, externalSecret.Labels)

	if shouldSkipClusterSecretStore(r, externalSecret) {
		log.V(1).Info("skipping cluster secret store as it is disabled")
		return ctrl.Result{}, nil
	}

	// skip when pointing to an unmanaged store
	skip, err := shouldSkipUnmanagedStore(ctx, req.Namespace, r, externalSecret)
	if skip {
		log.V(1).Info("skipping unmanaged store as it points to a unmanaged controllerClass")
		return ctrl.Result{}, nil
	}

	// Target Secret Name should default to the ExternalSecret name if not explicitly specified
	secretName := externalSecret.Spec.Target.Name
	if secretName == "" {
		secretName = externalSecret.Name
	}

	// fetch the existing secret (from the partial cache)
	//  - please note that the ~partial cache~ is different from the ~full cache~
	//    so there can be race conditions between the two caches
	//  - the WatchesMetadata(v1.Secret{}) in SetupWithManager() is using the partial cache
	//    so we might receive a reconcile request before the full cache is updated
	//  - furthermore, when `--enable-managed-secrets-caching` is true, the full cache
	//    will ONLY include secrets with the "managed" label, so we cant use the full cache
	//    to reliably determine if a secret exists or not
	secretPartial := &metav1.PartialObjectMetadata{}
	secretPartial.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
	err = r.Get(ctx, client.ObjectKey{Name: secretName, Namespace: externalSecret.Namespace}, secretPartial)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, logErrorGetSecret, "secretName", secretName, "secretNamespace", externalSecret.Namespace)
		syncCallsError.With(resourceLabels).Inc()
		return ctrl.Result{}, err
	}

	// if the secret exists but does not have the "managed" label, add the label
	// using a PATCH so it is visible in the cache, then requeue immediately
	if secretPartial.UID != "" && secretPartial.Labels[esv1beta1.LabelManaged] != esv1beta1.LabelManagedValue {
		fqdn := fmt.Sprintf(fieldOwnerTemplate, externalSecret.Name)
		patch := client.MergeFrom(secretPartial.DeepCopy())
		if secretPartial.Labels == nil {
			secretPartial.Labels = make(map[string]string)
		}
		secretPartial.Labels[esv1beta1.LabelManaged] = esv1beta1.LabelManagedValue
		err = r.Patch(ctx, secretPartial, patch, client.FieldOwner(fqdn))
		if err != nil {
			log.Error(err, logErrorPatchSecret, "secretName", secretName, "secretNamespace", externalSecret.Namespace)
			syncCallsError.With(resourceLabels).Inc()
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// fetch existing secret (from the full cache)
	// NOTE: we are using the `r.SecretClient` which is a special client we only used for managed secrets
	existingSecret := &v1.Secret{}
	err = r.SecretClient.Get(ctx, client.ObjectKey{Name: secretName, Namespace: externalSecret.Namespace}, existingSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, logErrorGetSecret, "secretName", secretName, "secretNamespace", externalSecret.Namespace)
		syncCallsError.With(resourceLabels).Inc()
		return ctrl.Result{}, err
	}

	// ensure the full cache is up-to-date
	if !equality.Semantic.DeepEqual(secretPartial.GetObjectMeta(), existingSecret.GetObjectMeta()) {
		log.V(1).Info("secret caches are not in sync, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	// refresh will be skipped if ALL the following conditions are met:
	// 1. refresh interval is not 0
	// 2. resource generation of the ExternalSecret has not changed
	// 3. the last refresh time of the ExternalSecret is within the refresh interval
	// 4. the target secret is valid:
	//     - it exists
	//     - it has the correct "managed" label
	//     - it has the correct "data-hash" annotation
	if !shouldRefresh(externalSecret) && isSecretValid(existingSecret) {
		log.V(1).Info("skipping refresh")
		return r.getRequeueResult(externalSecret), nil
	}

	// update status of the ExternalSecret when this function returns, if needed.
	// we use the ability of deferred functions to update named return values `result` and `err`.
	// NOTE: we dereference the DeepCopy of the status field because status fields are NOT pointers,
	//       so otherwise the `equality.Semantic.DeepEqual` will always return false.
	currentStatus := *externalSecret.Status.DeepCopy()
	defer func() {
		if !equality.Semantic.DeepEqual(currentStatus, externalSecret.Status) {
			updateErr := r.Status().Update(ctx, externalSecret)
			if updateErr != nil {
				// if we got an update conflict, we should requeue immediately
				if apierrors.IsConflict(updateErr) {
					log.V(1).Info("conflict while updating status, will requeue")

					// only update the result if there was no error, otherwise, we get an annoying log
					// saying that results are ignored when there is an error, and errors are retried anyway.
					if err == nil {
						result = ctrl.Result{Requeue: true}
					}
				} else {
					log.Error(updateErr, logErrorUpdateESStatus)
					// only update the error if there is no error already
					if err == nil {
						err = updateErr
					}
				}
			}
		}
	}()

	// retrieve the provider secret data.
	dataMap, err := r.getProviderSecretData(ctx, externalSecret)
	if err != nil {
		r.markAsFailed(msgErrorGetSecretData, err, externalSecret, syncCallsError.With(resourceLabels))
		return ctrl.Result{}, err
	}

	// if no data was found we can delete the secret if needed.
	if len(dataMap) == 0 {
		switch externalSecret.Spec.Target.DeletionPolicy {
		// delete secret and return early.
		case esv1beta1.DeletionPolicyDelete:
			// safeguard that we only can delete secrets we own
			// this is also implemented in the es validation webhook
			creationPolicy := externalSecret.Spec.Target.CreationPolicy
			if creationPolicy != esv1beta1.CreatePolicyOwner {
				err := fmt.Errorf(errDeleteCreatePolicy, secretName, creationPolicy)
				r.markAsFailed(msgErrorDeleteSecret, err, externalSecret, syncCallsError.With(resourceLabels))
				return ctrl.Result{}, err
			}

			// delete the secret, if it exists
			if existingSecret.UID != "" {
				if err := r.Delete(ctx, existingSecret); err != nil && !apierrors.IsNotFound(err) {
					r.markAsFailed(msgErrorDeleteSecret, err, externalSecret, syncCallsError.With(resourceLabels))
					return ctrl.Result{}, err
				}
			}

			r.markAsDone(externalSecret, start, log, esv1beta1.ConditionReasonSecretDeleted, msgDeleted)
			return r.getRequeueResult(externalSecret), nil
		// In case provider secrets don't exist the kubernetes secret will be kept as-is.
		case esv1beta1.DeletionPolicyRetain:
			r.markAsDone(externalSecret, start, log, esv1beta1.ConditionReasonSecretSynced, msgSyncedRetain)
			return r.getRequeueResult(externalSecret), nil
		// noop, handled below
		case esv1beta1.DeletionPolicyMerge:
		}
	}

	// mutationFunc is a function which can be applied to a secret to make it match the desired state.
	mutationFunc := func(secret *v1.Secret) error {
		if externalSecret.Spec.Target.CreationPolicy == esv1beta1.CreatePolicyOwner {
			err = controllerutil.SetControllerReference(externalSecret, secret, r.Scheme)
			if err != nil {
				return fmt.Errorf(errSetCtrlReference, err)
			}
		}

		// initialize maps within the secret so it's safe to set values
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string)
		}
		if secret.Labels == nil {
			secret.Labels = make(map[string]string)
		}
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}

		// get the list of keys that are managed by this ExternalSecret
		keys, err := getManagedDataKeys(secret, externalSecret.Name)
		if err != nil {
			return err
		}

		// remove any data keys that are managed by this ExternalSecret, so we can re-add them
		// this ensures keys added by templates are not left behind when they are removed from the template
		for _, key := range keys {
			delete(secret.Data, key)
		}

		err = r.applyTemplate(ctx, externalSecret, secret, dataMap)
		if err != nil {
			return fmt.Errorf(errApplyTemplate, err)
		}

		// set the immutable flag on the secret if requested by the ExternalSecret
		if externalSecret.Spec.Target.Immutable {
			secret.Immutable = ptr.To(true)
		}

		if externalSecret.Spec.Target.CreationPolicy == esv1beta1.CreatePolicyOwner {
			lblValue := utils.ObjectHash(fmt.Sprintf("%v/%v", externalSecret.Namespace, externalSecret.Name))
			secret.Labels[esv1beta1.LabelOwner] = lblValue
		}

		secret.Labels[esv1beta1.LabelManaged] = esv1beta1.LabelManagedValue
		secret.Annotations[esv1beta1.AnnotationDataHash] = utils.ObjectHash(secret.Data)

		return nil
	}

	switch externalSecret.Spec.Target.CreationPolicy { //nolint:exhaustive
	case esv1beta1.CreatePolicyMerge:
		// update the secret, if it exists
		if existingSecret.UID != "" {
			err = r.updateSecret(ctx, existingSecret, mutationFunc, externalSecret, secretName)
		} else {
			// if the secret does not exist, we wait until the next refresh interval
			// rather than returning an error which would requeue immediately
			r.markAsDone(externalSecret, start, log, esv1beta1.ConditionReasonSecretMissing, msgMissing)
			return r.getRequeueResult(externalSecret), nil
		}
	case esv1beta1.CreatePolicyNone:
		log.V(1).Info("secret creation skipped due to creationPolicy=None")
		err = nil
	default:
		// create the secret, if it does not exist
		if existingSecret.UID == "" {
			err = r.createSecret(ctx, mutationFunc, externalSecret, secretName)

			// we may have orphaned secrets to clean up,
			// for example, if the target secret name was changed
			if err == nil {
				delErr := deleteOrphanedSecrets(ctx, r.Client, externalSecret, secretName)
				if delErr != nil {
					r.markAsFailed(msgErrorDeleteOrphaned, delErr, externalSecret, syncCallsError.With(resourceLabels))
					return ctrl.Result{}, delErr
				}
			}
		} else {
			// update the secret, if it exists
			err = r.updateSecret(ctx, existingSecret, mutationFunc, externalSecret, secretName)
		}
	}
	if err != nil {
		// if we got an update conflict, we should requeue immediately
		if apierrors.IsConflict(err) {
			log.V(1).Info("conflict while updating secret, will requeue")
			return ctrl.Result{Requeue: true}, nil
		}

		// if the secret is immutable, we will never be able to update it
		// so we don't return an error (which would requeue immediately)
		if strings.Contains(err.Error(), fmt.Sprintf(errUpdateImmutable, secretName)) {
			r.markAsFailed(msgErrorUpdateImmutable, err, externalSecret, syncCallsError.With(resourceLabels))
			return ctrl.Result{}, nil
		}

		r.markAsFailed(msgErrorUpdateSecret, err, externalSecret, syncCallsError.With(resourceLabels))
		return ctrl.Result{}, err
	}

	r.markAsDone(externalSecret, start, log, esv1beta1.ConditionReasonSecretSynced, msgSynced)
	return r.getRequeueResult(externalSecret), nil
}

// getRequeueResult create a result with requeueAfter based on the ExternalSecret refresh interval.
func (r *Reconciler) getRequeueResult(externalSecret *esv1beta1.ExternalSecret) ctrl.Result {
	// default to the global requeue interval
	// note, this will never be used because the CRD has a default value of 1 hour
	refreshInterval := r.RequeueInterval
	if externalSecret.Spec.RefreshInterval != nil {
		refreshInterval = externalSecret.Spec.RefreshInterval.Duration
	}

	// if the refresh interval is <= 0, we should not requeue
	if refreshInterval <= 0 {
		return ctrl.Result{}
	}

	timeSinceLastRefresh := 0 * time.Second
	if !externalSecret.Status.RefreshTime.IsZero() {
		lastRefreshTime := externalSecret.Status.RefreshTime.Time
		timeSinceLastRefresh = time.Since(lastRefreshTime)
	}

	// if the last refresh time is in the future, we should requeue immediately
	// this should not happen as we call this function after updating the last refresh time
	// to when the reconciliation started
	if timeSinceLastRefresh < 0 {
		return ctrl.Result{Requeue: true}
	}

	if timeSinceLastRefresh < refreshInterval {
		// requeue after the remaining time
		return ctrl.Result{RequeueAfter: refreshInterval - timeSinceLastRefresh}
	} else {
		// requeue immediately
		return ctrl.Result{Requeue: true}
	}
}

func (r *Reconciler) markAsDone(externalSecret *esv1beta1.ExternalSecret, start time.Time, log logr.Logger, reason, msg string) {
	oldReadyCondition := GetExternalSecretCondition(externalSecret.Status, esv1beta1.ExternalSecretReady)
	newReadyCondition := NewExternalSecretCondition(esv1beta1.ExternalSecretReady, v1.ConditionTrue, reason, msg)
	SetExternalSecretCondition(externalSecret, *newReadyCondition)

	externalSecret.Status.RefreshTime = metav1.NewTime(start)
	externalSecret.Status.SyncedResourceVersion = getResourceVersion(externalSecret)

	// if the status or reason has changed, log at the appropriate verbosity level
	if oldReadyCondition == nil || oldReadyCondition.Status != newReadyCondition.Status || oldReadyCondition.Reason != newReadyCondition.Reason {
		if newReadyCondition.Reason == esv1beta1.ConditionReasonSecretDeleted {
			log.Info("deleted secret")
		} else {
			log.Info("reconciled secret")
		}
	} else {
		log.V(1).Info("reconciled secret")
	}
}

func (r *Reconciler) markAsFailed(msg string, err error, externalSecret *esv1beta1.ExternalSecret, counter prometheus.Counter) {
	r.recorder.Event(externalSecret, v1.EventTypeWarning, esv1beta1.ReasonUpdateFailed, err.Error())
	conditionSynced := NewExternalSecretCondition(esv1beta1.ExternalSecretReady, v1.ConditionFalse, esv1beta1.ConditionReasonSecretSyncedError, msg)
	SetExternalSecretCondition(externalSecret, *conditionSynced)
	counter.Inc()
}

func deleteOrphanedSecrets(ctx context.Context, cl client.Client, externalSecret *esv1beta1.ExternalSecret, secretName string) error {
	ownerLabel := utils.ObjectHash(fmt.Sprintf("%v/%v", externalSecret.Namespace, externalSecret.Name))

	secretListPartial := &metav1.PartialObjectMetadataList{}
	secretListPartial.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("SecretList"))
	listOpts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			esv1beta1.LabelOwner: ownerLabel,
		}),
		Namespace: externalSecret.Namespace,
	}
	if err := cl.List(ctx, secretListPartial, listOpts); err != nil {
		return err
	}

	// delete all secrets that are not the target secret
	for _, secretPartial := range secretListPartial.Items {
		if secretPartial.GetName() != secretName {
			if err := cl.Delete(ctx, &secretPartial); err != nil {
				return err
			}
		}
	}

	return nil
}

// createSecret creates a new secret with the given mutation function.
func (r *Reconciler) createSecret(ctx context.Context, mutationFunc func(secret *v1.Secret) error, es *esv1beta1.ExternalSecret, secretName string) error {
	fqdn := fmt.Sprintf(fieldOwnerTemplate, es.Name)

	// define and mutate the new secret
	newSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: es.Namespace,
		},
		Data: make(map[string][]byte),
	}
	if err := mutationFunc(newSecret); err != nil {
		return err
	}

	// note, we set field owner even for Create
	if err := r.Create(ctx, newSecret, client.FieldOwner(fqdn)); err != nil {
		return err
	}

	// set the binding reference to the secret
	// https://github.com/external-secrets/external-secrets/pull/2263
	es.Status.Binding = v1.LocalObjectReference{Name: newSecret.Name}

	r.recorder.Event(es, v1.EventTypeNormal, esv1beta1.ReasonCreated, "Created Secret")
	return nil
}

func (r *Reconciler) updateSecret(ctx context.Context, existingSecret *v1.Secret, mutationFunc func(secret *v1.Secret) error, es *esv1beta1.ExternalSecret, secretName string) error {
	fqdn := fmt.Sprintf(fieldOwnerTemplate, es.Name)

	// fail if the secret does not exist
	// this should never happen because we check this before calling this function
	if existingSecret.UID == "" {
		return fmt.Errorf(errUpdateNotFound, secretName)
	}

	// set the binding reference to the secret
	// https://github.com/external-secrets/external-secrets/pull/2263
	es.Status.Binding = v1.LocalObjectReference{Name: secretName}

	// mutate a copy of the existing secret with the mutation function
	updatedSecret := existingSecret.DeepCopy()
	if err := mutationFunc(updatedSecret); err != nil {
		return fmt.Errorf(errMutate, updatedSecret.Name, err)
	}

	// if the secret does not need to be updated, return early
	if equality.Semantic.DeepEqual(existingSecret, updatedSecret) {
		return nil
	}

	// if the existing secret is immutable, we can only update the object metadata
	if existingSecret.Immutable != nil && *existingSecret.Immutable {
		// check if the metadata was changed
		metadataChanged := !equality.Semantic.DeepEqual(existingSecret.ObjectMeta, updatedSecret.ObjectMeta)

		// check if the immutable data/type was changed
		var dataChanged bool
		if metadataChanged {
			// update the `existingSecret` object with the metadata from `updatedSecret`
			// this lets us compare the objects to see if the immutable data/type was changed
			existingSecret.ObjectMeta = *updatedSecret.ObjectMeta.DeepCopy()
			dataChanged = !equality.Semantic.DeepEqual(existingSecret, updatedSecret)
		} else {
			// we know there was some change in the secret (or we would have returned early)
			// we know the metadata was NOT changed (metadataChanged == false)
			// so, the only thing that could have changed is the immutable data/type fields
			dataChanged = true
		}

		// because we use labels and annotations to keep track of the secret,
		// we need to update the metadata, even if the data is immutable
		if metadataChanged {
			// NOTE: we are using the `existingSecret` object here, as we ONLY want to update the metadata,
			//       and we previously copied the metadata from the `updatedSecret` object
			if err := r.Update(ctx, existingSecret, client.FieldOwner(fqdn)); err != nil {
				// if we get a conflict, we should return early to requeue immediately
				// note, we don't wrap this error so we can handle it in the caller
				if apierrors.IsConflict(err) {
					return err
				}
				return fmt.Errorf(errUpdate, existingSecret.Name, err)
			}
		}

		// if the immutable data was changed, we should return an error
		if dataChanged {
			return fmt.Errorf(errUpdateImmutable, existingSecret.Name)
		}
	}

	// update the secret
	if err := r.Update(ctx, updatedSecret, client.FieldOwner(fqdn)); err != nil {
		// if we get a conflict, we should return early to requeue immediately
		// note, we don't wrap this error so we can handle it in the caller
		if apierrors.IsConflict(err) {
			return err
		}
		return fmt.Errorf(errUpdate, updatedSecret.Name, err)
	}

	r.recorder.Event(es, v1.EventTypeNormal, esv1beta1.ReasonUpdated, "Updated Secret")
	return nil
}

// getManagedDataKeys returns the list of data keys in a secret which are managed by a specified owner.
func getManagedDataKeys(secret *v1.Secret, fieldOwner string) ([]string, error) {
	return getManagedFieldKeys(secret, fieldOwner, func(fields map[string]any) []string {
		dataFields := fields["f:data"]
		if dataFields == nil {
			return nil
		}
		df, ok := dataFields.(map[string]any)
		if !ok {
			return nil
		}

		return slices.Collect(maps.Keys(df))
	})
}

func getManagedFieldKeys(
	secret *v1.Secret,
	fieldOwner string,
	process func(fields map[string]any) []string,
) ([]string, error) {
	fqdn := fmt.Sprintf(fieldOwnerTemplate, fieldOwner)
	var keys []string
	for _, v := range secret.ObjectMeta.ManagedFields {
		if v.Manager != fqdn {
			continue
		}
		fields := make(map[string]any)
		err := json.Unmarshal(v.FieldsV1.Raw, &fields)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling managed fields: %w", err)
		}
		for _, key := range process(fields) {
			if key == "." {
				continue
			}
			keys = append(keys, strings.TrimPrefix(key, "f:"))
		}
	}
	return keys, nil
}

func getResourceVersion(es *esv1beta1.ExternalSecret) string {
	return fmt.Sprintf("%d-%s", es.ObjectMeta.GetGeneration(), hashMeta(es.ObjectMeta))
}

// hashMeta returns a consistent hash of the `metadata.labels` and `metadata.annotations` fields of the given object.
func hashMeta(m metav1.ObjectMeta) string {
	type meta struct {
		annotations map[string]string
		labels      map[string]string
	}
	objectMeta := meta{
		annotations: m.Annotations,
		labels:      m.Labels,
	}
	return utils.ObjectHash(objectMeta)
}

func shouldSkipClusterSecretStore(r *Reconciler, es *esv1beta1.ExternalSecret) bool {
	return !r.ClusterSecretStoreEnabled && es.Spec.SecretStoreRef.Kind == esv1beta1.ClusterSecretStoreKind
}

// shouldSkipUnmanagedStore iterates over all secretStore references in the externalSecret spec,
// fetches the store and evaluates the controllerClass property.
// Returns true if any storeRef points to store with a non-matching controllerClass.
func shouldSkipUnmanagedStore(ctx context.Context, namespace string, r *Reconciler, es *esv1beta1.ExternalSecret) (bool, error) {
	var storeList []esv1beta1.SecretStoreRef

	if es.Spec.SecretStoreRef.Name != "" {
		storeList = append(storeList, es.Spec.SecretStoreRef)
	}

	for _, ref := range es.Spec.Data {
		if ref.SourceRef != nil {
			storeList = append(storeList, ref.SourceRef.SecretStoreRef)
		}
	}

	for _, ref := range es.Spec.DataFrom {
		if ref.SourceRef != nil && ref.SourceRef.SecretStoreRef != nil {
			storeList = append(storeList, *ref.SourceRef.SecretStoreRef)
		}

		// verify that generator's controllerClass matches
		if ref.SourceRef != nil && ref.SourceRef.GeneratorRef != nil {
			_, obj, err := resolvers.GeneratorRef(ctx, r.RestConfig, namespace, ref.SourceRef.GeneratorRef)
			if err != nil {
				return false, err
			}
			skipGenerator, err := shouldSkipGenerator(r, obj)
			if err != nil {
				return false, err
			}
			if skipGenerator {
				return true, nil
			}
		}
	}

	for _, ref := range storeList {
		var store esv1beta1.GenericStore

		switch ref.Kind {
		case esv1beta1.SecretStoreKind, "":
			store = &esv1beta1.SecretStore{}
		case esv1beta1.ClusterSecretStoreKind:
			store = &esv1beta1.ClusterSecretStore{}
			namespace = ""
		}

		err := r.Get(ctx, types.NamespacedName{
			Name:      ref.Name,
			Namespace: namespace,
		}, store)
		if err != nil {
			return false, err
		}
		class := store.GetSpec().Controller
		if class != "" && class != r.ControllerClass {
			return true, nil
		}
	}
	return false, nil
}

func shouldRefresh(es *esv1beta1.ExternalSecret) bool {
	// if the refresh interval is 0, and we have synced previously, we should not refresh
	if es.Spec.RefreshInterval.Duration <= 0 && es.Status.SyncedResourceVersion != "" {
		return false
	}

	// if the ExternalSecret has been updated, we should refresh
	if es.Status.SyncedResourceVersion != getResourceVersion(es) {
		return true
	}

	// if the last refresh time is zero, we should refresh
	if es.Status.RefreshTime.IsZero() {
		return true
	}

	// if the last refresh time + refresh interval is before now, we should refresh
	return es.Status.RefreshTime.Add(es.Spec.RefreshInterval.Duration).Before(time.Now())
}

// isSecretValid checks if the secret exists, and it's data is consistent with the calculated hash.
func isSecretValid(existingSecret *v1.Secret) bool {
	// if target secret doesn't exist, we need to refresh
	if existingSecret.UID == "" {
		return false
	}

	// if the managed label is missing or incorrect, then it's invalid
	if existingSecret.Labels[esv1beta1.LabelManaged] != esv1beta1.LabelManagedValue {
		return false
	}

	// if the data-hash annotation is missing or incorrect, then it's invalid
	// this is how we know if the data has chanced since we last updated the secret
	if existingSecret.Annotations[esv1beta1.AnnotationDataHash] != utils.ObjectHash(existingSecret.Data) {
		return false
	}

	return true
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("external-secrets")

	// index ExternalSecrets based on the target secret name,
	// this lets us quickly find all ExternalSecrets which target a specific Secret
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &esv1beta1.ExternalSecret{}, indexESTargetSecretNameField, func(obj client.Object) []string {
		es := obj.(*esv1beta1.ExternalSecret)
		// if the target name is set, use that as the index
		if es.Spec.Target.Name != "" {
			return []string{es.Spec.Target.Name}
		} else {
			// otherwise, use the ExternalSecret name
			return []string{es.Name}
		}
	}); err != nil {
		return err
	}

	// predicate function to ignore secret events unless they have the "managed" label
	secretHasESLabel := predicate.NewPredicateFuncs(func(object client.Object) bool {
		value, hasLabel := object.GetLabels()[esv1beta1.LabelManaged]
		return hasLabel && value == esv1beta1.LabelManagedValue
	})

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esv1beta1.ExternalSecret{}).
		// we cant use Owns(), as we don't set ownerReferences when the creationPolicy is not Owner.
		// we use WatchesMetadata() to reduce memory usage, as otherwise we have to process full secret objects.
		WatchesMetadata(
			&v1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, secretHasESLabel),
		).
		Complete(r)
}

func (r *Reconciler) findObjectsForSecret(ctx context.Context, secret client.Object) []reconcile.Request {
	externalSecretsList := &esv1beta1.ExternalSecretList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(indexESTargetSecretNameField, secret.GetName()),
		Namespace:     secret.GetNamespace(),
	}
	err := r.List(ctx, externalSecretsList, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(externalSecretsList.Items))
	for i, item := range externalSecretsList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

func BuildManagedSecretClient(mgr ctrl.Manager) (client.Client, error) {
	// secrets we manage will have the `reconcile.external-secrets.io/managed=true` label
	managedLabelReq, _ := labels.NewRequirement(esv1beta1.LabelManaged, selection.Equals, []string{esv1beta1.LabelManagedValue})
	managedLabelSelector := labels.NewSelector().Add(*managedLabelReq)

	// create a new cache with a label selector for managed secrets
	// NOTE: this means that the cache/client will be unable to see secrets without the "managed" label
	secretCacheOpts := cache.Options{
		HTTPClient: mgr.GetHTTPClient(),
		Scheme:     mgr.GetScheme(),
		Mapper:     mgr.GetRESTMapper(),
		ByObject: map[client.Object]cache.ByObject{
			&v1.Secret{}: {
				Label: managedLabelSelector,
			},
		},
		// this requires us to explicitly start an informer for each object type
		// and helps avoid people mistakenly using the secret client for other resources
		ReaderFailOnMissingInformer: true,
	}
	secretCache, err := cache.New(mgr.GetConfig(), secretCacheOpts)
	if err != nil {
		return nil, err
	}

	// start an informer for secrets
	// this is required because we set ReaderFailOnMissingInformer to true
	_, err = secretCache.GetInformer(context.Background(), &v1.Secret{})
	if err != nil {
		return nil, err
	}

	// add the secret cache to the manager, so that it starts at the same time
	err = mgr.Add(secretCache)
	if err != nil {
		return nil, err
	}

	// create a new client that uses the secret cache
	secretClient, err := client.New(mgr.GetConfig(), client.Options{
		HTTPClient: mgr.GetHTTPClient(),
		Scheme:     mgr.GetScheme(),
		Mapper:     mgr.GetRESTMapper(),
		Cache: &client.CacheOptions{
			Reader: secretCache,
		},
	})
	if err != nil {
		return nil, err
	}

	return secretClient, nil
}
