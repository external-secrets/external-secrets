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
package templates

import (
	"testing"
)

func TestPreprocessTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "global variable",
			template: "Hello $name",
			expected: "Hello {{ .global.variables.name }}",
		},
		{
			name:     "step variable",
			template: "Result: $job1.step1.result",
			expected: "Result: {{ .global.jobs.job1.step1.result }}",
		},
		{
			name:     "mixed variables",
			template: "Hello $name, your result is $job1.step1.result",
			expected: "Hello {{ .global.variables.name }}, your result is {{ .global.jobs.job1.step1.result }}",
		},
		{
			name:     "existing syntax",
			template: "Hello {{ .global.variables.name }}",
			expected: "Hello {{ .global.variables.name }}",
		},
		{
			name:     "dollar sign in text",
			template: "Cost: $100",
			expected: "Cost: $100", // Should not be replaced since it doesn't match the pattern
		},
		{
			name:     "dollar sign with valid variable",
			template: "Hello $user123",
			expected: "Hello {{ .global.variables.user123 }}",
		},
		{
			name:     "escaped dollar sign",
			template: "Cost: $$100",
			expected: "Cost: $100",
		},
		{
			name:     "escaped dollar sign with variable",
			template: "Cost: $$100 for $item",
			expected: "Cost: $100 for {{ .global.variables.item }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessTemplate(tt.template)
			if result != tt.expected {
				t.Errorf("preprocessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResolveTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "global variable with $ syntax",
			template: "Hello $name",
			data: map[string]interface{}{
				"global": map[string]interface{}{
					"variables": map[string]interface{}{
						"name": "ESOInc.",
					},
				},
			},
			expected: "Hello ESOInc.",
			wantErr:  false,
		},
		{
			name:     "step variable with $ syntax",
			template: "Result: $job1.step1.result",
			data: map[string]interface{}{
				"global": map[string]interface{}{
					"jobs": map[string]interface{}{
						"job1": map[string]interface{}{
							"step1": map[string]interface{}{
								"result": "success",
							},
						},
					},
				},
			},
			expected: "Result: success",
			wantErr:  false,
		},
		{
			name:     "mixed variables with $ syntax",
			template: "Hello $name, your result is $job1.step1.result",
			data: map[string]interface{}{
				"global": map[string]interface{}{
					"variables": map[string]interface{}{
						"name": "ESOInc.",
					},
					"jobs": map[string]interface{}{
						"job1": map[string]interface{}{
							"step1": map[string]interface{}{
								"result": "success",
							},
						},
					},
				},
			},
			expected: "Hello ESOInc., your result is success",
			wantErr:  false,
		},
		{
			name:     "existing syntax",
			template: "Hello {{ .global.variables.name }}",
			data: map[string]interface{}{
				"global": map[string]interface{}{
					"variables": map[string]interface{}{
						"name": "ESOInc.",
					},
				},
			},
			expected: "Hello ESOInc.",
			wantErr:  false,
		},
		{
			name:     "escaped dollar sign",
			template: "Cost: $$100",
			data: map[string]interface{}{
				"global": map[string]interface{}{},
			},
			expected: "Cost: $100",
			wantErr:  false,
		},
		{
			name:     "escaped dollar sign with variable",
			template: "Cost: $$100 for $item",
			data: map[string]interface{}{
				"global": map[string]interface{}{
					"variables": map[string]interface{}{
						"item": "apples",
					},
				},
			},
			expected: "Cost: $100 for apples",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveTemplate(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ResolveTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}
