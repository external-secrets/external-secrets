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

package grpc

import (
	"context"
	"fmt"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// CheckHealth verifies that a runtime is reachable and reports SERVING via gRPC health checks.
func CheckHealth(ctx context.Context, address string, tlsConfig *TLSConfig) error {
	if tlsConfig == nil {
		return fmt.Errorf("tls config is required for health checks")
	}

	grpcTLSConfig, err := tlsConfig.ToGRPCTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to create TLS config: %w", err)
	}

	conn, err := gogrpc.DialContext(ctx, address, gogrpc.WithTransportCredentials(credentials.NewTLS(grpcTLSConfig)))
	if err != nil {
		return fmt.Errorf("failed to dial runtime: %w", err)
	}
	defer func() { _ = conn.Close() }()

	resp, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("provider runtime is not serving: %s", resp.GetStatus().String())
	}

	return nil
}
