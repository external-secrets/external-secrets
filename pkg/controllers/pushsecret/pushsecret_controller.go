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

package pushsecret

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	v1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errFailedGetSecret        = "could not get source secret"
	errPatchStatus            = "error merging"
	errGetSecretStore         = "could not get SecretStore %q, %w"
	errGetClusterSecretStore  = "could not get ClusterSecretStore %q, %w"
	errGetProviderFailed      = "could not start provider"
	errGetSecretsClientFailed = "could not start secrets client"
	errCloseStoreClient       = "error when calling provider close method"
	errSetSecretFailed        = "could not write remote ref %v to target secretstore %v: %v"
	errFailedSetSecret        = "set secret failed: %v"
	pushSecretFinalizer       = "pushsecret.externalsecrets.io/finalizer"
)

type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	RequeueInterval time.Duration
	ControllerClass string
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("pushsecret", req.NamespacedName)
	var ps esapi.PushSecret
	err := r.Get(ctx, req.NamespacedName, &ps)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
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
		if err != nil {
			log.Error(err, errPatchStatus)
		}
	}()
	// finalizer logic
	if ps.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&ps, pushSecretFinalizer) {
			controllerutil.AddFinalizer(&ps, pushSecretFinalizer)
			err := r.Client.Update(ctx, &ps, &client.UpdateOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("could not update finalizers: %w", err)
			}
			return ctrl.Result{}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(&ps, pushSecretFinalizer) {
			// trigger a cleanup with no Synced Map
			badState, err := r.DeleteSecretFromProviders(ctx, &ps, esapi.SyncedPushSecretsMap{})
			if err != nil {
				msg := fmt.Sprintf("Failed to Delete Secrets from Provider: %v", err)
				cond := NewPushSecretCondition(esapi.PushSecretReady, v1.ConditionFalse, esapi.ReasonErrored, msg)
				ps = SetPushSecretCondition(ps, *cond)
				r.SetSyncedSecrets(&ps, badState)
				r.recorder.Event(&ps, v1.EventTypeWarning, esapi.ReasonErrored, msg)
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&ps, pushSecretFinalizer)
			err = r.Client.Update(ctx, &ps, &client.UpdateOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("could not update finalizers: %w", err)
			}
			return ctrl.Result{}, nil
		}
	}

	secret, err := r.GetSecret(ctx, ps)
	if err != nil {
		cond := NewPushSecretCondition(esapi.PushSecretReady, v1.ConditionFalse, esapi.ReasonErrored, errFailedGetSecret)
		ps = SetPushSecretCondition(ps, *cond)
		r.recorder.Event(&ps, v1.EventTypeWarning, esapi.ReasonErrored, errFailedGetSecret)
		return ctrl.Result{}, err
	}
	secretStores, err := r.GetSecretStores(ctx, ps)
	if err != nil {
		cond := NewPushSecretCondition(esapi.PushSecretReady, v1.ConditionFalse, esapi.ReasonErrored, err.Error())
		ps = SetPushSecretCondition(ps, *cond)
		r.recorder.Event(&ps, v1.EventTypeWarning, esapi.ReasonErrored, err.Error())
		return ctrl.Result{}, err
	}

	syncedSecrets, err := r.PushSecretToProviders(ctx, secretStores, ps, secret)
	if err != nil {
		msg := fmt.Sprintf(errFailedSetSecret, err)
		cond := NewPushSecretCondition(esapi.PushSecretReady, v1.ConditionFalse, esapi.ReasonErrored, msg)
		ps = SetPushSecretCondition(ps, *cond)
		r.SetSyncedSecrets(&ps, syncedSecrets)
		r.recorder.Event(&ps, v1.EventTypeWarning, esapi.ReasonErrored, msg)
		return ctrl.Result{}, err
	}
	switch ps.Spec.DeletionPolicy {
	case esapi.PushSecretDeletionPolicyDelete:
		badSyncState, err := r.DeleteSecretFromProviders(ctx, &ps, syncedSecrets)
		if err != nil {
			msg := fmt.Sprintf("Failed to Delete Secrets from Provider: %v", err)
			cond := NewPushSecretCondition(esapi.PushSecretReady, v1.ConditionFalse, esapi.ReasonErrored, msg)
			ps = SetPushSecretCondition(ps, *cond)
			r.SetSyncedSecrets(&ps, badSyncState)
			r.recorder.Event(&ps, v1.EventTypeWarning, esapi.ReasonErrored, msg)
			return ctrl.Result{}, err
		}
	case esapi.PushSecretDeletionPolicyNone:
	default:
	}
	msg := "PushSecret synced successfully"
	cond := NewPushSecretCondition(esapi.PushSecretReady, v1.ConditionTrue, esapi.ReasonSynced, msg)
	ps = SetPushSecretCondition(ps, *cond)
	r.SetSyncedSecrets(&ps, syncedSecrets)
	r.recorder.Event(&ps, v1.EventTypeNormal, esapi.ReasonSynced, msg)
	return ctrl.Result{RequeueAfter: refreshInt}, nil
}
func (r *Reconciler) SetSyncedSecrets(ps *esapi.PushSecret, status esapi.SyncedPushSecretsMap) {
	ps.Status.SyncedPushSecrets = status
}

func (r *Reconciler) DeleteSecretFromProviders(ctx context.Context, ps *esapi.PushSecret, newMap esapi.SyncedPushSecretsMap) (esapi.SyncedPushSecretsMap, error) {
	out := newMap.DeepCopy()
	for k, v := range ps.Status.SyncedPushSecrets {
		_, ok := out[k]
		if !ok {
			out[k] = make(map[string]esapi.PushSecretData)
		}
		for kk, vv := range v {
			out[k][kk] = vv
		}
	}
	for storeName, oldData := range ps.Status.SyncedPushSecrets {
		storeRef := esapi.PushSecretStoreRef{
			Name: strings.Split(storeName, "/")[1],
			Kind: strings.Split(storeName, "/")[0],
		}
		store, err := r.getSecretStoreFromName(ctx, storeRef, ps.Namespace)
		if err != nil {
			return out, err
		}
		client, err := r.getClientFromStore(ctx, store, ps)
		if err != nil {
			return out, fmt.Errorf("could not get secrets client for store %v: %w", store.GetName(), err)
		}
		defer func() { //nolint
			err := client.Close(ctx)
			if err != nil {
				r.Log.Error(err, errCloseStoreClient)
			}
		}()
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
				delete(out[storeName], oldRef.Match.RemoteRef.RemoteKey)
			}
		}
	}
	return out, nil
}

func (r *Reconciler) DeleteAllSecretsFromStore(ctx context.Context, client v1beta1.SecretsClient, data map[string]esapi.PushSecretData) error {
	for _, v := range data {
		err := r.DeleteSecretFromStore(ctx, client, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) DeleteSecretFromStore(ctx context.Context, client v1beta1.SecretsClient, data esapi.PushSecretData) error {
	return client.DeleteSecret(ctx, data.Match.RemoteRef)
}

func (r *Reconciler) getClientFromStore(ctx context.Context, store v1beta1.GenericStore, ps *esapi.PushSecret) (v1beta1.SecretsClient, error) {
	provider, err := v1beta1.GetProvider(store)
	if err != nil {
		return nil, fmt.Errorf(errGetProviderFailed)
	}
	client, err := provider.NewClient(ctx, store, r.Client, ps.Namespace)
	if err != nil {
		return nil, fmt.Errorf(errGetSecretsClientFailed)
	}
	return client, nil
}

func (r *Reconciler) PushSecretToProviders(ctx context.Context, stores map[esapi.PushSecretStoreRef]v1beta1.GenericStore, ps esapi.PushSecret, secret *v1.Secret) (esapi.SyncedPushSecretsMap, error) {
	out := esapi.SyncedPushSecretsMap{}
	for ref, store := range stores {
		storeKey := fmt.Sprintf("%v/%v", ref.Kind, store.GetName())
		out[storeKey] = make(map[string]esapi.PushSecretData)
		client, err := r.getClientFromStore(ctx, store, &ps)
		if err != nil {
			return out, fmt.Errorf("could not get secrets client for store %v: %w", store.GetName(), err)
		}
		defer func() { //nolint
			err := client.Close(ctx)
			if err != nil {
				r.Log.Error(err, errCloseStoreClient)
			}
		}()
		for _, ref := range ps.Spec.Data {
			secretValue, ok := secret.Data[ref.Match.SecretKey]
			if !ok {
				return out, fmt.Errorf("secret key %v does not exist", ref.Match.SecretKey)
			}
			err := client.SetSecret(ctx, secretValue, ref.Match.RemoteRef)
			if err != nil {
				return out, fmt.Errorf(errSetSecretFailed, ref.Match.SecretKey, store.GetName(), err)
			}
			out[storeKey][ref.Match.RemoteRef.RemoteKey] = ref
		}
	}
	return out, nil
}
func (r *Reconciler) GetSecret(ctx context.Context, ps esapi.PushSecret) (*v1.Secret, error) {
	secretName := types.NamespacedName{Name: ps.Spec.Selector.Secret.Name, Namespace: ps.Namespace}
	secret := &v1.Secret{}
	err := r.Client.Get(ctx, secretName, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (r *Reconciler) GetSecretStores(ctx context.Context, ps esapi.PushSecret) (map[esapi.PushSecretStoreRef]v1beta1.GenericStore, error) {
	stores := make(map[esapi.PushSecretStoreRef]v1beta1.GenericStore)
	for _, refStore := range ps.Spec.SecretStoreRefs {
		if refStore.LabelSelector != nil {
			labelSelector, err := metav1.LabelSelectorAsSelector(refStore.LabelSelector)
			if err != nil {
				return nil, fmt.Errorf("could not convert labels: %w", err)
			}
			if refStore.Kind == v1beta1.ClusterSecretStoreKind {
				clusterSecretStoreList := v1beta1.ClusterSecretStoreList{}
				err = r.List(ctx, &clusterSecretStoreList, &client.ListOptions{LabelSelector: labelSelector})
				if err != nil {
					return nil, fmt.Errorf("could not list cluster Secret Stores: %w", err)
				}
				for k, v := range clusterSecretStoreList.Items {
					key := esapi.PushSecretStoreRef{
						Name: v.Name,
						Kind: v1beta1.ClusterSecretStoreKind,
					}
					stores[key] = &clusterSecretStoreList.Items[k]
				}
			} else {
				secretStoreList := v1beta1.SecretStoreList{}
				err = r.List(ctx, &secretStoreList, &client.ListOptions{LabelSelector: labelSelector})
				if err != nil {
					return nil, fmt.Errorf("could not list Secret Stores: %w", err)
				}
				for k, v := range secretStoreList.Items {
					key := esapi.PushSecretStoreRef{
						Name: v.Name,
						Kind: v1beta1.SecretStoreKind,
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

func (r *Reconciler) getSecretStoreFromName(ctx context.Context, refStore esapi.PushSecretStoreRef, ns string) (v1beta1.GenericStore, error) {
	if refStore.Name == "" {
		return nil, fmt.Errorf("refStore Name must be provided")
	}
	ref := types.NamespacedName{
		Name: refStore.Name,
	}
	if refStore.Kind == v1beta1.ClusterSecretStoreKind {
		var store v1beta1.ClusterSecretStore
		err := r.Get(ctx, ref, &store)
		if err != nil {
			return nil, fmt.Errorf(errGetClusterSecretStore, ref.Name, err)
		}
		return &store, nil
	}
	ref.Namespace = ns
	var store v1beta1.SecretStore
	err := r.Get(ctx, ref, &store)
	if err != nil {
		return nil, fmt.Errorf(errGetSecretStore, ref.Name, err)
	}
	return &store, nil
}
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("pushsecret")

	return ctrl.NewControllerManagedBy(mgr).
		For(&esapi.PushSecret{}).
		Complete(r)
}

func NewPushSecretCondition(condType esapi.PushSecretConditionType, status v1.ConditionStatus, reason, message string) *esapi.PushSecretStatusCondition {
	return &esapi.PushSecretStatusCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

func SetPushSecretCondition(gs esapi.PushSecret, condition esapi.PushSecretStatusCondition) esapi.PushSecret {
	status := gs.Status
	currentCond := GetPushSecretCondition(status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status &&
		currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
		return gs
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	status.Conditions = append(filterOutCondition(status.Conditions, condition.Type), condition)
	gs.Status = status
	return gs
}

// filterOutCondition returns an empty set of conditions with the provided type.
func filterOutCondition(conditions []esapi.PushSecretStatusCondition, condType esapi.PushSecretConditionType) []esapi.PushSecretStatusCondition {
	newConditions := make([]esapi.PushSecretStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

// GetSecretStoreCondition returns the condition with the provided type.
func GetPushSecretCondition(status esapi.PushSecretStatus, condType esapi.PushSecretConditionType) *esapi.PushSecretStatusCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}
