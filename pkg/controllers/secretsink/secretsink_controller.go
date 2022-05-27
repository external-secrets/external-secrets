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

package secretsink

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
	log := r.Log.WithValues("secretsink", req.NamespacedName)
	var ss esapi.SecretSink
	err := r.Get(ctx, req.NamespacedName, &ss)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "unable to get SecretSink")
		return ctrl.Result{}, fmt.Errorf("get resource: %w", err)
	}

	p := client.MergeFrom(ss.DeepCopy())
	defer func() {
		err := r.Client.Status().Patch(ctx, &ss, p)
		if err != nil {
			log.Error(err, errPatchStatus)
		}
	}()
	secret, err := r.GetSecret(ctx, ss)
	if err != nil {
		cond := NewSecretSinkCondition(esapi.SecretSinkReady, v1.ConditionFalse, "SecretSyncFailed", errFailedGetSecret)
		ss = SetSecretSinkCondition(ss, *cond)
		return ctrl.Result{}, err
	}
	secretStores, err := r.GetSecretStores(ctx, ss)
	if err != nil {
		cond := NewSecretSinkCondition(esapi.SecretSinkReady, v1.ConditionFalse, "SecretSyncFailed", err.Error())
		ss = SetSecretSinkCondition(ss, *cond)
	}
	err = r.SetSecretToProviders(ctx, secretStores, ss, secret)
	if err != nil {
		msg := fmt.Sprintf(errFailedSetSecret, err)
		cond := NewSecretSinkCondition(esapi.SecretSinkReady, v1.ConditionFalse, "SecretSyncFailed", msg)
		ss = SetSecretSinkCondition(ss, *cond)
		return ctrl.Result{}, err
	}
	cond := NewSecretSinkCondition(esapi.SecretSinkReady, v1.ConditionTrue, "SecretSynced", "SecretSink synced successfully")
	ss = SetSecretSinkCondition(ss, *cond)
	// Set status for SecretSink
	return ctrl.Result{}, nil
}

func (r *Reconciler) SetSecretToProviders(ctx context.Context, stores []v1beta1.GenericStore, ss esapi.SecretSink, secret *v1.Secret) error {
	for _, store := range stores {
		provider, err := v1beta1.GetProvider(store)
		if err != nil {
			return fmt.Errorf(errGetProviderFailed)
		}
		client, err := provider.NewClient(ctx, store, r.Client, ss.Namespace)
		if err != nil {
			return fmt.Errorf(errGetSecretsClientFailed)
		}
		defer func() {
			err := client.Close(ctx)
			if err != nil {
				r.Log.Error(err, errCloseStoreClient)
			}
		}()
		var secretKey string
		var remoteKey string
		for _, ref := range ss.Spec.Data {
			for _, match := range ref.Match {
				secretKey = match.SecretKey
				secretValue, ok := secret.Data[secretKey]
				if !ok {
					return fmt.Errorf("secret key %v does not exist", secretKey)
				}
				for _, rK := range match.RemoteRefs {
					remoteKey = rK.RemoteKey
				}
				err := client.SetSecret(remoteKey, string(secretValue))
				if err != nil {
					return fmt.Errorf(errSetSecretFailed, match.SecretKey, store.GetName(), err)
				}
			}
		}
	}
	return nil
}

func (r *Reconciler) GetSecret(ctx context.Context, ss esapi.SecretSink) (*v1.Secret, error) {
	secretName := types.NamespacedName{Name: ss.Spec.Selector.Secret.Name, Namespace: ss.Namespace}
	secret := &v1.Secret{}
	err := r.Client.Get(ctx, secretName, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (r *Reconciler) GetSecretStores(ctx context.Context, ss esapi.SecretSink) ([]v1beta1.GenericStore, error) {
	stores := make([]v1beta1.GenericStore, 0)
	for _, refStore := range ss.Spec.SecretStoreRefs {
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
			ref.Namespace = ss.Namespace

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
	r.recorder = mgr.GetEventRecorderFor("secret-sink")

	return ctrl.NewControllerManagedBy(mgr).
		For(&esapi.SecretSink{}).
		Complete(r)
}

func NewSecretSinkCondition(condType esapi.SecretSinkConditionType, status v1.ConditionStatus, reason, message string) *esapi.SecretSinkStatusCondition {
	return &esapi.SecretSinkStatusCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

func SetSecretSinkCondition(gs esapi.SecretSink, condition esapi.SecretSinkStatusCondition) esapi.SecretSink {
	status := gs.Status
	currentCond := GetSecretSinkCondition(status, condition.Type)
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
func filterOutCondition(conditions []esapi.SecretSinkStatusCondition, condType esapi.SecretSinkConditionType) []esapi.SecretSinkStatusCondition {
	newConditions := make([]esapi.SecretSinkStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

// GetSecretStoreCondition returns the condition with the provided type.
func GetSecretSinkCondition(status esapi.SecretSinkStatus, condType esapi.SecretSinkConditionType) *esapi.SecretSinkStatusCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}
