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

package workflow

import (
	"context"
	"fmt"
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
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
)

const (
	errFailedGetSecret       = "could not get source secret"
	errPatchStatus           = "error merging"
	errGetSecretStore        = "could not get SecretStore %q, %w"
	errGetClusterSecretStore = "could not get ClusterSecretStore %q, %w"
	errSetSecretFailed       = "could not write remote ref %v to target secretstore %v: %v"
	errFailedSetSecret       = "set secret failed: %v"
	errConvert               = "could not apply conversion strategy to keys: %v"
	pushSecretFinalizer      = "pushsecret.externalsecrets.io/finalizer"
)

type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	RequeueInterval time.Duration
	ControllerClass string
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("workflow")

	return ctrl.NewControllerManagedBy(mgr).
		For(&esapi.Workflow{}).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("workflow", req.NamespacedName)

	var workflow esapi.Workflow
	mgr := secretstore.NewManager(r.Client, r.ControllerClass, false)
	defer mgr.Close(ctx)

	if err := r.Get(ctx, req.NamespacedName, &workflow); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		msg := "unable to get Workflow"
		r.recorder.Event(&workflow, v1.EventTypeWarning, esapi.ReasonErrored, msg)
		log.Error(err, msg)
		return ctrl.Result{}, fmt.Errorf("get resource: %w", err)
	}

	refreshInt := r.RequeueInterval
	if workflow.Spec.RefreshInterval != nil {
		refreshInt = workflow.Spec.RefreshInterval.Duration
	}

	p := client.MergeFrom(workflow.DeepCopy())
	defer func() {
		if err := r.Client.Status().Patch(ctx, &workflow, p); err != nil {
			log.Error(err, errPatchStatus)
		}
	}()

	err := NewWorkflowRunner(ctx, r.Client, workflow.Namespace, workflow.Spec.Workflows, log).Run()
	if err != nil {
		r.markAsFailed(workflow, err)
		return ctrl.Result{RequeueAfter: refreshInt}, nil
	}

	r.markAsDone(&workflow)
	return ctrl.Result{RequeueAfter: refreshInt}, nil
}

func (r *Reconciler) markAsFailed(workflow esapi.Workflow, err error) {
	msg := err.Error()
	r.recorder.Event(&workflow, v1.EventTypeWarning, esapi.ReasonErrored, msg)
	r.Log.Error(err, msg)
	cond := newWorkflowCondition(esapi.WorkflowReady, v1.ConditionFalse, esapi.ReasonErrored, msg)
	setWorkflowCondition(&workflow, *cond)
}

func (r *Reconciler) markAsDone(workflow *esapi.Workflow) {
	msg := "Workflow ran successfully"
	cond := newWorkflowCondition(esapi.WorkflowReady, v1.ConditionTrue, esapi.ReasonSynced, msg)
	setWorkflowCondition(workflow, *cond)
	r.recorder.Event(workflow, v1.EventTypeNormal, esapi.ReasonSynced, msg)
}

func newWorkflowCondition(condType esapi.WorkflowConditionType, status v1.ConditionStatus, reason, message string) *esapi.WorkflowStatusCondition {
	return &esapi.WorkflowStatusCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

func setWorkflowCondition(workflow *esapi.Workflow, condition esapi.WorkflowStatusCondition) {
	currentCond := getWorkflowCondition(workflow.Status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status &&
		currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	workflow.Status.Conditions = append(filterOutCondition(workflow.Status.Conditions, condition.Type), condition)
}

// filterOutCondition returns an empty set of conditions with the provided type.
func filterOutCondition(conditions []esapi.WorkflowStatusCondition, condType esapi.WorkflowConditionType) []esapi.WorkflowStatusCondition {
	newConditions := make([]esapi.WorkflowStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

// getWorkflowCondition returns the condition with the provided type.
func getWorkflowCondition(status esapi.WorkflowStatus, condType esapi.WorkflowConditionType) *esapi.WorkflowStatusCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}
