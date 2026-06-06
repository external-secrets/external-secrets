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

package keepersecurity

import (
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	ksm "github.com/keeper-security/secrets-manager-go/core"
)

// Resilience knobs (env-overridable):
//   - Throttle/429 retry is ON by default (pure resilience; only triggers on rate-limit errors).
//   - The shared record cache is OPT-IN via KEEPER_RECORD_CACHE_TTL_MS (>0 enables it).
func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// waitFn sleeps for d but returns early if ctx is cancelled. Overridable in tests.
var waitFn = func(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// backoff returns base*2^attempt capped, plus up to 25% jitter to avoid
// synchronized retry waves across many ExternalSecrets.
func backoff(attempt int) time.Duration {
	base := time.Duration(envInt("KEEPER_THROTTLE_RETRY_BASE_MS", 500)) * time.Millisecond
	maxd := time.Duration(envInt("KEEPER_THROTTLE_RETRY_MAX_MS", 10000)) * time.Millisecond
	d := base << attempt
	if d > maxd {
		d = maxd
	}
	// Add up to 25% jitter to de-synchronize retries across ExternalSecrets.
	// crypto/rand is used (not math/rand) only to satisfy weak-PRNG linters; the
	// jitter itself is not security-sensitive.
	if j := int64(d) / 4; j > 0 {
		if n, err := crand.Int(crand.Reader, big.NewInt(j)); err == nil {
			d += time.Duration(n.Int64())
		}
	}
	return d
}

// isRateLimited matches both the keeperapp app-throttle (403 {"error":"throttled"})
// and the edge nginx limit (429 Too Many Requests) — the latter is what real prod
// clients hit first and is NOT covered by the SDK's 403-only retry.
func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "throttled") ||
		strings.Contains(s, "429") ||
		strings.Contains(s, "too many request")
}

// retryAttempts is clamped to >=1 so a misconfigured env value can never turn the
// retry loops into zero-attempt no-ops (which would make reads return empty and
// writes silently skip). Note: envInt is NOT clamped because 0 is meaningful for
// other knobs (e.g. KEEPER_RECORD_CACHE_TTL_MS=0 disables the cache).
func retryAttempts() int {
	if n := envInt("KEEPER_THROTTLE_RETRY_ATTEMPTS", 4); n > 0 {
		return n
	}
	return 1
}

// getSecretsWithRetry wraps the SDK GetSecrets with backoff on throttle (403) / 429.
func getSecretsWithRetry(ctx context.Context, cl SecurityClient, filter []string) ([]*ksm.Record, error) {
	attempts := retryAttempts()
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		recs, err := cl.GetSecrets(filter)
		if err == nil {
			return recs, nil
		}
		if !isRateLimited(err) {
			return nil, err
		}
		last = err
		if attempt == attempts-1 {
			break
		}
		if werr := waitFn(ctx, backoff(attempt)); werr != nil {
			return nil, werr
		}
	}
	return nil, last
}

// getFoldersWithRetry wraps the SDK GetFolders with backoff on throttle (403) / 429.
func getFoldersWithRetry(ctx context.Context, cl SecurityClient) ([]*ksm.KeeperFolder, error) {
	attempts := retryAttempts()
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		folders, err := cl.GetFolders()
		if err == nil {
			return folders, nil
		}
		if !isRateLimited(err) {
			return nil, err
		}
		last = err
		if attempt == attempts-1 {
			break
		}
		if werr := waitFn(ctx, backoff(attempt)); werr != nil {
			return nil, werr
		}
	}
	return nil, last
}

// withRateLimitRetry retries a write op on throttle/429. Safe because a rate-limit
// response means the request was rejected before processing (no partial write).
func withRateLimitRetry(ctx context.Context, op func() error) error {
	attempts := retryAttempts()
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		err := op()
		if err == nil || !isRateLimited(err) {
			return err
		}
		last = err
		if attempt == attempts-1 {
			break
		}
		if werr := waitFn(ctx, backoff(attempt)); werr != nil {
			return werr
		}
	}
	return last
}

// ---- shared record cache (opt-in) ----
// One get_secret returns ALL records the app can read, so a short-lived shared
// cache collapses a reconcile wave of N ExternalSecrets into a single backend call.

type recordCacheEntry struct {
	records []*ksm.Record
	fetched time.Time
}

type recordCache struct {
	mu sync.Mutex
	m  map[string]recordCacheEntry
}

var sharedRecordCache = &recordCache{m: map[string]recordCacheEntry{}}

// folder cache (folders change rarely; reuses the same TTL knob)
type folderCacheEntry struct {
	folders []*ksm.KeeperFolder
	fetched time.Time
}

type folderCache struct {
	mu sync.Mutex
	m  map[string]folderCacheEntry
}

var sharedFolderCache = &folderCache{m: map[string]folderCacheEntry{}}

func (fc *folderCache) get(key string, ttl time.Duration) ([]*ksm.KeeperFolder, bool) {
	if ttl <= 0 || key == "" {
		return nil, false
	}
	fc.mu.Lock()
	defer fc.mu.Unlock()
	e, ok := fc.m[key]
	if !ok {
		return nil, false
	}
	if time.Since(e.fetched) > ttl {
		delete(fc.m, key) // evict instead of leaking stale entries
		return nil, false
	}
	return e.folders, true
}

func (fc *folderCache) set(key string, folders []*ksm.KeeperFolder) {
	if key == "" {
		return
	}
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.m[key] = folderCacheEntry{folders: folders, fetched: time.Now()}
}

func cacheTTL() time.Duration {
	return time.Duration(envInt("KEEPER_RECORD_CACHE_TTL_MS", 0)) * time.Millisecond
}

func (rc *recordCache) get(key string, ttl time.Duration) ([]*ksm.Record, bool) {
	if ttl <= 0 || key == "" {
		return nil, false
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	e, ok := rc.m[key]
	if !ok {
		return nil, false
	}
	if time.Since(e.fetched) > ttl {
		delete(rc.m, key) // evict instead of leaking stale entries
		return nil, false
	}
	return e.records, true
}

func (rc *recordCache) set(key string, recs []*ksm.Record) {
	if key == "" {
		return
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.m[key] = recordCacheEntry{records: recs, fetched: time.Now()}
}

// invalidate drops a key so a subsequent read re-fetches (called after writes).
func (rc *recordCache) invalidate(key string) {
	if key == "" {
		return
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	delete(rc.m, key)
}

func resetRecordCache() {
	sharedRecordCache.mu.Lock()
	sharedRecordCache.m = map[string]recordCacheEntry{}
	sharedRecordCache.mu.Unlock()
	sharedFolderCache.mu.Lock()
	sharedFolderCache.m = map[string]folderCacheEntry{}
	sharedFolderCache.mu.Unlock()
}

func hashConfig(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])
}
