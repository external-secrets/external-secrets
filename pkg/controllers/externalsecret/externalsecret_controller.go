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
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/templating"
	"github.com/external-secrets/external-secrets/pkg/utils"

	// Loading registered generators and providers.
	_ "github.com/external-secrets/external-secrets/pkg/generator/register"
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

const (
	// condition messages for "SecretSynced" reason.
	msgSynced       = "secret synced"
	msgSyncedRetain = "secret retained due to DeletionPolicy=Retain"

	// condition messages for "SecretDeleted" reason.
	msgDeleted = "secret deleted due to DeletionPolicy=Delete"

	// condition messages for "SecretMissing" reason.
	msgMissing = "secret will not be created due to CreationPolicy=Merge"

	// log messages.
	logErrorGetES                = "unable to get ExternalSecret"
	logErrorUpdateESStatus       = "unable to update ExternalSecret status"
	logErrorGetSecret            = "unable to get Secret"
	logErrorPatchSecret          = "unable to patch Secret"
	logErrorSecretCacheNotSynced = "controller caches for Secret are not in sync"
	logErrorUnmanagedStore       = "unable to determine if store is managed"

	// error formats.
	errConvert                  = "error applying conversion strategy %s to keys: %w"
	errRewrite                  = "error applying rewrite to keys: %w"
	errDecode                   = "error applying decoding strategy %s to data: %w"
	errGenerate                 = "error using generator: %w"
	errInvalidKeys              = "invalid secret keys (TIP: use rewrite or conversionStrategy to change keys): %w"
	errFetchTplFrom             = "could not fetch templateFrom data: %w"
	errApplyTemplate            = "could not apply template: %w"
	errExecTpl                  = "could not execute template: %w"
	errMutate                   = "could not mutate secret %s: %w"
	errCreate                   = "could not create secret %s: %w"
	errUpdate                   = "could not update secret %s: %w"
	errUpdateNotFound           = "could not update secret %s: not found"
	errDeleteSecret             = "could not delete secret %s: %w"
	errDeleteSecretOrphaned     = "could not delete orphaned secrets: %w"
	errDeleteSecretCreatePolicy = "unable to delete secret %s: creationPolicy=%s is not Owner"
	errSecretCachesNotSynced    = "controller caches for secret %s are not in sync"
	errSourcesNotExists         = "some referenced data sources do not exist: %v"
	errSourcesNotReady          = "some referenced data sources are not ready: %v"
	errProviderData             = "could not get data from providers: %w"
	errDefaultStoreMissing      = "default store is missing (spec.secretStoreRef), but some entries omit store references (spec.data[] or spec.dataFrom[])"
	errBecomeOwner              = "could not take ownership of secret %s: %w"
	errRemoveOwner              = "could not remove ownership from secret %s: %w"
	errSpecData                 = "error processing spec.data[%d] (key: %s), err: %w"
	errSpecDataFromFind         = "error processing spec.dataFrom[%d].find, err: %w"
	errSpecDataFromExtract      = "error processing spec.dataFrom[%d].extract (key: %s), err: %w"
	errSpecDataFromGenerator    = "error processing spec.dataFrom[%d].sourceRef.generatorRef, err: %w"

	// event messages.
	eventCreated         = "secret created"
	eventUpdated         = "secret updated"
	eventDeleted         = "secret deleted due to DeletionPolicy=Delete"
	eventDeletedOrphaned = "secret deleted because it was orphaned"
)

// these errors are explicitly defined so we can detect them with `errors.Is()`.
var (
	ErrSecretImmutable     = fmt.Errorf("secret is immutable")
	ErrSecretIsOwned       = fmt.Errorf("secret is owned by another ExternalSecret")
	ErrSecretSetCtrlRef    = fmt.Errorf("could not set controller reference on secret")
	ErrSecretRemoveCtrlRef = fmt.Errorf("could not remove controller reference from secret")
)

const indexESTargetSecretNameField = ".metadata.targetSecretName"
const indexESTriggerSecretsField = ".metadata.triggerSecrets"

// Reconciler reconciles a ExternalSecret object.
type Reconciler struct {
	client.Client
	SecretClient                  client.Client
	Log                           logr.Logger
	Scheme                        *runtime.Scheme
	RestConfig                    *rest.Config
	ControllerClass               string
	RequeueInterval               time.Duration
	ClusterSecretStoreEnabled     bool
	EnableFloodGate               bool
	EnableTriggerInClusterSecrets bool
	recorder                      record.EventRecorder
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
		log.V(1).Info("skipping ExternalSecret, it is marked for deletion")
		return ctrl.Result{}, nil
	}

	// if extended metrics is enabled, refine the time series vector
	resourceLabels = ctrlmetrics.RefineLabels(resourceLabels, externalSecret.Labels)

	// skip this ExternalSecret if it uses a ClusterSecretStore and the feature is disabled
	if r.shouldSkipClusterSecretStore(externalSecret) {
		log.V(1).Info("skipping ExternalSecret, ClusterSecretStore feature is disabled")
		return ctrl.Result{}, nil
	}

	// skip this ExternalSecret if it uses any SecretStore not managed by this controller
	// NOTE: we capture information about some sync errors but do not act on them until
	//       later in the reconciliation so we can update the status of the ExternalSecret
	//       (notExistSources, notReadySources, defaultStoreMissing)
	skip, sourcesResult, err := r.getDataSourcesOrSkip(ctx, externalSecret)
	if err != nil {
		log.Error(err, logErrorUnmanagedStore)
		syncCallsError.With(resourceLabels).Inc()
		return ctrl.Result{}, err
	}
	if skip {
		log.V(1).Info("skipping ExternalSecret, it uses unmanaged SecretStores or Generators")
		return ctrl.Result{}, nil
	}

	// the target secret name defaults to the ExternalSecret name, if not explicitly set
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
		fqdn := fmt.Sprintf(templating.FieldOwnerTemplate, externalSecret.Name)
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
	// NOTE: we are using the `r.SecretClient` which we only use for managed secrets.
	//       when `enableManagedSecretsCache` is true, this is a cached client that only sees our managed secrets,
	//       otherwise it will be the normal controller-runtime client which may be cached or make direct API calls,
	//       depending on if `enabledSecretCache` is true or false.
	existingSecret := &v1.Secret{}
	err = r.SecretClient.Get(ctx, client.ObjectKey{Name: secretName, Namespace: externalSecret.Namespace}, existingSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, logErrorGetSecret, "secretName", secretName, "secretNamespace", externalSecret.Namespace)
		syncCallsError.With(resourceLabels).Inc()
		return ctrl.Result{}, err
	}

	// ensure the full cache is up-to-date
	// NOTE: this prevents race conditions between the partial and full cache.
	//       we return an error so we get an exponential backoff if we end up looping,
	//       for example, during high cluster load and frequent updates to the target secret by other controllers.
	if secretPartial.UID != existingSecret.UID || secretPartial.ResourceVersion != existingSecret.ResourceVersion {
		err = fmt.Errorf(errSecretCachesNotSynced, secretName)
		log.Error(err, logErrorSecretCacheNotSynced, "secretName", secretName, "secretNamespace", externalSecret.Namespace)
		syncCallsError.With(resourceLabels).Inc()
		return ctrl.Result{}, err
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
	// NOTE: we use the ability of deferred functions to update named return values `result` and `err`
	// NOTE: we dereference the DeepCopy of the status field because status fields are NOT pointers,
	//       so otherwise the `equality.Semantic.DeepEqual` will always return false.
	currentStatus := *externalSecret.Status.DeepCopy()
	defer func() {
		// use the ssInfoMap and cssInfoMap to update the sources and triggers in the status
		r.updateStatusSourcesAndTriggers(externalSecret, sourcesResult.ssInfoMap, sourcesResult.cssInfoMap)

		// if the status has not changed, we don't need to update it
		// WARNING: be VERY careful to ensure that you haven't set empty `omitempty` fields to their empty values,
		//          as when we get the ExternalSecret from the API server, these fields will be seen as different
		//          from your true empty values, and the status will be updated every time.
		if equality.Semantic.DeepEqual(currentStatus, externalSecret.Status) {
			return
		}

		// update the status of the ExternalSecret, storing any error in a new variable
		// if there was no new error, we don't need to change the `result` or `err` values
		updateErr := r.Status().Update(ctx, externalSecret)
		if updateErr == nil {
			return
		}

		// if we got an update conflict, we should requeue immediately
		if apierrors.IsConflict(updateErr) {
			log.V(1).Info("conflict while updating status, will requeue")

			// we only explicitly request a requeue if the main function did not return an `err`.
			// otherwise, we get an annoying log saying that results are ignored when there is an error,
			// as errors are always retried.
			if err == nil {
				result = ctrl.Result{Requeue: true}
			}
			return
		}

		// for other errors, log and update the `err` variable if there is no error already
		// so the reconciler will requeue the request
		log.Error(updateErr, logErrorUpdateESStatus)
		if err == nil {
			err = updateErr
		}
	}()

	// handle errors we detected before this point
	// NOTE: we were holding on to these errors as before this point, we were
	//       still determining if we should be doing a sync or not, so could not
	//       update the status of the ExternalSecret.
	if sourcesResult.defaultStoreMissing {
		// NOTE: this error cant be fixed by retrying so we don't return an error (which would requeue immediately)
		err = errors.New(errDefaultStoreMissing)
		r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
		return ctrl.Result{}, nil
	}
	if len(sourcesResult.notExistSources) > 0 {
		err = fmt.Errorf(errSourcesNotExists, sourcesResult.notExistSources)
		r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
		return ctrl.Result{}, err
	}
	if r.EnableFloodGate && len(sourcesResult.notReadySources) > 0 {
		err = fmt.Errorf(errSourcesNotReady, sourcesResult.notReadySources)
		r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
		return ctrl.Result{}, err
	}

	// retrieve the provider secret data.
	dataMap, err := r.getProviderSecretData(ctx, externalSecret, sourcesResult.ssInfoMap, sourcesResult.cssInfoMap)
	if err != nil {
		err = fmt.Errorf(errProviderData, err)
		r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
		return ctrl.Result{}, err
	}

	// if no data was found we can delete the secret if needed.
	if len(dataMap) == 0 {
		switch externalSecret.Spec.Target.DeletionPolicy {
		// delete secret and return early.
		case esv1beta1.DeletionPolicyDelete:
			// safeguard that we only can delete secrets we own.
			// this is also implemented in the es validation webhook.
			// NOTE: this error cant be fixed by retrying so we don't return an error (which would requeue immediately)
			creationPolicy := externalSecret.Spec.Target.CreationPolicy
			if creationPolicy != esv1beta1.CreatePolicyOwner {
				err = fmt.Errorf(errDeleteSecretCreatePolicy, secretName, creationPolicy)
				r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
				return ctrl.Result{}, nil
			}

			// delete the secret, if it exists
			if existingSecret.UID != "" {
				err = r.Delete(ctx, existingSecret)
				if err != nil && !apierrors.IsNotFound(err) {
					err = fmt.Errorf(errDeleteSecret, secretName, err)
					r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
					return ctrl.Result{}, err
				}
				r.recorder.Event(externalSecret, v1.EventTypeNormal, esv1beta1.EventReasonDeleted, eventDeleted)
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

	switch externalSecret.Spec.Target.CreationPolicy {
	case esv1beta1.CreatePolicyNone:
		log.V(1).Info("secret creation skipped due to CreationPolicy=None")
		err = nil
	case esv1beta1.CreatePolicyMerge:
		// update the secret, if it exists
		if existingSecret.UID != "" {
			err = r.updateSecret(ctx, existingSecret, externalSecret, dataMap, secretName)
		} else {
			// if the secret does not exist, we wait until the next refresh interval
			// rather than returning an error which would requeue immediately
			r.markAsDone(externalSecret, start, log, esv1beta1.ConditionReasonSecretMissing, msgMissing)
			return r.getRequeueResult(externalSecret), nil
		}
	case esv1beta1.CreatePolicyOrphan:
		// create the secret, if it does not exist
		if existingSecret.UID == "" {
			err = r.createSecret(ctx, externalSecret, dataMap, secretName)
		} else {
			// if the secret exists, we should update it
			err = r.updateSecret(ctx, existingSecret, externalSecret, dataMap, secretName)
		}
	case esv1beta1.CreatePolicyOwner:
		// we may have orphaned secrets to clean up,
		// for example, if the target secret name was changed
		err = r.deleteOrphanedSecrets(ctx, externalSecret, secretName)
		if err != nil {
			err = fmt.Errorf(errDeleteSecretOrphaned, err)
			r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
			return ctrl.Result{}, err
		}

		// create the secret, if it does not exist
		if existingSecret.UID == "" {
			err = r.createSecret(ctx, externalSecret, dataMap, secretName)
		} else {
			// if the secret exists, we should update it
			err = r.updateSecret(ctx, existingSecret, externalSecret, dataMap, secretName)
		}
	}
	if err != nil {
		// if we got an update conflict, we should requeue immediately
		if apierrors.IsConflict(err) {
			log.V(1).Info("conflict while updating secret, will requeue")
			return ctrl.Result{Requeue: true}, nil
		}

		// detect errors indicating that we failed to set ourselves as the owner of the secret
		// NOTE: this error cant be fixed by retrying so we don't return an error (which would requeue immediately)
		if errors.Is(err, ErrSecretSetCtrlRef) || errors.Is(err, ErrSecretIsOwned) {
			err = fmt.Errorf(errBecomeOwner, secretName, err)
			r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
			return ctrl.Result{}, nil
		}

		// detect errors indicating that we failed to remove ourselves as the owner of the secret
		// NOTE: this error cant be fixed by retrying so we don't return an error (which would requeue immediately)
		if errors.Is(err, ErrSecretRemoveCtrlRef) {
			err = fmt.Errorf(errRemoveOwner, secretName, err)
			r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
			return ctrl.Result{}, nil
		}

		// detect errors indicating that the secret is immutable
		// NOTE: this error cant be fixed by retrying so we don't return an error (which would requeue immediately)
		if errors.Is(err, ErrSecretImmutable) {
			r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
			return ctrl.Result{}, nil
		}

		r.markAsFailed(err, externalSecret, syncCallsError.With(resourceLabels))
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

	// if the last refresh time is not set, requeue after the refresh interval
	// note, this should not happen, as we only call this function on ExternalSecrets
	// that have been reconciled at least once
	if externalSecret.Status.RefreshTime.IsZero() {
		return ctrl.Result{RequeueAfter: refreshInterval}
	}

	timeSinceLastRefresh := time.Since(externalSecret.Status.RefreshTime.Time)

	// if the last refresh time is in the future, we should requeue immediately
	// note, this should not happen, as we always refresh an ExternalSecret
	// that has a last refresh time in the future
	if timeSinceLastRefresh < 0 {
		return ctrl.Result{Requeue: true}
	}

	// if there is time remaining, requeue after the remaining time
	if timeSinceLastRefresh < refreshInterval {
		return ctrl.Result{RequeueAfter: refreshInterval - timeSinceLastRefresh}
	}

	// otherwise, requeue immediately
	return ctrl.Result{Requeue: true}
}

// updateStatusSourcesAndTriggers updates the status of the ExternalSecret with the data sources and triggers.
func (r *Reconciler) updateStatusSourcesAndTriggers(externalSecret *esv1beta1.ExternalSecret, ssInfoMap, cssInfoMap map[string]*StoreInfo) {
	// create a map to track the in-cluster trigger secrets we have found (for kubernetes providers)
	foundTriggerSecrets := make(map[esv1beta1.TriggerInClusterSecret]bool)

	// compareProviderSources compares two ProviderSourceInfo objects and returns an integer indicating their order.
	compareProviderSources := func(a, b esv1beta1.ProviderSourceInfo) int {
		return strings.Compare(a.Name, b.Name)
	}

	// compareTriggerSecrets compares two TriggerInClusterSecret objects and returns an integer indicating their order.
	compareTriggerSecrets := func(a, b esv1beta1.TriggerInClusterSecret) int {
		if n := strings.Compare(a.Namespace, b.Namespace); n != 0 {
			return n
		}
		return strings.Compare(a.Name, b.Name)
	}

	// convert the SecretStore map into a slice of ProviderSourceInfo
	// NOTE: we sort the resulting slice so the status is deterministic
	ssProviderSources := make([]esv1beta1.ProviderSourceInfo, 0, len(ssInfoMap))
	for storeName, storeInfo := range ssInfoMap {
		sourceInfo := r.storeInfoToProviderSource(storeName, storeInfo, foundTriggerSecrets)
		ssProviderSources = append(ssProviderSources, sourceInfo)
	}
	slices.SortFunc(ssProviderSources, compareProviderSources)

	// convert the ClusterSecretStore map into a slice of ProviderSourceInfo
	// NOTE: we sort the resulting slice so the status is deterministic
	cssProviderSources := make([]esv1beta1.ProviderSourceInfo, 0, len(cssInfoMap))
	for storeName, storeInfo := range cssInfoMap {
		sourceInfo := r.storeInfoToProviderSource(storeName, storeInfo, foundTriggerSecrets)
		cssProviderSources = append(cssProviderSources, sourceInfo)
	}
	slices.SortFunc(cssProviderSources, compareProviderSources)

	// convert the foundTriggerSecrets map into a slice of TriggerInClusterSecret
	// NOTE: we sort the resulting slice so the status is deterministic
	triggerSecretsSlice := make([]esv1beta1.TriggerInClusterSecret, 0, len(foundTriggerSecrets))
	for triggerSecret := range foundTriggerSecrets {
		triggerSecretsSlice = append(triggerSecretsSlice, triggerSecret)
	}
	slices.SortFunc(triggerSecretsSlice, compareTriggerSecrets)

	// update the status of the ExternalSecret
	externalSecret.Status.Sources = esv1beta1.ExternalSecretStatusSources{
		SecretStores:        ssProviderSources,
		ClusterSecretStores: cssProviderSources,
	}
	externalSecret.Status.Triggers = esv1beta1.ExternalSecretStatusTriggers{
		InClusterSecrets: triggerSecretsSlice,
	}
}

// storeInfoToProviderSource converts a StoreInfo to a ProviderSourceInfo and updates the foundTriggerSecrets map.
func (r *Reconciler) storeInfoToProviderSource(storeName string, storeInfo *StoreInfo, foundTriggerSecrets map[esv1beta1.TriggerInClusterSecret]bool) esv1beta1.ProviderSourceInfo {
	// compareSourceListKeys compares two SourceListedKey objects and returns an integer indicating their order.
	compareSourceListKeys := func(a, b esv1beta1.SourceListedKey) int {
		return strings.Compare(a.Key, b.Key)
	}

	// compareSourceFoundKeys compares two SourceFoundKey objects and returns an integer indicating their order.
	compareSourceFoundKeys := func(a, b esv1beta1.SourceFoundKey) int {
		return strings.Compare(a.Key, b.Key)
	}

	// convert the listedKeys map to a slice
	// NOTE: we sort the resulting slice so the status is deterministic
	listedKeys := make([]esv1beta1.SourceListedKey, 0, len(storeInfo.ListedKeys))
	for key, existsInProvider := range storeInfo.ListedKeys {
		sourceKey := esv1beta1.SourceListedKey{
			Key: key,
		}
		if !existsInProvider {
			// NOTE: we set this conditionally so that false values are not serialized in the status to save space
			sourceKey.NotExists = true
		}
		listedKeys = append(listedKeys, sourceKey)

		// if this is an in-cluster kubernetes provider, add these keys to the trigger secrets
		if r.EnableTriggerInClusterSecrets && storeInfo.InClusterKubernetesNamespace != "" {
			triggerSecret := esv1beta1.TriggerInClusterSecret{
				Name:      key,
				Namespace: storeInfo.InClusterKubernetesNamespace,
			}
			foundTriggerSecrets[triggerSecret] = true
		}
	}
	slices.SortFunc(listedKeys, compareSourceListKeys)

	// convert the foundKeys map to a slice
	// NOTE: we sort the resulting slice so the status is deterministic
	foundKeys := make([]esv1beta1.SourceFoundKey, 0, len(storeInfo.FoundKeys))
	for key := range storeInfo.FoundKeys {
		sourceKey := esv1beta1.SourceFoundKey{
			Key: key,
		}
		foundKeys = append(foundKeys, sourceKey)

		// if this is an in-cluster kubernetes provider, add these keys to the trigger secrets
		if r.EnableTriggerInClusterSecrets && storeInfo.InClusterKubernetesNamespace != "" {
			triggerSecret := esv1beta1.TriggerInClusterSecret{
				Name:      key,
				Namespace: storeInfo.InClusterKubernetesNamespace,
			}
			foundTriggerSecrets[triggerSecret] = true
		}
	}
	slices.SortFunc(foundKeys, compareSourceFoundKeys)

	providerInfo := esv1beta1.ProviderSourceInfo{
		Name:       storeName,
		ListedKeys: listedKeys,
		FoundKeys:  foundKeys,
	}

	// set the NotExists and NotReady flags if they are true
	// NOTE: we set them conditionally so that false values are not serialized in the status to save space
	if storeInfo.NotExists {
		providerInfo.NotExists = true
	}
	if storeInfo.NotReady {
		providerInfo.NotReady = true
	}

	return providerInfo
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

func (r *Reconciler) markAsFailed(err error, externalSecret *esv1beta1.ExternalSecret, counter prometheus.Counter) {
	conditionSynced := NewExternalSecretCondition(esv1beta1.ExternalSecretReady, v1.ConditionFalse, esv1beta1.ConditionReasonSecretSyncedError, err.Error())
	SetExternalSecretCondition(externalSecret, *conditionSynced)
	counter.Inc()
}

func (r *Reconciler) deleteOrphanedSecrets(ctx context.Context, externalSecret *esv1beta1.ExternalSecret, secretName string) error {
	ownerLabel := utils.ObjectHash(fmt.Sprintf("%v/%v", externalSecret.Namespace, externalSecret.Name))

	// we use a PartialObjectMetadataList to avoid loading the full secret objects
	// and because the Secrets partials are always cached due to WatchesMetadata() in SetupWithManager()
	secretListPartial := &metav1.PartialObjectMetadataList{}
	secretListPartial.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("SecretList"))
	listOpts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			esv1beta1.LabelOwner: ownerLabel,
		}),
		Namespace: externalSecret.Namespace,
	}
	if err := r.List(ctx, secretListPartial, listOpts); err != nil {
		return err
	}

	// delete all secrets that are not the target secret
	for _, secretPartial := range secretListPartial.Items {
		if secretPartial.GetName() != secretName {
			err := r.Delete(ctx, &secretPartial)
			if err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf(errDeleteSecret, secretPartial.GetName(), err)
			}
			r.recorder.Event(externalSecret, v1.EventTypeNormal, esv1beta1.EventReasonDeleted, eventDeletedOrphaned)
		}
	}

	return nil
}

// mutateSecret mutates the provided secret object to match the desired state of an ExternalSecret.
func (r *Reconciler) mutateSecret(ctx context.Context, es *esv1beta1.ExternalSecret, secret *v1.Secret, dataMap map[string][]byte) error {
	// get information about the current owner of the secret
	//  - we ignore the API version as it can change over time
	//  - we ignore the UID for consistency with the SetControllerReference function
	currentOwner := metav1.GetControllerOf(secret)
	ownerIsESKind := false
	ownerIsCurrentES := false
	if currentOwner != nil {
		currentOwnerGK := schema.FromAPIVersionAndKind(currentOwner.APIVersion, currentOwner.Kind).GroupKind()
		ownerIsESKind = currentOwnerGK.String() == esv1beta1.ExtSecretGroupKind
		ownerIsCurrentES = ownerIsESKind && currentOwner.Name == es.Name
	}

	// if another ExternalSecret is the owner, we should return an error
	// otherwise the controller will fight with itself to update the secret.
	// note, this does not prevent other controllers from owning the secret.
	if ownerIsESKind && !ownerIsCurrentES {
		return fmt.Errorf("%w: %s", ErrSecretIsOwned, currentOwner.Name)
	}

	// if the CreationPolicy is Owner, we should set ourselves as the owner of the secret
	if es.Spec.Target.CreationPolicy == esv1beta1.CreatePolicyOwner {
		err := controllerutil.SetControllerReference(es, secret, r.Scheme)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrSecretSetCtrlRef, err)
		}
	}

	// if the creation policy is not Owner, we should remove ourselves as the owner
	// this could happen if the creation policy was changed after the secret was created
	if es.Spec.Target.CreationPolicy != esv1beta1.CreatePolicyOwner && ownerIsCurrentES {
		err := controllerutil.RemoveControllerReference(es, secret, r.Scheme)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrSecretRemoveCtrlRef, err)
		}
	}

	// WARNING: this will remove any labels or annotations managed by this ExternalSecret
	//          so any updates to labels and annotations should be done AFTER this point
	err := r.applyTemplate(ctx, es, secret, dataMap)
	if err != nil {
		return fmt.Errorf(errApplyTemplate, err)
	}

	// set the immutable flag on the secret if requested by the ExternalSecret
	if es.Spec.Target.Immutable {
		secret.Immutable = ptr.To(true)
	}

	// we also use a label to keep track of the owner of the secret
	// this lets us remove secrets that are no longer needed if the target secret name changes
	if es.Spec.Target.CreationPolicy == esv1beta1.CreatePolicyOwner {
		lblValue := utils.ObjectHash(fmt.Sprintf("%v/%v", es.Namespace, es.Name))
		secret.Labels[esv1beta1.LabelOwner] = lblValue
	} else {
		// the label should not be set if the creation policy is not Owner
		delete(secret.Labels, esv1beta1.LabelOwner)
	}

	secret.Labels[esv1beta1.LabelManaged] = esv1beta1.LabelManagedValue
	secret.Annotations[esv1beta1.AnnotationDataHash] = utils.ObjectHash(secret.Data)

	return nil
}

// createSecret creates a new secret with the given mutation function.
func (r *Reconciler) createSecret(ctx context.Context, es *esv1beta1.ExternalSecret, dataMap map[string][]byte, secretName string) error {
	fqdn := fmt.Sprintf(templating.FieldOwnerTemplate, es.Name)

	// define and mutate the new secret
	newSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: es.Namespace,
		},
		Data: make(map[string][]byte),
	}
	if err := r.mutateSecret(ctx, es, newSecret, dataMap); err != nil {
		return fmt.Errorf(errMutate, newSecret.Name, err)
	}

	// note, we set field owner even for Create
	if err := r.Create(ctx, newSecret, client.FieldOwner(fqdn)); err != nil {
		return fmt.Errorf(errCreate, newSecret.Name, err)
	}

	// set the binding reference to the secret
	// https://github.com/external-secrets/external-secrets/pull/2263
	es.Status.Binding = v1.LocalObjectReference{Name: newSecret.Name}

	r.recorder.Event(es, v1.EventTypeNormal, esv1beta1.EventReasonCreated, eventCreated)
	return nil
}

func (r *Reconciler) updateSecret(ctx context.Context, existingSecret *v1.Secret, es *esv1beta1.ExternalSecret, dataMap map[string][]byte, secretName string) error {
	fqdn := fmt.Sprintf(templating.FieldOwnerTemplate, es.Name)

	// fail if the secret does not exist
	// this should never happen because we check this before calling this function
	if existingSecret.UID == "" {
		return fmt.Errorf(errUpdateNotFound, existingSecret.Name)
	}

	// set the binding reference to the secret
	// https://github.com/external-secrets/external-secrets/pull/2263
	es.Status.Binding = v1.LocalObjectReference{Name: secretName}

	// mutate a copy of the existing secret with the mutation function
	updatedSecret := existingSecret.DeepCopy()
	if err := r.mutateSecret(ctx, es, updatedSecret, dataMap); err != nil {
		return fmt.Errorf(errMutate, updatedSecret.Name, err)
	}

	// if the secret does not need to be updated, return early
	if equality.Semantic.DeepEqual(existingSecret, updatedSecret) {
		return nil
	}

	// if the existing secret is immutable, we can only update the object metadata
	if ptr.Deref(existingSecret.Immutable, false) {
		// check if the metadata was changed
		metadataChanged := !equality.Semantic.DeepEqual(existingSecret.ObjectMeta, updatedSecret.ObjectMeta)

		// check if the immutable data/type was changed
		var dataChanged bool
		if metadataChanged {
			// update the `existingSecret` object with the metadata from `updatedSecret`
			// this lets us compare the objects to see if the immutable data/type was changed
			existingSecret.ObjectMeta = *updatedSecret.ObjectMeta.DeepCopy()
			dataChanged = !equality.Semantic.DeepEqual(existingSecret, updatedSecret)

			// because we use labels and annotations to keep track of the secret,
			// we need to update the metadata, regardless of if the immutable data was changed
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
		} else {
			// we know there was some change in the secret (or we would have returned early)
			// we know the metadata was NOT changed (metadataChanged == false)
			// so, the only thing that could have changed is the immutable data/type fields
			dataChanged = true
		}

		// if the immutable data was changed, we should return an error
		if dataChanged {
			return fmt.Errorf(errUpdate, existingSecret.Name, ErrSecretImmutable)
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

	r.recorder.Event(es, v1.EventTypeNormal, esv1beta1.EventReasonUpdated, eventUpdated)
	return nil
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

	// if the last refresh time is in the future, we should refresh
	if es.Status.RefreshTime.Time.After(time.Now()) {
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
		}
		// otherwise, use the ExternalSecret name
		return []string{es.Name}
	}); err != nil {
		return err
	}

	// index ExternalSecrets based on the in-cluster secret triggers
	// this lets us quickly find all ExternalSecrets which should be reconciled when a secret changes
	if r.EnableTriggerInClusterSecrets {
		if err := mgr.GetFieldIndexer().IndexField(context.Background(), &esv1beta1.ExternalSecret{}, indexESTriggerSecretsField, func(obj client.Object) []string {
			es := obj.(*esv1beta1.ExternalSecret)

			// convert the in-cluster secret triggers in the status to a list of index values
			// with the format `<SECRET_NAMESPACE>/<SECRET_NAME>`
			indexValues := make([]string, len(es.Status.Triggers.InClusterSecrets))
			for i, trigger := range es.Status.Triggers.InClusterSecrets {
				indexValues[i] = fmt.Sprintf("%s/%s", trigger.Namespace, trigger.Name)
			}

			return indexValues
		}); err != nil {
			return err
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esv1beta1.ExternalSecret{}).
		// we cant use Owns(), as we don't set ownerReferences when the creationPolicy is not Owner.
		// we use WatchesMetadata() to reduce memory usage, as otherwise we have to process full secret objects.
		WatchesMetadata(
			&v1.Secret{},
			r.secretEventHandler(),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
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
