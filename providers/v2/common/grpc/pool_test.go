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
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/external-secrets/external-secrets/proto/provider"
)

func TestConnectionPoolGetReleaseReuse(t *testing.T) {
	address, tlsConfig := newPoolTestServer(t)

	pool := NewConnectionPool(PoolConfig{
		MaxIdleTime:         time.Minute,
		MaxLifetime:         time.Minute,
		HealthCheckInterval: time.Hour,
	})
	defer func() {
		_ = pool.Close()
	}()

	client1, err := pool.Get(context.Background(), address, tlsConfig)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	client2, err := pool.Get(context.Background(), address, tlsConfig)
	if err != nil {
		t.Fatalf("second Get() error = %v", err)
	}

	if client1 != client2 {
		t.Fatal("expected pooled client to be reused")
	}

	key := pool.connectionKey(address, tlsConfig)
	pooled := pool.connections[key]
	if pooled == nil {
		t.Fatalf("expected pooled connection for key %q", key)
	}
	if pooled.references != 2 {
		t.Fatalf("expected references=2, got %d", pooled.references)
	}

	pool.Release(address, tlsConfig)
	if pooled.references != 1 {
		t.Fatalf("expected references=1 after release, got %d", pooled.references)
	}
	pool.Release(address, tlsConfig)
	if pooled.references != 0 {
		t.Fatalf("expected references=0 after second release, got %d", pooled.references)
	}
}

func TestConnectionPoolGetReplacesExpiredConnection(t *testing.T) {
	address, tlsConfig := newPoolTestServer(t)

	pool := NewConnectionPool(PoolConfig{
		MaxIdleTime:         time.Minute,
		MaxLifetime:         time.Minute,
		HealthCheckInterval: time.Hour,
	})
	defer func() {
		_ = pool.Close()
	}()

	client1, err := pool.Get(context.Background(), address, tlsConfig)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	pool.Release(address, tlsConfig)

	key := pool.connectionKey(address, tlsConfig)
	pooled := pool.connections[key]
	if pooled == nil {
		t.Fatalf("expected pooled connection for key %q", key)
	}
	pooled.mu.Lock()
	pooled.created = time.Now().Add(-2 * time.Hour)
	pooled.mu.Unlock()

	client2, err := pool.Get(context.Background(), address, tlsConfig)
	if err != nil {
		t.Fatalf("second Get() error = %v", err)
	}

	if client1 == client2 {
		t.Fatal("expected expired pooled client to be replaced")
	}
}

func TestConnectionPoolCleanupIdleConnectionsRemovesReleasedConnection(t *testing.T) {
	address, tlsConfig := newPoolTestServer(t)

	pool := NewConnectionPool(PoolConfig{
		MaxIdleTime:         time.Second,
		MaxLifetime:         time.Minute,
		HealthCheckInterval: time.Hour,
	})
	defer func() {
		_ = pool.Close()
	}()

	_, err := pool.Get(context.Background(), address, tlsConfig)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	pool.Release(address, tlsConfig)

	key := pool.connectionKey(address, tlsConfig)
	pooled := pool.connections[key]
	if pooled == nil {
		t.Fatalf("expected pooled connection for key %q", key)
	}
	pooled.mu.Lock()
	pooled.lastUsed = time.Now().Add(-2 * time.Second)
	pooled.mu.Unlock()

	pool.cleanupIdleConnections()

	if _, ok := pool.connections[key]; ok {
		t.Fatalf("expected idle pooled connection %q to be removed", key)
	}
}

func TestConnectionPoolCheckConnectionHealthRemovesShutdownConnection(t *testing.T) {
	address, tlsConfig := newPoolTestServer(t)

	pool := NewConnectionPool(PoolConfig{
		MaxIdleTime:         time.Minute,
		MaxLifetime:         time.Minute,
		HealthCheckInterval: time.Hour,
	})
	defer func() {
		_ = pool.Close()
	}()

	_, err := pool.Get(context.Background(), address, tlsConfig)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	pool.Release(address, tlsConfig)

	key := pool.connectionKey(address, tlsConfig)
	pooled := pool.connections[key]
	if pooled == nil {
		t.Fatalf("expected pooled connection for key %q", key)
	}

	if err := pooled.conn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	pool.checkConnectionHealth()

	if _, ok := pool.connections[key]; ok {
		t.Fatalf("expected unhealthy pooled connection %q to be removed", key)
	}
}

func TestConnectionPoolCloseClearsTrackedConnections(t *testing.T) {
	address, tlsConfig := newPoolTestServer(t)

	pool := NewConnectionPool(PoolConfig{
		MaxIdleTime:         time.Minute,
		MaxLifetime:         time.Minute,
		HealthCheckInterval: time.Hour,
	})

	_, err := pool.Get(context.Background(), address, tlsConfig)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	pool.Release(address, tlsConfig)

	if len(pool.connections) != 1 {
		t.Fatalf("expected one tracked connection, got %d", len(pool.connections))
	}

	if err := pool.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if len(pool.connections) != 0 {
		t.Fatalf("expected no tracked connections after close, got %d", len(pool.connections))
	}
}

func newPoolTestServer(t *testing.T) (string, *TLSConfig) {
	t.Helper()

	serverCert, serverKey, clientCert, clientKey, caCert := newTLSArtifactsForTest(t, "127.0.0.1")

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		t.Fatal("failed to append CA cert")
	}
	tlsCert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		t.Fatalf("X509KeyPair() error = %v", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{tlsCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})))
	pb.RegisterSecretStoreProviderServer(server, &mockServer{})
	go func() {
		_ = server.Serve(lis)
	}()

	t.Cleanup(func() {
		server.Stop()
		_ = lis.Close()
	})

	return lis.Addr().String(), &TLSConfig{
		CACert:     caCert,
		ClientCert: clientCert,
		ClientKey:  clientKey,
		ServerName: "127.0.0.1",
	}
}
