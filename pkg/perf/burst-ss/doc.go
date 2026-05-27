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
