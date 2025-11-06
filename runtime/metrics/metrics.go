/*
Copyright © 2025 ESO Maintainer Team

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

// Package metrics provides functionality for collecting and managing metrics in the external-secrets system.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/external-secrets/external-secrets/runtime/constants"
)

const (
	// ExternalSecretSubsystem is the subsystem name used for external secret metrics.
	ExternalSecretSubsystem = "externalsecret"

	providerAPICalls = "provider_api_calls_count"
)

var (
	syncCallsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      providerAPICalls,
		Help:      "Number of API calls towards the secret provider",
	}, []string{"provider", "call", "status"})
)

// ObserveAPICall records metrics for an API call to a provider.
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
