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
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// mockSecretsClient implements the SecretsClient interface for testing.
type mockSecretsClient struct {
	esv1.SecretsClient
	getSecretErr     error
	getSecretValue   []byte
	getAllSecretsErr error
	getAllSecretsMap map[string][]byte
	getSecretMapErr  error
	getSecretMapData map[string][]byte
}

func (m *mockSecretsClient) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if m.getSecretErr != nil {
		return nil, m.getSecretErr
	}
	return m.getSecretValue, nil
}

func (m *mockSecretsClient) GetAllSecrets(_ context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if m.getAllSecretsErr != nil {
		return nil, m.getAllSecretsErr
	}
	return m.getAllSecretsMap, nil
}

func (m *mockSecretsClient) GetSecretMap(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if m.getSecretMapErr != nil {
		return nil, m.getSecretMapErr
	}
	return m.getSecretMapData, nil
}

func (m *mockSecretsClient) Close(_ context.Context) error {
	return nil
}

// mockManager implements the secretstore.ManagerInterface for testing.
type mockManager struct {
	secretsClient esv1.SecretsClient
	getClientErr  error
}

// Get implements the ManagerInterface.Get method.
func (m *mockManager) Get(_ context.Context, _ esv1.SecretStoreRef, _ string, _ *esv1.StoreGeneratorSourceRef) (esv1.SecretsClient, error) {
	if m.getClientErr != nil {
		return nil, m.getClientErr
	}
	return m.secretsClient, nil
}

// GetFromStore implements the ManagerInterface.GetFromStore method.
func (m *mockManager) GetFromStore(_ context.Context, _ esv1.GenericStore, _ string) (esv1.SecretsClient, error) {
	if m.getClientErr != nil {
		return nil, m.getClientErr
	}
	return m.secretsClient, nil
}

// Close implements the ManagerInterface.Close method.
func (m *mockManager) Close(_ context.Context) error {
	return nil
}

// Execute method includes JSON parsing capability

func TestPullStepExecutor_Execute(t *testing.T) {
	tests := []struct {
		name           string
		step           *workflows.PullStep
		mockClient     *mockSecretsClient
		managerErr     error
		expectedOutput map[string]interface{}
		expectsError   bool
	}{
		{
			name: "successful pull with data",
			step: &workflows.PullStep{
				Source: esv1.StoreSourceRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
						Kind: esv1.SecretStoreKind,
					},
				},
				Data: []esv1.ExternalSecretData{
					{
						SecretKey: "key1",
						RemoteRef: esv1.ExternalSecretDataRemoteRef{
							Key: "remote-key1",
						},
					},
					{
						SecretKey: "key2",
						RemoteRef: esv1.ExternalSecretDataRemoteRef{
							Key: "remote-key2",
						},
					},
				},
			},
			mockClient: &mockSecretsClient{
				getSecretValue: []byte("test-value"),
			},
			expectedOutput: map[string]interface{}{
				"key1": "test-value",
				"key2": "test-value",
			},
			expectsError: false,
		},
		{
			name: "successful pull with dataFrom.find",
			step: &workflows.PullStep{
				Source: esv1.StoreSourceRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
					},
				},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
					{
						Find: &esv1.ExternalSecretFind{
							Name: &esv1.FindName{
								RegExp: "test-*",
							},
						},
					},
				},
			},
			mockClient: &mockSecretsClient{
				getAllSecretsMap: map[string][]byte{
					"test-key1": []byte("test-value1"),
					"test-key2": []byte("test-value2"),
				},
			},
			expectedOutput: map[string]interface{}{
				"test-key1": "test-value1",
				"test-key2": "test-value2",
			},
			expectsError: false,
		},
		{
			name: "successful pull with dataFrom.extract",
			step: &workflows.PullStep{
				Source: esv1.StoreSourceRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
					},
				},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
					{
						Extract: &esv1.ExternalSecretDataRemoteRef{
							Key: "extract-key",
						},
					},
				},
			},
			mockClient: &mockSecretsClient{
				getSecretMapData: map[string][]byte{
					"extracted-1": []byte("value1"),
					"extracted-2": []byte("value2"),
				},
			},
			expectedOutput: map[string]interface{}{
				"extracted-1": "value1",
				"extracted-2": "value2",
			},
			expectsError: false,
		},
		{
			name: "error getting store client",
			step: &workflows.PullStep{
				Source: esv1.StoreSourceRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
					},
				},
				Data: []esv1.ExternalSecretData{
					{
						SecretKey: "key1",
						RemoteRef: esv1.ExternalSecretDataRemoteRef{
							Key: "remote-key1",
						},
					},
				},
			},
			managerErr:     errors.New("store error"),
			expectedOutput: nil,
			expectsError:   true,
		},
		{
			name: "error fetching secret",
			step: &workflows.PullStep{
				Source: esv1.StoreSourceRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
					},
				},
				Data: []esv1.ExternalSecretData{
					{
						SecretKey: "key1",
						RemoteRef: esv1.ExternalSecretDataRemoteRef{
							Key: "remote-key1",
						},
					},
				},
			},
			mockClient: &mockSecretsClient{
				getSecretErr: errors.New("secret fetch error"),
			},
			expectedOutput: nil,
			expectsError:   true,
		},
		{
			name: "error with dataFrom.find",
			step: &workflows.PullStep{
				Source: esv1.StoreSourceRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
					},
				},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
					{
						Find: &esv1.ExternalSecretFind{
							Name: &esv1.FindName{
								RegExp: "test-*",
							},
						},
					},
				},
			},
			mockClient: &mockSecretsClient{
				getAllSecretsErr: errors.New("find error"),
			},
			expectedOutput: nil,
			expectsError:   true,
		},
		{
			name: "error with dataFrom.extract",
			step: &workflows.PullStep{
				Source: esv1.StoreSourceRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
					},
				},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
					{
						Extract: &esv1.ExternalSecretDataRemoteRef{
							Key: "extract-key",
						},
					},
				},
			},
			mockClient: &mockSecretsClient{
				getSecretMapErr: errors.New("extract error"),
			},
			expectedOutput: nil,
			expectsError:   true,
		},
		{
			name: "error with invalid dataFrom specification",
			step: &workflows.PullStep{
				Source: esv1.StoreSourceRef{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "store1",
					},
				},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
					{}, // Neither Find nor Extract specified
				},
			},
			mockClient:     &mockSecretsClient{},
			expectedOutput: nil,
			expectsError:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a mock client - nil is fine for tests
			mockClient := client.Client(nil)

			// Create a mock manager that returns our mock client
			mockMgr := &mockManager{
				secretsClient: test.mockClient,
				getClientErr:  test.managerErr,
			}

			// Create executor with our mock manager
			executor := NewPullStepExecutor(test.step, mockClient, mockMgr)

			// Execute using the real implementation
			output, err := executor.Execute(context.Background(), mockClient, &workflows.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
			}, map[string]interface{}{}, "test-job")

			if test.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedOutput, output)
			}
		})
	}
}

func TestNewPullStepExecutor(t *testing.T) {
	mockClient := client.Client(nil) // Use nil for interface in test
	step := &workflows.PullStep{
		Source: esv1.StoreSourceRef{
			SecretStoreRef: esv1.SecretStoreRef{
				Name: "store1",
			},
		},
	}

	// Test with a mock manager
	mockMgr := &mockManager{
		secretsClient: &mockSecretsClient{},
	}
	executor := NewPullStepExecutor(step, mockClient, mockMgr)
	assert.Equal(t, step, executor.Step)
	assert.Equal(t, mockClient, executor.Client)
	assert.Equal(t, mockMgr, executor.Manager)
}
