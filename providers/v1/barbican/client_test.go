/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package barbican

import (
	"context"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/stretchr/testify/assert"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestExtractUUIDFromRef(t *testing.T) {
	testCases := []struct {
		name         string
		secretRef    string
		expectedUUID string
	}{
		{
			name:         "valid barbican secret ref",
			secretRef:    "https://barbican.example.com/v1/secrets/12345678-1234-1234-1234-123456789abc",
			expectedUUID: "12345678-1234-1234-1234-123456789abc",
		},
		{
			name:         "secret ref without protocol",
			secretRef:    "barbican.example.com/v1/secrets/87654321-4321-4321-4321-cba987654321",
			expectedUUID: "87654321-4321-4321-4321-cba987654321",
		},
		{
			name:         "empty string",
			secretRef:    "",
			expectedUUID: "",
		},
		{
			name:         "trailing slash",
			secretRef:    "https://barbican.example.com/v1/secrets/12345678-1234-1234-1234-123456789abc/",
			expectedUUID: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uuid := extractUUIDFromRef(tc.secretRef)
			assert.Equal(t, tc.expectedUUID, uuid)
		})
	}
}

func TestGetSecretPayloadProperty(t *testing.T) {
	testPayload := []byte(`{"username":"admin","password":"secret123","nested":{"key":"value"}}`)

	testCases := []struct {
		name         string
		payload      []byte
		property     string
		expectError  bool
		errorMessage string
		expectedData []byte
	}{
		{
			name:         "empty property returns full payload",
			payload:      testPayload,
			property:     "",
			expectError:  false,
			expectedData: testPayload,
		},
		{
			name:         "valid property extraction",
			payload:      testPayload,
			property:     "username",
			expectError:  false,
			expectedData: []byte(`"admin"`),
		},
		{
			name:         "nested property extraction",
			payload:      testPayload,
			property:     "nested",
			expectError:  false,
			expectedData: []byte(`{"key":"value"}`),
		},
		{
			name:         "property not found",
			payload:      testPayload,
			property:     "nonexistent",
			expectError:  true,
			errorMessage: "property nonexistent not found in secret payload",
		},
		{
			name:         "invalid JSON",
			payload:      []byte("invalid-json"),
			property:     "username",
			expectError:  true,
			errorMessage: "barbican client",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := getSecretPayloadProperty(tc.payload, tc.property)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMessage)
				assert.Nil(t, data)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedData, data)
			}
		})
	}
}

func TestUnsupportedOperations(t *testing.T) {
	client := &Client{
		keyManager: &gophercloud.ServiceClient{},
	}

	// Test PushSecret
	err := client.PushSecret(context.Background(), nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support pushing secrets")

	// Test SecretExists
	exists, err := client.SecretExists(context.Background(), nil)
	assert.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "barbican provider does not pushing secrets with update policy IfNotExists")

	// Test DeleteSecret
	err = client.DeleteSecret(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support deleting secrets")
}

func TestValidateAndClose(t *testing.T) {
	client := &Client{
		keyManager: &gophercloud.ServiceClient{},
	}

	// Test Validate
	result, err := client.Validate()
	assert.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultUnknown, result)

	// Test Close
	err = client.Close(context.Background())
	assert.NoError(t, err)
}

func TestGetAllSecretsValidation(t *testing.T) {
	client := &Client{
		keyManager: &gophercloud.ServiceClient{},
	}

	testCases := []struct {
		name         string
		findRef      esv1.ExternalSecretFind
		expectError  bool
		errorMessage string
	}{
		{
			name: "no name specified should return error",
			findRef: esv1.ExternalSecretFind{
				Name: nil,
			},
			expectError:  true,
			errorMessage: "missing field",
		},
		{
			name: "empty name regex should return error",
			findRef: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "",
				},
			},
			expectError:  true,
			errorMessage: "missing field",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.GetAllSecrets(context.Background(), tc.findRef)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMessage)
			} else if err != nil {
					assert.Contains(t, err.Error(), "barbican client")
			}
		})
	}
}
