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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	ctrl "sigs.k8s.io/controller-runtime"

	v2 "github.com/external-secrets/external-secrets/providers/v2/common"
)

// ConnectionPool manages a pool of gRPC connections to providers.
// It handles connection reuse, health checking, and graceful shutdown.
type ConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*pooledConnection
	maxIdle     time.Duration
	maxLifetime time.Duration
	healthCheck time.Duration
	log         logr.Logger
}

// pooledConnection wraps a gRPC connection with metadata for pooling.
type pooledConnection struct {
	conn       *grpc.ClientConn
	client     v2.Provider
	created    time.Time
	lastUsed   time.Time
	references int32 // Number of active users
	mu         sync.Mutex
}

// PoolConfig configures the connection pool.
type PoolConfig struct {
	// MaxIdleTime is how long a connection can be idle before being closed
	MaxIdleTime time.Duration
	// MaxLifetime is the maximum lifetime of a connection
	MaxLifetime time.Duration
	// HealthCheckInterval is how often to check connection health
	HealthCheckInterval time.Duration
}

// DefaultPoolConfig returns sensible defaults for connection pooling.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxIdleTime:         5 * time.Minute,
		MaxLifetime:         30 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}
}

// NewConnectionPool creates a new connection pool with the given configuration.
func NewConnectionPool(cfg PoolConfig) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string]*pooledConnection),
		maxIdle:     cfg.MaxIdleTime,
		maxLifetime: cfg.MaxLifetime,
		healthCheck: cfg.HealthCheckInterval,
		log:         ctrl.Log.WithName("grpc-pool"),
	}

	pool.log.Info("connection pool initialized",
		"maxIdleTime", cfg.MaxIdleTime.String(),
		"maxLifetime", cfg.MaxLifetime.String(),
		"healthCheckInterval", cfg.HealthCheckInterval.String())

	// Start background goroutine for cleanup and health checks
	go pool.maintenance()

	return pool
}

// Get retrieves or creates a connection to the specified provider address.
// The caller must call Release() when done with the connection.
func (p *ConnectionPool) Get(ctx context.Context, address string, tlsConfig *TLSConfig) (v2.Provider, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := p.connectionKey(address, tlsConfig)

	p.log.V(1).Info("getting connection from pool",
		"address", address,
		"key", key)

	// Check if we have a valid cached connection
	if pooled, exists := p.connections[key]; exists {
		pooled.mu.Lock()
		defer pooled.mu.Unlock()

		p.log.V(1).Info("found cached connection",
			"address", address,
			"state", pooled.conn.GetState().String(),
			"references", pooled.references,
			"age", time.Since(pooled.created).String(),
			"idleTime", time.Since(pooled.lastUsed).String())

		// Check if connection is still valid
		if p.isConnectionValid(pooled) {
			pooled.references++
			pooled.lastUsed = time.Now()
			p.log.Info("reusing cached connection",
				"address", address,
				"references", pooled.references)
			// Record cache hit
			poolMetrics.RecordHit(address, tlsConfig != nil)
			return pooled.client, nil
		}

		// Connection is invalid, clean it up
		p.log.Info("cached connection invalid, cleaning up",
			"address", address,
			"state", pooled.conn.GetState().String())
		pooled.conn.Close()
		delete(p.connections, key)
	}

	// Create new connection
	p.log.Info("creating new connection",
		"address", address,
		"tlsEnabled", tlsConfig != nil)

	// Record cache miss
	poolMetrics.RecordMiss(address, tlsConfig != nil)

	providerClient, err := NewClient(address, tlsConfig)
	if err != nil {
		p.log.Error(err, "failed to create new connection", "address", address)
		// Record connection error
		poolMetrics.RecordConnectionError(address, tlsConfig != nil)
		return nil, fmt.Errorf("failed to create new connection: %w", err)
	}

	// Extract the underlying connection for pooling
	grpcClient, ok := providerClient.(*grpcProviderClient)
	if !ok {
		return nil, fmt.Errorf("unexpected client type")
	}

	pooled := &pooledConnection{
		conn:       grpcClient.conn,
		client:     providerClient,
		created:    time.Now(),
		lastUsed:   time.Now(),
		references: 1,
	}

	p.connections[key] = pooled

	p.log.Info("new connection added to pool",
		"address", address,
		"state", grpcClient.conn.GetState().String(),
		"target", grpcClient.conn.Target())

	return providerClient, nil
}

// Release marks a connection as no longer in use.
// This should be called in a defer after Get().
func (p *ConnectionPool) Release(address string, tlsConfig *TLSConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := p.connectionKey(address, tlsConfig)

	if pooled, exists := p.connections[key]; exists {
		pooled.mu.Lock()
		defer pooled.mu.Unlock()

		if pooled.references > 0 {
			pooled.references--
			p.log.V(1).Info("released connection",
				"address", address,
				"remainingReferences", pooled.references)
		}
	}
}

// Close shuts down the connection pool and closes all connections.
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pooled := range p.connections {
		pooled.mu.Lock()
		if pooled.conn != nil {
			pooled.conn.Close()
		}
		pooled.mu.Unlock()
	}

	p.connections = make(map[string]*pooledConnection)

	return nil
}

// maintenance runs periodic cleanup and health checks.
func (p *ConnectionPool) maintenance() {
	ticker := time.NewTicker(p.healthCheck)
	defer ticker.Stop()

	for range ticker.C {
		p.cleanupIdleConnections()
		p.checkConnectionHealth()
		p.updatePoolMetrics()
	}
}

// cleanupIdleConnections removes connections that have been idle too long.
func (p *ConnectionPool) cleanupIdleConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	toRemove := make([]string, 0)
	evictions := make(map[string]string) // key -> reason

	for key, pooled := range p.connections {
		pooled.mu.Lock()

		// Skip connections that are in use
		if pooled.references > 0 {
			pooled.mu.Unlock()
			continue
		}

		// Check if connection is too old or idle too long
		idleTooLong := now.Sub(pooled.lastUsed) > p.maxIdle
		tooOld := now.Sub(pooled.created) > p.maxLifetime

		if idleTooLong {
			pooled.conn.Close()
			toRemove = append(toRemove, key)
			evictions[key] = "idle_timeout"
		} else if tooOld {
			pooled.conn.Close()
			toRemove = append(toRemove, key)
			evictions[key] = "max_lifetime"
		}

		pooled.mu.Unlock()
	}

	for _, key := range toRemove {
		address, tlsEnabled := p.parseConnectionKey(key)
		poolMetrics.RecordEviction(address, tlsEnabled, evictions[key])
		delete(p.connections, key)
	}
}

// checkConnectionHealth verifies that pooled connections are still healthy.
func (p *ConnectionPool) checkConnectionHealth() {
	p.mu.Lock()
	defer p.mu.Unlock()

	toRemove := make([]string, 0)

	for key, pooled := range p.connections {
		pooled.mu.Lock()

		// Check connection state
		state := pooled.conn.GetState()
		if state == connectivity.TransientFailure || state == connectivity.Shutdown {
			pooled.conn.Close()
			toRemove = append(toRemove, key)
		}

		pooled.mu.Unlock()
	}

	for _, key := range toRemove {
		address, tlsEnabled := p.parseConnectionKey(key)
		poolMetrics.RecordEviction(address, tlsEnabled, "health_check")
		delete(p.connections, key)
	}
}

// isConnectionValid checks if a pooled connection is still usable.
func (p *ConnectionPool) isConnectionValid(pooled *pooledConnection) bool {
	// Check age
	if time.Since(pooled.created) > p.maxLifetime {
		return false
	}

	// Check connection state
	state := pooled.conn.GetState()
	if state == connectivity.Shutdown || state == connectivity.TransientFailure {
		return false
	}

	return true
}

// connectionKey generates a unique key for caching connections.
func (p *ConnectionPool) connectionKey(address string, tlsConfig *TLSConfig) string {
	if tlsConfig != nil {
		return fmt.Sprintf("%s-tls", address)
	}
	return fmt.Sprintf("%s-insecure", address)
}

// parseConnectionKey extracts address and TLS status from a connection key.
func (p *ConnectionPool) parseConnectionKey(key string) (address string, tlsEnabled bool) {
	if len(key) > 4 && key[len(key)-4:] == "-tls" {
		return key[:len(key)-4], true
	}
	if len(key) > 9 && key[len(key)-9:] == "-insecure" {
		return key[:len(key)-9], false
	}
	return key, false
}

// updatePoolMetrics updates pool state metrics.
func (p *ConnectionPool) updatePoolMetrics() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Track stats per address/TLS combination
	stats := make(map[string]struct {
		active int
		idle   int
		total  int
	})

	now := time.Now()

	for key, pooled := range p.connections {
		pooled.mu.Lock()
		address, tlsEnabled := p.parseConnectionKey(key)
		
		statKey := key
		s := stats[statKey]
		s.total++
		
		if pooled.references > 0 {
			s.active++
		} else {
			s.idle++
		}
		
		stats[statKey] = s

		// Record connection age and idle time
		poolMetrics.RecordConnectionAge(address, tlsEnabled, now.Sub(pooled.created))
		if pooled.references == 0 {
			poolMetrics.RecordConnectionIdle(address, tlsEnabled, now.Sub(pooled.lastUsed))
		}
		
		pooled.mu.Unlock()
	}

	// Update gauges
	for key, s := range stats {
		address, tlsEnabled := p.parseConnectionKey(key)
		poolMetrics.UpdatePoolState(address, tlsEnabled, s.active, s.idle, s.total)
	}
}
