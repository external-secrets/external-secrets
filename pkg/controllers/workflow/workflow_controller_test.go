// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.
package workflow

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
)

// (For testing purposes, ensure that your v1alpha1 package registers the Workflow type into the scheme.)
// Here we assume that v1alpha1 has an AddToScheme(scheme) function.
func addToScheme(scheme *runtime.Scheme) error {
	return workflows.AddToScheme(scheme)
}

//
// Reconcile() tests
//

// TestReconcileNotFound verifies that when the workflow is not found the reconciler does nothing.
func TestReconcileNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := addToScheme(scheme); err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}
	// Create a fake client with no objects.
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)

	r := &Reconciler{
		Client:   fakeClient,
		Log:      logr.Discard(),
		Scheme:   scheme,
		Recorder: recorder,
		Manager:  secretstore.NewManager(fakeClient, "", false),
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// Expect no requeue.
	if res.Requeue || res.RequeueAfter != 0 {
		t.Errorf("expected no requeue, got: %+v", res)
	}
}

// TestReconcileTerminalWorkflow verifies that a workflow already in a terminal state is ignored.
func TestReconcileTerminalWorkflow(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := addToScheme(scheme); err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}

	workflowObj := &workflows.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "terminal",
			Namespace: "default",
		},
		Spec: workflows.WorkflowSpec{
			Version: "v1",
			Name:    "terminal",
			Jobs: map[string]workflows.Job{
				"job1": {
					Standard: &workflows.StandardJob{
						Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "test"}}},
					},
				},
			},
		},
		Status: workflows.WorkflowStatus{
			Phase: workflows.PhaseSucceeded,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(workflowObj).Build()
	recorder := record.NewFakeRecorder(10)

	r := &Reconciler{
		Client:   fakeClient,
		Log:      logr.Discard(),
		Scheme:   scheme,
		Recorder: recorder,
		Manager:  secretstore.NewManager(fakeClient, "", false),
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "terminal",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Since the workflow is already complete, no further action is expected.
	if res.Requeue || res.RequeueAfter != 0 {
		t.Errorf("expected no requeue, got: %+v", res)
	}
}

//
// calculateExecutionOrder and dependency tests
//

func TestCalculateExecutionOrder(t *testing.T) {
	r := &Reconciler{}
	jobs := map[string]workflows.Job{
		"job1": {
			DependsOn: []string{},
			Standard: &workflows.StandardJob{
				Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "job1"}}},
			},
		},
		"job2": {
			DependsOn: []string{"job1"},
			Standard: &workflows.StandardJob{
				Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "job2"}}},
			},
		},
		"job3": {
			DependsOn: []string{"job2"},
			Standard: &workflows.StandardJob{
				Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "job3"}}},
			},
		},
	}
	order, err := r.calculateExecutionOrder(jobs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"job1", "job2", "job3"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("expected order %v, got %v", expected, order)
	}
}

func TestCalculateExecutionOrderCycle(t *testing.T) {
	r := &Reconciler{}
	jobs := map[string]workflows.Job{
		"job1": {
			DependsOn: []string{"job3"},
			Standard: &workflows.StandardJob{
				Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "job1"}}},
			},
		},
		"job2": {
			DependsOn: []string{"job1"},
			Standard: &workflows.StandardJob{
				Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "job2"}}},
			},
		},
		"job3": {
			DependsOn: []string{"job2"},
			Standard: &workflows.StandardJob{
				Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "job3"}}},
			},
		},
	}
	_, err := r.calculateExecutionOrder(jobs)
	if err == nil {
		t.Fatalf("expected error due to cyclic dependency, got nil")
	}
	if !strings.Contains(err.Error(), "cyclic dependencies") {
		t.Errorf("expected cyclic dependency error, got %v", err)
	}
}

func TestJobDependenciesMet(t *testing.T) {
	r := &Reconciler{}
	job := workflows.Job{
		DependsOn: []string{"job1", "job2"},
		Standard: &workflows.StandardJob{
			Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "test"}}},
		},
	}
	statuses := map[string]workflows.JobStatus{
		"job1": {Phase: workflows.JobPhaseSucceeded},
		"job2": {Phase: workflows.JobPhaseSucceeded},
	}
	if !r.jobDependenciesMet(job, statuses) {
		t.Errorf("expected dependencies to be met")
	}
	// Change one dependency to not succeeded.
	statuses["job2"] = workflows.JobStatus{Phase: workflows.JobPhaseRunning}
	if r.jobDependenciesMet(job, statuses) {
		t.Errorf("expected dependencies not to be met")
	}
}

func TestResolveJobVariables(t *testing.T) {
	r := &Reconciler{}
	job := &workflows.Job{
		Variables: map[string]string{
			"var1": "prefix-{{ .global.variables.base }}-suffix",
		},
		Standard: &workflows.StandardJob{
			Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "test"}}},
		},
	}
	wf := &workflows.Workflow{
		Spec: workflows.WorkflowSpec{
			Variables: apiextensionsv1.JSON{Raw: []byte(`{"base": "core"}`)},
		},
		Status: workflows.WorkflowStatus{
			JobStatuses: map[string]workflows.JobStatus{},
		},
	}
	vars, err := r.resolveJobVariables(job, wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "prefix-core-suffix"
	if vars["var1"] != expected {
		t.Errorf("expected variable value %q, got %q", expected, vars["var1"])
	}
}

//
// Status update and initialization tests
//

func TestMarkWorkflowFailed(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := addToScheme(scheme); err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}

	wf := &workflows.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "workflows.external-secrets.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failwf",
			Namespace: "default",
		},
		Spec: workflows.WorkflowSpec{
			Version: "v1",
			Name:    "failwf",
			Jobs: map[string]workflows.Job{
				"job1": {
					Standard: &workflows.StandardJob{
						Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "test"}}},
					},
				},
			},
		},
		Status: workflows.WorkflowStatus{},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(wf).
		WithStatusSubresource(&workflows.Workflow{}).
		Build()

	recorder := record.NewFakeRecorder(10)
	r := &Reconciler{
		Client:   fakeClient,
		Log:      logr.Discard(),
		Scheme:   scheme,
		Recorder: recorder,
		Manager:  secretstore.NewManager(fakeClient, "", false),
	}
	res, err := r.markWorkflowFailed(context.Background(), wf, "TestFailure", "failure message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Requeue || res.RequeueAfter != 0 {
		t.Errorf("expected no requeue, got: %+v", res)
	}
	updatedWf := &workflows.Workflow{}
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: "failwf", Namespace: "default"}, updatedWf); err != nil {
		t.Fatalf("failed to get updated workflow: %v", err)
	}
	if updatedWf.Status.Phase != workflows.PhaseFailed {
		t.Errorf("expected phase %q, got %q", workflows.PhaseFailed, updatedWf.Status.Phase)
	}
	// Also check that a condition was set.
	cond := meta.FindStatusCondition(updatedWf.Status.Conditions, "Failed")
	if cond == nil || cond.Reason != "TestFailure" {
		t.Errorf("expected Failed condition with reason TestFailure, got: %v", updatedWf.Status.Conditions)
	}
}

func TestInitializeWorkflow(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := addToScheme(scheme); err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}

	wf := &workflows.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "workflows.external-secrets.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "initwf",
			Namespace: "default",
		},
		Spec: workflows.WorkflowSpec{
			Version: "v1",
			Name:    "initwf",
			Jobs: map[string]workflows.Job{
				"job1": {
					Standard: &workflows.StandardJob{
						Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "test"}}},
					},
				},
				"job2": {
					Standard: &workflows.StandardJob{
						Steps: []workflows.Step{{Debug: &workflows.DebugStep{Message: "test"}}},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(wf).
		WithStatusSubresource(&workflows.Workflow{}).
		Build()

	recorder := record.NewFakeRecorder(10)
	r := &Reconciler{
		Client:   fakeClient,
		Log:      logr.Discard(),
		Scheme:   scheme,
		Recorder: recorder,
		Manager:  secretstore.NewManager(fakeClient, "", false),
	}

	ctx := context.Background()
	// Do not call Create here; wf is already in the client store.
	res, err := r.initializeWorkflow(ctx, wf, r.Log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Requeue {
		t.Errorf("expected requeue true, got: %+v", res)
	}
	if wf.Status.Phase != workflows.PhasePending {
		t.Errorf("expected phase %q, got %q", workflows.PhasePending, wf.Status.Phase)
	}
	if len(wf.Status.JobStatuses) != 2 {
		t.Errorf("expected 2 job statuses, got %d", len(wf.Status.JobStatuses))
	}
}
