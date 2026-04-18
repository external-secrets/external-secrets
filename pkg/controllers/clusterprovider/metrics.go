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

// Package clusterprovider exposes compatibility metrics for v2 ClusterProviderStore resources.
package clusterprovider

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	// ClusterProviderSubsystem is the subsystem name for ClusterProvider metrics.
	ClusterProviderSubsystem = "clusterprovider"

	// ClusterProviderReconcileDurationKey is the key for the reconcile duration metric.
	ClusterProviderReconcileDurationKey = "reconcile_duration"

	// StatusConditionKey is the key for the status condition metric.
	StatusConditionKey = "status_condition"
)

var (
	gaugeVecMetrics     = map[string]*prometheus.GaugeVec{}
	registerMetricsOnce sync.Once
)

// SetUpMetrics initializes the metrics for the ClusterProvider controller.
func SetUpMetrics() {
	registerMetricsOnce.Do(func() {
		clusterProviderReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Subsystem: ClusterProviderSubsystem,
			Name:      ClusterProviderReconcileDurationKey,
			Help:      "The duration time to reconcile the ClusterProvider",
		}, []string{"name"})

		clusterProviderCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Subsystem: ClusterProviderSubsystem,
			Name:      StatusConditionKey,
			Help:      "The status condition of a specific ClusterProvider",
		}, []string{"name", "condition", "status"})

		metrics.Registry.MustRegister(clusterProviderReconcileDuration, clusterProviderCondition)

		gaugeVecMetrics = map[string]*prometheus.GaugeVec{
			ClusterProviderReconcileDurationKey: clusterProviderReconcileDuration,
			StatusConditionKey:                  clusterProviderCondition,
		}
	})
}

// GetGaugeVec returns the GaugeVec for the given key.
func GetGaugeVec(key string) *prometheus.GaugeVec {
	return gaugeVecMetrics[key]
}

// RemoveMetrics deletes all metrics published by the resource.
func RemoveMetrics(name string) {
	for _, gaugeVecMetric := range gaugeVecMetrics {
		gaugeVecMetric.DeletePartialMatch(
			map[string]string{
				"name": name,
			},
		)
	}
}

// UpdateStatusCondition updates the legacy ClusterProvider condition metrics for a v2 ClusterProviderStore.
func UpdateStatusCondition(name, conditionType, conditionStatus string) {
	clusterProviderConditionGauge := GetGaugeVec(StatusConditionKey)
	if clusterProviderConditionGauge == nil {
		return
	}

	if conditionType == "Ready" {
		switch conditionStatus {
		case "False":
			clusterProviderConditionGauge.WithLabelValues(
				name,
				"Ready",
				"True",
			).Set(0)
		case "True":
			clusterProviderConditionGauge.WithLabelValues(
				name,
				"Ready",
				"False",
			).Set(0)
		case "Unknown":
			break
		}
	}

	clusterProviderConditionGauge.WithLabelValues(
		name,
		conditionType,
		conditionStatus,
	).Set(1)
}

// RecordReconcileDuration updates the legacy ClusterProvider reconcile duration metric for a v2 ClusterProviderStore.
func RecordReconcileDuration(name string, seconds float64) {
	clusterProviderReconcileDurationGauge := GetGaugeVec(ClusterProviderReconcileDurationKey)
	if clusterProviderReconcileDurationGauge == nil {
		return
	}
	clusterProviderReconcileDurationGauge.WithLabelValues(name).Set(seconds)
}
