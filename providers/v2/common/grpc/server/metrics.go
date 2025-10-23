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

package server

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	// gRPC latency buckets optimized for typical RPC call durations
	grpcLatencyBuckets = []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30}

	// gRPC Server Counters
	serverRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total number of gRPC server requests",
		},
		[]string{"method", "status"},
	)

	// gRPC Server Histograms
	serverRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_request_duration_seconds",
			Help:    "Duration of gRPC server requests in seconds",
			Buckets: grpcLatencyBuckets,
		},
		[]string{"method"},
	)

	// gRPC Server Gauges
	serverActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "grpc_server_active_connections",
			Help: "Number of active gRPC server connections",
		},
	)
)

// ServerMetrics interface for testability
type ServerMetrics interface {
	RecordRequest(method string, err error, duration time.Duration)
	IncrementActiveConnections()
	DecrementActiveConnections()
}

// defaultServerMetrics implements ServerMetrics using Prometheus
type defaultServerMetrics struct{}

// RecordRequest records metrics for a server request
func (m *defaultServerMetrics) RecordRequest(method string, err error, duration time.Duration) {
	status := "success"
	if err != nil {
		status = "error"
	}

	serverRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
	serverRequestsTotal.WithLabelValues(method, status).Inc()
}

// IncrementActiveConnections increments the active connections gauge
func (m *defaultServerMetrics) IncrementActiveConnections() {
	serverActiveConnections.Inc()
}

// DecrementActiveConnections decrements the active connections gauge
func (m *defaultServerMetrics) DecrementActiveConnections() {
	serverActiveConnections.Dec()
}

// Global instance
var serverMetrics ServerMetrics = &defaultServerMetrics{}

// MetricsUnaryInterceptor returns a gRPC unary server interceptor that records metrics
func MetricsUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start)
		serverMetrics.RecordRequest(info.FullMethod, err, duration)

		return resp, err
	}
}

// ConnectionCountingStreamServerInterceptor returns a gRPC stream server interceptor that tracks active connections
func ConnectionCountingStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		serverMetrics.IncrementActiveConnections()
		defer serverMetrics.DecrementActiveConnections()

		return handler(srv, ss)
	}
}

// RegisterMetrics registers all server metrics with Prometheus
func RegisterMetrics(registry prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		serverRequestsTotal,
		serverRequestDuration,
		serverActiveConnections,
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			// Check if already registered
			if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
				continue
			}
			return fmt.Errorf("failed to register server metric: %w", err)
		}
	}

	return nil
}

// GetServerMetrics returns the server metrics instance (for testing)
func GetServerMetrics() ServerMetrics {
	return serverMetrics
}

// getStatusCode extracts the gRPC status code from an error
func getStatusCode(err error) string {
	if err == nil {
		return "OK"
	}
	st, ok := status.FromError(err)
	if !ok {
		return "Unknown"
	}
	return st.Code().String()
}

