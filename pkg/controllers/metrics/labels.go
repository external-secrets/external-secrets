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
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	NonConditionMetricLabelNames = make([]string, 0)

	ConditionMetricLabelNames = make([]string, 0)

	NonConditionMetricLabels = make(map[string]string)

	ConditionMetricLabels = make(map[string]string)
)

var nonAlphanumericRegex *regexp.Regexp

func init() {
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
}

// SetUpLabelNames initializes both non-conditional and conditional metric labels and label names.
func SetUpLabelNames(addKubeStandardLabels bool) {
	NonConditionMetricLabelNames = []string{"name", "namespace"}
	ConditionMetricLabelNames = []string{"name", "namespace", "condition", "status"}

	// Figure out what the labels for the metrics are
	if addKubeStandardLabels {
		NonConditionMetricLabelNames = append(
			NonConditionMetricLabelNames,
			"app_kubernetes_io_name", "app_kubernetes_io_instance",
			"app_kubernetes_io_version", "app_kubernetes_io_component",
			"app_kubernetes_io_part_of", "app_kubernetes_io_managed_by",
		)

		ConditionMetricLabelNames = append(
			ConditionMetricLabelNames,
			"app_kubernetes_io_name", "app_kubernetes_io_instance",
			"app_kubernetes_io_version", "app_kubernetes_io_component",
			"app_kubernetes_io_part_of", "app_kubernetes_io_managed_by",
		)
	}

	// Set default values for each label
	for _, k := range NonConditionMetricLabelNames {
		NonConditionMetricLabels[k] = ""
	}

	for _, k := range ConditionMetricLabelNames {
		ConditionMetricLabels[k] = ""
	}
}

// RefineLabels refines the given Prometheus Labels with values from a map `newLabels`
// Only overwrite a value if the corresponding key is present in the
// Prometheus' Labels already to avoid adding label names which are
// not defined in a metric's description. Note that non-alphanumeric
// characters from keys of `newLabels` are replaced by an underscore
// because Prometheus does not accept non-alphanumeric, non-underscore
// characters in label names.
func RefineLabels(promLabels prometheus.Labels, newLabels map[string]string) prometheus.Labels {
	var refinement = prometheus.Labels{}

	for k, v := range promLabels {
		refinement[k] = v
	}

	for k, v := range newLabels {
		cleanKey := nonAlphanumericRegex.ReplaceAllString(k, "_")
		if _, ok := refinement[cleanKey]; ok {
			refinement[cleanKey] = v
		}
	}

	return refinement
}

func RefineNonConditionMetricLabels(labels map[string]string) prometheus.Labels {
	return RefineLabels(NonConditionMetricLabels, labels)
}

func RefineConditionMetricLabels(labels map[string]string) prometheus.Labels {
	return RefineLabels(ConditionMetricLabels, labels)
}
