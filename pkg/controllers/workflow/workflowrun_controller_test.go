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

package workflow

import (
	"context"
	"encoding/json"
	"testing"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type RunReconcilerTestSuite struct {
	suite.Suite
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	builder  *fake.ClientBuilder
}

func (s *RunReconcilerTestSuite) SetupTest() {
	s.scheme = runtime.NewScheme()
	workflows.AddToScheme(s.scheme)
	scanv1alpha1.AddToScheme(s.scheme)
	corev1.AddToScheme(s.scheme)

	s.recorder = record.NewFakeRecorder(20)
	s.builder = fake.NewClientBuilder().WithScheme(s.scheme)
}

func TestRunReconcilerTestSuite(t *testing.T) {
	suite.Run(t, new(RunReconcilerTestSuite))
}

func (s *RunReconcilerTestSuite) TestReconcileWorkflowCreated() {
	template := &workflows.WorkflowTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WorkflowTemplate",
			APIVersion: "workflows.external-secrets.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "default",
		},
		Spec: workflows.WorkflowTemplateSpec{
			Version: "v1",
			Name:    "Sample Workflow Template",
			Jobs: map[string]workflows.Job{
				"start": {
					Standard: &workflows.StandardJob{
						Steps: []workflows.Step{
							{
								Name: "step1",
								Debug: &workflows.DebugStep{
									Message: "Starting workflow",
								},
							},
						},
					},
				},
			},
		},
	}

	run := &workflows.WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testrun",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "WorkflowRun",
			APIVersion: "workflows.external-secrets.io/v1alpha1",
		},
		Spec: workflows.WorkflowRunSpec{
			TemplateRef: workflows.TemplateRef{
				Name: "test-template",
			},
		},
	}

	cl := s.builder.WithObjects(template, run).WithStatusSubresource(template, run).Build()
	reconciler := &RunReconciler{
		Client:   cl,
		Log:      logr.Discard(),
		Scheme:   s.scheme,
		Recorder: s.recorder,
	}

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(run),
	})

	require.NoError(s.T(), err)

	updatedRun := &workflows.WorkflowRun{}
	err = cl.Get(context.Background(), client.ObjectKeyFromObject(run), updatedRun)
	require.NoError(s.T(), err)
	assert.Len(s.T(), updatedRun.Status.Conditions, 1)
	assert.Equal(s.T(), "WorkflowCreated", updatedRun.Status.Conditions[0].Type)
	assert.Equal(s.T(), metav1.ConditionTrue, updatedRun.Status.Conditions[0].Status)
}

func (s *RunReconcilerTestSuite) TestReconcileTemplateNotFound() {
	run := &workflows.WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testrun",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "WorkflowRun",
			APIVersion: "workflows.external-secrets.io/v1alpha1",
		},
		Spec: workflows.WorkflowRunSpec{
			TemplateRef: workflows.TemplateRef{
				Name: "nonexistent-template",
			},
		},
	}

	cl := s.builder.WithObjects(run).WithStatusSubresource(run).Build()
	reconciler := &RunReconciler{
		Client:   cl,
		Log:      logr.Discard(),
		Scheme:   s.scheme,
		Recorder: s.recorder,
	}

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(run),
	})

	require.NoError(s.T(), err)

	updatedRun := &workflows.WorkflowRun{}
	err = cl.Get(context.Background(), client.ObjectKeyFromObject(run), updatedRun)
	require.NoError(s.T(), err)
	assert.Len(s.T(), updatedRun.Status.Conditions, 1)
	assert.Equal(s.T(), "TemplateFound", updatedRun.Status.Conditions[0].Type)
	assert.Equal(s.T(), metav1.ConditionFalse, updatedRun.Status.Conditions[0].Status)
}

func (s *RunReconcilerTestSuite) TestResolveWorkflowFromTemplateFinding() {
	// Simulate a Finding resource that returns a location
	finding := &scanv1alpha1.Finding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "finding1",
			Namespace: "default",
		},
		Status: scanv1alpha1.FindingStatus{
			Locations: []scanv1alpha1.SecretInStoreRef{{
				Name:       "secret-store",
				Kind:       "SecretStore",
				APIVersion: "external-secrets.io/v1",
				RemoteRef: scanv1alpha1.RemoteRef{
					Key:      "secret-key",
					Property: "secret-property",
				},
			}},
		},
	}

	secondFinding := &scanv1alpha1.Finding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "finding2",
			Namespace: "default",
		},
		Status: scanv1alpha1.FindingStatus{
			Locations: []scanv1alpha1.SecretInStoreRef{{
				Name:       "second-secret-store",
				Kind:       "SecretStore",
				APIVersion: "external-secrets.io/v1",
				RemoteRef: scanv1alpha1.RemoteRef{
					Key:      "secret-key",
					Property: "secret-property",
				},
			}},
		},
	}

	params := []workflows.Parameter{
		{
			Name: "param1",
			Type: workflows.ParameterTypeFinding,
		},
		{
			Name: "param2",
			Type: workflows.ParameterTypeFindingArray,
		},
	}

	template := &workflows.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "template1",
			Namespace: "default",
		},
		Spec: workflows.WorkflowTemplateSpec{
			Version: "v1",
			Name:    "test-template",
			ParameterGroups: []workflows.ParameterGroup{
				{Parameters: params},
			},
		},
	}

	// Arguments: simulate passing a finding array parameter
	argsJSON, _ := json.Marshal(map[string]any{
		"param1": map[string]string{"name": "finding1"},
		"param2": []map[string]string{
			{"name": "finding2"},
		},
	})

	run := &workflows.WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "default",
		},
		Spec: workflows.WorkflowRunSpec{
			TemplateRef: workflows.TemplateRef{Name: "template1"},
			Arguments: apiextensionsv1.JSON{
				Raw: argsJSON,
			},
		},
	}

	cl := s.builder.WithObjects(finding, secondFinding).Build()
	reconciler := &RunReconciler{
		Client:   cl,
		Log:      logr.Discard(),
		Scheme:   s.scheme,
		Recorder: s.recorder,
	}

	workflow, err := reconciler.resolveWorkflowFromTemplate(context.Background(), template, run)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), workflow)
	assert.Contains(s.T(), string(workflow.Spec.Variables.Raw), "secret-store")
	assert.Contains(s.T(), string(workflow.Spec.Variables.Raw), "second-secret-store")
}

func (s *RunReconcilerTestSuite) TestResolveWorkflowFromTemplateCustomObject() {
	// Simulate a Finding resource that returns a location
	finding := &scanv1alpha1.Finding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "finding1",
			Namespace: "default",
		},
		Status: scanv1alpha1.FindingStatus{
			Locations: []scanv1alpha1.SecretInStoreRef{{
				Name:       "secret-store",
				Kind:       "SecretStore",
				APIVersion: "external-secrets.io/v1",
				RemoteRef: scanv1alpha1.RemoteRef{
					Key:      "secret-key",
					Property: "secret-property",
				},
			}},
		},
	}

	secondFinding := &scanv1alpha1.Finding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "finding2",
			Namespace: "default",
		},
		Status: scanv1alpha1.FindingStatus{
			Locations: []scanv1alpha1.SecretInStoreRef{{
				Name:       "secret-store2",
				Kind:       "SecretStore",
				APIVersion: "external-secrets.io/v1",
				RemoteRef: scanv1alpha1.RemoteRef{
					Key:      "secret-key",
					Property: "secret-property",
				},
			}},
		},
	}

	params := []workflows.Parameter{
		{
			Name: "param1",
			Type: workflows.ParameterType("object[customName]finding"),
		},
		{
			Name: "param2",
			Type: workflows.ParameterType("object[customName]array[finding]"),
		},
	}

	template := &workflows.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "template1",
			Namespace: "default",
		},
		Spec: workflows.WorkflowTemplateSpec{
			Version: "v1",
			Name:    "test-template",
			ParameterGroups: []workflows.ParameterGroup{
				{Parameters: params},
			},
		},
	}

	argsJSON, _ := json.Marshal(map[string]any{
		"param1": map[string]interface{}{
			"key1": map[string]string{"name": "finding1"},
		},
		"param2": map[string]interface{}{
			"key2": []map[string]string{
				{"name": "finding1"},
				{"name": "finding2"},
			},
		},
	})

	run := &workflows.WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "default",
		},
		Spec: workflows.WorkflowRunSpec{
			TemplateRef: workflows.TemplateRef{Name: "template1"},
			Arguments: apiextensionsv1.JSON{
				Raw: argsJSON,
			},
		},
	}

	cl := s.builder.WithObjects(finding, secondFinding).Build()
	reconciler := &RunReconciler{
		Client:   cl,
		Log:      logr.Discard(),
		Scheme:   s.scheme,
		Recorder: s.recorder,
	}

	workflow, err := reconciler.resolveWorkflowFromTemplate(context.Background(), template, run)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), workflow)

	var parsed map[string]interface{}
	err = json.Unmarshal(workflow.Spec.Variables.Raw, &parsed)
	require.NoError(s.T(), err)

	param1, ok := parsed["param1"].(map[string]interface{})
	assert.True(s.T(), ok, "param1 should be a map")

	key1, ok := param1["key1"].([]interface{})
	require.True(s.T(), ok, "key1 should be a list")

	found := false
	for _, item := range key1 {
		if m, ok := item.(map[string]interface{}); ok && m["name"] == "secret-store" {
			found = true
			break
		}
	}
	assert.True(s.T(), found, "expected to find an object with name 'secret-store' in key1")

	param2, ok := parsed["param2"].(map[string]interface{})
	assert.True(s.T(), ok, "param2 should be a map")

	key2, ok := param2["key2"].([]interface{})
	require.True(s.T(), ok, "key1 should be a list")

	found = false
	for _, item := range key2 {
		if m, ok := item.(map[string]interface{}); ok && m["name"] == "secret-store" {
			found = true
			break
		}
	}
	assert.True(s.T(), found, "expected to find an object with name 'secret-store' in key1")
	assert.Equal(s.T(), 2, len(key2))
}
