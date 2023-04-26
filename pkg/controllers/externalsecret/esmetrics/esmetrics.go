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

package esmetrics

import (
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	ExternalSecretSubsystem            = "externalsecret"
	SyncCallsKey                       = "sync_calls_total"
	SyncCallsErrorKey                  = "sync_calls_error"
	ExternalSecretStatusConditionKey   = "status_condition"
	ExternalSecretReconcileDurationKey = "reconcile_duration"
)

var (
	NonConditionMetricLabelNames = make([]string, 0)

	ConditionMetricLabelNames = make([]string, 0)

	NonConditionMetricLabels = make(map[string]string)

	ConditionMetricLabels = make(map[string]string)
)

var counterVecMetrics map[string]*prometheus.CounterVec = map[string]*prometheus.CounterVec{}

var gaugeVecMetrics map[string]*prometheus.GaugeVec = map[string]*prometheus.GaugeVec{}

// Called at the root to set-up the metric logic using the
// config flags provided.
func SetUpMetrics(addKubeStandardLabels bool) {
	// Figure out what the labels for the metrics are
	if addKubeStandardLabels {
		NonConditionMetricLabelNames = []string{
			"name", "namespace",
			"app_kubernetes_io_name", "app_kubernetes_io_instance",
			"app_kubernetes_io_version", "app_kubernetes_io_component",
			"app_kubernetes_io_part_of", "app_kubernetes_io_managed_by",
		}

		ConditionMetricLabelNames = []string{
			"name", "namespace",
			"condition", "status",
			"app_kubernetes_io_name", "app_kubernetes_io_instance",
			"app_kubernetes_io_version", "app_kubernetes_io_component",
			"app_kubernetes_io_part_of", "app_kubernetes_io_managed_by",
		}
	} else {
		NonConditionMetricLabelNames = []string{"name", "namespace"}

		ConditionMetricLabelNames = []string{"name", "namespace", "condition", "status"}
	}

	// Set default values for each label
	for _, k := range NonConditionMetricLabelNames {
		NonConditionMetricLabels[k] = ""
	}

	for _, k := range ConditionMetricLabelNames {
		ConditionMetricLabels[k] = ""
	}

	// Obtain the prometheus metrics and register
	syncCallsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsKey,
		Help:      "Total number of the External Secret sync calls",
	}, NonConditionMetricLabelNames)

	syncCallsError := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsErrorKey,
		Help:      "Total number of the External Secret sync errors",
	}, NonConditionMetricLabelNames)

	externalSecretCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      ExternalSecretStatusConditionKey,
		Help:      "The status condition of a specific External Secret",
	}, ConditionMetricLabelNames)

	externalSecretReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      ExternalSecretReconcileDurationKey,
		Help:      "The duration time to reconcile the External Secret",
	}, NonConditionMetricLabelNames)

	metrics.Registry.MustRegister(syncCallsTotal, syncCallsError, externalSecretCondition, externalSecretReconcileDuration)

	counterVecMetrics = map[string]*prometheus.CounterVec{
		SyncCallsKey:      syncCallsTotal,
		SyncCallsErrorKey: syncCallsError,
	}

	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		ExternalSecretStatusConditionKey:   externalSecretCondition,
		ExternalSecretReconcileDurationKey: externalSecretReconcileDuration,
	}
}

func UpdateExternalSecretCondition(es *esv1beta1.ExternalSecret, condition *esv1beta1.ExternalSecretStatusCondition, value float64) {
	esInfo := make(map[string]string)
	esInfo["name"] = es.Name
	esInfo["namespace"] = es.Namespace
	for k, v := range es.Labels {
		esInfo[k] = v
	}
	conditionLabels := RefineConditionMetricLabels(esInfo)
	externalSecretCondition := GetGaugeVec(ExternalSecretStatusConditionKey)

	switch condition.Type {
	case esv1beta1.ExternalSecretDeleted:
		// Remove condition=Ready metrics when the object gets deleted.
		externalSecretCondition.Delete(RefineLabels(conditionLabels,
			map[string]string{
				"condition": string(esv1beta1.ExternalSecretReady),
				"status":    string(v1.ConditionFalse),
			}))

		externalSecretCondition.Delete(RefineLabels(conditionLabels,
			map[string]string{
				"condition": string(esv1beta1.ExternalSecretReady),
				"status":    string(v1.ConditionTrue),
			}))

	case esv1beta1.ExternalSecretReady:
		// Remove condition=Deleted metrics when the object gets ready.
		externalSecretCondition.Delete(RefineLabels(conditionLabels,
			map[string]string{
				"condition": string(esv1beta1.ExternalSecretDeleted),
				"status":    string(v1.ConditionFalse),
			}))

		externalSecretCondition.Delete(RefineLabels(conditionLabels,
			map[string]string{
				"condition": string(esv1beta1.ExternalSecretDeleted),
				"status":    string(v1.ConditionTrue),
			}))

		// Toggle opposite Status to 0
		switch condition.Status {
		case v1.ConditionFalse:
			externalSecretCondition.With(RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esv1beta1.ExternalSecretReady),
					"status":    string(v1.ConditionTrue),
				})).Set(0)
		case v1.ConditionTrue:
			externalSecretCondition.With(RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esv1beta1.ExternalSecretReady),
					"status":    string(v1.ConditionFalse),
				})).Set(0)
		case v1.ConditionUnknown:
			break
		default:
			break
		}

	default:
		break
	}

	externalSecretCondition.With(RefineLabels(conditionLabels,
		map[string]string{
			"condition": string(condition.Type),
			"status":    string(condition.Status),
		})).Set(value)
}

func GetCounterVec(key string) *prometheus.CounterVec {
	return counterVecMetrics[key]
}

func GetGaugeVec(key string) *prometheus.GaugeVec {
	return gaugeVecMetrics[key]
}

// Refine the given Prometheus Labels with values from a map `newLabels`
// Only overwrite a value if the corresponding key is present in the
// Prometheus' Labels already to avoid adding label names which are
// not defined in a metric's description. Note that non-alphanumeric
// characters from keys of `newLabels` are replaced by an underscore
// because Prometheus does not accept non-alphanumeric, non-underscore
// characters in label names.
func RefineLabels(promLabels prometheus.Labels, newLabels map[string]string) prometheus.Labels {
	nonAlphanumericRegex := regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
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
