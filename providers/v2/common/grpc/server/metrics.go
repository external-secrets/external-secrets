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

package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

var (
	// gRPC latency buckets optimized for typical RPC call durations.
	grpcLatencyBuckets = []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30}

	// gRPC server counters.
	serverRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total number of gRPC server requests",
		},
		[]string{"method", "status"},
	)

	// gRPC server histograms.
	serverRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_request_duration_seconds",
			Help:    "Duration of gRPC server requests in seconds",
			Buckets: grpcLatencyBuckets,
		},
		[]string{"method"},
	)

	// gRPC server gauges.
	serverActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "grpc_server_active_connections",
			Help: "Number of active gRPC server connections",
		},
	)
)

// Metrics provides testable hooks for server metrics.
type Metrics interface {
	RecordRequest(method string, err error, duration time.Duration)
	IncrementActiveConnections()
	DecrementActiveConnections()
}

// defaultMetrics implements Metrics using Prometheus.
type defaultMetrics struct{}

// RecordRequest records metrics for a server request.
func (m *defaultMetrics) RecordRequest(method string, err error, duration time.Duration) {
	status := "success"
	if err != nil {
		status = "error"
	}

	serverRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
	serverRequestsTotal.WithLabelValues(method, status).Inc()
}

// IncrementActiveConnections increments the active connections gauge.
func (m *defaultMetrics) IncrementActiveConnections() {
	serverActiveConnections.Inc()
}

// DecrementActiveConnections decrements the active connections gauge.
func (m *defaultMetrics) DecrementActiveConnections() {
	serverActiveConnections.Dec()
}

var serverMetrics Metrics = &defaultMetrics{}

// MetricsUnaryInterceptor returns a gRPC unary server interceptor that records metrics.
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

// ConnectionCountingStreamServerInterceptor returns a gRPC stream server interceptor that tracks active connections.
func ConnectionCountingStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		serverMetrics.IncrementActiveConnections()
		defer serverMetrics.DecrementActiveConnections()

		return handler(srv, ss)
	}
}

// RegisterMetrics registers all server metrics with Prometheus.
func RegisterMetrics(registry prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		serverRequestsTotal,
		serverRequestDuration,
		serverActiveConnections,
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			var alreadyRegistered prometheus.AlreadyRegisteredError
			if errors.As(err, &alreadyRegistered) {
				continue
			}
			return fmt.Errorf("failed to register server metric: %w", err)
		}
	}

	return nil
}

// GetMetrics returns the server metrics instance.
func GetMetrics() Metrics {
	return serverMetrics
}
