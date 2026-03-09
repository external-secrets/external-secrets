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

package secretstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrlreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/metrics"

	// Load registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/register"
)

const (
	errStoreClient          = "could not get provider client: %w"
	errValidationFailed     = "could not validate provider: %w"
	errValidationUnknownMsg = "could not determine validation status"
	errPatchStatus          = "unable to patch status: %w"
	errUnableCreateClient   = "unable to create client"
	errUnableValidateStore  = "unable to validate store"

	msgStoreValidated     = "store validated"
	msgStoreNotMaintained = "store isn't currently maintained. Please plan and prepare accordingly."
	msgStoreDeprecated    = "store is deprecated and will be removed on the next minor release. Please plan and prepare accordingly."

	// Finalizer for SecretStores when they have PushSecrets with DeletionPolicy=Delete.
	secretStoreFinalizer = "secretstore.externalsecrets.io/finalizer"
)

var errValidationUnknown = errors.New(errValidationUnknownMsg)

// Opts holds the options for the reconcile function.
type Opts struct {
	ControllerClass string
	GaugeVecGetter  metrics.GaugeVevGetter
	Recorder        record.EventRecorder
	RequeueInterval time.Duration
}

func reconcile(ctx context.Context, req ctrl.Request, ss esapi.GenericStore, cl client.Client, isPushSecretEnabled bool, log logr.Logger, opts Opts) (ctrl.Result, error) {
	if !ShouldProcessStore(ss, opts.ControllerClass) {
		log.V(1).Info("skip store")
		return ctrl.Result{}, nil
	}

	// Manage finalizer if PushSecret feature is enabled.
	if isPushSecretEnabled {
		finalizersUpdated, err := handleFinalizer(ctx, cl, ss)
		if err != nil {
			return ctrl.Result{}, err
		}

		if finalizersUpdated {
			log.V(1).Info("updating resource with finalizer changes")
			if err := cl.Update(ctx, ss); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	requeueInterval := opts.RequeueInterval

	if ss.GetSpec().RefreshInterval != 0 {
		requeueInterval = time.Second * time.Duration(ss.GetSpec().RefreshInterval)
	}

	// patch status when done processing
	p := client.MergeFrom(ss.Copy())
	defer func() {
		err := cl.Status().Patch(ctx, ss, p)
		if err != nil {
			log.Error(err, errPatchStatus)
		}
	}()

	// validateStore modifies the store conditions
	// we have to patch the status
	log.V(1).Info("validating")
	err := validateStore(ctx, req.Namespace, opts.ControllerClass, ss, cl, opts.GaugeVecGetter, opts.Recorder)
	if err != nil {
		log.Error(err, "unable to validate store")
		// in case of validation status unknown, validateStore will mark
		// the store as ready but we should show ReasonValidationUnknown
		if errors.Is(err, errValidationUnknown) {
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
		return ctrl.Result{}, err
	}
	storeProvider, err := esapi.GetProvider(ss)
	if err != nil {
		return ctrl.Result{}, err
	}

	isMaintained, err := esapi.GetMaintenanceStatus(ss)
	if err != nil {
		return ctrl.Result{}, err
	}
	annotations := ss.GetAnnotations()
	_, ok := annotations["external-secrets.io/ignore-maintenance-checks"]
	if !ok {
		switch isMaintained {
		case esapi.MaintenanceStatusNotMaintained:
			opts.Recorder.Event(ss, v1.EventTypeWarning, esapi.StoreUnmaintained, msgStoreNotMaintained)
		case esapi.MaintenanceStatusDeprecated:
			opts.Recorder.Event(ss, v1.EventTypeWarning, esapi.StoreDeprecated, msgStoreDeprecated)
		case esapi.MaintenanceStatusMaintained:
		default:
			// no warnings
		}
	}

	capStatus := esapi.SecretStoreStatus{
		Capabilities: storeProvider.Capabilities(),
		Conditions:   ss.GetStatus().Conditions,
	}
	ss.SetStatus(capStatus)

	opts.Recorder.Event(ss, v1.EventTypeNormal, esapi.ReasonStoreValid, msgStoreValidated)
	cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionTrue, esapi.ReasonStoreValid, msgStoreValidated)
	SetExternalSecretCondition(ss, *cond, opts.GaugeVecGetter)

	return ctrl.Result{
		RequeueAfter: requeueInterval,
	}, err
}

// validateStore tries to construct a new client
// if it fails sets a condition and writes events.
func validateStore(ctx context.Context, namespace, controllerClass string, store esapi.GenericStore,
	client client.Client, gaugeVecGetter metrics.GaugeVevGetter, recorder record.EventRecorder) error {
	mgr := NewManager(client, controllerClass, false)
	defer func() {
		_ = mgr.Close(ctx)
	}()
	cl, err := mgr.GetFromStore(ctx, store, namespace)
	if err != nil {
		cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionFalse, esapi.ReasonInvalidProviderConfig, errUnableCreateClient)
		SetExternalSecretCondition(store, *cond, gaugeVecGetter)
		recorder.Event(store, v1.EventTypeWarning, esapi.ReasonInvalidProviderConfig, err.Error())
		return fmt.Errorf(errStoreClient, err)
	}
	validationResult, err := cl.Validate()
	if err != nil {
		if validationResult == esapi.ValidationResultUnknown {
			cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionTrue, esapi.ReasonValidationUnknown, errValidationUnknownMsg)
			SetExternalSecretCondition(store, *cond, gaugeVecGetter)
			recorder.Event(store, v1.EventTypeWarning, esapi.ReasonValidationUnknown, err.Error())
			return errValidationUnknown
		}
		cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionFalse, esapi.ReasonInvalidProviderConfig, errUnableValidateStore)
		SetExternalSecretCondition(store, *cond, gaugeVecGetter)
		recorder.Event(store, v1.EventTypeWarning, esapi.ReasonInvalidProviderConfig, err.Error())
		return fmt.Errorf(errValidationFailed, err)
	}

	return nil
}

// ShouldProcessStore returns true if the store should be processed.
func ShouldProcessStore(store esapi.GenericStore, class string) bool {
	if store == nil || store.GetSpec().Controller == "" || store.GetSpec().Controller == class {
		return true
	}

	return false
}

// handleFinalizer manages the finalizer for ClusterSecretStores and SecretStores.
func handleFinalizer(ctx context.Context, cl client.Client, store esapi.GenericStore) (finalizersUpdated bool, err error) {
	log := logr.FromContextOrDiscard(ctx)
	hasPushSecretsWithDeletePolicy, err := hasPushSecretsWithDeletePolicy(ctx, cl, store)
	if err != nil {
		return false, fmt.Errorf("failed to check PushSecrets: %w", err)
	}

	storeKind := store.GetKind()

	// If the store is being deleted and has the finalizer, check if we can remove it
	if !store.GetObjectMeta().DeletionTimestamp.IsZero() {
		if hasPushSecretsWithDeletePolicy {
			log.Info("cannot remove finalizer, there are still PushSecrets with DeletionPolicy=Delete that reference this store")
			return false, nil
		}

		if controllerutil.RemoveFinalizer(store, secretStoreFinalizer) {
			log.Info(fmt.Sprintf("removed finalizer from %s during deletion", storeKind))
			return true, nil
		}

		return false, nil
	}

	// If the store is not being deleted, manage the finalizer based on PushSecrets
	if hasPushSecretsWithDeletePolicy {
		if controllerutil.AddFinalizer(store, secretStoreFinalizer) {
			log.Info(fmt.Sprintf("added finalizer to %s due to PushSecrets with DeletionPolicy=Delete", storeKind))
			return true, nil
		}
	} else {
		if controllerutil.RemoveFinalizer(store, secretStoreFinalizer) {
			log.Info(fmt.Sprintf("removed finalizer from %s, no more PushSecrets with DeletionPolicy=Delete", storeKind))
			return true, nil
		}
	}

	return false, nil
}

// hasPushSecretsWithDeletePolicy checks if there are any PushSecrets with DeletionPolicy=Delete
// that reference this SecretStore using the controller-runtime index.
func hasPushSecretsWithDeletePolicy(ctx context.Context, cl client.Client, store esapi.GenericStore) (bool, error) {
	// Search for PushSecrets that have already synced from this store.
	found, err := hasSyncedPushSecrets(ctx, cl, store)
	if err != nil {
		return false, fmt.Errorf("failed to check for synced push secrets: %w", err)
	}
	if found {
		return true, nil
	}

	// Search for PushSecrets that reference this store, but may not have synced yet.
	found, err = hasUnsyncedPushSecretRefs(ctx, cl, store)
	if err != nil {
		return false, fmt.Errorf("failed to check for unsynced push secret refs: %w", err)
	}

	return found, nil
}

// hasSyncedPushSecrets uses the 'status.syncedPushSecrets' index from PushSecrets to efficiently find
// PushSecrets with DeletionPolicy=Delete that have already been synced from the given store.
func hasSyncedPushSecrets(ctx context.Context, cl client.Client, store esapi.GenericStore) (bool, error) {
	storeKey := fmt.Sprintf("%s/%s", store.GetKind(), store.GetName())

	opts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.syncedPushSecrets", storeKey),
	}

	if store.GetKind() == esapi.SecretStoreKind {
		opts.Namespace = store.GetNamespace()
	}

	var pushSecretList esv1alpha1.PushSecretList
	if err := cl.List(ctx, &pushSecretList, opts); err != nil {
		return false, err
	}

	// If any PushSecrets are found, return true. The index ensures they have DeletionPolicy=Delete.
	return len(pushSecretList.Items) > 0, nil
}

// hasUnsyncedPushSecretRefs searches for all PushSecrets with DeletionPolicy=Delete
// and checks if any of them reference the given store (by name or labelSelector).
// This is necessary for cases where the reference exists, but synchronization has not occurred yet.
func hasUnsyncedPushSecretRefs(ctx context.Context, cl client.Client, store esapi.GenericStore) (bool, error) {
	opts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.deletionPolicy", string(esv1alpha1.PushSecretDeletionPolicyDelete)),
	}

	if store.GetKind() == esapi.SecretStoreKind {
		opts.Namespace = store.GetNamespace()
	}

	var pushSecretList esv1alpha1.PushSecretList
	if err := cl.List(ctx, &pushSecretList, opts); err != nil {
		return false, err
	}

	for _, ps := range pushSecretList.Items {
		for _, storeRef := range ps.Spec.SecretStoreRefs {
			if storeMatchesRef(store, storeRef) {
				return true, nil
			}
		}
	}

	return false, nil
}

// findStoresForPushSecret finds SecretStores or ClusterSecretStores that should be reconciled when a PushSecret changes.
func findStoresForPushSecret(ctx context.Context, c client.Client, obj client.Object, storeList client.ObjectList) []ctrlreconcile.Request {
	ps, ok := obj.(*esv1alpha1.PushSecret)
	if !ok {
		return nil
	}

	var isClusterScoped bool
	switch storeList.(type) {
	case *esapi.ClusterSecretStoreList:
		isClusterScoped = true
	case *esapi.SecretStoreList:
		isClusterScoped = false
	default:
		return nil
	}

	listOpts := make([]client.ListOption, 0)
	if !isClusterScoped {
		listOpts = append(listOpts, client.InNamespace(ps.GetNamespace()))
	}

	if err := c.List(ctx, storeList, listOpts...); err != nil {
		return nil
	}

	requests := make([]ctrlreconcile.Request, 0)
	var stores []esapi.GenericStore

	switch sl := storeList.(type) {
	case *esapi.SecretStoreList:
		for i := range sl.Items {
			stores = append(stores, &sl.Items[i])
		}
	case *esapi.ClusterSecretStoreList:
		for i := range sl.Items {
			stores = append(stores, &sl.Items[i])
		}
	}

	for _, store := range stores {
		if shouldReconcileSecretStoreForPushSecret(store, ps) {
			req := ctrlreconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: store.GetName(),
				},
			}
			if !isClusterScoped {
				req.NamespacedName.Namespace = store.GetNamespace()
			}
			requests = append(requests, req)
		}
	}

	return requests
}
