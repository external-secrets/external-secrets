//go:build perf

package steady_es

import (
	"context"
	"fmt"
	"time"

	perf "github.com/external-secrets/external-secrets/pkg/perf"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExternalSecret steady-state reconciliation", func() {
	DescribeTable("N ExternalSecrets, observe K requeue cycles after warmup",
		func(n int) {
			ctx := context.Background()
			concurrency := perf.EnvOrInt("PERF_ES_CONCURRENCY", 4)
			requeueSecs := perf.EnvOrInt("PERF_REQUEUE_INTERVAL_SECS", 30)
			refreshInterval := time.Duration(requeueSecs) * time.Second
			observationCycles := perf.EnvOrInt("PERF_OBSERVATION_CYCLES", 3)
			scenario := fmt.Sprintf("steady-n%d-c%d-r%ds-k%d", n, concurrency, requeueSecs, observationCycles)

			By(fmt.Sprintf("creating %d ExternalSecrets with refreshInterval=%s", n, refreshInterval))
			Expect(perf.CreateExternalSecrets(ctx, k8sClient, testNamespace, n, storeRef, refreshInterval)).To(Succeed())

			By("waiting for all to reach Ready (warmup, not measured)")
			_, err := perf.WaitAllReady(ctx, k8sClient, testNamespace, n, 30*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			// Wait 2 full refresh intervals before starting measurement.
			// This lets the initial requeue wave drain through the controller before we start capturing metrics
			// so the observation window reflects only routine steady-state re-reconciliations. 
			By(fmt.Sprintf("settling: waiting 2 * refreshInterval (%s)", 2*refreshInterval))
			time.Sleep(2 * refreshInterval)

			By(fmt.Sprintf("observation window: %d * refreshInterval (%s)", observationCycles, time.Duration(observationCycles)*refreshInterval))
			before := perf.Snapshot("externalsecret")
			windowStart := time.Now()
			time.Sleep(time.Duration(observationCycles) * refreshInterval)
			after := perf.Snapshot("externalsecret")
			wallTime := time.Since(windowStart)

			reconcileDelta, errorsDelta, heapDelta, gcDelta := perf.DiffSnapshots(before, after)

			result := perf.PerfResult{
				Plan:          "steady-state",
				Scenario:      scenario,
				NumStores:     1,
				NumESPerStore: n,
				Concurrency:   concurrency,
				WallTimeSec:   wallTime.Seconds(),
				// ThroughputRPS reflects how many re-reconciliations the controller
				// completed per second during the observation window, not a fixed N/t.
				ThroughputRPS:  reconcileDelta / wallTime.Seconds(),
				ReconcileP50ms: perf.HistogramPercentile(after.ReconcileTime, 0.50) * 1000,
				ReconcileP90ms: perf.HistogramPercentile(after.ReconcileTime, 0.90) * 1000,
				ReconcileP99ms: perf.HistogramPercentile(after.ReconcileTime, 0.99) * 1000,
				QueueP50ms:     perf.HistogramPercentile(after.QueueDuration, 0.50) * 1000,
				QueueP90ms:     perf.HistogramPercentile(after.QueueDuration, 0.90) * 1000,
				HeapDeltaMB:    float64(heapDelta) / (1024 * 1024),
				NumGCDelta:     gcDelta,
				PauseTotalMs:   float64(after.PauseTotalNs-before.PauseTotalNs) / 1e6,
				ErrorsDelta:    errorsDelta,
				TotalObjects:   n + 1, // n ES + 1 Store
				// HeapBytesPerObject and GCsPerKObject normalise cost by the number of
				// managed objects so results are comparable across different N values.
				HeapBytesPerObject: float64(heapDelta) / float64(n+1),
				GCsPerKObject:      float64(gcDelta) / float64(n+1) * 1000,
			}

			AddReportEntry("perf-result", result)
			allResults = append(allResults, result)

			GinkgoWriter.Printf(
				"\n[steady] n=%d concurrency=%d refreshInterval=%s observationCycles=%d wall=%.2fs rps=%.1f p50=%.2fms p90=%.2fms p99=%.2fms heap/obj=%.0fB gc/kobj=%.2f errors=%.0f\n",
				n, concurrency, refreshInterval, observationCycles,
				result.WallTimeSec, result.ThroughputRPS,
				result.ReconcileP50ms, result.ReconcileP90ms, result.ReconcileP99ms,
				result.HeapBytesPerObject, result.GCsPerKObject, errorsDelta,
			)
		},
		Entry("N=100", 100),
		Entry("N=500", 500),
		Entry("N=1000", 1000),
		Entry("N=5000", 5000),
	)
})
