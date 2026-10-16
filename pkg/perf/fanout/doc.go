// /*
// Copyright © The ESO Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

//go:build perf

// Package fanout contains Plan 3 of the ESO performance test suite:
// PERF_NUM_STORES SecretStores each with PERF_ES_PER_STORE ExternalSecrets,
// testing the combined fan-out case under realistic multi-tenant conditions.
//
// Default: 100 stores × 100 ES = 10k ExternalSecrets total.
// Full scale: 1000 × 100 = 100k (dedicated machine).
// Maximum scale (1M ES) requires a real cluster, not envtest.
//
// Run with:
//
//	go test -v -tags=perf -timeout=60m ./pkg/perf/fanout/...
//
// Environment variables:
//
//	PERF_NUM_STORES        SecretStores / namespaces (default 100)
//	PERF_ES_PER_STORE      ExternalSecrets per store (default 100)
//	PERF_ES_CONCURRENCY    MaxConcurrentReconciles for ES controller (default 16)
//	PERF_STORE_CONCURRENCY MaxConcurrentReconciles for SecretStore controller (default 16)
//	PERF_QPS               REST client QPS (default 500)
//	PERF_BURST             REST client burst (default 1000)
package fanout
