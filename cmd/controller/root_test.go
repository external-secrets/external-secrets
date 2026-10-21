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

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// fakeCache reports a fixed cache-sync state.
type fakeCache struct {
	runtimecache.Cache
	synced bool
}

func (f *fakeCache) WaitForCacheSync(_ context.Context) bool {
	return f.synced
}

// fakeManager captures the checkers registered by setupHealthChecks, keyed by
// endpoint and name, so tests can invoke them directly.
type fakeManager struct {
	ctrl.Manager
	cache    runtimecache.Cache
	checkers map[string]healthz.Checker
}

func newFakeManager(synced bool) *fakeManager {
	return &fakeManager{
		cache:    &fakeCache{synced: synced},
		checkers: map[string]healthz.Checker{},
	}
}

func (f *fakeManager) GetCache() runtimecache.Cache {
	return f.cache
}

func (f *fakeManager) AddHealthzCheck(name string, c healthz.Checker) error {
	f.checkers["healthz/"+name] = c
	return nil
}

func (f *fakeManager) AddReadyzCheck(name string, c healthz.Checker) error {
	f.checkers["readyz/"+name] = c
	return nil
}

// readyzChecker returns the checker registered at /readyz.
func (f *fakeManager) readyzChecker(t *testing.T) healthz.Checker {
	t.Helper()
	c, ok := f.checkers["readyz/readyz"]
	require.True(t, ok, `setupHealthChecks must register a readyz check named "readyz"`)
	return c
}

// TestSetupHealthChecks verifies that setupHealthChecks registers
// a readiness check that gates on cache sync.
func TestSetupHealthChecks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		synced  bool
		wantErr bool
	}{
		{
			name:    "cache already synced returns nil",
			synced:  true,
			wantErr: false,
		},
		{
			name:    "cache not synced returns error",
			synced:  false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := newFakeManager(tt.synced)
			require.NoError(t, setupHealthChecks(mgr))

			// Use a real *http.Request so the checker can use req.Context().
			req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
			err := mgr.readyzChecker(t)(req)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cache not yet synced")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSetupHealthChecks_NilRequest verifies that the readyz checker
// doesn't panic when called with a nil request.
func TestSetupHealthChecks_NilRequest(t *testing.T) {
	t.Parallel()
	mgr := newFakeManager(true)
	require.NoError(t, setupHealthChecks(mgr))
	require.NotPanics(t, func() {
		require.NoError(t, mgr.readyzChecker(t)(nil))
	})
}
