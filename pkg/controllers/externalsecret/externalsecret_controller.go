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
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
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

	// Loading registered generators.
	_ "github.com/external-secrets/external-secrets/pkg/generator/register"
	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

const (
	fieldOwnerTemplate      = "externalsecrets.external-secrets.io/%v"
	errGetES                = "could not get ExternalSecret"
	errConvert              = "could not apply conversion strategy to keys: %v"
	errDecode               = "could not apply decoding strategy to %v[%d]: %v"
	errGenerate             = "could not generate [%d]: %w"
	errRewrite              = "could not rewrite spec.dataFrom[%d]: %v"
	errInvalidKeys          = "secret keys from spec.dataFrom.%v[%d] can only have alphanumeric,'-', '_' or '.' characters. Convert them using rewrite (https://external-secrets.io/latest/guides-datafrom-rewrite)"
	errUpdateSecret         = "could not update Secret"
	errPatchStatus          = "unable to patch status"
	errGetExistingSecret    = "could not get existing secret: %w"
	errSetCtrlReference     = "could not set ExternalSecret controller reference: %w"
	errFetchTplFrom         = "error fetching templateFrom data: %w"
	errGetSecretData        = "could not get secret data from provider"
	errDeleteSecret         = "could not delete secret"
	errApplyTemplate        = "could not apply template: %w"
	errExecTpl              = "could not execute template: %w"
	errInvalidCreatePolicy  = "invalid creationPolicy=%s. Can not delete secret i do not own"
	errPolicyMergeNotFound  = "the desired secret %s was not found. With creationPolicy=Merge the secret won't be created"
	errPolicyMergeGetSecret = "unable to get secret %s: %w"
	errPolicyMergeMutate    = "unable to mutate secret %s: %w"
	errPolicyMergePatch     = "unable to patch secret %s: %w"
)

const externalSecretSecretNameKey = ".spec.target.name"

// Reconciler reconciles a ExternalSecret object.
type Reconciler struct {
	client.Client
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
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ExternalSecret", req.NamespacedName)

	resourceLabels := ctrlmetrics.RefineNonConditionMetricLabels(map[string]string{"name": req.Name, "namespace": req.Namespace})
	start := time.Now()

	syncCallsError := esmetrics.GetCounterVec(esmetrics.SyncCallsErrorKey)

	// use closures to dynamically update resourceLabels
	defer func() {
		esmetrics.GetGaugeVec(esmetrics.ExternalSecretReconcileDurationKey).With(resourceLabels).Set(float64(time.Since(start)))
		esmetrics.GetCounterVec(esmetrics.SyncCallsKey).With(resourceLabels).Inc()
	}()

	var externalSecret esv1beta1.ExternalSecret
	err := r.Get(ctx, req.NamespacedName, &externalSecret)

	if err != nil {
		if apierrors.IsNotFound(err) {
			conditionSynced := NewExternalSecretCondition(esv1beta1.ExternalSecretDeleted, v1.ConditionFalse, esv1beta1.ConditionReasonSecretDeleted, "Secret was deleted")
			SetExternalSecretCondition(&esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      req.Name,
					Namespace: req.Namespace,
				},
			}, *conditionSynced)

			return ctrl.Result{}, nil
		}

		log.Error(err, errGetES)
		syncCallsError.With(resourceLabels).Inc()

		return ctrl.Result{}, err
	}

	timeSinceLastRefresh := 0 * time.Second
	if !externalSecret.Status.RefreshTime.IsZero() {
		timeSinceLastRefresh = time.Since(externalSecret.Status.RefreshTime.Time)
	}

	// skip reconciliation if deletion timestamp is set on external secret
	if externalSecret.DeletionTimestamp != nil {
		log.Info("skipping as it is in deletion")
		return ctrl.Result{}, nil
	}

	// if extended metrics is enabled, refine the time series vector
	resourceLabels = ctrlmetrics.RefineLabels(resourceLabels, externalSecret.Labels)

	if shouldSkipClusterSecretStore(r, externalSecret) {
		log.Info("skipping cluster secret store as it is disabled")
		return ctrl.Result{}, nil
	}

	// skip when pointing to an unmanaged store
	skip, err := shouldSkipUnmanagedStore(ctx, req.Namespace, r, externalSecret)
	if skip {
		log.Info("skipping unmanaged store as it points to a unmanaged controllerClass")
		return ctrl.Result{}, nil
	}

	refreshInt := r.RequeueInterval
	if externalSecret.Spec.RefreshInterval != nil {
		refreshInt = externalSecret.Spec.RefreshInterval.Duration
	}

	// Target Secret Name should default to the ExternalSecret name if not explicitly specified
	secretName := externalSecret.Spec.Target.Name
	if secretName == "" {
		secretName = externalSecret.ObjectMeta.Name
	}

	// fetch external secret, we need to ensure that it exists, and it's hashmap corresponds
	var existingSecret v1.Secret
	err = r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: externalSecret.Namespace,
	}, &existingSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, errGetExistingSecret)
		return ctrl.Result{}, err
	}

	// refresh should be skipped if
	// 1. resource generation hasn't changed
	// 2. refresh interval is 0
	// 3. if we're still within refresh-interval
	if !shouldRefresh(externalSecret) && isSecretValid(existingSecret) {
		refreshInt = (externalSecret.Spec.RefreshInterval.Duration - timeSinceLastRefresh) + 5*time.Second
		log.V(1).Info("skipping refresh", "rv", getResourceVersion(externalSecret), "nr", refreshInt.Seconds())
		return ctrl.Result{RequeueAfter: refreshInt}, nil
	}
	if !shouldReconcile(externalSecret) {
		log.V(1).Info("stopping reconciling", "rv", getResourceVersion(externalSecret))
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

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: externalSecret.Namespace,
		},
		Immutable: &externalSecret.Spec.Target.Immutable,
		Data:      make(map[string][]byte),
	}

	dataMap, err := r.getProviderSecretData(ctx, &externalSecret)
	if err != nil {
		r.markAsFailed(log, errGetSecretData, err, &externalSecret, syncCallsError.With(resourceLabels))
		return ctrl.Result{}, err
	}

	// if no data was found we can delete the secret if needed.
	if len(dataMap) == 0 {
		switch externalSecret.Spec.Target.DeletionPolicy {
		// delete secret and return early.
		case esv1beta1.DeletionPolicyDelete:
			// safeguard that we only can delete secrets we own
			// this is also implemented in the es validation webhook
			if externalSecret.Spec.Target.CreationPolicy != esv1beta1.CreatePolicyOwner {
				err := fmt.Errorf(errInvalidCreatePolicy, externalSecret.Spec.Target.CreationPolicy)
				r.markAsFailed(log, errDeleteSecret, err, &externalSecret, syncCallsError.With(resourceLabels))
				return ctrl.Result{}, err
			}

			if err := r.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
				r.markAsFailed(log, errDeleteSecret, err, &externalSecret, syncCallsError.With(resourceLabels))
				return ctrl.Result{}, err
			}

			conditionSynced := NewExternalSecretCondition(esv1beta1.ExternalSecretReady, v1.ConditionTrue, esv1beta1.ConditionReasonSecretDeleted, "secret deleted due to DeletionPolicy")
			SetExternalSecretCondition(&externalSecret, *conditionSynced)
			return ctrl.Result{RequeueAfter: refreshInt}, nil
		// In case provider secrets don't exist the kubernetes secret will be kept as-is.
		case esv1beta1.DeletionPolicyRetain:
			r.markAsDone(&externalSecret, start, log)
			return ctrl.Result{RequeueAfter: refreshInt}, nil
		// noop, handled below
		case esv1beta1.DeletionPolicyMerge:
		}
	}

	mutationFunc := func() error {
		if externalSecret.Spec.Target.CreationPolicy == esv1beta1.CreatePolicyOwner {
			err = controllerutil.SetControllerReference(&externalSecret, &secret.ObjectMeta, r.Scheme)
			if err != nil {
				return fmt.Errorf(errSetCtrlReference, err)
			}
		}
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		// diff existing keys
		keys, err := getManagedDataKeys(&existingSecret, externalSecret.Name)
		if err != nil {
			return err
		}
		// Sanitize data map for any updates on the ES
		for _, key := range keys {
			if dataMap[key] == nil {
				secret.Data[key] = nil
				// Sanitizing any templated / updated keys
				delete(secret.Data, key)
			}
		}
		err = r.applyTemplate(ctx, &externalSecret, secret, dataMap)
		if err != nil {
			return fmt.Errorf(errApplyTemplate, err)
		}
		if externalSecret.Spec.Target.CreationPolicy == esv1beta1.CreatePolicyOwner {
			lblValue := utils.ObjectHash(fmt.Sprintf("%v/%v", externalSecret.Namespace, externalSecret.Name))
			secret.Labels[esv1beta1.LabelOwner] = lblValue
		}

		secret.Annotations[esv1beta1.AnnotationDataHash] = r.computeDataHashAnnotation(&existingSecret, secret)

		return nil
	}

	switch externalSecret.Spec.Target.CreationPolicy { //nolint:exhaustive
	case esv1beta1.CreatePolicyMerge:
		err = r.patchSecret(ctx, secret, mutationFunc, &externalSecret)
		if err == nil {
			externalSecret.Status.Binding = v1.LocalObjectReference{Name: secret.Name}
		}
	case esv1beta1.CreatePolicyNone:
		log.V(1).Info("secret creation skipped due to creationPolicy=None")
		err = nil
	default:
		var created bool
		created, err = r.createOrUpdateSecret(ctx, secret, mutationFunc, &externalSecret)
		if err == nil {
			externalSecret.Status.Binding = v1.LocalObjectReference{Name: secret.Name}
		}
		// cleanup orphaned secrets
		if created {
			delErr := deleteOrphanedSecrets(ctx, r.Client, &externalSecret)
			if delErr != nil {
				msg := fmt.Sprintf("failed to clean up orphaned secrets: %v", delErr)
				r.markAsFailed(log, msg, delErr, &externalSecret, syncCallsError.With(resourceLabels))
				return ctrl.Result{}, delErr
			}
		}
	}

	if err != nil {
		r.markAsFailed(log, errUpdateSecret, err, &externalSecret, syncCallsError.With(resourceLabels))
		return ctrl.Result{}, err
	}

	r.markAsDone(&externalSecret, start, log)

	return ctrl.Result{
		RequeueAfter: refreshInt,
	}, nil
}

func (r *Reconciler) markAsDone(externalSecret *esv1beta1.ExternalSecret, start time.Time, log logr.Logger) {
	conditionSynced := NewExternalSecretCondition(esv1beta1.ExternalSecretReady, v1.ConditionTrue, esv1beta1.ConditionReasonSecretSynced, "Secret was synced")
	currCond := GetExternalSecretCondition(externalSecret.Status, esv1beta1.ExternalSecretReady)
	SetExternalSecretCondition(externalSecret, *conditionSynced)
	externalSecret.Status.RefreshTime = metav1.NewTime(start)
	externalSecret.Status.SyncedResourceVersion = getResourceVersion(*externalSecret)
	if currCond == nil || currCond.Status != conditionSynced.Status {
		log.Info("reconciled secret") // Log once if on success in any verbosity
	} else {
		log.V(1).Info("reconciled secret") // Log all reconciliation cycles if higher verbosity applied
	}
}

func (r *Reconciler) markAsFailed(log logr.Logger, msg string, err error, externalSecret *esv1beta1.ExternalSecret, counter prometheus.Counter) {
	log.Error(err, msg)
	r.recorder.Event(externalSecret, v1.EventTypeWarning, esv1beta1.ReasonUpdateFailed, err.Error())
	conditionSynced := NewExternalSecretCondition(esv1beta1.ExternalSecretReady, v1.ConditionFalse, esv1beta1.ConditionReasonSecretSyncedError, msg)
	SetExternalSecretCondition(externalSecret, *conditionSynced)
	counter.Inc()
}

func deleteOrphanedSecrets(ctx context.Context, cl client.Client, externalSecret *esv1beta1.ExternalSecret) error {
	secretList := v1.SecretList{}
	lblValue := utils.ObjectHash(fmt.Sprintf("%v/%v", externalSecret.Namespace, externalSecret.Name))
	ls := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			esv1beta1.LabelOwner: lblValue,
		},
	}
	labelSelector, err := metav1.LabelSelectorAsSelector(ls)
	if err != nil {
		return err
	}
	err = cl.List(ctx, &secretList, &client.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return err
	}
	for key, secret := range secretList.Items {
		if externalSecret.Spec.Target.Name != "" && secret.Name != externalSecret.Spec.Target.Name {
			err = cl.Delete(ctx, &secretList.Items[key])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Reconciler) createOrUpdateSecret(ctx context.Context, secret *v1.Secret, mutationFunc func() error, es *esv1beta1.ExternalSecret) (bool, error) {
	fqdn := fmt.Sprintf(fieldOwnerTemplate, es.Name)
	key := client.ObjectKeyFromObject(secret)
	if err := r.Client.Get(ctx, key, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		if err := mutationFunc(); err != nil {
			return false, err
		}
		// Setting Field Owner even for CreationPolicy==Create
		if err := r.Client.Create(ctx, secret, client.FieldOwner(fqdn)); err != nil {
			return false, err
		}
		r.recorder.Event(es, v1.EventTypeNormal, esv1beta1.ReasonCreated, "Created Secret")
		return true, nil
	}

	existing := secret.DeepCopyObject()
	if err := mutationFunc(); err != nil {
		return false, err
	}

	if equality.Semantic.DeepEqual(existing, secret) {
		return false, nil
	}

	if err := r.Client.Update(ctx, secret, client.FieldOwner(fqdn)); err != nil {
		return false, err
	}
	r.recorder.Event(es, v1.EventTypeNormal, esv1beta1.ReasonUpdated, "Updated Secret")
	return false, nil
}

func (r *Reconciler) patchSecret(ctx context.Context, secret *v1.Secret, mutationFunc func() error, es *esv1beta1.ExternalSecret) error {
	fqdn := fmt.Sprintf(fieldOwnerTemplate, es.Name)
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret.DeepCopy())
	if apierrors.IsNotFound(err) {
		return fmt.Errorf(errPolicyMergeNotFound, secret.Name)
	}
	if err != nil {
		return fmt.Errorf(errPolicyMergeGetSecret, secret.Name, err)
	}

	existing := secret.DeepCopyObject()
	if err = mutationFunc(); err != nil {
		return fmt.Errorf(errPolicyMergeMutate, secret.Name, err)
	}

	// GVK is missing in the Secret, see:
	// https://github.com/kubernetes-sigs/controller-runtime/issues/526
	// https://github.com/kubernetes-sigs/controller-runtime/issues/1517
	// https://github.com/kubernetes/kubernetes/issues/80609
	// we need to manually set it before doing a Patch() as it depends on the GVK
	gvks, unversioned, err := r.Scheme.ObjectKinds(secret)
	if err != nil {
		return err
	}
	if !unversioned && len(gvks) == 1 {
		secret.SetGroupVersionKind(gvks[0])
	}

	if equality.Semantic.DeepEqual(existing, secret) {
		return nil
	}
	// Cleaning up Managed fields manually as to keep patch coherence
	secret.ObjectMeta.ManagedFields = nil
	// we're not able to resolve conflicts so we force ownership
	// see: https://kubernetes.io/docs/reference/using-api/server-side-apply/#using-server-side-apply-in-a-controller
	if err := r.Client.Patch(ctx, secret, client.Apply, client.FieldOwner(fqdn), client.ForceOwnership); err != nil {
		return fmt.Errorf(errPolicyMergePatch, secret.Name, err)
	}
	r.recorder.Event(es, v1.EventTypeNormal, esv1beta1.ReasonUpdated, "Updated Secret")
	return nil
}

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
		var keys []string
		for k := range df {
			keys = append(keys, k)
		}
		return keys
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

func getResourceVersion(es esv1beta1.ExternalSecret) string {
	return fmt.Sprintf("%d-%s", es.ObjectMeta.GetGeneration(), hashMeta(es.ObjectMeta))
}

func hashMeta(m metav1.ObjectMeta) string {
	type meta struct {
		annotations map[string]string
		labels      map[string]string
	}
	return utils.ObjectHash(meta{
		annotations: m.Annotations,
		labels:      m.Labels,
	})
}

func shouldSkipClusterSecretStore(r *Reconciler, es esv1beta1.ExternalSecret) bool {
	return !r.ClusterSecretStoreEnabled && es.Spec.SecretStoreRef.Kind == esv1beta1.ClusterSecretStoreKind
}

// shouldSkipUnmanagedStore iterates over all secretStore references in the externalSecret spec,
// fetches the store and evaluates the controllerClass property.
// Returns true if any storeRef points to store with a non-matching controllerClass.
func shouldSkipUnmanagedStore(ctx context.Context, namespace string, r *Reconciler, es esv1beta1.ExternalSecret) (bool, error) {
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
			genDef, err := r.getGeneratorDefinition(ctx, namespace, ref.SourceRef.GeneratorRef)
			if err != nil {
				return false, err
			}
			skipGenerator, err := shouldSkipGenerator(r, genDef)
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

		err := r.Client.Get(ctx, types.NamespacedName{
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

func shouldRefresh(es esv1beta1.ExternalSecret) bool {
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
	return es.Status.RefreshTime.Add(es.Spec.RefreshInterval.Duration).Before(time.Now())
}

func shouldReconcile(es esv1beta1.ExternalSecret) bool {
	if es.Spec.Target.Immutable && hasSyncedCondition(es) {
		return false
	}
	return true
}

func hasSyncedCondition(es esv1beta1.ExternalSecret) bool {
	for _, condition := range es.Status.Conditions {
		if condition.Reason == "SecretSynced" {
			return true
		}
	}
	return false
}

// isSecretValid checks if the secret exists, and it's data is consistent with the calculated hash.
func isSecretValid(existingSecret v1.Secret) bool {
	// if target secret doesn't exist, or annotations as not set, we need to refresh
	if existingSecret.UID == "" || existingSecret.Annotations == nil {
		return false
	}

	// if the calculated hash is different from the calculation, then it's invalid
	if existingSecret.Annotations[esv1beta1.AnnotationDataHash] != utils.ObjectHash(existingSecret.Data) {
		return false
	}
	return true
}

// computeDataHashAnnotation generate a hash of the secret data combining the old key with the new keys to add or override.
func (r *Reconciler) computeDataHashAnnotation(existing, secret *v1.Secret) string {
	data := make(map[string][]byte)
	for k, v := range existing.Data {
		data[k] = v
	}
	for k, v := range secret.Data {
		data[k] = v
	}
	return utils.ObjectHash(data)
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("external-secrets")

	// Index .Spec.Target.Name to reconcile ExternalSecrets effectively when secrets have changed
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &esv1beta1.ExternalSecret{}, externalSecretSecretNameKey, func(obj client.Object) []string {
		es := obj.(*esv1beta1.ExternalSecret)

		if name := es.Spec.Target.Name; name != "" {
			return []string{name}
		}
		return []string{es.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esv1beta1.ExternalSecret{}).
		// Cannot use Owns since the controller does not set owner reference when creation policy is not Owner
		Watches(
			&v1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
			builder.OnlyMetadata,
		).
		Complete(r)
}

func (r *Reconciler) findObjectsForSecret(ctx context.Context, secret client.Object) []reconcile.Request {
	var externalSecrets esv1beta1.ExternalSecretList
	err := r.List(
		ctx,
		&externalSecrets,
		client.InNamespace(secret.GetNamespace()),
		client.MatchingFields{externalSecretSecretNameKey: secret.GetName()},
	)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(externalSecrets.Items))
	for i := range externalSecrets.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      externalSecrets.Items[i].GetName(),
				Namespace: externalSecrets.Items[i].GetNamespace(),
			},
		}
	}
	return requests
}
