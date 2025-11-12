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
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCircuitBreaker_States(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	})

	ctx := context.Background()

	// Initially closed
	if cb.State() != StateClosed {
		t.Errorf("expected initial state to be Closed, got %v", cb.State())
	}

	// Simulate 3 failures to open the circuit
	for i := 0; i < 3; i++ {
		err := cb.Call(ctx, func() error {
			return status.Error(codes.Unavailable, "service unavailable")
		})
		if err == nil {
			t.Fatal("expected error from failing call")
		}
	}

	// Should now be open
	if cb.State() != StateOpen {
		t.Errorf("expected state to be Open after failures, got %v", cb.State())
	}

	// Requests should fail fast
	err := cb.Call(ctx, func() error {
		t.Fatal("should not execute function in open state")
		return nil
	})
	if err == nil || err.Error() != "circuit breaker is open" {
		t.Errorf("expected circuit open error, got: %v", err)
	}

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)

	// Should now be half-open
	err = cb.Call(ctx, func() error {
		return nil // Success
	})
	if err != nil {
		t.Fatalf("expected success in half-open state, got: %v", err)
	}

	// Should transition back to closed
	if cb.State() != StateClosed {
		t.Errorf("expected state to be Closed after success, got %v", cb.State())
	}
}

func TestRetry_Backoff(t *testing.T) {
	attempts := 0
	config := RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false, // Disable for predictable testing
	}

	ctx := context.Background()
	start := time.Now()

	err := WithRetry(ctx, config, func(ctx context.Context, attempt int) error {
		attempts++
		if attempt < 2 {
			return status.Error(codes.Unavailable, "unavailable")
		}
		return nil // Success on 3rd attempt
	})

	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	// Expected: 10ms + 20ms = 30ms minimum
	if elapsed < 30*time.Millisecond {
		t.Errorf("expected at least 30ms elapsed, got %v", elapsed)
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	attempts := 0
	config := DefaultRetryConfig()

	ctx := context.Background()

	err := WithRetry(ctx, config, func(ctx context.Context, attempt int) error {
		attempts++
		return status.Error(codes.NotFound, "not found")
	})

	if err == nil {
		t.Fatal("expected error")
	}

	// Should not retry NotFound
	if attempts != 1 {
		t.Errorf("expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	config := DefaultRetryConfig()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	err := WithRetry(ctx, config, func(ctx context.Context, attempt int) error {
		return status.Error(codes.Unavailable, "unavailable")
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "Unavailable is retryable",
			err:       status.Error(codes.Unavailable, "test"),
			retryable: true,
		},
		{
			name:      "DeadlineExceeded is retryable",
			err:       status.Error(codes.DeadlineExceeded, "test"),
			retryable: true,
		},
		{
			name:      "NotFound is not retryable",
			err:       status.Error(codes.NotFound, "test"),
			retryable: false,
		},
		{
			name:      "PermissionDenied is not retryable",
			err:       status.Error(codes.PermissionDenied, "test"),
			retryable: false,
		},
		{
			name:      "Context canceled is not retryable",
			err:       context.Canceled,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryable(tt.err)
			if result != tt.retryable {
				t.Errorf("expected retryable=%v, got %v", tt.retryable, result)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	config := RetryConfig{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1 * time.Second}, // Capped at MaxBackoff
		{5, 1 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		result := calculateBackoff(tt.attempt, config)
		if result != tt.expected {
			t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
		}
	}
}
