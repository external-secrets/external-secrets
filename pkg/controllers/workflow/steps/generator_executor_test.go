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
package steps

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// mockClient implements the client.Client interface for testing.
type generatorMockClient struct {
	client.Client
	getErr        error
	fakeGenerator *genv1alpha1.Fake
}

func (m *generatorMockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if m.getErr != nil {
		return m.getErr
	}

	// If this is a request for our fake generator, fill in the object
	if key.Name == "my-generator" && m.fakeGenerator != nil {
		// Set the TypeMeta and ObjectMeta on the fake generator
		fakeGen, ok := obj.(*genv1alpha1.Fake)
		if ok {
			fakeGen.TypeMeta = metav1.TypeMeta{
				APIVersion: "generators.external-secrets.io/v1alpha1",
				Kind:       "FakeGenerator",
			}
			fakeGen.ObjectMeta = metav1.ObjectMeta{
				Name:      "my-generator",
				Namespace: key.Namespace,
			}
			fakeGen.Spec = m.fakeGenerator.Spec
			return nil
		}
	}
	return nil
}

// mockGenerator implements the Generator interface for testing.
type mockGenerator struct {
	generateErr error
	secretMap   map[string][]byte
}

// Generate implements the Generator interface.
func (m *mockGenerator) Generate(ctx context.Context, obj *apiextensions.JSON, client client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if m.generateErr != nil {
		return nil, nil, m.generateErr
	}
	// Create an empty JSON object as the state
	emptyState := &apiextensions.JSON{Raw: []byte("{}")}
	return m.secretMap, emptyState, nil
}

// Cleanup implements the Generator interface.
func (m *mockGenerator) Cleanup(ctx context.Context, obj *apiextensions.JSON, status genv1alpha1.GeneratorProviderState, kube client.Client, namespace string) error {
	return nil
}

// GetCleanupPolicy implements the Generator interface.
func (g *mockGenerator) GetCleanupPolicy(obj *apiextensions.JSON) (*genv1alpha1.CleanupPolicy, error) {
	return nil, nil
}

func (g *mockGenerator) LastActivityTime(ctx context.Context, obj *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, kube client.Client, namespace string) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

// GetKey implements the Generator interface.
func (m *mockGenerator) GetKeys() map[string]string {
	return nil
}

// Register the mock generator for testing.
func registerMockGenerator(secretMap map[string][]byte, err error) {
	// Use ForceRegister to overwrite if already registered
	genv1alpha1.ForceRegister("FakeGenerator", &mockGenerator{
		secretMap:   secretMap,
		generateErr: err,
	})
}

// FakeSpec defines the configuration for the fake generator.
type FakeSpec struct {
	Data map[string]string `json:"data"`
}

func TestGeneratorStepExecutor_Execute(t *testing.T) {
	tests := []struct {
		name            string
		step            *workflows.GeneratorStep
		mockSecretMap   map[string][]byte
		mockGenerateErr error
		data            map[string]interface{}
		expectedOutput  map[string]interface{}
		expectsError    bool
	}{
		{
			name: "successful execution with inline generator",
			step: &workflows.GeneratorStep{
				Kind: "FakeGenerator",
				Generator: &genv1alpha1.GeneratorSpec{
					FakeSpec: &genv1alpha1.FakeSpec{
						Data: map[string]string{
							"test": "data",
						},
					},
				},
			},
			mockSecretMap: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
			data: map[string]interface{}{},
			expectedOutput: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expectsError: false,
		},
		{
			name: "successful execution with inline generator",
			step: &workflows.GeneratorStep{
				Kind: "FakeGenerator",
				Generator: &genv1alpha1.GeneratorSpec{
					FakeSpec: &genv1alpha1.FakeSpec{
						Data: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			mockSecretMap: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
			data: map[string]interface{}{},
			expectedOutput: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expectsError: false,
		},
		{
			name:           "error when no generator specified",
			step:           &workflows.GeneratorStep{},
			mockSecretMap:  map[string][]byte{},
			data:           map[string]interface{}{},
			expectedOutput: nil,
			expectsError:   true,
		},
		{
			name: "error when generate fails",
			step: &workflows.GeneratorStep{
				Kind: "FakeGenerator",
				Generator: &genv1alpha1.GeneratorSpec{
					FakeSpec: &genv1alpha1.FakeSpec{
						Data: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			mockSecretMap:   map[string][]byte{},
			mockGenerateErr: fmt.Errorf("generate error"),
			data:            map[string]interface{}{},
			expectedOutput:  nil,
			expectsError:    true,
		},
		{
			name: "successful with rewrite rules",
			step: &workflows.GeneratorStep{
				Kind: "FakeGenerator",
				Generator: &genv1alpha1.GeneratorSpec{
					FakeSpec: &genv1alpha1.FakeSpec{
						Data: map[string]string{
							"foo": "bar",
						},
					},
				},
				Rewrite: []esv1.ExternalSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: "key1",
							Target: "newKey1",
						},
					},
				},
			},
			mockSecretMap: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
			data: map[string]interface{}{},
			expectedOutput: map[string]interface{}{
				"newKey1": "value1",
				"key2":    "value2",
			},
			expectsError: false,
		},
		{
			name: "invalid key names",
			step: &workflows.GeneratorStep{
				Kind: "FakeGenerator",
				Generator: &genv1alpha1.GeneratorSpec{
					FakeSpec: &genv1alpha1.FakeSpec{
						Data: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			mockSecretMap: map[string][]byte{
				"invalid key": []byte("value1"),
			},
			data:           map[string]interface{}{},
			expectedOutput: nil,
			expectsError:   true,
		},
		{
			name: "parses JSON values correctly",
			step: &workflows.GeneratorStep{
				Kind: "FakeGenerator",
				Generator: &genv1alpha1.GeneratorSpec{
					FakeSpec: &genv1alpha1.FakeSpec{
						Data: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			mockSecretMap: map[string][]byte{
				"jsonKey":   []byte(`{"nested": "object", "array": [1, 2, 3]}`),
				"stringKey": []byte("simple string"),
			},
			data: map[string]interface{}{},
			expectedOutput: map[string]interface{}{
				"jsonKey": map[string]interface{}{
					"nested": "object",
					"array":  []interface{}{float64(1), float64(2), float64(3)},
				},
				"stringKey": "simple string",
			},
			expectsError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Register the test's mock generator
			registerMockGenerator(test.mockSecretMap, test.mockGenerateErr)

			// Create a scheme and add the required types to it
			scheme := runtime.NewScheme()
			// If this is a generatorRef test, set up the scheme to handle the Fake type
			if test.step.GeneratorRef != nil && test.step.GeneratorRef.Kind == "FakeGenerator" {
				_ = genv1alpha1.AddToScheme(scheme)
			}

			// Create a fake generator for the client to return
			fakeGen := &genv1alpha1.Fake{
				Spec: genv1alpha1.FakeSpec{
					Data: map[string]string{
						"foo": "bar",
					},
				},
			}

			// Create a mock client with the fake generator
			mockClient := &generatorMockClient{
				fakeGenerator: fakeGen,
			}

			// Create a mock manager
			mockMgr := &mockManager{
				secretsClient: &mockSecretsClient{},
			}

			executor := NewGeneratorStepExecutor(
				test.step,
				mockClient,
				scheme,
				mockMgr,
			)

			// Execute the step
			output, err := executor.Execute(context.Background(), mockClient, &workflows.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
			}, test.data, "job-test")

			// Verify results
			if test.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedOutput, output)
			}
		})
	}
}

func TestNewGeneratorStepExecutor(t *testing.T) {
	step := &workflows.GeneratorStep{
		GeneratorRef: &esv1.GeneratorRef{
			Name: "test-generator",
		},
	}
	mockClient := &generatorMockClient{}
	scheme := runtime.NewScheme()

	// Create a mock manager
	mockMgr := &mockManager{
		secretsClient: &mockSecretsClient{},
	}

	executor := NewGeneratorStepExecutor(step, mockClient, scheme, mockMgr)

	assert.Equal(t, step, executor.Step)
	assert.Equal(t, mockClient, executor.Client)
	assert.Equal(t, scheme, executor.Scheme)
	assert.Equal(t, mockMgr, executor.Manager)
}
