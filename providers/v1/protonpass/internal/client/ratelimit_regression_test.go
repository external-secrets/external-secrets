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

package client

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentSessionMint(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Code":1000,"Session":{"SessionUID":"synthetic","AccessToken":"synthetic","AccessExpirationTime":9999999999}}`))
	}))
	defer server.Close()
	pat := "pst_concurrentregression::" + base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	start := make(chan struct{})
	errs := make(chan error, 32)
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			c, err := NewClient(pat, WithBaseURL(server.URL))
			if err == nil {
				<-start
				_, err = c.getSession(context.Background())
			}
			errs <- err
		})
	}
	close(start)
	workers.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("32 concurrent clients minted %d sessions, want 1", n)
	}
}

func TestSession2028HonorsObservedRetryAfter(t *testing.T) {
	// Direct Proton API observation on 2026-09-11: HTTP 429, Code 2028,
	// Retry-After: 879. Keep the original five-minute minimum for shorter values.
	for _, tc := range []struct {
		name, header string
		minimum      time.Duration
	}{
		{"observed-879-seconds", "879", 878 * time.Second},
		{"shorter-than-minimum", "1", rateLimitCooldown - time.Second},
		{"missing", "", rateLimitCooldown - time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) > 1 {
					_, _ = w.Write([]byte(`{"Code":1000,"Session":{"SessionUID":"u","AccessToken":"a","AccessExpirationTime":9999999999}}`))
					return
				}
				w.Header().Set("Retry-After", tc.header)
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"Code":2028}`))
			}))
			defer server.Close()
			pat := "pst_" + t.Name() + "::" + base64.RawURLEncoding.EncodeToString(make([]byte, 32))
			c, err := NewClient(pat, WithBaseURL(server.URL))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { setCooldown(c.sessionKey().Name, time.Time{}) })
			if _, err = c.getSession(context.Background()); err == nil {
				t.Fatal("expected rate limit")
			}
			other, err := NewClient(pat, WithBaseURL(server.URL))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = other.getSession(context.Background()); err == nil {
				t.Fatal("expected cooldown")
			}
			if calls.Load() != 1 {
				t.Fatalf("unexpected login calls: %d", calls.Load())
			}
			mintCooldownMu.Lock()
			remaining := time.Until(mintCooldown[c.sessionKey().Name])
			mintCooldownMu.Unlock()
			if remaining < tc.minimum {
				t.Fatalf("cooldown %s, want at least %s", remaining, tc.minimum)
			}
			setCooldown(c.sessionKey().Name, time.Now().Add(-time.Second))
			if _, err = other.getSession(context.Background()); err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 2 {
				t.Fatal("did not recover after cooldown expiry")
			}
		})
	}
}

func TestHTTPTimeoutAndRecovery(t *testing.T) {
	for _, endpoint := range []string{"session", "read"} {
		for _, phase := range []string{"headers", "body"} {
			t.Run(endpoint+"/"+phase, func(t *testing.T) {
				var stall atomic.Bool
				stall.Store(true)
				release := make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					if stall.Load() {
						if phase == "body" {
							w.WriteHeader(http.StatusOK)
							w.(http.Flusher).Flush()
						}
						select {
						case <-r.Context().Done():
						case <-release:
						}
						return
					}
					if endpoint == "session" {
						_, _ = w.Write([]byte(`{"Code":1000,"Session":{"SessionUID":"u","AccessToken":"a","AccessExpirationTime":9999999999}}`))
					} else {
						_, _ = w.Write([]byte(`{}`))
					}
				}))
				defer server.Close()
				defer close(release)
				pat := "pst_" + t.Name() + "::" + base64.RawURLEncoding.EncodeToString(make([]byte, 32))
				c, err := NewClient(pat, WithBaseURL(server.URL))
				if err != nil {
					t.Fatal(err)
				}
				if c.httpClient.Timeout != 30*time.Second {
					t.Fatalf("default timeout = %s, want 30s", c.httpClient.Timeout)
				}
				// Exercise the same client with a short timeout to keep the test fast.
				c.httpClient.Timeout = 50 * time.Millisecond
				call := func(ctx context.Context) error {
					if endpoint == "session" {
						_, err := c.mintSession(ctx)
						return err
					}
					_, _, _, err := c.raw(ctx, http.MethodGet, "/read", nil, &session{})
					return err
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err = call(ctx)
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected timeout, got %v", err)
				}
				if ctx.Err() != nil {
					t.Fatal("caller deadline fired instead of HTTP timeout")
				}
				stall.Store(false)
				if err = call(ctx); err != nil {
					t.Fatalf("next request failed after timeout: %v", err)
				}
				canceled, cancelNow := context.WithCancel(context.Background())
				cancelNow()
				if err = call(canceled); !errors.Is(err, context.Canceled) {
					t.Fatalf("caller cancellation lost: %v", err)
				}
			})
		}
	}
}
