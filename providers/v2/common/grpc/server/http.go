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
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// DefaultMetricsPort is the default port for the HTTP metrics server
	DefaultMetricsPort = 8081
	// DefaultMetricsPath is the default path for the metrics endpoint
	DefaultMetricsPath = "/metrics"
)

var metricsLog = ctrl.Log.WithName("metrics-server")

// MetricsServer serves Prometheus metrics via HTTP
type MetricsServer struct {
	server   *http.Server
	registry *prometheus.Registry
}

// NewMetricsServer creates a new HTTP metrics server
func NewMetricsServer(port int, registry *prometheus.Registry) *MetricsServer {
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	mux := http.NewServeMux()
	mux.Handle(DefaultMetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
	}))

	// Add health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	return &MetricsServer{
		server:   server,
		registry: registry,
	}
}

// Start starts the HTTP metrics server
func (m *MetricsServer) Start(ctx context.Context) error {
	metricsLog.Info("Starting metrics server", "addr", m.server.Addr, "path", DefaultMetricsPath)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		metricsLog.Info("Shutting down metrics server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("metrics server shutdown error: %w", err)
		}
		return nil
	case err := <-errChan:
		return err
	}
}

// GetRegistry returns the Prometheus registry
func (m *MetricsServer) GetRegistry() *prometheus.Registry {
	return m.registry
}

