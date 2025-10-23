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

package provider

import (
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

const (
	// ProviderSubsystem is the subsystem name for Provider metrics.
	ProviderSubsystem = "provider"

	// ProviderReconcileDurationKey is the key for the reconcile duration metric.
	ProviderReconcileDurationKey = "reconcile_duration"

	// StatusConditionKey is the key for the status condition metric.
	StatusConditionKey = "status_condition"
)

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// SetUpMetrics initializes the metrics for the Provider controller.
func SetUpMetrics() {
	providerReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ProviderSubsystem,
		Name:      ProviderReconcileDurationKey,
		Help:      "The duration time to reconcile the Provider",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	providerCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ProviderSubsystem,
		Name:      StatusConditionKey,
		Help:      "The status condition of a specific Provider",
	}, ctrlmetrics.ConditionMetricLabelNames)

	metrics.Registry.MustRegister(providerReconcileDuration, providerCondition)

	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		ProviderReconcileDurationKey: providerReconcileDuration,
		StatusConditionKey:           providerCondition,
	}
}

// GetGaugeVec returns the GaugeVec for the given key.
func GetGaugeVec(key string) *prometheus.GaugeVec {
	return gaugeVecMetrics[key]
}

// RemoveMetrics deletes all metrics published by the resource.
func RemoveMetrics(namespace, name string) {
	for _, gaugeVecMetric := range gaugeVecMetrics {
		gaugeVecMetric.DeletePartialMatch(
			map[string]string{
				"namespace": namespace,
				"name":      name,
			},
		)
	}
}

// UpdateStatusCondition updates the condition metrics for a Provider.
func UpdateStatusCondition(provider *esapi.Provider, condition esapi.ProviderCondition) {
	providerInfo := make(map[string]string)
	providerInfo["name"] = provider.GetName()
	providerInfo["namespace"] = provider.GetNamespace()
	for k, v := range provider.GetLabels() {
		providerInfo[k] = v
	}
	conditionLabels := ctrlmetrics.RefineConditionMetricLabels(providerInfo)
	providerConditionGauge := GetGaugeVec(StatusConditionKey)

	if condition.Type == esapi.ProviderReady {
		switch condition.Status {
		case metav1.ConditionFalse:
			providerConditionGauge.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esapi.ProviderReady),
					"status":    string(metav1.ConditionTrue),
				})).Set(0)
		case metav1.ConditionTrue:
			providerConditionGauge.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esapi.ProviderReady),
					"status":    string(metav1.ConditionFalse),
				})).Set(0)
		case metav1.ConditionUnknown:
			break
		}
	}

	providerConditionGauge.With(ctrlmetrics.RefineLabels(conditionLabels,
		map[string]string{
			"condition": string(condition.Type),
			"status":    string(condition.Status),
		})).Set(1)
}

