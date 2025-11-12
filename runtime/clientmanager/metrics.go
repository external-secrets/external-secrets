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

package clientmanager

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ClientManager Gauges
	clientsCachedTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "clientmanager_clients_cached_total",
			Help: "Total number of cached provider clients",
		},
		[]string{"provider_type"},
	)

	// ClientManager Counters
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

// ClientManagerMetrics interface for testability
type ClientManagerMetrics interface {
	RecordCacheHit(providerType string)
	RecordCacheMiss(providerType string)
	RecordCacheInvalidation(providerType string, reason string)
	UpdateCachedClients(providerType string, count int)
}

// defaultClientManagerMetrics implements ClientManagerMetrics using Prometheus
type defaultClientManagerMetrics struct{}

// RecordCacheHit records a cache hit
func (m *defaultClientManagerMetrics) RecordCacheHit(providerType string) {
	cacheHitsTotal.WithLabelValues(providerType).Inc()
}

// RecordCacheMiss records a cache miss
func (m *defaultClientManagerMetrics) RecordCacheMiss(providerType string) {
	// Cache misses are implicit - we don't track them separately
	// The absence of a hit implies a miss
}

// RecordCacheInvalidation records a cache invalidation
func (m *defaultClientManagerMetrics) RecordCacheInvalidation(providerType string, reason string) {
	cacheInvalidationsTotal.WithLabelValues(providerType, reason).Inc()
}

// UpdateCachedClients updates the total cached clients gauge
func (m *defaultClientManagerMetrics) UpdateCachedClients(providerType string, count int) {
	clientsCachedTotal.WithLabelValues(providerType).Set(float64(count))
}

// Global instance
var clientManagerMetrics ClientManagerMetrics = &defaultClientManagerMetrics{}

// RegisterMetrics registers all client manager metrics with the controller-runtime metrics registry
func RegisterMetrics() error {
	collectors := []prometheus.Collector{
		clientsCachedTotal,
		cacheHitsTotal,
		cacheInvalidationsTotal,
	}

	for _, collector := range collectors {
		if err := metrics.Registry.Register(collector); err != nil {
			// Check if already registered
			if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
				continue
			}
			return fmt.Errorf("failed to register clientmanager metric: %w", err)
		}
	}

	// Initialize metrics with zero values so they appear in /metrics output
	// This ensures metrics are visible even before any cache operations occur
	for _, providerType := range []string{"provider", "cluster-provider"} {
		clientsCachedTotal.WithLabelValues(providerType).Set(0)
		cacheHitsTotal.WithLabelValues(providerType).Add(0)
		cacheInvalidationsTotal.WithLabelValues(providerType, "generation_change").Add(0)
		cacheInvalidationsTotal.WithLabelValues(providerType, "store_mismatch").Add(0)
	}

	return nil
}

// GetClientManagerMetrics returns the client manager metrics instance (for testing)
func GetClientManagerMetrics() ClientManagerMetrics {
	return clientManagerMetrics
}

