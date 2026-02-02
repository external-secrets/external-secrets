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
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	secrets, err := r.resolveSecrets(ctx, &ps)
	if err != nil {
		// Handle source secret deletion with DeletionPolicy=Delete
		if ps.Spec.DeletionPolicy == esapi.PushSecretDeletionPolicyDelete && len(ps.Status.SyncedPushSecrets) > 0 {
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
		out, err := r.handlePushSecretDataForStore(ctx, ps, secret, out, mgr, store.GetName(), ref.Kind)
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
	storeName, refKind string,
) (esapi.SyncedPushSecretsMap, error) {
	storeKey := fmt.Sprintf("%v/%v", refKind, storeName)
	out[storeKey] = make(map[string]esapi.PushSecretData)
	storeRef := esv1.SecretStoreRef{
		Name: storeName,
		Kind: refKind,
	}
	secretClient, err := mgr.Get(ctx, storeRef, ps.GetNamespace(), nil)
	if err != nil {
		return out, fmt.Errorf("could not get secrets client for store %v: %w", storeName, err)
	}

	// Create a copy of the secret for this store to avoid mutating the shared secret
	storeSecret := secret.DeepCopy()

	// Expand dataTo entries into PushSecretData
	dataToEntries, err := r.expandDataTo(&ps, storeSecret)
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
		secretData, err := esutils.ReverseKeys(data.ConversionStrategy, originalStoreSecretData)
		if err != nil {
			return nil, fmt.Errorf(errConvert, err)
		}
		storeSecret.Data = secretData
		key := data.GetSecretKey()
		if !secretKeyExists(key, storeSecret) {
			return out, fmt.Errorf("secret key %v does not exist", key)
		}
		switch ps.Spec.UpdatePolicy {
		case esapi.PushSecretUpdatePolicyIfNotExists:
			exists, err := secretClient.SecretExists(ctx, data.Match.RemoteRef)
			if err != nil {
				return out, fmt.Errorf("could not verify if secret exists in store: %w", err)
			} else if exists {
				out[storeKey][statusRef(data)] = data
				continue
			}
		case esapi.PushSecretUpdatePolicyReplace:
		default:
		}
		if err := secretClient.PushSecret(ctx, storeSecret, data); err != nil {
			return out, fmt.Errorf(errSetSecretFailed, key, storeName, err)
		}
		out[storeKey][statusRef(data)] = data
	}
	return out, nil
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

// expandDataTo converts dataTo entries into PushSecretData by matching keys and applying rewrites.
func (r *Reconciler) expandDataTo(ps *esapi.PushSecret, secret *v1.Secret) ([]esapi.PushSecretData, error) {
	if len(ps.Spec.DataTo) == 0 {
		return nil, nil
	}

	allData := make([]esapi.PushSecretData, 0)

	// Track remote keys across all dataTo entries to detect duplicates
	overallRemoteKeys := make(map[string]string) // remoteKey -> "dataTo[i]:sourceKey"

	// Process each dataTo entry
	for i, dataTo := range ps.Spec.DataTo {
		// Apply conversion strategy BEFORE matching and rewriting
		// This ensures that keys are matched against their converted names
		convertedData, err := esutils.ReverseKeys(dataTo.ConversionStrategy, secret.Data)
		if err != nil {
			return nil, fmt.Errorf("dataTo[%d]: conversion failed: %w", i, err)
		}

		// Get all keys from the converted secret data
		allKeys := make([]string, 0, len(convertedData))
		for key := range convertedData {
			allKeys = append(allKeys, key)
		}

		// Match keys based on pattern (using converted keys)
		matchedKeys, err := matchKeys(allKeys, dataTo.Match)
		if err != nil {
			return nil, fmt.Errorf("dataTo[%d]: match failed: %w", i, err)
		}

		if len(matchedKeys) == 0 {
			r.Log.Info("dataTo entry matched no keys", "index", i)
			continue
		}

		// Filter convertedData to only include matched keys
		matchedData := make(map[string][]byte, len(matchedKeys))
		for _, key := range matchedKeys {
			matchedData[key] = convertedData[key]
		}

		// Apply rewrites using the shared esutils.RewriteMap function
		rewrittenData, err := esutils.RewriteMap(dataTo.Rewrite, matchedData)
		if err != nil {
			return nil, fmt.Errorf("dataTo[%d]: rewrite failed: %w", i, err)
		}

		// Build sourceKey -> remoteKey mapping by comparing original and rewritten keys
		keyMap := buildKeyMapping(matchedData, rewrittenData)

		// Validate that no remote key is empty
		for sourceKey, remoteKey := range keyMap {
			if remoteKey == "" {
				return nil, fmt.Errorf("dataTo[%d]: empty remote key produced for source key %q", i, sourceKey)
			}
		}

		// Check for duplicate remote keys within this dataTo entry and across all dataTo entries
		for sourceKey, remoteKey := range keyMap {
			if existingSource, exists := overallRemoteKeys[remoteKey]; exists {
				return nil, fmt.Errorf("dataTo[%d]: duplicate remote key %q from source key %q (conflicts with %s)", i, remoteKey, sourceKey, existingSource)
			}
			overallRemoteKeys[remoteKey] = fmt.Sprintf("dataTo[%d]:%s", i, sourceKey)
		}

		// Create PushSecretData entries
		// Note: SecretKey now references the converted key name
		for sourceKey, remoteKey := range keyMap {
			data := esapi.PushSecretData{
				Match: esapi.PushSecretMatch{
					SecretKey: sourceKey, // This is now the converted key name
					RemoteRef: esapi.PushSecretRemoteRef{
						RemoteKey: remoteKey,
					},
				},
				Metadata:           dataTo.Metadata,
				ConversionStrategy: dataTo.ConversionStrategy,
			}
			allData = append(allData, data)
		}

		r.Log.Info("expanded dataTo entry", "index", i, "matchedKeys", len(matchedKeys), "created", len(keyMap))
	}

	return allData, nil
}

// buildKeyMapping maps original keys to rewritten keys by matching values.
func buildKeyMapping(original, rewritten map[string][]byte) map[string]string {
	result := make(map[string]string, len(original))

	// Reverse lookup: value -> rewritten key (rewrites only change keys, not values)
	valueToRewrittenKey := make(map[string]string, len(rewritten))
	for key, value := range rewritten {
		valueToRewrittenKey[string(value)] = key
	}

	// Map original keys to their rewritten counterparts by matching values
	for originalKey, originalValue := range original {
		if rewrittenKey, exists := valueToRewrittenKey[string(originalValue)]; exists {
			result[originalKey] = rewrittenKey
		} else {
			// If no rewrite happened, key stays the same
			result[originalKey] = originalKey
		}
	}

	return result
}

// mergeDataEntries merges dataTo-expanded entries with explicit data entries.
// Explicit data entries override dataTo entries for the same source secret key.
// Returns an error if duplicate remote keys with the same property are detected.
func mergeDataEntries(dataToEntries []esapi.PushSecretData, explicitData []esapi.PushSecretData) ([]esapi.PushSecretData, error) {
	// Create a map of source secretKey -> data from explicit data
	explicitMap := make(map[string]esapi.PushSecretData)
	for _, data := range explicitData {
		key := data.GetSecretKey()
		explicitMap[key] = data
	}

	// Add dataTo entries that don't conflict with explicit data (based on source key)
	result := make([]esapi.PushSecretData, 0, len(dataToEntries)+len(explicitData))
	for _, data := range dataToEntries {
		key := data.GetSecretKey()
		if _, exists := explicitMap[key]; !exists {
			result = append(result, data)
		}
	}

	// Add all explicit data entries
	result = append(result, explicitData...)

	// Check for duplicate remote keys with the same property.
	// Use a struct as the map key to avoid collisions from string concatenation.
	type compositeKey struct {
		remoteKey string
		property  string
	}
	type compositeKeyInfo struct {
		sourceKey string
	}
	remoteKeys := make(map[compositeKey]compositeKeyInfo)
	for _, data := range result {
		remoteKey := data.GetRemoteKey()
		property := data.GetProperty()
		sourceKey := data.GetSecretKey()

		// Create composite key using struct (collision-free)
		key := compositeKey{
			remoteKey: remoteKey,
			property:  property,
		}

		if existing, exists := remoteKeys[key]; exists {
			if property != "" {
				return nil, fmt.Errorf("duplicate remote key %q with property %q: source keys %q and %q both map to the same remote key and property", remoteKey, property, existing.sourceKey, sourceKey)
			}
			return nil, fmt.Errorf("duplicate remote key %q: source keys %q and %q both map to the same remote key", remoteKey, existing.sourceKey, sourceKey)
		}
		remoteKeys[key] = compositeKeyInfo{
			sourceKey: sourceKey,
		}
	}

	return result, nil
}
