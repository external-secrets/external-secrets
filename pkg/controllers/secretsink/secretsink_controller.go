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
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

const (
	errNotImplemented = "secret sink not implemented"
	errPatchStatus    = "error merging"
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
		return ctrl.Result{}, err
	}
	p := client.MergeFrom(ss.DeepCopy())
	defer func() {
		err := r.Client.Status().Patch(ctx, &ss, p)
		if err != nil {
			log.Error(err, errPatchStatus)
		}
	}()
	cond := NewSecretSinkCondition(esapi.SecretSinkReady, v1.ConditionFalse, "NotImplementedError", errNotImplemented)
	ss = SetSecretSinkCondition(ss, *cond)
	// Set status for SecretSink
	return ctrl.Result{}, nil
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
