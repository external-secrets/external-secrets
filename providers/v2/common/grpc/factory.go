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
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	ctrl "sigs.k8s.io/controller-runtime"

	pb "github.com/external-secrets/external-secrets/proto/provider"
	v2 "github.com/external-secrets/external-secrets/providers/v2/common"
)

// NewClient creates a new gRPC client that connects to the provider at the given address.
// If tlsConfig is nil, an insecure connection is used (not recommended for production).
// If log is nil, a default logger will be used.
func NewClient(address string, tlsConfig *TLSConfig) (v2.Provider, error) {
	return NewClientWithLogger(address, tlsConfig, ctrl.Log.WithName("grpc-client"))
}

// NewClientWithLogger creates a new gRPC client with a custom logger.
func NewClientWithLogger(address string, tlsConfig *TLSConfig, log logr.Logger) (v2.Provider, error) {
	if address == "" {
		return nil, fmt.Errorf("provider address cannot be empty")
	}

	log.Info("creating gRPC client",
		"address", address,
		"tlsEnabled", tlsConfig != nil)

	// Set up connection options
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second, // Send keepalive pings every 10 seconds
			Timeout:             5 * time.Second,  // Wait 5 seconds for ping ack
			PermitWithoutStream: true,             // Allow pings when no streams are active
		}),
	}

	log.V(1).Info("configured keepalive parameters",
		"time", "10s",
		"timeout", "5s",
		"permitWithoutStream", true)

	// Configure TLS or insecure credentials
	if tlsConfig == nil {
		return nil, fmt.Errorf("tlsConfig cannot be nil; insecure connections are not allowed in production")
	}

	log.V(1).Info("configuring TLS",
		"serverName", tlsConfig.ServerName,
		"hasCACert", len(tlsConfig.CACert) > 0,
		"hasClientCert", len(tlsConfig.ClientCert) > 0,
		"hasClientKey", len(tlsConfig.ClientKey) > 0)

	grpcTLSConfig, err := tlsConfig.ToGRPCTLSConfig()
	if err != nil {
		log.Error(err, "failed to create TLS config")
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(grpcTLSConfig)))
	log.Info("TLS configured successfully")

	// Dial the provider
	log.Info("dialing provider", "address", address)
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		log.Error(err, "failed to dial provider", "address", address)
		return nil, fmt.Errorf("failed to dial provider at %s: %w", address, err)
	}

	log.Info("gRPC connection established",
		"address", address,
		"target", conn.Target(),
		"state", conn.GetState().String())

	// Create the gRPC client stub
	grpcClient := pb.NewSecretStoreProviderClient(conn)

	return &grpcProviderClient{
		conn:   conn,
		client: grpcClient,
		log:    log.WithValues("target", conn.Target()),
	}, nil
}

// NewClientWithConn creates a client from an existing gRPC connection.
// This is useful for testing or when you need more control over connection setup.
func NewClientWithConn(conn *grpc.ClientConn) v2.Provider {
	return &grpcProviderClient{
		conn:   conn,
		client: pb.NewSecretStoreProviderClient(conn),
		log:    ctrl.Log.WithName("grpc-client").WithValues("target", conn.Target()),
	}
}

// NewResilientProviderClient creates a production-ready provider client with
// connection pooling, retry logic, and circuit breaking.
// This is the recommended way to create provider clients for production use.
func NewResilientProviderClient(address string, tlsConfig *TLSConfig) (v2.Provider, error) {
	config := DefaultResilientClientConfig(address, tlsConfig)
	return NewResilientClient(config)
}

// NewResilientProviderClientWithConfig creates a resilient provider client with custom configuration.
func NewResilientProviderClientWithConfig(config ResilientClientConfig) (v2.Provider, error) {
	return NewResilientClient(config)
}
