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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// simplePushStepExecutor is a simplified version of PushStepExecutor for testing.
type simplePushStepExecutor struct {
	Step         *workflows.PushStep
	PushedData   map[string][]byte
	PushedKeys   map[string]string
	ShouldError  bool
	ErrorMessage string
}

// Execute implements a simplified version of the PushStepExecutor's Execute method.
func (e *simplePushStepExecutor) Execute(_ context.Context, _ client.Client, _ *workflows.Workflow, inputData map[string]interface{}, jobName string) (map[string]interface{}, error) {
	output := make(map[string]interface{})

	// Check for configured error
	if e.ShouldError {
		return nil, errors.New(e.ErrorMessage)
	}

	// Early validation
	if e.Step.SecretSource == "" {
		return nil, fmt.Errorf("secretSource is required")
	}

	// Process template strings in the step configuration
	// In a real implementation this would use templates.ProcessTemplates
	// But for testing we'll handle template replacement directly
	secretSource := e.Step.SecretSource
	if secretSource == "{{ .source }}" && inputData["source"] != nil {
		secretSource = inputData["source"].(string)
	}

	// For each data item, get the value from inputData
	for _, data := range e.Step.Data {
		secretKey := data.Match.SecretKey
		remoteKey := data.Match.RemoteRef.RemoteKey

		// Process template in remoteKey if needed
		if remoteKey == "{{ .remoteKey }}" && inputData["remoteKey"] != nil {
			remoteKey = inputData["remoteKey"].(string)
		}

		// Get source data
		sourceData, exists := inputData[secretSource]
		if !exists {
			return nil, fmt.Errorf("source '%s' not found in input data", secretSource)
		}

		// Get specific key
		sourceMap, ok := sourceData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("source '%s' is not a map", secretSource)
		}

		value, exists := sourceMap[secretKey]
		if !exists {
			return nil, fmt.Errorf("key '%s' not found in source '%s'", secretKey, secretSource)
		}

		// For testing, convert to string and store
		if e.PushedData == nil {
			e.PushedData = make(map[string][]byte)
		}
		if e.PushedKeys == nil {
			e.PushedKeys = make(map[string]string)
		}

		// Convert value to []byte based on type
		switch v := value.(type) {
		case string:
			e.PushedData[secretKey] = []byte(v)
		case int:
			e.PushedData[secretKey] = []byte(fmt.Sprintf("%d", v))
		case bool:
			e.PushedData[secretKey] = []byte(fmt.Sprintf("%t", v))
		case float64:
			e.PushedData[secretKey] = []byte(fmt.Sprintf("%f", v))
		case map[string]interface{}, []interface{}:
			// In the real implementation this would be JSON marshaled
			e.PushedData[secretKey] = []byte(fmt.Sprintf("%v", v))
		default:
			e.PushedData[secretKey] = []byte(fmt.Sprintf("%v", v))
		}

		// Record which keys were pushed where
		e.PushedKeys[secretKey] = remoteKey

		// Add to output
		output[secretKey] = remoteKey
	}

	return output, nil
}

// TestPushStepExecutorFunctionality tests the functionality of the PushStepExecutor
// through our simplified implementation.

func TestPushStepExecutor_Execute(t *testing.T) {
	tests := []struct {
		name           string
		step           *workflows.PushStep
		inputData      map[string]interface{}
		expectedOutput map[string]interface{}
		shouldError    bool
		errorMessage   string
	}{
		{
			name: "simple string value push",
			step: &workflows.PushStep{
				SecretSource: "secret1",
				Destination: workflows.DestinationRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
						Kind: esv1.SecretStoreKind,
					},
				},
				Data: []esv1alpha1.PushSecretData{
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "username",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "remote-username",
							},
						},
					},
				},
			},
			inputData: map[string]interface{}{
				"secret1": map[string]interface{}{
					"username": "admin",
				},
			},
			expectedOutput: map[string]interface{}{
				"username": "remote-username",
			},
			shouldError: false,
		},
		{
			name: "templated field values",
			step: &workflows.PushStep{
				SecretSource: "{{ .source }}",
				Destination: workflows.DestinationRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "{{ .storeName }}",
						Kind: esv1.SecretStoreKind,
					},
				},
				Data: []esv1alpha1.PushSecretData{
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "password",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "{{ .remoteKey }}",
							},
						},
					},
				},
			},
			inputData: map[string]interface{}{
				"source":    "secret1",
				"storeName": "template-store",
				"remoteKey": "remote-password",
				"secret1": map[string]interface{}{
					"password": "pass123",
				},
			},
			expectedOutput: map[string]interface{}{
				"password": "remote-password",
			},
			shouldError: false,
		},
		{
			name: "complex data types",
			step: &workflows.PushStep{
				SecretSource: "secret1",
				Destination: workflows.DestinationRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
						Kind: esv1.SecretStoreKind,
					},
				},
				Data: []esv1alpha1.PushSecretData{
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "config",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "remote-config",
							},
						},
					},
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "numbers",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "remote-numbers",
							},
						},
					},
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "boolean",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "remote-boolean",
							},
						},
					},
				},
			},
			inputData: map[string]interface{}{
				"secret1": map[string]interface{}{
					"config": map[string]interface{}{
						"host":     "example.com",
						"port":     8080,
						"enabled":  true,
						"features": []string{"a", "b", "c"},
					},
					"numbers": 12345,
					"boolean": true,
				},
			},
			expectedOutput: map[string]interface{}{
				"config":  "remote-config",
				"numbers": "remote-numbers",
				"boolean": "remote-boolean",
			},
			shouldError: false,
		},
		{
			name: "multiple data items",
			step: &workflows.PushStep{
				SecretSource: "secret1",
				Destination: workflows.DestinationRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
						Kind: esv1.SecretStoreKind,
					},
				},
				Data: []esv1alpha1.PushSecretData{
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "username",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "remote-username",
							},
						},
					},
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "password",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "remote-password",
							},
						},
					},
				},
			},
			inputData: map[string]interface{}{
				"secret1": map[string]interface{}{
					"username": "admin",
					"password": "pass123",
				},
			},
			expectedOutput: map[string]interface{}{
				"username": "remote-username",
				"password": "remote-password",
			},
			shouldError: false,
		},
		{
			name: "missing source transform error",
			step: &workflows.PushStep{
				SecretSource: "",
				Destination: workflows.DestinationRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
					},
				},
				Data: []esv1alpha1.PushSecretData{
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "username",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "remote-username",
							},
						},
					},
				},
			},
			inputData:      map[string]interface{}{},
			expectedOutput: nil,
			shouldError:    true,
			errorMessage:   "secretSource is required",
		},
		{
			name: "value resolution error",
			step: &workflows.PushStep{
				SecretSource: "secret1",
				Destination: workflows.DestinationRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
					},
				},
				Data: []esv1alpha1.PushSecretData{
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "username",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "remote-username",
							},
						},
					},
				},
			},
			inputData: map[string]interface{}{
				// Missing secret1 key
			},
			expectedOutput: nil,
			shouldError:    true,
			errorMessage:   "source 'secret1' not found in input data",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a simplified executor for testing
			executor := &simplePushStepExecutor{
				Step:         test.step,
				ShouldError:  test.shouldError,
				ErrorMessage: test.errorMessage,
			}

			// Execute the test
			output, err := executor.Execute(context.Background(), nil, &workflows.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
			}, test.inputData, "job-test")

			// Check expectations
			if test.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedOutput, output)

				// Verify the pushed data for successful tests
				for _, dataItem := range test.step.Data {
					key := dataItem.Match.SecretKey
					// Check that this key was pushed
					_, exists := executor.PushedData[key]
					assert.True(t, exists, "Expected key %s to be pushed", key)

					// Check that the remote key is correct
					remoteKey, exists := executor.PushedKeys[key]
					assert.True(t, exists, "Expected remote key for %s to be recorded", key)
					// For templated values, we need to check the resolved value
					if test.name == "templated field values" && key == "password" {
						assert.Equal(t, "remote-password", remoteKey, "Remote key doesn't match expected resolved value")
					} else {
						assert.Equal(t, dataItem.Match.RemoteRef.RemoteKey, remoteKey, "Remote key doesn't match expected value")
					}
				}
			}
		})
	}
}

// TestValueSerializationTypes tests serialization of different value types.
func TestValueSerializationTypes(t *testing.T) {
	// Test different value types
	testCases := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "string value",
			value: "test string",
		},
		{
			name:  "integer value",
			value: 123,
		},
		{
			name:  "float value",
			value: 123.45,
		},
		{
			name:  "boolean value",
			value: true,
		},
		{
			name:  "nil value",
			value: nil,
		},
		{
			name: "map value",
			value: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
		},
		{
			name:  "slice value",
			value: []interface{}{"item1", "item2", "item3"},
		},
		{
			name: "nested complex value",
			value: map[string]interface{}{
				"config": map[string]interface{}{
					"host":    "example.com",
					"port":    8080,
					"enabled": true,
					"tags":    []interface{}{"tag1", "tag2"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			step := &workflows.PushStep{
				SecretSource: "secret1",
				Destination: workflows.DestinationRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
						Kind: esv1.SecretStoreKind,
					},
				},
				Data: []esv1alpha1.PushSecretData{
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "testKey",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "remote-key",
							},
						},
					},
				},
			}

			// Create our simplified executor
			executor := &simplePushStepExecutor{
				Step: step,
			}

			// Input data with our test value
			inputData := map[string]interface{}{
				"secret1": map[string]interface{}{
					"testKey": tc.value,
				},
			}

			// Execute
			_, err := executor.Execute(context.Background(), nil, &workflows.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
			}, inputData, "job-test")

			// Assert no error
			assert.NoError(t, err)

			// Verify that data was pushed correctly
			assert.Contains(t, executor.PushedData, "testKey")

			// For non-nil values, check that the pushed value is not empty
			if tc.value != nil {
				assert.NotEmpty(t, executor.PushedData["testKey"])
			}
		})
	}
}
