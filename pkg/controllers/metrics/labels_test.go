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

package metrics

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
)

func TestSetUpLabelNames(t *testing.T) {
	testCases := []struct {
		description                          string
		addKubeStandardLabels                bool
		expectedNonConditionMetricLabelNames []string
		expectedConditionMetricLabelNames    []string
		expectedNonConditionMetricLabels     map[string]string
		expectedConditionMetricLabels        map[string]string
	}{
		{
			description:           "Add standard labels disabled",
			addKubeStandardLabels: false,
			expectedNonConditionMetricLabelNames: []string{
				"name",
				"namespace",
			},
			expectedConditionMetricLabelNames: []string{
				"name",
				"namespace",
				"condition",
				"status",
			},
			expectedNonConditionMetricLabels: map[string]string{
				"name":      "",
				"namespace": "",
			},
			expectedConditionMetricLabels: map[string]string{
				"name":      "",
				"namespace": "",
				"condition": "",
				"status":    "",
			},
		},
		{
			description:           "Add standard labels enabled",
			addKubeStandardLabels: true,
			expectedNonConditionMetricLabelNames: []string{
				"name",
				"namespace",
				"app_kubernetes_io_name",
				"app_kubernetes_io_instance",
				"app_kubernetes_io_version",
				"app_kubernetes_io_component",
				"app_kubernetes_io_part_of",
				"app_kubernetes_io_managed_by",
			},
			expectedConditionMetricLabelNames: []string{
				"name",
				"namespace",
				"condition",
				"status",
				"app_kubernetes_io_name",
				"app_kubernetes_io_instance",
				"app_kubernetes_io_version",
				"app_kubernetes_io_component",
				"app_kubernetes_io_part_of",
				"app_kubernetes_io_managed_by",
			},
			expectedNonConditionMetricLabels: map[string]string{
				"name":                         "",
				"namespace":                    "",
				"app_kubernetes_io_name":       "",
				"app_kubernetes_io_instance":   "",
				"app_kubernetes_io_version":    "",
				"app_kubernetes_io_component":  "",
				"app_kubernetes_io_part_of":    "",
				"app_kubernetes_io_managed_by": "",
			},
			expectedConditionMetricLabels: map[string]string{
				"name":                         "",
				"namespace":                    "",
				"condition":                    "",
				"status":                       "",
				"app_kubernetes_io_name":       "",
				"app_kubernetes_io_instance":   "",
				"app_kubernetes_io_version":    "",
				"app_kubernetes_io_component":  "",
				"app_kubernetes_io_part_of":    "",
				"app_kubernetes_io_managed_by": "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			SetUpLabelNames(tc.addKubeStandardLabels)

			if diff := cmp.Diff(NonConditionMetricLabelNames, tc.expectedNonConditionMetricLabelNames); diff != "" {
				t.Errorf("NonConditionMetricLabelNames does not match the expected value. (-got +want)\n%s", diff)
			}

			if diff := cmp.Diff(ConditionMetricLabelNames, tc.expectedConditionMetricLabelNames); diff != "" {
				t.Errorf("ConditionMetricLabelNames does not match the expected value. (-got +want)\n%s", diff)
			}

			if diff := cmp.Diff(NonConditionMetricLabels, tc.expectedNonConditionMetricLabels); diff != "" {
				t.Errorf("NonConditionMetricLabels are not initialized with empty strings. (-got +want)\n%s", diff)
			}

			if diff := cmp.Diff(ConditionMetricLabels, tc.expectedConditionMetricLabels); diff != "" {
				t.Errorf("ConditionMetricLabels are not initialized with empty strings. (-got +want)\n%s", diff)
			}
		})
	}
}

func TestRefineLabels(t *testing.T) {
	testCases := []struct {
		description        string
		promLabels         prometheus.Labels
		newLabels          map[string]string
		expectedRefinement prometheus.Labels
	}{
		{
			description: "No new labels",
			promLabels: prometheus.Labels{
				"label1": "value1",
				"label2": "value2",
			},
			newLabels:          map[string]string{},
			expectedRefinement: prometheus.Labels{"label1": "value1", "label2": "value2"},
		},
		{
			description: "Add unregistered labels",
			promLabels: prometheus.Labels{
				"label1": "value1",
				"label2": "value2",
			},
			newLabels: map[string]string{
				"new_label1": "new_value1",
				"new_label2": "new_value2",
			},
			expectedRefinement: prometheus.Labels{
				"label1": "value1",
				"label2": "value2",
			},
		},
		{
			description: "Overwrite existing labels",
			promLabels: prometheus.Labels{
				"label1": "value1",
				"label2": "value2",
			},
			newLabels: map[string]string{
				"label1": "new_value1",
				"label2": "new_value2",
			},
			expectedRefinement: prometheus.Labels{
				"label1": "new_value1",
				"label2": "new_value2",
			},
		},
		{
			description: "Clean non-alphanumeric characters in new labels",
			promLabels: prometheus.Labels{
				"label_1": "value1",
			},
			newLabels: map[string]string{
				"label@1": "new_value",
			},
			expectedRefinement: prometheus.Labels{
				"label_1": "new_value",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			refinement := RefineLabels(tc.promLabels, tc.newLabels)

			if diff := cmp.Diff(refinement, tc.expectedRefinement); diff != "" {
				t.Errorf("Refinement does not match the expected value. (-got +want)\n%s", diff)
			}
		})
	}
}
