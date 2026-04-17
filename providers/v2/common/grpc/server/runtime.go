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

package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// RuntimeOptions configures a provider gRPC server instance.
type RuntimeOptions struct {
	ProviderName string
	Port         int
	EnableTLS    bool
	Verbose      bool
	Register     func(grpc.ServiceRegistrar)
}

// RunProviderServer starts the provider gRPC and metrics servers with standard wiring.
func RunProviderServer(opts RuntimeOptions) error {
	grpcServer, err := NewGRPCServer(Options{
		EnableTLS: opts.EnableTLS,
		Verbose:   opts.Verbose,
	})
	if err != nil {
		return fmt.Errorf("create gRPC server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsServer := NewMetricsServer(DefaultMetricsPort, nil)
	if err := RegisterMetrics(metricsServer.GetRegistry()); err != nil {
		return fmt.Errorf("register provider metrics: %w", err)
	}
	go func() {
		if err := metricsServer.Start(ctx); err != nil {
			log.Fatalf("Failed to start provider metrics server: %v", err)
		}
	}()

	if opts.Register != nil {
		opts.Register(grpcServer)
	}

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", opts.Port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		log.Printf("Received signal: %v, shutting down gracefully...", sig)
		cancel()
		grpcServer.GracefulStop()
	}()

	log.Printf("%s Provider listening on %s", opts.ProviderName, lis.Addr().String())
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}
