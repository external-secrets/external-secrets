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

package ssmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	commonmetrics "github.com/external-secrets/external-secrets/pkg/controllers/secretstore/metrics"
)

const (
	SecretStoreSubsystem            = "secretstore"
	SecretStoreReconcileDurationKey = "reconcile_duration"
)

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// SetUpMetrics is called at the root to set-up the metric logic using the
// config flags provided.
func SetUpMetrics() {
	secretStoreReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: SecretStoreSubsystem,
		Name:      SecretStoreReconcileDurationKey,
		Help:      "The duration time to reconcile the Secret Store",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	secretStoreCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: SecretStoreSubsystem,
		Name:      commonmetrics.StatusConditionKey,
		Help:      "The status condition of a specific Secret Store",
	}, ctrlmetrics.ConditionMetricLabelNames)

	metrics.Registry.MustRegister(secretStoreReconcileDuration, secretStoreCondition)

	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		SecretStoreReconcileDurationKey:  secretStoreReconcileDuration,
		commonmetrics.StatusConditionKey: secretStoreCondition,
	}
}

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
