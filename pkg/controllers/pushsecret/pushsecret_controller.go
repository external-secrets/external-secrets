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
		log.Error(err, "unable to get PushSecret")
		return ctrl.Result{}, fmt.Errorf("get resource: %w", err)
	}

	p := client.MergeFrom(ps.DeepCopy())
	defer func() {
		err := r.Client.Status().Patch(ctx, &ps, p)
		if err != nil {
			log.Error(err, errPatchStatus)
		}
	}()
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
	err = r.SetSecretToProviders(ctx, secretStores, ps, secret)
	if err != nil {
		msg := fmt.Sprintf(errFailedSetSecret, err)
		cond := NewPushSecretCondition(esapi.PushSecretReady, v1.ConditionFalse, esapi.ReasonErrored, msg)
		ps = SetPushSecretCondition(ps, *cond)
		r.recorder.Event(&ps, v1.EventTypeWarning, esapi.ReasonErrored, msg)
		return ctrl.Result{}, err
	}
	msg := "PushSecret synced successfully"
	cond := NewPushSecretCondition(esapi.PushSecretReady, v1.ConditionTrue, esapi.ReasonSynced, msg)
	ps = SetPushSecretCondition(ps, *cond)
	// Set status for PushSecret
	r.recorder.Event(&ps, v1.EventTypeNormal, esapi.ReasonSynced, msg)
	return ctrl.Result{}, nil
}

func (r *Reconciler) SetSecretToProviders(ctx context.Context, stores []v1beta1.GenericStore, ps esapi.PushSecret, secret *v1.Secret) error {
	for _, store := range stores {
		provider, err := v1beta1.GetProvider(store)
		if err != nil {
			return fmt.Errorf(errGetProviderFailed)
		}
		client, err := provider.NewClient(ctx, store, r.Client, ps.Namespace)
		if err != nil {
			return fmt.Errorf(errGetSecretsClientFailed)
		}
		defer func() {
			err := client.Close(ctx)
			if err != nil {
				r.Log.Error(err, errCloseStoreClient)
			}
		}()
		for _, ref := range ps.Spec.Data {
			for _, match := range ref.Match {
				secretValue, ok := secret.Data[match.SecretKey]
				if !ok {
					return fmt.Errorf("secret key %v does not exist", match.SecretKey)
				}
				for _, rK := range match.RemoteRefs {
					err := client.SetSecret(ctx, secretValue, rK)
					if err != nil {
						return fmt.Errorf(errSetSecretFailed, match.SecretKey, store.GetName(), err)
					}
				}
			}
		}
	}
	return nil
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

func (r *Reconciler) GetSecretStores(ctx context.Context, ps esapi.PushSecret) ([]v1beta1.GenericStore, error) {
	stores := make([]v1beta1.GenericStore, 0)
	for _, refStore := range ps.Spec.SecretStoreRefs {
		ref := types.NamespacedName{
			Name: refStore.Name,
		}

		if refStore.Kind == v1beta1.ClusterSecretStoreKind {
			var store v1beta1.ClusterSecretStore
			err := r.Get(ctx, ref, &store)
			if err != nil {
				return nil, fmt.Errorf(errGetClusterSecretStore, ref.Name, err)
			}
			stores = append(stores, &store)
		} else {
			ref.Namespace = ps.Namespace

			var store v1beta1.SecretStore
			err := r.Get(ctx, ref, &store)
			if err != nil {
				return nil, fmt.Errorf(errGetSecretStore, ref.Name, err)
			}
			stores = append(stores, &store)
		}
	}
	return stores, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("push-secret")

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
