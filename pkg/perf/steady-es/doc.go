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

// Package steady_es measures ExternalSecret reconciliation during steady-state operation:
// objects are created and allowed to settle (2× refreshInterval) before any metrics are
// captured, so the observation window reflects only routine re-reconciliation behaviour —
// not the initial creation burst.
//
// Run with:
//
//	go test -v -tags=perf -timeout=60m ./pkg/perf/steady-es/...
//
// Environment variables:
//
//	PERF_ES_CONCURRENCY        MaxConcurrentReconciles for the ES controller (default 4)
//	PERF_REQUEUE_INTERVAL_SECS RefreshInterval set on each ES, and the controller fallback (default 30)
//	PERF_OBSERVATION_CYCLES    Number of refreshInterval-lengths to observe (default 3)
package steady_es
