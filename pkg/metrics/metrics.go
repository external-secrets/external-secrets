//Copyright External Secrets Inc. All Rights Reserved

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/external-secrets/external-secrets/pkg/constants"
)

const (
	ExternalSecretSubsystem = "externalsecret"
	providerAPICalls        = "provider_api_calls_count"
)

var (
	syncCallsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      providerAPICalls,
		Help:      "Number of API calls towards the secret provider",
	}, []string{"provider", "call", "status"})
)

func ObserveAPICall(provider, call string, err error) {
	syncCallsTotal.WithLabelValues(provider, call, deriveStatus(err)).Inc()
}

func deriveStatus(err error) string {
	if err != nil {
		return constants.StatusError
	}
	return constants.StatusSuccess
}

func init() {
	metrics.Registry.MustRegister(syncCallsTotal)
}
