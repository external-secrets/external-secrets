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

// Package pushsecret implements the controller for managing PushSecret resources.
package pushsecret

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/pushsecret/psmetrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	ctrlutil "github.com/external-secrets/external-secrets/pkg/controllers/util"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/statemanager"
	estemplate "github.com/external-secrets/external-secrets/runtime/template/v2"
	"github.com/external-secrets/external-secrets/runtime/util/locks"

	// Load registered generators.
	_ "github.com/external-secrets/external-secrets/pkg/register"
)

const (
	errFailedGetSecret         = "could not get source secret"
	errPatchStatus             = "error merging"
	errGetSecretStore          = "could not get SecretStore %q, %w"
	errGetClusterSecretStore   = "could not get ClusterSecretStore %q, %w"
	errSetSecretFailed         = "could not write remote ref %v to target secretstore %v: %v"
	errFailedSetSecret         = "set secret failed: %v"
	errConvert                 = "could not apply conversion strategy to keys: %v"
	pushSecretFinalizer        = "pushsecret.externalsecrets.io/finalizer"
	errCloudNotUpdateFinalizer = "could not update finalizers: %w"
)

// Reconciler is the controller for PushSecret resources.
// It manages the lifecycle of PushSecrets, ensuring that secrets are pushed to
// specified secret stores according to the defined policies and templates.
type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	RestConfig      *rest.Config
	RequeueInterval time.Duration
	ControllerClass string
}

// storeInfo holds the identifying attributes of a secret store for per-store processing.
type storeInfo struct {
	Name   string
	Kind   string
	Labels map[string]string
}

// SetupWithManager sets up the controller with the Manager.
// It configures the controller to watch PushSecret resources and
// manages indexing for efficient lookups based on secret stores and deletion policies.
func (r *Reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("pushsecret")

	// Index PushSecrets by the stores they have pushed to (for finalizer management on store deletion)
	// Refer to common.go for more details on the index function
	if err := mgr.GetFieldIndexer().IndexField(ctx, &esapi.PushSecret{}, "status.syncedPushSecrets", func(obj client.Object) []string {
		ps := obj.(*esapi.PushSecret)

		// Only index PushSecrets with DeletionPolicy=Delete for efficiency
		if ps.Spec.DeletionPolicy != esapi.PushSecretDeletionPolicyDelete {
			return nil
		}

		// Format is typically "Kind/Name" (e.g., "SecretStore/store1", "ClusterSecretStore/clusterstore1")
		storeKeys := make([]string, 0, len(ps.Status.SyncedPushSecrets))
		for storeKey := range ps.Status.SyncedPushSecrets {
			storeKeys = append(storeKeys, storeKey)
		}
		return storeKeys
	}); err != nil {
		return err
	}

	// Index PushSecrets by deletionPolicy for quick filtering
	if err := mgr.GetFieldIndexer().IndexField(ctx, &esapi.PushSecret{}, "spec.deletionPolicy", func(obj client.Object) []string {
		ps := obj.(*esapi.PushSecret)
		return []string{string(ps.Spec.DeletionPolicy)}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esapi.PushSecret{}).
		Complete(r)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("pushsecret", req.NamespacedName)

	resourceLabels := ctrlmetrics.RefineNonConditionMetricLabels(map[string]string{"name": req.Name, "namespace": req.Namespace})
	start := time.Now()

	pushSecretReconcileDuration := psmetrics.GetGaugeVec(psmetrics.PushSecretReconcileDurationKey)
	defer func() { pushSecretReconcileDuration.With(resourceLabels).Set(float64(time.Since(start))) }()

	var ps esapi.PushSecret
	mgr := secretstore.NewManager(r.Client, r.ControllerClass, false)
	defer func() {
		_ = mgr.Close(ctx)
	}()

	if err := r.Get(ctx, req.NamespacedName, &ps); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		msg := "unable to get PushSecret"
		r.recorder.Event(&ps, v1.EventTypeWarning, esapi.ReasonErrored, msg)
		log.Error(err, msg)

		return ctrl.Result{}, fmt.Errorf("get resource: %w", err)
	}

	refreshInt := r.RequeueInterval
	if ps.Spec.RefreshInterval != nil {
		refreshInt = ps.Spec.RefreshInterval.Duration
	}

	p := client.MergeFrom(ps.DeepCopy())
	defer func() {
		err := r.Client.Status().Patch(ctx, &ps, p)
		if err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, errPatchStatus)
		}
	}()
	switch ps.Spec.DeletionPolicy {
	case esapi.PushSecretDeletionPolicyDelete:
		// finalizer logic. Only added if we should delete the secrets
		if ps.ObjectMeta.DeletionTimestamp.IsZero() {
			if added := controllerutil.AddFinalizer(&ps, pushSecretFinalizer); added {
				if err := r.Client.Update(ctx, &ps, &client.UpdateOptions{}); err != nil {
					return ctrl.Result{}, fmt.Errorf(errCloudNotUpdateFinalizer, err)
				}
				return ctrl.Result{Requeue: true}, nil
			}
		} else if controllerutil.ContainsFinalizer(&ps, pushSecretFinalizer) {
			// trigger a cleanup with no Synced Map
			badState, err := r.DeleteSecretFromProviders(ctx, &ps, esapi.SyncedPushSecretsMap{}, mgr)
			if err != nil {
				msg := fmt.Sprintf("Failed to Delete Secrets from Provider: %v", err)
				r.markAsFailed(msg, &ps, badState)
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&ps, pushSecretFinalizer)
			if err := r.Client.Update(ctx, &ps, &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, fmt.Errorf("could not update finalizers: %w", err)
			}

			return ctrl.Result{}, nil
		}
	case esapi.PushSecretDeletionPolicyNone:
		if controllerutil.ContainsFinalizer(&ps, pushSecretFinalizer) {
			controllerutil.RemoveFinalizer(&ps, pushSecretFinalizer)
			if err := r.Client.Update(ctx, &ps, &client.UpdateOptions{}); err != nil {
				return ctrl.Result{}, fmt.Errorf(errCloudNotUpdateFinalizer, err)
			}
		}
	default:
	}

	timeSinceLastRefresh := 0 * time.Second
	if !ps.Status.RefreshTime.IsZero() {
		timeSinceLastRefresh = time.Since(ps.Status.RefreshTime.Time)
	}
	if !shouldRefresh(ps) {
		refreshInt = (ps.Spec.RefreshInterval.Duration - timeSinceLastRefresh) + 5*time.Second
		log.V(1).Info("skipping refresh", "rv", ctrlutil.GetResourceVersion(ps.ObjectMeta), "nr", refreshInt.Seconds())
		return ctrl.Result{RequeueAfter: refreshInt}, nil
	}

	// Validate dataTo storeRefs
	if err := validateDataToStoreRefs(ps.Spec.DataTo, ps.Spec.SecretStoreRefs); err != nil {
		r.markAsFailed(err.Error(), &ps, nil)
		return ctrl.Result{}, err
	}

	secrets, err := r.resolveSecrets(ctx, &ps)
	if err != nil {
		// Handle source secret deletion with DeletionPolicy=Delete
		isSecretSelector := ps.Spec.Selector.Secret != nil && ps.Spec.Selector.Secret.Name != ""
		if apierrors.IsNotFound(err) && isSecretSelector &&
			ps.Spec.DeletionPolicy == esapi.PushSecretDeletionPolicyDelete &&
			len(ps.Status.SyncedPushSecrets) > 0 {
			return r.handleSourceSecretDeleted(ctx, &ps, mgr, err)
		}
		r.markAsFailed(errFailedGetSecret, &ps, nil)
		return ctrl.Result{}, err
	}
	secretStores, err := r.GetSecretStores(ctx, ps)
	if err != nil {
		r.markAsFailed(err.Error(), &ps, nil)

		return ctrl.Result{}, err
	}

	// Filter out SecretStores that are being deleted to avoid finalizer conflicts
	activeSecretStores := make(map[esapi.PushSecretStoreRef]esv1.GenericStore, len(secretStores))
	for ref, store := range secretStores {
		// Skip stores that are being deleted
		if !store.GetDeletionTimestamp().IsZero() {
			log.Info("skipping SecretStore that is being deleted", "storeName", store.GetName(), "storeKind", store.GetKind())
			continue
		}
		activeSecretStores[ref] = store
	}

	secretStores, err = removeUnmanagedStores(ctx, req.Namespace, r, activeSecretStores)
	if err != nil {
		r.markAsFailed(err.Error(), &ps, nil)
		return ctrl.Result{}, err
	}
	// if no stores are managed by this controller
	if len(secretStores) == 0 {
		return ctrl.Result{}, nil
	}

	allSyncedSecrets := make(esapi.SyncedPushSecretsMap)
	for _, secret := range secrets {
		if err := r.applyTemplate(ctx, &ps, &secret); err != nil {
			return ctrl.Result{}, err
		}

		syncedSecrets, err := r.PushSecretToProviders(ctx, secretStores, ps, &secret, mgr)
		if err != nil {
			if errors.Is(err, locks.ErrConflict) {
				log.Info("retry to acquire lock to update the secret later", "error", err)
				return ctrl.Result{Requeue: true}, nil
			}

			totalSecrets := mergeSecretState(syncedSecrets, ps.Status.SyncedPushSecrets)
			msg := fmt.Sprintf(errFailedSetSecret, err)
			r.markAsFailed(msg, &ps, totalSecrets)

			return ctrl.Result{}, err
		}
		switch ps.Spec.DeletionPolicy {
		case esapi.PushSecretDeletionPolicyDelete:
			badSyncState, err := r.DeleteSecretFromProviders(ctx, &ps, syncedSecrets, mgr)
			if err != nil {
				msg := fmt.Sprintf("Failed to Delete Secrets from Provider: %v", err)
				r.markAsFailed(msg, &ps, badSyncState)
				return ctrl.Result{}, err
			}
		case esapi.PushSecretDeletionPolicyNone:
		default:
		}

		allSyncedSecrets = mergeSecretState(allSyncedSecrets, syncedSecrets)
	}

	r.markAsDone(&ps, allSyncedSecrets, start)

	return ctrl.Result{RequeueAfter: refreshInt}, nil
}

// handleSourceSecretDeleted cleans up provider secrets when source Secret is unavailable.
//
//nolint:unparam // Returns (ctrl.Result, error) for consistency with Reconcile pattern.
func (r *Reconciler) handleSourceSecretDeleted(ctx context.Context, ps *esapi.PushSecret, mgr *secretstore.Manager, resolveErr error) (ctrl.Result, error) {
	log := r.Log.WithValues("pushsecret", client.ObjectKeyFromObject(ps))
	log.Info("source secret unavailable, cleaning up provider secrets", "syncedSecrets", len(ps.Status.SyncedPushSecrets))

	// Delete all previously synced secrets from providers
	badState, err := r.DeleteSecretFromProviders(ctx, ps, esapi.SyncedPushSecretsMap{}, mgr)
	if err != nil {
		msg := fmt.Sprintf("failed to cleanup provider secrets: %v", err)
		r.markAsFailed(msg, ps, badState)
		return ctrl.Result{}, err
	}

	// Clear synced secrets and mark as failed (source unavailable)
	r.setSecrets(ps, esapi.SyncedPushSecretsMap{})
	r.markAsFailed(errFailedGetSecret, ps, nil)
	return ctrl.Result{}, resolveErr
}

func shouldRefresh(ps esapi.PushSecret) bool {
	if ps.Status.SyncedResourceVersion != ctrlutil.GetResourceVersion(ps.ObjectMeta) {
		return true
	}
	if ps.Spec.RefreshInterval.Duration == 0 && ps.Status.SyncedResourceVersion != "" {
		return false
	}
	if ps.Status.RefreshTime.IsZero() {
		return true
	}
	return ps.Status.RefreshTime.Add(ps.Spec.RefreshInterval.Duration).Before(time.Now())
}

func (r *Reconciler) markAsFailed(msg string, ps *esapi.PushSecret, syncState esapi.SyncedPushSecretsMap) {
	cond := NewPushSecretCondition(esapi.PushSecretReady, v1.ConditionFalse, esapi.ReasonErrored, msg)
	SetPushSecretCondition(ps, *cond)
	if syncState != nil {
		r.setSecrets(ps, syncState)
	}
	r.recorder.Event(ps, v1.EventTypeWarning, esapi.ReasonErrored, msg)
}

func (r *Reconciler) markAsDone(ps *esapi.PushSecret, secrets esapi.SyncedPushSecretsMap, start time.Time) {
	msg := "PushSecret synced successfully"
	if ps.Spec.UpdatePolicy == esapi.PushSecretUpdatePolicyIfNotExists {
		msg += ". Existing secrets in providers unchanged."
	}
	cond := NewPushSecretCondition(esapi.PushSecretReady, v1.ConditionTrue, esapi.ReasonSynced, msg)
	SetPushSecretCondition(ps, *cond)
	r.setSecrets(ps, secrets)
	ps.Status.RefreshTime = metav1.NewTime(start)
	ps.Status.SyncedResourceVersion = ctrlutil.GetResourceVersion(ps.ObjectMeta)
	r.recorder.Event(ps, v1.EventTypeNormal, esapi.ReasonSynced, msg)
}

func (r *Reconciler) setSecrets(ps *esapi.PushSecret, status esapi.SyncedPushSecretsMap) {
	ps.Status.SyncedPushSecrets = status
}

func mergeSecretState(newMap, old esapi.SyncedPushSecretsMap) esapi.SyncedPushSecretsMap {
	if newMap == nil {
		return old
	}

	out := newMap.DeepCopy()
	for k, v := range old {
		_, ok := out[k]
		if !ok {
			out[k] = make(map[string]esapi.PushSecretData)
		}
		maps.Insert(out[k], maps.All(v))
	}
	return out
}

// DeleteSecretFromProviders removes secrets from providers that are no longer needed.
// It compares the existing synced secrets in the PushSecret status with the new desired state,
// and deletes any secrets that are no longer present in the new state.
func (r *Reconciler) DeleteSecretFromProviders(ctx context.Context, ps *esapi.PushSecret, newMap esapi.SyncedPushSecretsMap, mgr *secretstore.Manager) (esapi.SyncedPushSecretsMap, error) {
	out := mergeSecretState(newMap, ps.Status.SyncedPushSecrets)
	for storeName, oldData := range ps.Status.SyncedPushSecrets {
		storeRef := esv1.SecretStoreRef{
			Name: strings.Split(storeName, "/")[1],
			Kind: strings.Split(storeName, "/")[0],
		}
		client, err := mgr.Get(ctx, storeRef, ps.Namespace, nil)
		if err != nil {
			return out, fmt.Errorf("could not get secrets client for store %v: %w", storeName, err)
		}
		newData, ok := newMap[storeName]
		if !ok {
			err = r.DeleteAllSecretsFromStore(ctx, client, oldData)
			if err != nil {
				return out, err
			}
			delete(out, storeName)
			continue
		}
		for oldEntry, oldRef := range oldData {
			_, ok := newData[oldEntry]
			if !ok {
				err = r.DeleteSecretFromStore(ctx, client, oldRef)
				if err != nil {
					return out, err
				}
				delete(out[storeName], oldEntry)
			}
		}
	}
	return out, nil
}

// DeleteAllSecretsFromStore removes all secrets from a given secret store.
func (r *Reconciler) DeleteAllSecretsFromStore(ctx context.Context, client esv1.SecretsClient, data map[string]esapi.PushSecretData) error {
	for _, v := range data {
		err := r.DeleteSecretFromStore(ctx, client, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteSecretFromStore removes a specific secret from a given secret store.
func (r *Reconciler) DeleteSecretFromStore(ctx context.Context, client esv1.SecretsClient, data esapi.PushSecretData) error {
	return client.DeleteSecret(ctx, data.Match.RemoteRef)
}

// PushSecretToProviders pushes the secret data to the specified secret stores.
// It iterates over each store and handles the push operation according to the
// defined update policies and conversion strategies.
func (r *Reconciler) PushSecretToProviders(
	ctx context.Context,
	stores map[esapi.PushSecretStoreRef]esv1.GenericStore,
	ps esapi.PushSecret,
	secret *v1.Secret,
	mgr *secretstore.Manager,
) (esapi.SyncedPushSecretsMap, error) {
	out := make(esapi.SyncedPushSecretsMap)
	for ref, store := range stores {
		si := storeInfo{Name: store.GetName(), Kind: ref.Kind, Labels: store.GetLabels()}
		out, err := r.handlePushSecretDataForStore(ctx, ps, secret, out, mgr, si)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

func (r *Reconciler) handlePushSecretDataForStore(
	ctx context.Context,
	ps esapi.PushSecret,
	secret *v1.Secret,
	out esapi.SyncedPushSecretsMap,
	mgr *secretstore.Manager,
	si storeInfo,
) (esapi.SyncedPushSecretsMap, error) {
	storeKey := fmt.Sprintf("%v/%v", si.Kind, si.Name)
	out[storeKey] = make(map[string]esapi.PushSecretData)
	storeRef := esv1.SecretStoreRef{
		Name: si.Name,
		Kind: si.Kind,
	}
	secretClient, err := mgr.Get(ctx, storeRef, ps.GetNamespace(), nil)
	if err != nil {
		return out, fmt.Errorf("could not get secrets client for store %v: %w", si.Name, err)
	}

	// Create a copy of the secret for this store to avoid mutating the shared secret
	storeSecret := secret.DeepCopy()

	// Filter dataTo entries for this specific store
	filteredDataTo, err := filterDataToForStore(ps.Spec.DataTo, si.Name, si.Kind, si.Labels)
	if err != nil {
		return out, fmt.Errorf("failed to filter dataTo: %w", err)
	}

	// Expand filtered dataTo entries into PushSecretData
	dataToEntries, err := r.expandDataTo(storeSecret, filteredDataTo)
	if err != nil {
		return out, fmt.Errorf("failed to expand dataTo: %w", err)
	}

	// Merge dataTo entries with explicit data (explicit data overrides)
	allData, err := mergeDataEntries(dataToEntries, ps.Spec.Data)
	if err != nil {
		return out, fmt.Errorf("failed to merge data entries: %w", err)
	}

	// Preserve the original secret data so each data entry's conversion
	// is applied to the original data, not to already-converted data
	originalStoreSecretData := storeSecret.Data

	for _, data := range allData {
		if err := r.pushSecretEntry(ctx, secretClient, storeSecret, data, ps.Spec.UpdatePolicy, originalStoreSecretData, si.Name); err != nil {
			return out, err
		}
		out[storeKey][statusRef(data)] = data
	}
	return out, nil
}

// pushSecretEntry converts, validates, and pushes a single data entry to the provider.
// If the update policy is IfNotExists and the secret already exists, the push is skipped.
func (r *Reconciler) pushSecretEntry(
	ctx context.Context,
	secretClient esv1.SecretsClient,
	storeSecret *v1.Secret,
	data esapi.PushSecretData,
	updatePolicy esapi.PushSecretUpdatePolicy,
	originalData map[string][]byte,
	storeName string,
) error {
	secretData, err := esutils.ReverseKeys(data.ConversionStrategy, originalData)
	if err != nil {
		return fmt.Errorf(errConvert, err)
	}
	storeSecret.Data = secretData

	key := data.GetSecretKey()
	if !secretKeyExists(key, storeSecret) {
		return fmt.Errorf("secret key %v does not exist", key)
	}

	if updatePolicy == esapi.PushSecretUpdatePolicyIfNotExists {
		exists, err := secretClient.SecretExists(ctx, data.Match.RemoteRef)
		if err != nil {
			return fmt.Errorf("could not verify if secret exists in store: %w", err)
		}
		if exists {
			return nil
		}
	}

	if err := secretClient.PushSecret(ctx, storeSecret, data); err != nil {
		return fmt.Errorf(errSetSecretFailed, key, storeName, err)
	}
	return nil
}

func secretKeyExists(key string, secret *v1.Secret) bool {
	_, ok := secret.Data[key]
	return key == "" || ok
}

const defaultGeneratorStateKey = "__pushsecret"

func (r *Reconciler) resolveSecrets(ctx context.Context, ps *esapi.PushSecret) ([]v1.Secret, error) {
	var err error
	generatorState := statemanager.New(ctx, r.Client, r.Scheme, ps.Namespace, ps)
	defer func() {
		if err != nil {
			if err := generatorState.Rollback(); err != nil {
				r.Log.Error(err, "error rolling back generator state")
			}

			return
		}
		if err := generatorState.Commit(); err != nil {
			r.Log.Error(err, "error committing generator state")
		}
	}()

	switch {
	case ps.Spec.Selector.Secret != nil && ps.Spec.Selector.Secret.Name != "":
		secretName := types.NamespacedName{Name: ps.Spec.Selector.Secret.Name, Namespace: ps.Namespace}
		secret := &v1.Secret{}
		if err := r.Client.Get(ctx, secretName, secret); err != nil {
			return nil, err
		}
		generatorState.EnqueueFlagLatestStateForGC(defaultGeneratorStateKey)

		return []v1.Secret{*secret}, nil
	case ps.Spec.Selector.GeneratorRef != nil:
		secret, err := r.resolveSecretFromGenerator(ctx, ps.Namespace, ps.Spec.Selector.GeneratorRef, generatorState)
		if err != nil {
			return nil, fmt.Errorf("could not resolve secret from generator ref %v: %w", ps.Spec.Selector.GeneratorRef, err)
		}

		return []v1.Secret{*secret}, nil
	case ps.Spec.Selector.Secret != nil && ps.Spec.Selector.Secret.Selector != nil:
		labelSelector, err := metav1.LabelSelectorAsSelector(ps.Spec.Selector.Secret.Selector)
		if err != nil {
			return nil, err
		}

		var secretList v1.SecretList
		err = r.List(ctx, &secretList, &client.ListOptions{LabelSelector: labelSelector, Namespace: ps.Namespace})
		if err != nil {
			return nil, err
		}

		return secretList.Items, err
	}

	return nil, errors.New("no secret selector provided")
}

func (r *Reconciler) resolveSecretFromGenerator(ctx context.Context, namespace string, generatorRef *esv1.GeneratorRef, generatorState *statemanager.Manager) (*v1.Secret, error) {
	gen, genResource, err := resolvers.GeneratorRef(ctx, r.Client, r.Scheme, namespace, generatorRef)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve generator: %w", err)
	}
	var prevState *genv1alpha1.GeneratorState
	if generatorState != nil {
		prevState, err = generatorState.GetLatestState(defaultGeneratorStateKey)
		if err != nil {
			return nil, fmt.Errorf("unable to get latest state: %w", err)
		}
	}
	secretMap, newState, err := gen.Generate(ctx, genResource, r.Client, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to generate: %w", err)
	}
	if prevState != nil && generatorState != nil {
		generatorState.EnqueueMoveStateToGC(defaultGeneratorStateKey)
	}
	if generatorState != nil {
		generatorState.EnqueueSetLatest(ctx, defaultGeneratorStateKey, namespace, genResource, gen, newState)
	}
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "___generated-secret",
			Namespace: namespace,
		},
		Data: secretMap,
	}, err
}

// GetSecretStores retrieves the SecretStore and ClusterSecretStore resources
// referenced in the PushSecret. It supports both direct references by name
// and label selectors to find multiple stores.
func (r *Reconciler) GetSecretStores(ctx context.Context, ps esapi.PushSecret) (map[esapi.PushSecretStoreRef]esv1.GenericStore, error) {
	stores := make(map[esapi.PushSecretStoreRef]esv1.GenericStore)
	for _, refStore := range ps.Spec.SecretStoreRefs {
		if refStore.LabelSelector != nil {
			labelSelector, err := metav1.LabelSelectorAsSelector(refStore.LabelSelector)
			if err != nil {
				return nil, fmt.Errorf("could not convert labels: %w", err)
			}
			if refStore.Kind == esv1.ClusterSecretStoreKind {
				clusterSecretStoreList := esv1.ClusterSecretStoreList{}
				err = r.List(ctx, &clusterSecretStoreList, &client.ListOptions{LabelSelector: labelSelector})
				if err != nil {
					return nil, fmt.Errorf("could not list cluster Secret Stores: %w", err)
				}
				for k, v := range clusterSecretStoreList.Items {
					key := esapi.PushSecretStoreRef{
						Name: v.Name,
						Kind: esv1.ClusterSecretStoreKind,
					}
					stores[key] = &clusterSecretStoreList.Items[k]
				}
			} else {
				secretStoreList := esv1.SecretStoreList{}
				err = r.List(ctx, &secretStoreList, &client.ListOptions{LabelSelector: labelSelector, Namespace: ps.Namespace})
				if err != nil {
					return nil, fmt.Errorf("could not list Secret Stores: %w", err)
				}
				for k, v := range secretStoreList.Items {
					key := esapi.PushSecretStoreRef{
						Name: v.Name,
						Kind: esv1.SecretStoreKind,
					}
					stores[key] = &secretStoreList.Items[k]
				}
			}
		} else {
			store, err := r.getSecretStoreFromName(ctx, refStore, ps.Namespace)
			if err != nil {
				return nil, err
			}
			stores[refStore] = store
		}
	}
	return stores, nil
}

func (r *Reconciler) getSecretStoreFromName(ctx context.Context, refStore esapi.PushSecretStoreRef, ns string) (esv1.GenericStore, error) {
	if refStore.Name == "" {
		return nil, errors.New("refStore Name must be provided")
	}
	ref := types.NamespacedName{
		Name: refStore.Name,
	}
	if refStore.Kind == esv1.ClusterSecretStoreKind {
		var store esv1.ClusterSecretStore
		err := r.Get(ctx, ref, &store)
		if err != nil {
			return nil, fmt.Errorf(errGetClusterSecretStore, ref.Name, err)
		}
		return &store, nil
	}
	ref.Namespace = ns
	var store esv1.SecretStore
	err := r.Get(ctx, ref, &store)
	if err != nil {
		return nil, fmt.Errorf(errGetSecretStore, ref.Name, err)
	}
	return &store, nil
}

// NewPushSecretCondition creates a new PushSecret condition.
func NewPushSecretCondition(condType esapi.PushSecretConditionType, status v1.ConditionStatus, reason, message string) *esapi.PushSecretStatusCondition {
	return &esapi.PushSecretStatusCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// SetPushSecretCondition updates the PushSecret to include the provided condition.
func SetPushSecretCondition(ps *esapi.PushSecret, condition esapi.PushSecretStatusCondition) {
	currentCond := GetPushSecretCondition(ps.Status.Conditions, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status &&
		currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
		psmetrics.UpdatePushSecretCondition(ps, &condition, 1.0)
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	ps.Status.Conditions = append(FilterOutCondition(ps.Status.Conditions, condition.Type), condition)

	if currentCond != nil {
		psmetrics.UpdatePushSecretCondition(ps, currentCond, 0.0)
	}

	psmetrics.UpdatePushSecretCondition(ps, &condition, 1.0)
}

// FilterOutCondition returns an empty set of conditions with the provided type.
func FilterOutCondition(conditions []esapi.PushSecretStatusCondition, condType esapi.PushSecretConditionType) []esapi.PushSecretStatusCondition {
	newConditions := make([]esapi.PushSecretStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

// GetPushSecretCondition returns the condition with the provided type.
func GetPushSecretCondition(conditions []esapi.PushSecretStatusCondition, condType esapi.PushSecretConditionType) *esapi.PushSecretStatusCondition {
	for i := range conditions {
		c := conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func statusRef(ref esv1.PushSecretData) string {
	if ref.GetProperty() != "" {
		return ref.GetRemoteKey() + "/" + ref.GetProperty()
	}
	return ref.GetRemoteKey()
}

// removeUnmanagedStores iterates over all SecretStore references and evaluates the controllerClass property.
// Returns a map containing only managed stores.
func removeUnmanagedStores(ctx context.Context, namespace string, r *Reconciler, ss map[esapi.PushSecretStoreRef]esv1.GenericStore) (map[esapi.PushSecretStoreRef]esv1.GenericStore, error) {
	for ref := range ss {
		var store esv1.GenericStore
		switch ref.Kind {
		case esv1.SecretStoreKind:
			store = &esv1.SecretStore{}
		case esv1.ClusterSecretStoreKind:
			store = &esv1.ClusterSecretStore{}
			namespace = ""
		}
		err := r.Client.Get(ctx, types.NamespacedName{
			Name:      ref.Name,
			Namespace: namespace,
		}, store)

		if err != nil {
			return ss, err
		}

		class := store.GetSpec().Controller
		if class != "" && class != r.ControllerClass {
			delete(ss, ref)
		}
	}
	return ss, nil
}

// matchKeys filters secret keys based on the provided match pattern.
// If pattern is nil or empty, all keys are matched.
func matchKeys(allKeys []string, match *esapi.PushSecretDataToMatch) ([]string, error) {
	// If no match pattern specified, return all keys
	if match == nil || match.RegExp == "" {
		return allKeys, nil
	}

	// Compile the regex pattern
	re, err := regexp.Compile(match.RegExp)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regexp pattern %q: %w", match.RegExp, err)
	}

	// Filter keys that match the pattern
	matched := make([]string, 0)
	for _, key := range allKeys {
		if re.MatchString(key) {
			matched = append(matched, key)
		}
	}

	return matched, nil
}

// filterDataToForStore returns dataTo entries that target the given store.
func filterDataToForStore(dataToList []esapi.PushSecretDataTo, storeName, storeKind string, storeLabels map[string]string) ([]esapi.PushSecretDataTo, error) {
	filtered := make([]esapi.PushSecretDataTo, 0, len(dataToList))
	for i, dataTo := range dataToList {
		matches, err := dataToMatchesStore(dataTo, storeName, storeKind, storeLabels)
		if err != nil {
			return nil, fmt.Errorf("dataTo[%d]: %w", i, err)
		}
		if matches {
			filtered = append(filtered, dataTo)
		}
	}
	return filtered, nil
}

// dataToMatchesStore reports whether a single dataTo entry targets the given store.
func dataToMatchesStore(dataTo esapi.PushSecretDataTo, storeName, storeKind string, storeLabels map[string]string) (bool, error) {
	if dataTo.StoreRef == nil {
		return false, fmt.Errorf("storeRef is required")
	}
	refKind := dataTo.StoreRef.Kind
	if refKind == "" {
		refKind = esv1.SecretStoreKind
	}
	// Match by name takes precedence over label selector.
	if dataTo.StoreRef.Name != "" {
		return dataTo.StoreRef.Name == storeName && refKind == storeKind, nil
	}
	// Match by label selector.
	if dataTo.StoreRef.LabelSelector == nil {
		return false, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(dataTo.StoreRef.LabelSelector)
	if err != nil {
		return false, fmt.Errorf("invalid labelSelector: %w", err)
	}
	return refKind == storeKind && selector.Matches(labels.Set(storeLabels)), nil
}

// expandDataTo expands dataTo entries into individual PushSecretData entries.
//
// Each matched key becomes a separate entry that is pushed independently. This enables:
//   - Individual key transformation via rewrite
//   - Per-key status tracking in SyncedPushSecrets
//   - Granular deletion when keys are removed (DeletionPolicy=Delete)
//   - Compatibility with all providers (no bulk API requirement)
//
// This mirrors how explicit `data` entries work - each entry maps one source key to
// one provider secret. The difference is dataTo generates these entries dynamically
// from patterns rather than requiring explicit configuration.
//
// Processing order when template is used:
//  1. Template is applied to source secret (creates/transforms keys)
//  2. dataTo matches against the templated secret keys
//  3. Rewrite transforms matched key names
//  4. Push to providers
func (r *Reconciler) expandDataTo(secret *v1.Secret, dataToList []esapi.PushSecretDataTo) ([]esapi.PushSecretData, error) {
	if len(dataToList) == 0 {
		return nil, nil
	}

	allData := make([]esapi.PushSecretData, 0)

	// Track remote keys across all dataTo entries to detect duplicates
	overallRemoteKeys := make(map[string]string) // remoteKey -> "dataTo[i]:sourceKey"

	for i, dataTo := range dataToList {
		entries, keyMap, err := r.expandSingleDataTo(secret, dataTo)
		if err != nil {
			return nil, fmt.Errorf("dataTo[%d]: %w", i, err)
		}
		if len(entries) == 0 {
			r.Log.Info("dataTo entry matched no keys", "index", i)
			continue
		}

		// Check for duplicate remote keys across all dataTo entries
		for sourceKey, remoteKey := range keyMap {
			if existingSource, exists := overallRemoteKeys[remoteKey]; exists {
				return nil, fmt.Errorf("dataTo[%d]: duplicate remote key %q from source key %q (conflicts with %s)", i, remoteKey, sourceKey, existingSource)
			}
			overallRemoteKeys[remoteKey] = fmt.Sprintf("dataTo[%d]:%s", i, sourceKey)
		}

		allData = append(allData, entries...)
		r.Log.Info("expanded dataTo entry", "index", i, "matchedKeys", len(entries), "created", len(keyMap))
	}

	return allData, nil
}

// expandSingleDataTo processes a single dataTo entry: converts keys, matches them
// against the pattern, applies rewrites, validates remote keys, and builds the
// resulting PushSecretData entries along with the source-to-remote key mapping.
func (r *Reconciler) expandSingleDataTo(secret *v1.Secret, dataTo esapi.PushSecretDataTo) ([]esapi.PushSecretData, map[string]string, error) {
	// Apply conversion strategy BEFORE matching and rewriting
	// This ensures that keys are matched against their converted names
	convertedData, err := esutils.ReverseKeys(dataTo.ConversionStrategy, secret.Data)
	if err != nil {
		return nil, nil, fmt.Errorf("conversion failed: %w", err)
	}

	allKeys := make([]string, 0, len(convertedData))
	for key := range convertedData {
		allKeys = append(allKeys, key)
	}

	matchedKeys, err := matchKeys(allKeys, dataTo.Match)
	if err != nil {
		return nil, nil, fmt.Errorf("match failed: %w", err)
	}
	if len(matchedKeys) == 0 {
		return nil, nil, nil
	}

	matchedData := make(map[string][]byte, len(matchedKeys))
	for _, key := range matchedKeys {
		matchedData[key] = convertedData[key]
	}

	keyMap, err := rewriteWithKeyMapping(dataTo.Rewrite, matchedData)
	if err != nil {
		return nil, nil, fmt.Errorf("rewrite failed: %w", err)
	}

	// Validate that no remote key is empty
	for sourceKey, remoteKey := range keyMap {
		if remoteKey == "" {
			return nil, nil, fmt.Errorf("empty remote key produced for source key %q", sourceKey)
		}
	}

	// Create PushSecretData entries
	// SecretKey references the converted key name
	entries := make([]esapi.PushSecretData, 0, len(keyMap))
	for sourceKey, remoteKey := range keyMap {
		entries = append(entries, esapi.PushSecretData{
			Match: esapi.PushSecretMatch{
				SecretKey: sourceKey,
				RemoteRef: esapi.PushSecretRemoteRef{
					RemoteKey: remoteKey,
				},
			},
			Metadata:           dataTo.Metadata,
			ConversionStrategy: dataTo.ConversionStrategy,
		})
	}

	return entries, keyMap, nil
}

// validateDataToStoreRefs checks that each dataTo entry has a valid storeRef.
func validateDataToStoreRefs(dataToList []esapi.PushSecretDataTo, storeRefs []esapi.PushSecretStoreRef) error {
	for i, d := range dataToList {
		if d.StoreRef == nil {
			return fmt.Errorf("dataTo[%d]: storeRef is required", i)
		}
		if d.StoreRef.Name == "" && d.StoreRef.LabelSelector == nil {
			return fmt.Errorf("dataTo[%d]: storeRef must have name or labelSelector", i)
		}
		if d.StoreRef.Name != "" && !storeRefExistsInList(d.StoreRef, storeRefs) {
			return fmt.Errorf("dataTo[%d]: storeRef %q not found in secretStoreRefs", i, d.StoreRef.Name)
		}
	}
	return nil
}

// storeRefExistsInList checks if ref matches any entry in storeRefs.
func storeRefExistsInList(ref *esapi.PushSecretStoreRef, storeRefs []esapi.PushSecretStoreRef) bool {
	refKind := ref.Kind
	if refKind == "" {
		refKind = esv1.SecretStoreKind
	}
	for _, sr := range storeRefs {
		srKind := sr.Kind
		if srKind == "" {
			srKind = esv1.SecretStoreKind
		}
		// Skip if kinds don't match
		if srKind != refKind {
			continue
		}
		if sr.Name != "" && sr.Name == ref.Name {
			return true
		}
		// Can't validate labelSelector statically - assume it could match if kinds are compatible
		if sr.LabelSelector != nil {
			return true
		}
	}
	return false
}

// rewriteWithKeyMapping applies rewrites and returns originalKey -> rewrittenKey mapping.
func rewriteWithKeyMapping(rewrites []esapi.PushSecretRewrite, data map[string][]byte) (map[string]string, error) {
	// Initialize: each key maps to itself
	keyMap := make(map[string]string, len(data))
	for k := range data {
		keyMap[k] = k
	}

	// Apply each rewrite operation
	for i, op := range rewrites {
		newKeyMap := make(map[string]string, len(keyMap))
		for origKey, currentKey := range keyMap {
			newKey, err := applyRewriteToKey(op, currentKey)
			if err != nil {
				return nil, fmt.Errorf("rewrite[%d] on key %q: %w", i, currentKey, err)
			}
			newKeyMap[origKey] = newKey
		}
		keyMap = newKeyMap
	}

	return keyMap, nil
}

// applyRewriteToKey applies a single rewrite operation to a key.
func applyRewriteToKey(op esapi.PushSecretRewrite, key string) (string, error) {
	switch {
	case op.Regexp != nil:
		re, err := regexp.Compile(op.Regexp.Source)
		if err != nil {
			return "", fmt.Errorf("invalid regexp: %w", err)
		}
		return re.ReplaceAllString(key, op.Regexp.Target), nil
	case op.Transform != nil:
		tmpl, err := template.New("t").Funcs(estemplate.FuncMap()).Parse(op.Transform.Template)
		if err != nil {
			return "", fmt.Errorf("invalid template: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, map[string]string{"value": key}); err != nil {
			return "", fmt.Errorf("template exec: %w", err)
		}
		return buf.String(), nil
	default:
		return key, nil
	}
}

// resolveSourceKeyConflicts merges dataTo and explicit data entries.
// When both reference the same source secret key, explicit data wins.
func resolveSourceKeyConflicts(dataToEntries, explicitData []esapi.PushSecretData) []esapi.PushSecretData {
	// Build set of source keys from explicit data
	explicitSourceKeys := make(map[string]struct{}, len(explicitData))
	for _, data := range explicitData {
		explicitSourceKeys[data.GetSecretKey()] = struct{}{}
	}

	// Keep dataTo entries whose source key is NOT in explicit data
	result := make([]esapi.PushSecretData, 0, len(dataToEntries)+len(explicitData))
	for _, data := range dataToEntries {
		if _, exists := explicitSourceKeys[data.GetSecretKey()]; !exists {
			result = append(result, data)
		}
	}

	// Add all explicit data entries (they always take precedence)
	return append(result, explicitData...)
}

// validateRemoteKeyUniqueness ensures no two entries push to the same remote location.
// The remote location is defined by (remoteKey, property) tuple.
func validateRemoteKeyUniqueness(entries []esapi.PushSecretData) error {
	type remoteLocation struct {
		remoteKey string
		property  string
	}

	seen := make(map[remoteLocation]string) // location -> source key (for error message)

	for _, data := range entries {
		loc := remoteLocation{
			remoteKey: data.GetRemoteKey(),
			property:  data.GetProperty(),
		}
		sourceKey := data.GetSecretKey()

		if existingSource, exists := seen[loc]; exists {
			if loc.property != "" {
				return fmt.Errorf(
					"duplicate remote key %q with property %q: source keys %q and %q both map to the same destination",
					loc.remoteKey, loc.property, existingSource, sourceKey)
			}
			return fmt.Errorf(
				"duplicate remote key %q: source keys %q and %q both map to the same destination",
				loc.remoteKey, existingSource, sourceKey)
		}
		seen[loc] = sourceKey
	}

	return nil
}

// mergeDataEntries combines dataTo and explicit data entries.
// It resolves source key conflicts (explicit wins) and validates no duplicate remote destinations.
func mergeDataEntries(dataToEntries, explicitData []esapi.PushSecretData) ([]esapi.PushSecretData, error) {
	merged := resolveSourceKeyConflicts(dataToEntries, explicitData)

	if err := validateRemoteKeyUniqueness(merged); err != nil {
		return nil, err
	}

	return merged, nil
}
