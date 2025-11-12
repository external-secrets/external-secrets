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

package grpc

import (
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// gRPC latency buckets optimized for typical RPC call durations
	grpcLatencyBuckets = []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30}

	// Connection Pool Gauges
	poolConnectionsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "grpc_pool_connections_active",
			Help: "Number of active gRPC connections in the pool with references > 0",
		},
		[]string{"address", "tls_enabled"},
	)

	poolConnectionsIdle = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "grpc_pool_connections_idle",
			Help: "Number of idle gRPC connections in the pool with references = 0",
		},
		[]string{"address", "tls_enabled"},
	)

	poolConnectionsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "grpc_pool_connections_total",
			Help: "Total number of gRPC connections in the pool",
		},
		[]string{"address", "tls_enabled"},
	)

	// Connection Pool Histograms
	connectionAge = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_connection_age_seconds",
			Help:    "Age of gRPC connections in seconds",
			Buckets: []float64{60, 300, 600, 900, 1800, 3600}, // 1m, 5m, 10m, 15m, 30m, 1h
		},
		[]string{"address", "tls_enabled"},
	)

	connectionIdle = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_connection_idle_seconds",
			Help:    "Idle time of gRPC connections in seconds",
			Buckets: []float64{30, 60, 120, 300, 600, 900}, // 30s, 1m, 2m, 5m, 10m, 15m
		},
		[]string{"address", "tls_enabled"},
	)

	// Connection Pool Counters
	poolHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_pool_hits_total",
			Help: "Total number of connection pool cache hits (connection reused)",
		},
		[]string{"address", "tls_enabled"},
	)

	poolMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_pool_misses_total",
			Help: "Total number of connection pool cache misses (new connection created)",
		},
		[]string{"address", "tls_enabled"},
	)

	poolEvictions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_pool_evictions_total",
			Help: "Total number of connection pool evictions",
		},
		[]string{"address", "tls_enabled", "eviction_reason"},
	)

	poolConnectionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_pool_connection_errors_total",
			Help: "Total number of failed connection attempts",
		},
		[]string{"address", "tls_enabled"},
	)

	// gRPC Client Metrics
	clientRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_client_request_duration_seconds",
			Help:    "Duration of gRPC client requests in seconds",
			Buckets: grpcLatencyBuckets,
		},
		[]string{"method", "target", "status"},
	)

	clientRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_client_requests_total",
			Help: "Total number of gRPC client requests",
		},
		[]string{"method", "target", "status"},
	)

	clientRequestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_client_request_errors_total",
			Help: "Total number of failed gRPC client requests",
		},
		[]string{"method", "target", "error_type"},
	)
)

// Metrics interface for testability
type PoolMetrics interface {
	RecordHit(address string, tlsEnabled bool)
	RecordMiss(address string, tlsEnabled bool)
	RecordEviction(address string, tlsEnabled bool, reason string)
	RecordConnectionError(address string, tlsEnabled bool)
	UpdatePoolState(address string, tlsEnabled bool, active, idle, total int)
	RecordConnectionAge(address string, tlsEnabled bool, age time.Duration)
	RecordConnectionIdle(address string, tlsEnabled bool, idle time.Duration)
}

// ClientMetrics interface for testability
type ClientMetrics interface {
	ObserveRequest(method, target string, err error, duration time.Duration)
}

// defaultPoolMetrics implements PoolMetrics using Prometheus
type defaultPoolMetrics struct{}

// RecordHit records a connection pool cache hit
func (m *defaultPoolMetrics) RecordHit(address string, tlsEnabled bool) {
	poolHits.WithLabelValues(address, strconv.FormatBool(tlsEnabled)).Inc()
}

// RecordMiss records a connection pool cache miss
func (m *defaultPoolMetrics) RecordMiss(address string, tlsEnabled bool) {
	poolMisses.WithLabelValues(address, strconv.FormatBool(tlsEnabled)).Inc()
}

// RecordEviction records a connection eviction with reason
func (m *defaultPoolMetrics) RecordEviction(address string, tlsEnabled bool, reason string) {
	poolEvictions.WithLabelValues(address, strconv.FormatBool(tlsEnabled), reason).Inc()
}

// RecordConnectionError records a failed connection attempt
func (m *defaultPoolMetrics) RecordConnectionError(address string, tlsEnabled bool) {
	poolConnectionErrors.WithLabelValues(address, strconv.FormatBool(tlsEnabled)).Inc()
}

// UpdatePoolState updates the current pool state gauges
func (m *defaultPoolMetrics) UpdatePoolState(address string, tlsEnabled bool, active, idle, total int) {
	labels := prometheus.Labels{"address": address, "tls_enabled": strconv.FormatBool(tlsEnabled)}
	poolConnectionsActive.With(labels).Set(float64(active))
	poolConnectionsIdle.With(labels).Set(float64(idle))
	poolConnectionsTotal.With(labels).Set(float64(total))
}

// RecordConnectionAge records the age of a connection
func (m *defaultPoolMetrics) RecordConnectionAge(address string, tlsEnabled bool, age time.Duration) {
	connectionAge.WithLabelValues(address, strconv.FormatBool(tlsEnabled)).Observe(age.Seconds())
}

// RecordConnectionIdle records the idle time of a connection
func (m *defaultPoolMetrics) RecordConnectionIdle(address string, tlsEnabled bool, idle time.Duration) {
	connectionIdle.WithLabelValues(address, strconv.FormatBool(tlsEnabled)).Observe(idle.Seconds())
}

// defaultClientMetrics implements ClientMetrics using Prometheus
type defaultClientMetrics struct{}

// ObserveRequest records metrics for a client request
func (m *defaultClientMetrics) ObserveRequest(method, target string, err error, duration time.Duration) {
	status := "success"
	if err != nil {
		status = "error"
		errorType := classifyError(err)
		clientRequestErrors.WithLabelValues(method, target, errorType).Inc()
	}

	clientRequestDuration.WithLabelValues(method, target, status).Observe(duration.Seconds())
	clientRequestsTotal.WithLabelValues(method, target, status).Inc()
}

// classifyError extracts error type for metrics
func classifyError(err error) string {
	if err == nil {
		return "none"
	}
	errStr := err.Error()
	// Classify common error patterns
	switch {
	case contains(errStr, "context deadline exceeded"):
		return "timeout"
	case contains(errStr, "connection refused"):
		return "connection_refused"
	case contains(errStr, "unavailable"):
		return "unavailable"
	case contains(errStr, "forbidden"):
		return "forbidden"
	case contains(errStr, "not found"):
		return "not_found"
	case contains(errStr, "unauthorized"):
		return "unauthorized"
	default:
		return "unknown"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Global instances
var (
	poolMetrics   PoolMetrics   = &defaultPoolMetrics{}
	clientMetrics ClientMetrics = &defaultClientMetrics{}
)

// RegisterMetrics registers all gRPC metrics with Prometheus
func RegisterMetrics(registry prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		poolConnectionsActive,
		poolConnectionsIdle,
		poolConnectionsTotal,
		connectionAge,
		connectionIdle,
		poolHits,
		poolMisses,
		poolEvictions,
		poolConnectionErrors,
		clientRequestDuration,
		clientRequestsTotal,
		clientRequestErrors,
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			// Check if already registered
			if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
				continue
			}
			return fmt.Errorf("failed to register metric: %w", err)
		}
	}

	return nil
}

// GetPoolMetrics returns the pool metrics instance (for testing)
func GetPoolMetrics() PoolMetrics {
	return poolMetrics
}

// GetClientMetrics returns the client metrics instance (for testing)
func GetClientMetrics() ClientMetrics {
	return clientMetrics
}

