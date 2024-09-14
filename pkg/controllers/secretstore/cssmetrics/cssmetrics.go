//Copyright External Secrets Inc. All Rights Reserved

package cssmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	commonmetrics "github.com/external-secrets/external-secrets/pkg/controllers/secretstore/metrics"
)

const (
	ClusterSecretStoreSubsystem            = "clustersecretstore"
	ClusterSecretStoreReconcileDurationKey = "reconcile_duration"
)

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// SetUpMetrics is called at the root to set-up the metric logic using the
// config flags provided.
func SetUpMetrics() {
	clusterSecretStoreReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ClusterSecretStoreSubsystem,
		Name:      ClusterSecretStoreReconcileDurationKey,
		Help:      "The duration time to reconcile the Cluster Secret Store",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	clusterSecretStoreCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ClusterSecretStoreSubsystem,
		Name:      commonmetrics.StatusConditionKey,
		Help:      "The status condition of a specific Cluster Secret Store",
	}, ctrlmetrics.ConditionMetricLabelNames)

	metrics.Registry.MustRegister(clusterSecretStoreReconcileDuration, clusterSecretStoreCondition)

	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		ClusterSecretStoreReconcileDurationKey: clusterSecretStoreReconcileDuration,
		commonmetrics.StatusConditionKey:       clusterSecretStoreCondition,
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
