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
	"crypto/tls"
	"crypto/x509"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestCheckHealthReturnsNilWhenServing(t *testing.T) {
	address, tlsConfig := newHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)

	err := CheckHealth(context.Background(), address, tlsConfig)
	if err != nil {
		t.Fatalf("CheckHealth() error = %v", err)
	}
}

func TestCheckHealthReturnsErrorWhenNotServing(t *testing.T) {
	address, tlsConfig := newHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	err := CheckHealth(context.Background(), address, tlsConfig)
	if err == nil {
		t.Fatal("expected non-serving runtime to fail health check")
	}
}

func newHealthServer(t *testing.T, status grpc_health_v1.HealthCheckResponse_ServingStatus) (string, *TLSConfig) {
	t.Helper()

	serverCert, serverKey, clientCert, clientKey, caCert := newTLSArtifactsForTest(t, testLoopbackAddress)

	caPool := x509.NewCertPool()
	require.True(t, caPool.AppendCertsFromPEM(caCert))

	tlsCert, err := tls.X509KeyPair(serverCert, serverKey)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", testLoopbackAddress+":0")
	require.NoError(t, err)

	healthServer := grpc_health.NewServer()
	healthServer.SetServingStatus("", status)

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{tlsCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})))
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = lis.Close()
	})

	return lis.Addr().String(), &TLSConfig{
		CACert:     caCert,
		ClientCert: clientCert,
		ClientKey:  clientKey,
		ServerName: testLoopbackAddress,
	}
}
