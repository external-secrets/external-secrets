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
	"math"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetryConfig configures retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
	// Jitter adds randomness to backoff to prevent thundering herd
	Jitter bool
}

// DefaultRetryConfig returns sensible defaults for retry behavior.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            true,
	}
}

// CircuitBreakerConfig configures circuit breaker behavior.
type CircuitBreakerConfig struct {
	// MaxFailures is the number of consecutive failures before opening
	MaxFailures int
	// Timeout is how long to wait in open state before trying again
	Timeout time.Duration
	// HalfOpenMaxRequests is max requests allowed in half-open state
	HalfOpenMaxRequests int
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:         5,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 1,
	}
}

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// StateClosed means the circuit is closed and requests flow normally
	StateClosed CircuitState = iota
	// StateOpen means the circuit is open and requests fail fast
	StateOpen
	// StateHalfOpen means the circuit is testing if the service has recovered
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitState
	failures        int
	lastFailureTime time.Time
	halfOpenReqs    int
	config          CircuitBreakerConfig
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:  StateClosed,
		config: config,
	}
}

// Call executes the given function with circuit breaker protection.
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	// Check if we should allow the request
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.afterRequest(err)

	return err
}

// beforeRequest checks if the request should be allowed.
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.halfOpenReqs = 0
			return nil
		}
		return fmt.Errorf("circuit breaker is open")

	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.halfOpenReqs >= cb.config.HalfOpenMaxRequests {
			return fmt.Errorf("circuit breaker is half-open, max requests reached")
		}
		cb.halfOpenReqs++
		return nil

	default:
		return fmt.Errorf("unknown circuit breaker state")
	}
}

// afterRequest records the result of a request.
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err == nil {
		// Success
		cb.onSuccess()
	} else {
		// Failure
		cb.onFailure()
	}
}

// onSuccess handles a successful request.
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failures = 0

	case StateHalfOpen:
		// Success in half-open state means we can close the circuit
		cb.state = StateClosed
		cb.failures = 0
		cb.halfOpenReqs = 0
	}
}

// onFailure handles a failed request.
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.MaxFailures {
			cb.state = StateOpen
		}

	case StateHalfOpen:
		// Failure in half-open state means we go back to open
		cb.state = StateOpen
		cb.halfOpenReqs = 0
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// RetryableFunc is a function that can be retried.
type RetryableFunc func(ctx context.Context, attempt int) error

// WithRetry executes a function with exponential backoff retry logic.
func WithRetry(ctx context.Context, config RetryConfig, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn(ctx, attempt)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return err
		}

		// Last attempt, don't wait
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate backoff
		backoff := calculateBackoff(attempt, config)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retry attempts (%d) exceeded: %w", config.MaxAttempts, lastErr)
}

// isRetryable determines if an error should trigger a retry.
func isRetryable(err error) bool {
	// Check gRPC status codes
	st, ok := status.FromError(err)
	if ok {
		switch st.Code() {
		case codes.Unavailable,
			codes.DeadlineExceeded,
			codes.ResourceExhausted,
			codes.Aborted:
			return true

		case codes.InvalidArgument,
			codes.NotFound,
			codes.AlreadyExists,
			codes.PermissionDenied,
			codes.Unauthenticated:
			return false

		default:
			// For unknown codes, retry to be safe
			return true
		}
	}

	// For non-gRPC errors, retry network-related issues
	// Context errors should not be retried
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Default: retry
	return true
}

// calculateBackoff calculates the backoff duration for a given attempt.
func calculateBackoff(attempt int, config RetryConfig) time.Duration {
	// Exponential backoff: initialBackoff * (multiplier ^ attempt)
	backoff := float64(config.InitialBackoff) * math.Pow(config.BackoffMultiplier, float64(attempt))

	// Apply max backoff
	if backoff > float64(config.MaxBackoff) {
		backoff = float64(config.MaxBackoff)
	}

	// Add jitter if enabled
	if config.Jitter {
		// Add random jitter between 0 and 25% of backoff
		jitter := rand.Float64() * backoff * 0.25
		backoff += jitter
	}

	return time.Duration(backoff)
}
