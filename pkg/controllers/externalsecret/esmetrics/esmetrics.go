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
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

const (
	ExternalSecretSubsystem            = "externalsecret"
	SyncCallsKey                       = "sync_calls_total"
	SyncCallsErrorKey                  = "sync_calls_error"
	ExternalSecretStatusConditionKey   = "status_condition"
	ExternalSecretReconcileDurationKey = "reconcile_duration"
)

var counterVecMetrics = map[string]*prometheus.CounterVec{}

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// Called at the root to set-up the metric logic using the
// config flags provided.
func SetUpMetrics() {
	// Obtain the prometheus metrics and register
	syncCallsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsKey,
		Help:      "Total number of the External Secret sync calls",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	syncCallsError := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsErrorKey,
		Help:      "Total number of the External Secret sync errors",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	externalSecretCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      ExternalSecretStatusConditionKey,
		Help:      "The status condition of a specific External Secret",
	}, ctrlmetrics.ConditionMetricLabelNames)

	externalSecretReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      ExternalSecretReconcileDurationKey,
		Help:      "The duration time to reconcile the External Secret",
	}, ctrlmetrics.NonConditionMetricLabelNames)

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
	conditionLabels := ctrlmetrics.RefineConditionMetricLabels(esInfo)
	externalSecretCondition := GetGaugeVec(ExternalSecretStatusConditionKey)

	switch condition.Type {
	case esv1beta1.ExternalSecretDeleted:
		// Remove condition=Ready metrics when the object gets deleted.
		externalSecretCondition.Delete(ctrlmetrics.RefineLabels(conditionLabels,
			map[string]string{
				"condition": string(esv1beta1.ExternalSecretReady),
				"status":    string(v1.ConditionFalse),
			}))

		externalSecretCondition.Delete(ctrlmetrics.RefineLabels(conditionLabels,
			map[string]string{
				"condition": string(esv1beta1.ExternalSecretReady),
				"status":    string(v1.ConditionTrue),
			}))

	case esv1beta1.ExternalSecretReady:
		// Remove condition=Deleted metrics when the object gets ready.
		externalSecretCondition.Delete(ctrlmetrics.RefineLabels(conditionLabels,
			map[string]string{
				"condition": string(esv1beta1.ExternalSecretDeleted),
				"status":    string(v1.ConditionFalse),
			}))

		externalSecretCondition.Delete(ctrlmetrics.RefineLabels(conditionLabels,
			map[string]string{
				"condition": string(esv1beta1.ExternalSecretDeleted),
				"status":    string(v1.ConditionTrue),
			}))

		// Toggle opposite Status to 0
		switch condition.Status {
		case v1.ConditionFalse:
			externalSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esv1beta1.ExternalSecretReady),
					"status":    string(v1.ConditionTrue),
				})).Set(0)
		case v1.ConditionTrue:
			externalSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
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

	externalSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
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
