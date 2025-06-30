/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParameterTagsToJSONString(t *testing.T) {
	tests := []struct {
		name     string
		tags     map[string]string
		expected string
		wantErr  bool
	}{
		{
			name: "Valid tags",
			tags: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: `{"key1":"value1","key2":"value2"}`,
			wantErr:  false,
		},
		{
			name:     "Empty tags",
			tags:     map[string]string{},
			expected: `{}`,
			wantErr:  false,
		},
		{
			name:     "Nil tags",
			tags:     nil,
			wantErr:  false,
			expected: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParameterTagsToJSONString(tt.tags)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				var resultMap map[string]string
				err := json.Unmarshal([]byte(result), &resultMap)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
