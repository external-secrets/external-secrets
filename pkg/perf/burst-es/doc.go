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

// Package burst_es contains the first aspect of the ESO performance test suite:
// N ExternalSecrets created simultaneously (burst) against a single SecretStore,
// measuring reconcile throughput and latency via controller-runtime Prometheus metrics.
//
// Run with:
//
//	go test -v -tags=perf -timeout=15m ./pkg/perf/burst-es/...
//
// Environment variables:
//
//	PERF_ES_CONCURRENCY  MaxConcurrentReconciles for the ES controller (default 4)
package burst_es
