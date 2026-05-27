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
