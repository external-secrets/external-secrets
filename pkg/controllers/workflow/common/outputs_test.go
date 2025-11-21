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

package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// func TestIsSensitive(t *testing.T) {
// 	tests := []struct {
// 		name           string
// 		key            string
// 		sensitiveKeys  []string
// 		expectedResult bool
// 	}{
// 		{
// 			name:           "explicit sensitive key",
// 			key:            "api-key",
// 			sensitiveKeys:  []string{"api-key", "other-key"},
// 			expectedResult: true,
// 		},
// 		{
// 			name:           "pattern match - password",
// 			key:            "user-password",
// 			sensitiveKeys:  []string{},
// 			expectedResult: true,
// 		},
// 		{
// 			name:           "pattern match - token",
// 			key:            "access-token",
// 			sensitiveKeys:  []string{},
// 			expectedResult: true,
// 		},
// 		{
// 			name:           "pattern match - key",
// 			key:            "encryption-key",
// 			sensitiveKeys:  []string{},
// 			expectedResult: true,
// 		},
// 		{
// 			name:           "pattern match - secret",
// 			key:            "client-secret",
// 			sensitiveKeys:  []string{},
// 			expectedResult: true,
// 		},
// 		{
// 			name:           "pattern match - credential",
// 			key:            "aws-credential",
// 			sensitiveKeys:  []string{},
// 			expectedResult: true,
// 		},
// 		{
// 			name:           "pattern match - auth",
// 			key:            "authorization",
// 			sensitiveKeys:  []string{},
// 			expectedResult: true,
// 		},
// 		{
// 			name:           "not sensitive",
// 			key:            "username",
// 			sensitiveKeys:  []string{},
// 			expectedResult: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			result := IsSensitive(tt.key, tt.sensitiveKeys)
// 			assert.Equal(t, tt.expectedResult, result)
// 		})
// 	}
// }

// No longer needed as we're only processing interface maps

func TestProcessOutputs(t *testing.T) {
	now := time.Now()
	jsonObj := map[string]interface{}{"foo": "bar"}
	jsonBytes, _ := json.Marshal(jsonObj)

	tests := []struct {
		name              string
		outputs           map[string]interface{}
		step              workflows.Step
		expected          map[string]string
		expectedSensitive map[string]string
		expectError       bool
	}{
		{
			name: "interface map",
			outputs: map[string]interface{}{
				"username":    "john",
				"password":    "secret123",
				"is_active":   true,
				"login_count": 42,
				"last_login":  now,
				"metadata":    jsonObj,
			},
			step: workflows.Step{
				Name: "test-step",
				Outputs: []workflows.OutputDefinition{
					{
						Name:      "password",
						Type:      workflows.OutputTypeMap,
						Sensitive: true,
					},
				},
			},
			expected: map[string]string{
				"username":    "john",
				"password":    MaskValue,
				"is_active":   "true",
				"login_count": "42",
				"last_login":  now.Format(time.RFC3339),
				"metadata":    string(jsonBytes),
			},
			expectedSensitive: map[string]string{
				"password": "secret123",
			},
			expectError: false,
		},
		{
			name:    "nil outputs",
			outputs: nil,
			step: workflows.Step{
				Name: "test-step",
			},
			expected:          nil,
			expectedSensitive: nil,
			expectError:       false,
		},
		{
			name: "already processed outputs",
			outputs: map[string]interface{}{
				"username":    "john",
				"is_active":   "true",
				"login_count": "42",
				"last_login":  now.Format(time.RFC3339),
				"metadata":    string(jsonBytes),
			},
			step: workflows.Step{
				Name: "test-step",
				Outputs: []workflows.OutputDefinition{
					{
						Name: "username",
						Type: workflows.OutputTypeString,
					},
					{
						Name: "is_active",
						Type: workflows.OutputTypeBool,
					},
					{
						Name: "login_count",
						Type: workflows.OutputTypeNumber,
					},
					{
						Name: "last_login",
						Type: workflows.OutputTypeTime,
					},
					{
						Name: "metadata",
						Type: workflows.OutputTypeMap,
					},
				},
			},
			expected: map[string]string{
				"username":    "john",
				"is_active":   "true",
				"login_count": "42",
				"last_login":  now.Format(time.RFC3339),
				"metadata":    string(jsonBytes),
			},
			expectedSensitive: map[string]string{},
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, sensitiveValues, err := ProcessOutputs(tt.outputs, tt.step)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
				assert.Equal(t, tt.expectedSensitive, sensitiveValues)
			}
		})
	}
}
