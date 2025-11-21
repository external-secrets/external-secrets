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
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"

	esapi "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

func TestJavaScriptExecutor(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		script   string
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name: "set string value",
			input: map[string]interface{}{
				"key": "value",
			},
			script:   "setString('result', 'test');",
			expected: map[string]interface{}{"result": "test"},
			wantErr:  false,
		},
		{
			name: "set boolean value",
			input: map[string]interface{}{
				"key": "value",
			},
			script:   "setBool('result', true);",
			expected: map[string]interface{}{"result": true},
			wantErr:  false,
		},
		{
			name: "set number value",
			input: map[string]interface{}{
				"key": "value",
			},
			script:   "setNumber('result', 123);",
			expected: map[string]interface{}{"result": int64(123)},
			wantErr:  false,
		},
		{
			name: "set date value",
			input: map[string]interface{}{
				"key": "value",
			},
			script:   "setDate('result', new Date('2025-01-01T00:00:00Z'));",
			expected: map[string]interface{}{"result": time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			wantErr:  false,
		},
		{
			name: "set JSON value",
			input: map[string]interface{}{
				"key": "value",
			},
			script:   `setJSON('result', '{"name":"test","value":true}');`,
			expected: map[string]interface{}{"result": map[string]interface{}{"name": "test", "value": true}},
			wantErr:  false,
		},
		{
			name: "set array value",
			input: map[string]interface{}{
				"key": "value",
			},
			script:   "setArray('result', ['test']);",
			expected: map[string]interface{}{"result": []interface{}{"test"}},
			wantErr:  false,
		},
		{
			name: "set map value",
			input: map[string]interface{}{
				"key": "value",
			},
			script:   "setMap('result', 'key', 'value');",
			expected: map[string]interface{}{"result": map[string]interface{}{"key": "value"}},
			wantErr:  false,
		},
		{
			name: "invalid script",
			input: map[string]interface{}{
				"key": "value",
			},
			script:   "setUnexisting('shouldnot', 'work')",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewJavaScriptExecutor(&esapi.JavaScriptStep{Script: tt.script}, logr.Discard())
			result, err := executor.Execute(context.Background(), nil, nil, tt.input, "job-test")

			if (err != nil) != tt.wantErr {
				t.Errorf("JavaScriptExecutor.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			require.Equal(t, tt.expected, result)
		})
	}
}
