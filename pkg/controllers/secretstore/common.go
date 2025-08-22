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

package secretstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrlreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/metrics"

	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

const (
	errStoreClient         = "could not get provider client: %w"
	errValidationFailed    = "could not validate provider: %w"
	errValidationUnknown   = "could not determine validation status: %s"
	errPatchStatus         = "unable to patch status: %w"
	errUnableCreateClient  = "unable to create client"
	errUnableValidateStore = "unable to validate store: %s"

	msgStoreValidated     = "store validated"
	msgStoreNotMaintained = "store isn't currently maintained. Please plan and prepare accordingly."

	// Finalizer for SecretStores when they have PushSecrets with DeletionPolicy=Delete.
	secretStoreFinalizer = "secretstore.externalsecrets.io/finalizer"
)

var validationUnknownError = errors.New("could not determine validation status")

type Opts struct {
	ControllerClass string
	GaugeVecGetter  metrics.GaugeVevGetter
	Recorder        record.EventRecorder
	RequeueInterval time.Duration
}

func reconcile(ctx context.Context, req ctrl.Request, ss esapi.GenericStore, cl client.Client, isPushSecretEnable bool, log logr.Logger, opts Opts) (ctrl.Result, error) {
	if !ShouldProcessStore(ss, opts.ControllerClass) {
		log.V(1).Info("skip store")
		return ctrl.Result{}, nil
	}

	finalizersUpdated, err := handleFinalizer(ctx, cl, ss, log, isPushSecretEnable)
	if err != nil {
		return ctrl.Result{}, err
	}

	if finalizersUpdated {
		log.V(1).Info("updating resource with finalizer changes")
		if err := cl.Update(ctx, ss); err != nil {
			return ctrl.Result{}, err
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
		if err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, errPatchStatus)
		}
	}()

	// validateStore modifies the store conditions
	// we have to patch the status
	log.V(1).Info("validating")
	err = validateStore(ctx, req.Namespace, opts.ControllerClass, ss, cl, opts.GaugeVecGetter, opts.Recorder)
	if err != nil {
		log.Error(err, "unable to validate store")
		// in case of validation status unknown, validateStore will mark
		// the store as ready but we should show ReasonValidationUnknown
		if errors.Is(err, validationUnknownError) {
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

	if !bool(isMaintained) && !ok {
		opts.Recorder.Event(ss, v1.EventTypeWarning, esapi.StoreUnmaintained, msgStoreNotMaintained)
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
			cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionTrue, esapi.ReasonValidationUnknown, fmt.Sprintf(errValidationUnknown, err))
			SetExternalSecretCondition(store, *cond, gaugeVecGetter)
			recorder.Event(store, v1.EventTypeWarning, esapi.ReasonValidationUnknown, err.Error())
			return validationUnknownError
		}
		cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionFalse, esapi.ReasonInvalidProviderConfig, fmt.Sprintf(errUnableValidateStore, err))
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
// It adds a finalizer when there are PushSecrets with DeletionPolicy=Delete that reference this store
// and removes it when there are no such PushSecrets.
func handleFinalizer(ctx context.Context, cl client.Client, store esapi.GenericStore, log logr.Logger, isPushSecretEnable bool) (finalizersUpdated bool, err error) {
	if !isPushSecretEnable {
		log.V(1).Info("skipping finalizer management, PushSecret feature is disabled")
		return false, nil
	}
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
	storeName := store.GetName()
	storeKind := store.GetKind()

	var storeNamespace string
	if storeKind == esapi.SecretStoreKind {
		storeNamespace = store.GetNamespace()
	}

	isKindMatch := func(refKind string) bool {
		if refKind == storeKind {
			return true
		}
		return refKind == "" && (storeKind == esapi.SecretStoreKind || storeKind == esapi.ClusterSecretStoreKind)
	}

	buildListOpts := func(namespace string) *client.ListOptions {
		opts := &client.ListOptions{}
		if namespace != "" {
			opts.Namespace = namespace
		}
		return opts
	}

	storeKeyToFind := fmt.Sprintf("%s/%s", storeKind, storeName)
	listOpts := buildListOpts(storeNamespace)
	listOpts.FieldSelector = fields.OneTermEqualSelector("status.syncedPushSecrets", storeKeyToFind)

	var pushSecretList esv1alpha1.PushSecretList
	if err := cl.List(ctx, &pushSecretList, listOpts); err != nil {
		return false, fmt.Errorf("failed to list PushSecrets by store index: %w", err)
	}

	// Check if any of these PushSecrets have DeletionPolicy=Delete
	for _, ps := range pushSecretList.Items {
		if ps.Spec.DeletionPolicy == esv1alpha1.PushSecretDeletionPolicyDelete {
			// Verify the store reference matches
			if _, hasPushed := ps.Status.SyncedPushSecrets[storeKeyToFind]; hasPushed {
				return true, nil
			}
		}
	}

	// Also check for PushSecrets that reference this store by name or labelSelector
	// but haven't pushed to it yet (for initial finalizer setup)
	listOpts = buildListOpts(storeNamespace)
	listOpts.FieldSelector = fields.OneTermEqualSelector("spec.deletionPolicy", string(esv1alpha1.PushSecretDeletionPolicyDelete))

	if err := cl.List(ctx, &pushSecretList, listOpts); err != nil {
		return false, fmt.Errorf("failed to list PushSecrets by deletionPolicy: %w", err)
	}

	for _, ps := range pushSecretList.Items {
		// Check if this PushSecret references our store by name or labelSelector
		for _, storeRef := range ps.Spec.SecretStoreRefs {
			// Check name match
			if storeRef.Name == storeName && isKindMatch(storeRef.Kind) {
				return true, nil
			}

			// Check labelSelector match
			if storeRef.LabelSelector != nil && isKindMatch(storeRef.Kind) {
				selector, err := metav1.LabelSelectorAsSelector(storeRef.LabelSelector)
				if err != nil {
					continue // Skip invalid selectors
				}
				if selector.Matches(labels.Set(store.GetLabels())) {
					return true, nil
				}
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

// shouldReconcileSecretStoreForPushSecret determines if a SecretStore should be reconciled
// when a PushSecret changes, based on whether the PushSecret references this store.
func shouldReconcileSecretStoreForPushSecret(store esapi.GenericStore, ps *esv1alpha1.PushSecret) bool {
	// Check if this PushSecret has pushed to this store
	storeKey := fmt.Sprintf("%s/%s", store.GetKind(), store.GetName())
	if _, hasPushed := ps.Status.SyncedPushSecrets[storeKey]; hasPushed {
		return true
	}
	// Also check if the PushSecret references this store in its spec
	for _, storeRef := range ps.Spec.SecretStoreRefs {
		refKind := storeRef.Kind
		if refKind == "" {
			refKind = esapi.SecretStoreKind
		}

		if storeRef.Name == store.GetName() && (storeRef.Kind == "" || (storeRef.Kind == esapi.SecretStoreKind || storeRef.Kind == esapi.ClusterSecretStoreKind)) {
			return true
		}
		// Check labelSelector match
		if storeRef.LabelSelector != nil && storeRef.Kind == esapi.SecretStoreKind {
			selector, err := metav1.LabelSelectorAsSelector(storeRef.LabelSelector)
			if err == nil && selector.Matches(labels.Set(store.GetLabels())) {
				return true
			}
		}
	}

	return false
}
