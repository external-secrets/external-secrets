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

package jobs

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

func TestLoopJobExecutorOutputs(t *testing.T) {
	// Create a test scheme
	scheme := runtime.NewScheme()
	_ = workflows.AddToScheme(scheme)

	// Create a fake client
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create a test workflow
	wf := &workflows.Workflow{
		Spec: workflows.WorkflowSpec{
			Jobs: map[string]workflows.Job{
				"test-job": {
					Loop: &workflows.LoopJob{
						Range: `{"key1": "value1", "key2": "value2"}`,
						Steps: []workflows.Step{
							{
								Name: "test-step",
								Debug: &workflows.DebugStep{
									Message: "Test message",
								},
							},
						},
					},
				},
			},
		},
		Status: workflows.WorkflowStatus{
			JobStatuses: map[string]workflows.JobStatus{
				"test-job": {
					StepStatuses: map[string]workflows.StepStatus{
						"test-step": {
							Phase:   workflows.StepPhasePending,
							Outputs: map[string]string{}, // Initialize empty outputs map
						},
					},
				},
			},
		},
	}

	// Create a loop job executor
	executor := NewLoopJobExecutor(wf.Spec.Jobs["test-job"].Loop, scheme, logr.Discard(), nil)

	// Execute the job
	jobStatus := wf.Status.JobStatuses["test-job"]
	err := executor.Execute(context.Background(), client, wf, "test-job", &jobStatus)
	assert.NoError(t, err)
	wf.Status.JobStatuses["test-job"] = jobStatus

	// Check that the outputs are structured correctly
	stepStatus := wf.Status.JobStatuses["test-job"].StepStatuses["test-step"]
	outputs := stepStatus.Outputs

	// Print the outputs for debugging
	t.Logf("Outputs: %v", outputs)

	// Verify that outputs contain both range keys
	assert.Contains(t, outputs, "key1")
	assert.Contains(t, outputs, "key2")

	// The debug step executor doesn't return any outputs, so we just verify the keys exist
	assert.NotEmpty(t, outputs["key1"])
	assert.NotEmpty(t, outputs["key2"])
}

func TestLoopJobExecutorWithSensitiveOutputs(t *testing.T) {
	// Create a test scheme
	scheme := runtime.NewScheme()
	_ = workflows.AddToScheme(scheme)

	// Create a fake client
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create a test workflow with sensitive outputs
	wf := &workflows.Workflow{
		Spec: workflows.WorkflowSpec{
			Jobs: map[string]workflows.Job{
				"test-job": {
					Loop: &workflows.LoopJob{
						Range: `{"key1": "value1", "key2": "value2"}`,
						Steps: []workflows.Step{
							{
								Name: "test-step",
								Debug: &workflows.DebugStep{
									Message: "Test message with password",
								},
								Outputs: []workflows.OutputDefinition{
									{
										Name:      "password",
										Type:      workflows.OutputTypeString,
										Sensitive: true,
									},
								},
							},
						},
					},
				},
			},
		},
		Status: workflows.WorkflowStatus{
			JobStatuses: map[string]workflows.JobStatus{
				"test-job": {
					StepStatuses: map[string]workflows.StepStatus{
						"test-step": {
							Phase: workflows.StepPhasePending,
						},
					},
				},
			},
		},
	}

	// Create a loop job executor
	executor := NewLoopJobExecutor(wf.Spec.Jobs["test-job"].Loop, scheme, logr.Discard(), nil)

	// Execute the job
	jobStatus := wf.Status.JobStatuses["test-job"]
	err := executor.Execute(context.Background(), client, wf, "test-job", &jobStatus)
	assert.NoError(t, err)
	wf.Status.JobStatuses["test-job"] = jobStatus

	// Check that the outputs are structured correctly
	stepStatus := wf.Status.JobStatuses["test-job"].StepStatuses["test-step"]
	outputs := stepStatus.Outputs

	// Verify that outputs contain both range keys
	assert.Contains(t, outputs, "key1")
	assert.Contains(t, outputs, "key2")

	// The debug step executor doesn't return any outputs, so we just verify the keys exist
	assert.NotEmpty(t, outputs["key1"])
	assert.NotEmpty(t, outputs["key2"])
}
