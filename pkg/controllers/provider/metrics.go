/*
Copyright © The ESO Authors

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

package provider

import (
	"maps"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

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

var (
	gaugeVecMetrics     = map[string]*prometheus.GaugeVec{}
	registerMetricsOnce sync.Once
)

// SetUpMetrics initializes the metrics for the Provider controller.
func SetUpMetrics() {
	registerMetricsOnce.Do(func() {
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
	})
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

// UpdateStatusCondition updates the legacy Provider condition metrics for a v2 ProviderStore.
func UpdateStatusCondition(name, namespace string, labels map[string]string, conditionType, conditionStatus string) {
	providerInfo := map[string]string{
		"name":      name,
		"namespace": namespace,
	}
	maps.Copy(providerInfo, labels)
	conditionLabels := ctrlmetrics.RefineConditionMetricLabels(providerInfo)
	providerConditionGauge := GetGaugeVec(StatusConditionKey)
	if providerConditionGauge == nil {
		return
	}

	if conditionType == "Ready" {
		switch conditionStatus {
		case "False":
			providerConditionGauge.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": "Ready",
					"status":    "True",
				})).Set(0)
		case "True":
			providerConditionGauge.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": "Ready",
					"status":    "False",
				})).Set(0)
		case "Unknown":
			break
		}
	}

	providerConditionGauge.With(ctrlmetrics.RefineLabels(conditionLabels,
		map[string]string{
			"condition": conditionType,
			"status":    conditionStatus,
		})).Set(1)
}

// RecordReconcileDuration updates the legacy Provider reconcile duration metric for a v2 ProviderStore.
func RecordReconcileDuration(name, namespace string, labels map[string]string, seconds float64) {
	providerInfo := map[string]string{
		"name":      name,
		"namespace": namespace,
	}
	maps.Copy(providerInfo, labels)
	providerReconcileDurationGauge := GetGaugeVec(ProviderReconcileDurationKey)
	if providerReconcileDurationGauge == nil {
		return
	}
	providerReconcileDurationGauge.With(ctrlmetrics.RefineNonConditionMetricLabels(providerInfo)).Set(seconds)
}
