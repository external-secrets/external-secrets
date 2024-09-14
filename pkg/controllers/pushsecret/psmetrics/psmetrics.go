//Copyright External Secrets Inc. All Rights Reserved

package psmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

const (
	PushSecretSubsystem            = "pushsecret"
	PushSecretReconcileDurationKey = "reconcile_duration"
)

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// SetUpMetrics is called at the root to set-up the metric logic using the
// config flags provided.
func SetUpMetrics() {
	pushSecretReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: PushSecretSubsystem,
		Name:      PushSecretReconcileDurationKey,
		Help:      "The duration time to reconcile the Push Secret",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	metrics.Registry.MustRegister(pushSecretReconcileDuration)

	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		PushSecretReconcileDurationKey: pushSecretReconcileDuration,
	}
}

func GetGaugeVec(key string) *prometheus.GaugeVec {
	return gaugeVecMetrics[key]
}
