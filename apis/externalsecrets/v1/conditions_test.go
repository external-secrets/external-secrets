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

package v1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
)

func TestGetExternalSecretCondition(t *testing.T) {
	status := ExternalSecretStatus{
		Conditions: []ExternalSecretStatusCondition{
			{
				Type:   ExternalSecretReady,
				Status: corev1.ConditionFalse,
			},
			{
				Type:   ExternalSecretReady,
				Status: corev1.ConditionTrue,
			},
		},
	}

	tests := []struct {
		name     string
		condType ExternalSecretConditionType
		expected *ExternalSecretStatusCondition
	}{
		{
			name:     "Status has a condition of the specified type",
			condType: ExternalSecretReady,
			expected: &ExternalSecretStatusCondition{
				Type:   ExternalSecretReady,
				Status: corev1.ConditionFalse,
			},
		},
		{
			name:     "Status does not have a condition of the specified type",
			condType: ExternalSecretDeleted,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExternalSecretCondition(status, tt.condType)

			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("(-got, +want)\n%s", diff)
			}
		})
	}
}
