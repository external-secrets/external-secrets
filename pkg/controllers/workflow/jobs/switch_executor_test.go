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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// mockSecretsClient implements the SecretsClient interface for testing.
type mockSecretsClient struct {
	esv1.SecretsClient
}

func (m *mockSecretsClient) GetSecret(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return []byte("test"), nil
}

func (m *mockSecretsClient) Close(_ context.Context) error {
	return nil
}

// mockManager implements the secretstore.ManagerInterface for testing.
type mockManager struct {
	secretsClient esv1.SecretsClient
}

// Get implements the ManagerInterface.Get method.
func (m *mockManager) Get(_ context.Context, _ esv1.SecretStoreRef, _ string, _ *esv1.StoreGeneratorSourceRef) (esv1.SecretsClient, error) {
	return m.secretsClient, nil
}

// GetFromStore implements the ManagerInterface.GetFromStore method.
func (m *mockManager) GetFromStore(_ context.Context, _ esv1.GenericStore, _ string) (esv1.SecretsClient, error) {
	return m.secretsClient, nil
}

// Close implements the ManagerInterface.Close method.
func (m *mockManager) Close(_ context.Context) error {
	return nil
}

func TestSwitchJobExecutor(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = workflows.AddToScheme(scheme)

	tests := []struct {
		name         string
		job          *workflows.SwitchJob
		variables    apiextensionsv1.JSON
		expectedCase int
		expectError  bool
	}{
		{
			name: "first case condition true",
			job: &workflows.SwitchJob{
				Cases: []workflows.SwitchCase{
					{
						Condition: "true",
						Steps: []workflows.Step{
							{
								Name: "step1",
								Debug: &workflows.DebugStep{
									Message: "First case executed",
								},
							},
						},
					},
					{
						Condition: "false",
						Steps: []workflows.Step{
							{
								Name: "step2",
								Debug: &workflows.DebugStep{
									Message: "Second case executed",
								},
							},
						},
					},
				},
			},
			expectedCase: 0,
			expectError:  false,
		},
		{
			name: "second case condition true",
			job: &workflows.SwitchJob{
				Cases: []workflows.SwitchCase{
					{
						Condition: "false",
						Steps: []workflows.Step{
							{
								Name: "step1",
								Debug: &workflows.DebugStep{
									Message: "First case executed",
								},
							},
						},
					},
					{
						Condition: "true",
						Steps: []workflows.Step{
							{
								Name: "step2",
								Debug: &workflows.DebugStep{
									Message: "Second case executed",
								},
							},
						},
					},
				},
			},
			expectedCase: 1,
			expectError:  false,
		},
		{
			name: "no case condition true",
			job: &workflows.SwitchJob{
				Cases: []workflows.SwitchCase{
					{
						Condition: "false",
						Steps: []workflows.Step{
							{
								Name: "step1",
								Debug: &workflows.DebugStep{
									Message: "First case executed",
								},
							},
						},
					},
					{
						Condition: "false",
						Steps: []workflows.Step{
							{
								Name: "step2",
								Debug: &workflows.DebugStep{
									Message: "Second case executed",
								},
							},
						},
					},
				},
			},
			expectedCase: -1, // No case should be executed
			expectError:  false,
		},
		{
			name: "condition with variable",
			job: &workflows.SwitchJob{
				Cases: []workflows.SwitchCase{
					{
						Condition: "{{ eq .global.variables.environment \"production\" }}",
						Steps: []workflows.Step{
							{
								Name: "step1",
								Debug: &workflows.DebugStep{
									Message: "Production case executed",
								},
							},
						},
					},
					{
						Condition: "{{ eq .global.variables.environment \"staging\" }}",
						Steps: []workflows.Step{
							{
								Name: "step2",
								Debug: &workflows.DebugStep{
									Message: "Staging case executed",
								},
							},
						},
					},
				},
			},
			variables: apiextensionsv1.JSON{
				Raw: []byte(`{
					"environment": "production"
				}`),
			},
			expectedCase: 0,
			expectError:  false,
		},
		{
			name: "invalid condition",
			job: &workflows.SwitchJob{
				Cases: []workflows.SwitchCase{
					{
						Condition: "not-a-boolean",
						Steps: []workflows.Step{
							{
								Name: "step1",
								Debug: &workflows.DebugStep{
									Message: "This should not execute",
								},
							},
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a workflow with the test job
			wf := &workflows.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
				Spec: workflows.WorkflowSpec{
					Version:   "v1alpha1",
					Name:      "test-workflow",
					Variables: tt.variables,
					Jobs: map[string]workflows.Job{
						"testJob": {
							Switch: tt.job,
						},
					},
				},
				Status: workflows.WorkflowStatus{
					JobStatuses: map[string]workflows.JobStatus{
						"testJob": {
							Phase:        workflows.JobPhaseRunning,
							StepStatuses: make(map[string]workflows.StepStatus),
						},
					},
				},
			}

			// Create a fake client
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(wf).Build()

			// Create a mock manager
			mockManager := &mockManager{
				secretsClient: &mockSecretsClient{},
			}

			// Create the executor
			executor := NewSwitchJobExecutor(tt.job, scheme, logr.Discard(), mockManager)

			// Execute the job
			jobStatus := wf.Status.JobStatuses["testJob"]
			err := executor.Execute(context.Background(), client, wf, "testJob", &jobStatus)

			// Check for expected error
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Check that the job is marked as succeeded
			assert.Equal(t, workflows.JobPhaseSucceeded, jobStatus.Phase)

			// Check that the expected case was executed
			if tt.expectedCase >= 0 {
				// Get the step name from the expected case
				stepName := tt.job.Cases[tt.expectedCase].Steps[0].Name

				// The step from the expected case should be in the step statuses
				stepStatus, exists := jobStatus.StepStatuses[stepName]
				assert.True(t, exists, "Expected step %s to be in step statuses", stepName)
				assert.Equal(t, workflows.StepPhaseSucceeded, stepStatus.Phase)

				// If there are multiple steps with the same name across different cases,
				// we need to verify that only the expected case's step was executed
				if tt.expectedCase > 0 {
					// Check that steps from other cases with the same name were not executed
					for i := 0; i < tt.expectedCase; i++ {
						for _, step := range tt.job.Cases[i].Steps {
							if step.Name == stepName {
								// This is a step with the same name from a different case
								// We need to make sure it wasn't executed
								// Since we're using the same key, we can only verify by checking the total number
								// of step statuses - there should only be one entry per unique step name
								assert.Equal(t, 1, len(jobStatus.StepStatuses),
									"Expected only one step status for step %s", stepName)
							}
						}
					}
				}
			} else {
				// No case should have been executed, so no step statuses should be added
				assert.Empty(t, jobStatus.StepStatuses)
			}
		})
	}
}
