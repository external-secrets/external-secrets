//Copyright External Secrets Inc. All Rights Reserved

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
