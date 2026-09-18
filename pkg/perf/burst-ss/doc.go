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

// Package burst_ss contains the second aspect of the ESO performance test suite:
// PERF_NUM_STORES SecretStores (one per namespace) reconciled simultaneously,
// measuring SecretStore controller throughput via controller-runtime Prometheus metrics.
//
// Run with:
//
//	go test -v -tags=perf -timeout=30m ./pkg/perf/burst-ss/...
//
// Environment variables:
//
//	PERF_NUM_STORES        Number of SecretStores / namespaces (default 10000)
//	PERF_STORE_CONCURRENCY MaxConcurrentReconciles for the SecretStore controller (default 16)
package burst_ss
