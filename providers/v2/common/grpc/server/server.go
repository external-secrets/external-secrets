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
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// ServerOptions holds configuration options for creating a gRPC server.
type ServerOptions struct {
	EnableTLS bool
	Verbose   bool
}

// NewGRPCServer creates a new gRPC server with standard configuration.
// It includes:
// - TLS/mTLS if enabled
// - Keepalive parameters
// - Connection tap handler (if verbose)
// - RPC logging interceptor
func NewGRPCServer(opts ServerOptions) (*grpc.Server, error) {
	var grpcOpts []grpc.ServerOption

	// Add keepalive parameters for better connection diagnostics
	grpcOpts = append(grpcOpts,
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Minute,
			Timeout:               1 * time.Minute,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second, // Allow pings every 5s (client sends every 10s)
			PermitWithoutStream: true,
		}),
	)

	if opts.Verbose {
		log.Printf("[CONFIG] Keepalive configured: idle=15m, age=30m, time=5m, timeout=1m, minPing=5s")
	}

	// Add connection tap handler for verbose mode
	if opts.Verbose {
		grpcOpts = append(grpcOpts, grpc.InTapHandle(ConnectionTapHandler))
		log.Printf("[CONFIG] Connection tap handler enabled")
	}

	// Add RPC interceptors: metrics and logging
	grpcOpts = append(grpcOpts, grpc.ChainUnaryInterceptor(
		MetricsUnaryInterceptor(),
		LoggingUnaryInterceptor(opts.Verbose),
	))
	log.Printf("[CONFIG] RPC metrics and logging interceptors enabled")

	// Configure TLS if enabled
	if opts.EnableTLS {
		log.Printf("[TLS] Loading TLS configuration...")
		tlsConfig, err := LoadTLSConfig(DefaultTLSConfig())
		if err != nil {
			return nil, err
		}

		// Log TLS configuration details
		log.Printf("[TLS] Configuration loaded successfully")
		log.Printf("[TLS]   Min TLS version: 0x%04x (%s)", tlsConfig.MinVersion, TLSVersionName(tlsConfig.MinVersion))
		log.Printf("[TLS]   Max TLS version: 0x%04x (%s)", tlsConfig.MaxVersion, TLSVersionName(tlsConfig.MaxVersion))
		log.Printf("[TLS]   Client auth required: %v", tlsConfig.ClientAuth == 4) // tls.RequireAndVerifyClientCert

		if tlsConfig.ClientCAs != nil {
			subjects := tlsConfig.ClientCAs.Subjects()
			log.Printf("[TLS]   Client CA pool has %d certificate(s)", len(subjects))
			if opts.Verbose {
				for i, subject := range subjects {
					log.Printf("[TLS]     CA %d: %s", i, string(subject))
				}
			}
		} else {
			log.Printf("[TLS]   WARNING: No client CA pool configured")
		}

		if len(tlsConfig.Certificates) > 0 {
			log.Printf("[TLS]   Server has %d certificate(s)", len(tlsConfig.Certificates))
			for i, cert := range tlsConfig.Certificates {
				if len(cert.Certificate) > 0 {
					log.Printf("[TLS]     Cert %d: raw certificate data present", i)
				}
			}
		} else {
			log.Printf("[TLS]   WARNING: No server certificates configured")
		}

		grpcOpts = append(grpcOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
		log.Printf("[TLS] mTLS enabled for provider server")
	} else {
		log.Printf("[SECURITY] WARNING: TLS DISABLED - NOT SUITABLE FOR PRODUCTION")
		log.Printf("[SECURITY] All traffic will be transmitted in PLAINTEXT")
	}

	return grpc.NewServer(grpcOpts...), nil
}
