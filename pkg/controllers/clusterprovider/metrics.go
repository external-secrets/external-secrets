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

package clusterprovider

import (
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	// ClusterProviderSubsystem is the subsystem name for ClusterProvider metrics.
	ClusterProviderSubsystem = "clusterprovider"

	// ClusterProviderReconcileDurationKey is the key for the reconcile duration metric.
	ClusterProviderReconcileDurationKey = "reconcile_duration"

	// StatusConditionKey is the key for the status condition metric.
	StatusConditionKey = "status_condition"
)

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// SetUpMetrics initializes the metrics for the ClusterProvider controller.
func SetUpMetrics() {
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

// UpdateStatusCondition updates the condition metrics for a ClusterProvider.
func UpdateStatusCondition(clusterProvider *esapi.ClusterProvider, condition esapi.ProviderCondition) {
	clusterProviderConditionGauge := GetGaugeVec(StatusConditionKey)

	if condition.Type == esapi.ProviderReady {
		switch condition.Status {
		case metav1.ConditionFalse:
			clusterProviderConditionGauge.WithLabelValues(
				clusterProvider.GetName(),
				string(esapi.ProviderReady),
				string(metav1.ConditionTrue),
			).Set(0)
		case metav1.ConditionTrue:
			clusterProviderConditionGauge.WithLabelValues(
				clusterProvider.GetName(),
				string(esapi.ProviderReady),
				string(metav1.ConditionFalse),
			).Set(0)
		case metav1.ConditionUnknown:
			break
		}
	}

	clusterProviderConditionGauge.WithLabelValues(
		clusterProvider.GetName(),
		string(condition.Type),
		string(condition.Status),
	).Set(1)
}

