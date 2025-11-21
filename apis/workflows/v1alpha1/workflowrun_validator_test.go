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

// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestValidateWorkflowRunParameters(t *testing.T) {
	// Create a test scheme
	scheme := runtime.NewScheme()
	_ = AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create test objects
	template := &WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "default",
		},
		Spec: WorkflowTemplateSpec{
			Version: "v1",
			Name:    "Test Template",
			ParameterGroups: []ParameterGroup{
				{
					Name: "Basic",
					Parameters: []Parameter{
						{
							Name:     "stringParam",
							Type:     ParameterTypeString,
							Required: true,
						},
						{
							Name:     "numberParam",
							Type:     ParameterTypeNumber,
							Required: true,
						},
						{
							Name:     "optionalParam",
							Type:     ParameterTypeString,
							Required: false,
							Default:  "default",
						},
						{
							Name:          "arrayParam",
							Type:          ParameterTypeString,
							Required:      false,
							AllowMultiple: true,
							Validation: &ParameterValidation{
								MinItems: intPtr(1),
								MaxItems: intPtr(3),
							},
						},
					},
				},
			},
			Jobs: map[string]Job{
				"test": {
					Standard: &StandardJob{
						Steps: []Step{
							{
								Name: "test",
								Debug: &DebugStep{
									Message: "Test",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create a fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(template).
		Build()

	// Set the validation client
	SetValidationClient(client)

	// Test cases
	tests := []struct {
		name        string
		workflowRun *WorkflowRun
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid parameters",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "test-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"stringParam": "value",
							"numberParam": 42
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing required parameter",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "test-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"stringParam": "value"
						}`), // numberParam is missing
					},
				},
			},
			wantErr: true,
			errMsg:  "required parameter \"numberParam\" is missing",
		},
		{
			name: "invalid number parameter",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "test-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"stringParam": "value",
							"numberParam": "not-a-number"
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid value for argument \"numberParam\": failed to parse as number",
		},
		{
			name: "undefined parameter",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "test-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"stringParam":    "value",
							"numberParam":    42,
							"undefinedParam": "value"
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "argument \"undefinedParam\" is not defined in the template",
		},
		{
			name: "valid array parameter",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "test-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"stringParam": "value",
							"numberParam": 42,
							"arrayParam":  ["item1", "item2"]
						}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "array parameter too few items",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "test-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"stringParam": "value",
							"numberParam": 42,
							"arrayParam":  []
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "requires at least 1 items, got 0",
		},
		{
			name: "array parameter too many items",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "test-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"stringParam": "value",
							"numberParam": 42,
							"arrayParam":  ["item1", "item2", "item3", "item4"]
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "allows at most 3 items, got 4",
		},
		{
			name: "template not found",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "non-existent-template",
					},
					Arguments: apiextensionsv1.JSON{
						Raw: []byte(`{
							"stringParam": "value"
						}`),
					},
				},
			},
			wantErr: true,
			errMsg:  "referenced WorkflowTemplate non-existent-template not found in namespace default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkflowRunParameters(tt.workflowRun)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateWorkflowRunParametersClientNotSet(t *testing.T) {
	// Save the current client
	originalClient := k8sClient
	defer func() {
		k8sClient = originalClient
	}()

	// Set client to nil
	k8sClient = nil

	workflowRun := &WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-run",
			Namespace: "default",
		},
		Spec: WorkflowRunSpec{
			TemplateRef: TemplateRef{
				Name: "test-template",
			},
			Arguments: apiextensionsv1.JSON{
				Raw: []byte(`{
				"stringParam": "value",
			}`)},
		},
	}

	err := validateWorkflowRunParameters(workflowRun)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation client not initialized")
}

// Helper function to create int pointers.
func intPtr(i int) *int {
	return &i
}
