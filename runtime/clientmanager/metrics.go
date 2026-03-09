// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// Client manager gauges.
	clientsCachedTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "clientmanager_clients_cached_total",
			Help: "Total number of cached provider clients",
		},
		[]string{"provider_type"},
	)

	// Client manager counters.
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

const (
	providerTypeProvider        = "provider"
	providerTypeClusterProvider = "cluster-provider"
)

// Metrics exposes client manager metrics operations.
type Metrics interface {
	RecordCacheHit(providerType string)
	RecordCacheMiss()
	RecordCacheInvalidation(providerType, reason string)
	UpdateCachedClients(providerType string, count int)
}

// defaultClientManagerMetrics implements Metrics using Prometheus.
type defaultClientManagerMetrics struct{}

// RecordCacheHit records a cache hit.
func (m *defaultClientManagerMetrics) RecordCacheHit(providerType string) {
	cacheHitsTotal.WithLabelValues(providerType).Inc()
}

// RecordCacheMiss records a cache miss.
func (m *defaultClientManagerMetrics) RecordCacheMiss() {
	// Cache misses are implicit - we don't track them separately
	// The absence of a hit implies a miss
}

// RecordCacheInvalidation records a cache invalidation.
func (m *defaultClientManagerMetrics) RecordCacheInvalidation(providerType, reason string) {
	cacheInvalidationsTotal.WithLabelValues(providerType, reason).Inc()
}

// UpdateCachedClients updates the total cached clients gauge.
func (m *defaultClientManagerMetrics) UpdateCachedClients(providerType string, count int) {
	clientsCachedTotal.WithLabelValues(providerType).Set(float64(count))
}

// Global instance.
var clientManagerMetrics Metrics = &defaultClientManagerMetrics{}

// RegisterMetrics registers all client manager metrics with the controller-runtime metrics registry.
func RegisterMetrics() error {
	collectors := []prometheus.Collector{
		clientsCachedTotal,
		cacheHitsTotal,
		cacheInvalidationsTotal,
	}

	for _, collector := range collectors {
		if err := metrics.Registry.Register(collector); err != nil {
			var alreadyRegisteredErr prometheus.AlreadyRegisteredError
			if errors.As(err, &alreadyRegisteredErr) {
				continue
			}
			return fmt.Errorf("failed to register clientmanager metric: %w", err)
		}
	}

	// Initialize metrics with zero values so they appear in /metrics output
	// This ensures metrics are visible even before any cache operations occur
	for _, providerType := range []string{providerTypeProvider, providerTypeClusterProvider} {
		clientsCachedTotal.WithLabelValues(providerType).Set(0)
		cacheHitsTotal.WithLabelValues(providerType).Add(0)
		cacheInvalidationsTotal.WithLabelValues(providerType, "generation_change").Add(0)
		cacheInvalidationsTotal.WithLabelValues(providerType, "store_mismatch").Add(0)
	}

	return nil
}

// GetClientManagerMetrics returns the client manager metrics instance for testing.
func GetClientManagerMetrics() Metrics {
	return clientManagerMetrics
}
