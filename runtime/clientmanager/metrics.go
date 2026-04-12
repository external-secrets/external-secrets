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

package clientmanager

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ClientManager gauges.
	clientsCachedTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "clientmanager_clients_cached_total",
			Help: "Total number of cached provider clients",
		},
		[]string{"provider_type"},
	)

	// ClientManager counters.
	cacheHitsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "clientmanager_cache_hits_total",
			Help: "Total number of client cache hits",
		},
		[]string{"provider_type"},
	)

	cacheInvalidationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "clientmanager_cache_invalidations_total",
			Help: "Total number of client cache invalidations",
		},
		[]string{"provider_type", "reason"},
	)
)

// Metrics provides test hooks for client manager metrics.
type Metrics interface {
	RecordCacheHit(providerType string)
	RecordCacheMiss(providerType string)
	RecordCacheInvalidation(providerType, reason string)
	UpdateCachedClients(providerType string, count int)
}

// defaultMetrics implements Metrics using Prometheus.
type defaultMetrics struct{}

// RecordCacheHit records a cache hit.
func (m *defaultMetrics) RecordCacheHit(providerType string) {
	cacheHitsTotal.WithLabelValues(providerType).Inc()
}

// RecordCacheMiss records a cache miss.
func (m *defaultMetrics) RecordCacheMiss(_ string) {
	// Cache misses are implicit - we don't track them separately
	// The absence of a hit implies a miss
}

// RecordCacheInvalidation records a cache invalidation.
func (m *defaultMetrics) RecordCacheInvalidation(providerType, reason string) {
	cacheInvalidationsTotal.WithLabelValues(providerType, reason).Inc()
}

// UpdateCachedClients updates the total cached clients gauge.
func (m *defaultMetrics) UpdateCachedClients(providerType string, count int) {
	clientsCachedTotal.WithLabelValues(providerType).Set(float64(count))
}

// Global instance.
var clientManagerMetrics Metrics = &defaultMetrics{}

// RegisterMetrics registers all client manager metrics with the controller-runtime metrics registry.
func RegisterMetrics() error {
	collectors := []prometheus.Collector{
		clientsCachedTotal,
		cacheHitsTotal,
		cacheInvalidationsTotal,
	}

	for _, collector := range collectors {
		if err := metrics.Registry.Register(collector); err != nil {
			var alreadyRegistered prometheus.AlreadyRegisteredError
			if errors.As(err, &alreadyRegistered) {
				continue
			}
			return fmt.Errorf("failed to register clientmanager metric: %w", err)
		}
	}

	// Initialize metrics with zero values so they appear in /metrics output
	// This ensures metrics are visible even before any cache operations occur
	for _, providerType := range []string{providerMetricsLabel, clusterProviderMetricsLabel} {
		clientsCachedTotal.WithLabelValues(providerType).Set(0)
		cacheHitsTotal.WithLabelValues(providerType).Add(0)
		cacheInvalidationsTotal.WithLabelValues(providerType, cacheInvalidationGeneration).Add(0)
		cacheInvalidationsTotal.WithLabelValues(providerType, cacheInvalidationMismatch).Add(0)
	}

	return nil
}

// GetClientManagerMetrics returns the client manager metrics instance for tests.
func GetClientManagerMetrics() Metrics {
	return clientManagerMetrics
}
