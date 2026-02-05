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

// Package sdk contains Nebius contains logic to create Nebius sdk for interaction with any Nebius service.
package sdk

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"time"

	"github.com/nebius/gosdk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// NewSDK initializes a new gosdk.SDK instance using the provided context, API domain, and CA certificate.
// It sets up TLS configuration, including support for custom CA certificates, and gRPC dial options.
// Returns the initialized SDK instance or an error if the setup fails.
func NewSDK(ctx context.Context, apiDomain string, caCertificate []byte) (*gosdk.SDK, error) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}

	if caCertificate != nil && len(caCertificate) > 0 {
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCertificate) {
			return nil, errors.New("failed to append CA certificate. PEM parse error")
		}
		tlsCfg.RootCAs = certPool
	}

	sdk, err := gosdk.New(
		ctx,
		gosdk.WithDomain(apiDomain),
		gosdk.WithDialOptions(
			grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                time.Second * 30,
				Timeout:             time.Second * 5,
				PermitWithoutStream: false,
			}),
		),
	)

	return sdk, err
}
