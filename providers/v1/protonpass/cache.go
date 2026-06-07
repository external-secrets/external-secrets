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

package protonpass

import "github.com/external-secrets/external-secrets/runtime/cache"

// sessionCacheSize bounds the number of distinct stores whose minted session is
// kept warm. Each entry is a single *apiClient (one Proton session); the cache
// is LRU so the oldest store's session is dropped past this bound.
const sessionCacheSize = 1024

// sessionCache memoizes each store's *apiClient — which holds a lazily-minted,
// reusable Proton session — across reconciles, keyed by store identity and
// versioned by the store's ResourceVersion.
//
// Proton rate-limits logins per ACCOUNT (API code 2028, "too many recent
// logins"). Without this cache every ExternalSecret reconcile AND every periodic
// SecretStore validation mints a fresh session, so even a handful of resources
// trips the throttle (and, once throttled, the validation requeue loop keeps
// re-minting and sustains it). Caching collapses minting to roughly once per
// store: a spec change (new ResourceVersion) evicts the entry and an expired
// session (401) is re-minted in place by apiClient.ensureSession, so the cached
// pointer stays valid across rotations.
//
// The apiClient is safe to share across reconcile goroutines — its session state
// is guarded by a mutex.
var sessionCache = cache.Must[*apiClient](sessionCacheSize, nil)
