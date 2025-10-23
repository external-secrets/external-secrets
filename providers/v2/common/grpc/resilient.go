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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	v2 "github.com/external-secrets/external-secrets/providers/v2/common"
)

// ResilientClient wraps a gRPC provider client with connection pooling,
// retry logic, and circuit breaking for production-ready reliability.
type ResilientClient struct {
	address        string
	tlsConfig      *TLSConfig
	pool           *ConnectionPool
	circuitBreaker *CircuitBreaker
	retryConfig    RetryConfig
	log            logr.Logger
}

// ResilientClientConfig configures the resilient client.
type ResilientClientConfig struct {
	Address       string
	TLSConfig     *TLSConfig
	PoolConfig    PoolConfig
	CircuitConfig CircuitBreakerConfig
	RetryConfig   RetryConfig
}

// DefaultResilientClientConfig returns sensible defaults.
func DefaultResilientClientConfig(address string, tlsConfig *TLSConfig) ResilientClientConfig {
	return ResilientClientConfig{
		Address:       address,
		TLSConfig:     tlsConfig,
		PoolConfig:    DefaultPoolConfig(),
		CircuitConfig: DefaultCircuitBreakerConfig(),
		RetryConfig:   DefaultRetryConfig(),
	}
}

// NewResilientClient creates a new resilient client with connection pooling,
// retry logic, and circuit breaking.
func NewResilientClient(config ResilientClientConfig) (*ResilientClient, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("provider address cannot be empty")
	}

	log := ctrl.Log.WithName("grpc-resilient").WithValues("address", config.Address)
	log.Info("creating resilient client",
		"poolMaxIdleTime", config.PoolConfig.MaxIdleTime.String(),
		"poolMaxLifetime", config.PoolConfig.MaxLifetime.String(),
		"retryMaxAttempts", config.RetryConfig.MaxAttempts,
		"circuitMaxFailures", config.CircuitConfig.MaxFailures)

	return &ResilientClient{
		address:        config.Address,
		tlsConfig:      config.TLSConfig,
		pool:           NewConnectionPool(config.PoolConfig),
		circuitBreaker: NewCircuitBreaker(config.CircuitConfig),
		retryConfig:    config.RetryConfig,
		log:            log,
	}, nil
}

// Ensure ResilientClient implements Provider interface
var _ v2.Provider = &ResilientClient{}

// PushSecret writes a secret with retry logic and circuit breaking.
func (rc *ResilientClient) PushSecret(ctx context.Context, secretData map[string][]byte, pushSecretData *pb.PushSecretData, providerRef *pb.ProviderReference, sourceNamespace string) error {
	return rc.executeWithResilience(ctx, func(client v2.Provider) error {
		return client.PushSecret(ctx, secretData, pushSecretData, providerRef, sourceNamespace)
	})
}

// DeleteSecret deletes a secret with retry logic and circuit breaking.
func (rc *ResilientClient) DeleteSecret(ctx context.Context, remoteRef *pb.PushSecretRemoteRef, providerRef *pb.ProviderReference, sourceNamespace string) error {
	return rc.executeWithResilience(ctx, func(client v2.Provider) error {
		return client.DeleteSecret(ctx, remoteRef, providerRef, sourceNamespace)
	})
}

// SecretExists checks if a secret exists with retry logic and circuit breaking.
func (rc *ResilientClient) SecretExists(ctx context.Context, remoteRef *pb.PushSecretRemoteRef, providerRef *pb.ProviderReference, sourceNamespace string) (bool, error) {
	var result bool

	err := rc.executeWithResilience(ctx, func(client v2.Provider) error {
		exists, err := client.SecretExists(ctx, remoteRef, providerRef, sourceNamespace)
		if err != nil {
			return err
		}
		result = exists
		return nil
	})

	return result, err
}

// GetSecret retrieves a secret with retry logic and circuit breaking.
func (rc *ResilientClient) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef, providerRef *pb.ProviderReference, sourceNamespace string) ([]byte, error) {
	var result []byte

	err := rc.executeWithResilience(ctx, func(client v2.Provider) error {
		secretData, err := client.GetSecret(ctx, ref, providerRef, sourceNamespace)
		if err != nil {
			return err
		}
		result = secretData
		return nil
	})

	return result, err
}

// GetAllSecrets retrieves multiple secrets with retry logic and circuit breaking.
func (rc *ResilientClient) GetAllSecrets(ctx context.Context, find esv1.ExternalSecretFind, providerRef *pb.ProviderReference, sourceNamespace string) (map[string][]byte, error) {
	var result map[string][]byte

	err := rc.executeWithResilience(ctx, func(client v2.Provider) error {
		secrets, err := client.GetAllSecrets(ctx, find, providerRef, sourceNamespace)
		if err != nil {
			return err
		}
		result = secrets
		return nil
	})

	return result, err
}

// Validate validates the provider configuration with retry logic.
func (rc *ResilientClient) Validate(ctx context.Context, providerRef *pb.ProviderReference, sourceNamespace string) error {
	return rc.executeWithResilience(ctx, func(client v2.Provider) error {
		return client.Validate(ctx, providerRef, sourceNamespace)
	})
}

// Capabilities retrieves the provider's capabilities with retry logic.
func (rc *ResilientClient) Capabilities(ctx context.Context, providerRef *pb.ProviderReference, sourceNamespace string) (pb.SecretStoreCapabilities, error) {
	var result pb.SecretStoreCapabilities

	err := rc.executeWithResilience(ctx, func(client v2.Provider) error {
		caps, err := client.Capabilities(ctx, providerRef, sourceNamespace)
		if err != nil {
			return err
		}
		result = caps
		return nil
	})

	return result, err
}

// Close closes the connection pool.
func (rc *ResilientClient) Close(ctx context.Context) error {
	return rc.pool.Close()
}

// executeWithResilience executes a function with connection pooling, retry, and circuit breaking.
func (rc *ResilientClient) executeWithResilience(ctx context.Context, fn func(v2.Provider) error) error {
	rc.log.V(1).Info("executing with resilience",
		"circuitState", rc.circuitBreaker.State())

	// Check circuit breaker first
	return rc.circuitBreaker.Call(ctx, func() error {
		// Execute with retry
		return WithRetry(ctx, rc.retryConfig, func(ctx context.Context, attempt int) error {
			if attempt > 1 {
				rc.log.Info("retrying operation",
					"attempt", attempt,
					"maxAttempts", rc.retryConfig.MaxAttempts)
			}

			// Get connection from pool
			rc.log.V(1).Info("getting connection from pool")
			client, err := rc.pool.Get(ctx, rc.address, rc.tlsConfig)
			if err != nil {
				rc.log.Error(err, "failed to get connection from pool",
					"attempt", attempt)
				return fmt.Errorf("failed to get connection from pool: %w", err)
			}
			defer rc.pool.Release(rc.address, rc.tlsConfig)

			// Execute the function
			rc.log.V(1).Info("executing provider operation", "attempt", attempt)
			err = fn(client)
			if err != nil {
				rc.log.Error(err, "provider operation failed",
					"attempt", attempt,
					"willRetry", attempt < rc.retryConfig.MaxAttempts)
			}
			return err
		})
	})
}

// GetCircuitState returns the current state of the circuit breaker.
func (rc *ResilientClient) GetCircuitState() CircuitState {
	return rc.circuitBreaker.State()
}
