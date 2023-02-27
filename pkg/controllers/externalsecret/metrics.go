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

package externalsecret

import (
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	ExternalSecretSubsystem            = "externalsecret"
	SyncCallsKey                       = "sync_calls_total"
	SyncCallsErrorKey                  = "sync_calls_error"
	externalSecretStatusConditionKey   = "status_condition"
	externalSecretReconcileDurationKey = "reconcile_duration"
)

var (
	NonConditionMetricLabelNames = []string{
		"name", "namespace",
		"app_kubernetes_io_name", "app_kubernetes_io_instance",
		"app_kubernetes_io_version", "app_kubernetes_io_component",
		"app_kubernetes_io_part_of", "app_kubernetes_io_managed_by",
	}

	NonConditionMetricLabels = prometheus.Labels{
		"name":                         "",
		"namespace":                    "",
		"app_kubernetes_io_name":       "",
		"app_kubernetes_io_instance":   "",
		"app_kubernetes_io_version":    "",
		"app_kubernetes_io_component":  "",
		"app_kubernetes_io_part_of":    "",
		"app_kubernetes_io_managed_by": "",
	}

	ConditionMetricLabelNames = []string{
		"name", "namespace",
		"condition", "status",
		"app_kubernetes_io_name", "app_kubernetes_io_instance",
		"app_kubernetes_io_version", "app_kubernetes_io_component",
		"app_kubernetes_io_part_of", "app_kubernetes_io_managed_by",
	}

	ConditionMetricLabels = prometheus.Labels{
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
	}
)

var (
	syncCallsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsKey,
		Help:      "Total number of the External Secret sync calls",
	}, NonConditionMetricLabelNames)

	syncCallsError = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsErrorKey,
		Help:      "Total number of the External Secret sync errors",
	}, NonConditionMetricLabelNames)

	externalSecretCondition = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      externalSecretStatusConditionKey,
		Help:      "The status condition of a specific External Secret",
	}, ConditionMetricLabelNames)

	externalSecretReconcileDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      externalSecretReconcileDurationKey,
		Help:      "The duration time to reconcile the External Secret",
	}, NonConditionMetricLabelNames)
)

// updateExternalSecretCondition updates the ExternalSecret conditions.
func updateExternalSecretCondition(es *esv1beta1.ExternalSecret, condition *esv1beta1.ExternalSecretStatusCondition, value float64) {
	conditionLabels := RefineLabelsWithMaps(ConditionMetricLabels, map[string]string{"name": es.Name, "namespace": es.Namespace}, es.Labels)

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

func init() {
	metrics.Registry.MustRegister(syncCallsTotal, syncCallsError, externalSecretCondition, externalSecretReconcileDuration)
}
