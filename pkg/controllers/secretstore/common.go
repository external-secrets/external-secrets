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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/metrics"

	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

const (
	errStoreClient         = "could not get provider client: %w"
	errValidationFailed    = "could not validate provider: %w"
	errPatchStatus         = "unable to patch status: %w"
	errUnableCreateClient  = "unable to create client"
	errUnableValidateStore = "unable to validate store: %s"

	msgStoreValidated     = "store validated"
	msgStoreNotMaintained = "store isn't currently maintained. Please plan and prepare accordingly."

	// Finalizer for SecretStores when they have PushSecrets with DeletionPolicy=Delete.
	secretStoreFinalizer = "secretstore.externalsecrets.io/finalizer"
)

type Opts struct {
	ControllerClass string
	GaugeVecGetter  metrics.GaugeVevGetter
	Recorder        record.EventRecorder
	RequeueInterval time.Duration
}

func reconcile(ctx context.Context, req ctrl.Request, ss esapi.GenericStore, cl client.Client, log logr.Logger, opts Opts) (ctrl.Result, error) {
	if !ShouldProcessStore(ss, opts.ControllerClass) {
		log.V(1).Info("skip store")
		return ctrl.Result{}, nil
	}

	updated, err := handleFinalizer(ctx, cl, ss, log)
	if err != nil {
		return ctrl.Result{}, err
	}
	if updated {
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
	if err != nil && validationResult != esapi.ValidationResultUnknown {
		// Use ReasonInvalidProviderConfig for validation errors that indicate
		// invalid configuration (like empty server URL)
		reason := esapi.ReasonValidationFailed
		if validationResult == esapi.ValidationResultError {
			reason = esapi.ReasonInvalidProviderConfig
		}
		cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionFalse, reason, fmt.Sprintf(errUnableValidateStore, err))
		SetExternalSecretCondition(store, *cond, gaugeVecGetter)
		recorder.Event(store, v1.EventTypeWarning, reason, err.Error())
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
func handleFinalizer(ctx context.Context, cl client.Client, store esapi.GenericStore, log logr.Logger) (bool, error) {

	hasPushSecretsWithDeletePolicy, err := hasPushSecretsWithDeletePolicy(ctx, cl, store)
	if err != nil {
		return false, fmt.Errorf("failed to check PushSecrets: %w", err)
	}

	// If the store is being deleted and has the finalizer, check if we can remove it
	if !store.GetObjectMeta().DeletionTimestamp.IsZero() {
		if store.ContainsFinalizer(secretStoreFinalizer) {
			if hasPushSecretsWithDeletePolicy {
				log.Info("cannot remove finalizer, there are still PushSecrets with DeletionPolicy=Delete that reference this store")
				return false, nil
			}
			store.RemoveFinalizer(secretStoreFinalizer)
			log.Info(fmt.Sprintf("removed finalizer from %s", store.GetKind()))
			return true, nil
		}
		return false, nil
	}

	// If not being deleted, manage the finalizer based on whether there are PushSecrets with Delete policy
	if hasPushSecretsWithDeletePolicy {
		store.AddFinalizer(secretStoreFinalizer)
		log.Info(fmt.Sprintf("added finalizer to %s due to PushSecrets with DeletionPolicy=Delete", store.GetKind()))

	} else {
		store.RemoveFinalizer(secretStoreFinalizer)
		log.Info(fmt.Sprintf("removed finalizer from %s, no PushSecrets with DeletionPolicy=Delete found", store.GetKind()))
	}

	return true, nil
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

	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("status.syncedPushSecrets", storeName),
	}
	if storeNamespace != "" {
		listOpts.Namespace = storeNamespace
	}

	var pushSecretList esv1alpha1.PushSecretList
	if err := cl.List(ctx, &pushSecretList, listOpts); err != nil {
		return false, fmt.Errorf("failed to list PushSecrets by store index: %w", err)
	}

	// Check if any of these PushSecrets have DeletionPolicy=Delete
	for _, ps := range pushSecretList.Items {
		if ps.Spec.DeletionPolicy == esv1alpha1.PushSecretDeletionPolicyDelete {
			// Verify the store reference matches
			storeKey := fmt.Sprintf("%s/%s", storeKind, storeName)
			if _, hasPushed := ps.Status.SyncedPushSecrets[storeKey]; hasPushed {
				return true, nil
			}
		}
	}

	// Also check for PushSecrets that reference this store by name or labelSelector
	// but haven't pushed to it yet (for initial finalizer setup)
	listOpts = &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.deletionPolicy", string(esv1alpha1.PushSecretDeletionPolicyDelete)),
	}
	if storeNamespace != "" {
		listOpts.Namespace = storeNamespace
	}
	if err := cl.List(ctx, &pushSecretList, listOpts); err != nil {
		return false, fmt.Errorf("failed to list PushSecrets by deletionPolicy: %w", err)
	}

	for _, ps := range pushSecretList.Items {
		// Check if this PushSecret references our store
		for _, storeRef := range ps.Spec.SecretStoreRefs {
			if storeRef.Name == storeName {
				// Check if kind matches
				if storeRef.Kind == storeKind ||
					(storeRef.Kind == "" && storeKind == esapi.SecretStoreKind) ||
					(storeRef.Kind == "" && storeKind == esapi.ClusterSecretStoreKind) {
					return true, nil
				}
			}
		}

		// Check labelSelector match
		for _, storeRef := range ps.Spec.SecretStoreRefs {
			if storeRef.LabelSelector != nil &&
				(storeRef.Kind == storeKind ||
					(storeRef.Kind == "" && storeKind == esapi.SecretStoreKind) ||
					(storeRef.Kind == "" && storeKind == esapi.ClusterSecretStoreKind)) {
				selector, err := metav1.LabelSelectorAsSelector(storeRef.LabelSelector)
				if err != nil {
					continue
				}
				if selector.Matches(labels.Set(store.GetLabels())) {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
