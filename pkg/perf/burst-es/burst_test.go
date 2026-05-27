//go:build perf

package burst_es

import (
	"context"
	"fmt"
	"time"

	perf "github.com/external-secrets/external-secrets/pkg/perf"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExternalSecret burst reconciliation", func() {
	DescribeTable("N ExternalSecrets on 1 SecretStore",
		func(n int) {
			ctx := context.Background()
			concurrency := perf.EnvOrInt("PERF_ES_CONCURRENCY", 4)
			scenario := fmt.Sprintf("burst-n%d-c%d", n, concurrency)

			before := perf.Snapshot("externalsecret")

			By(fmt.Sprintf("creating %d ExternalSecrets", n))
			Expect(perf.CreateExternalSecrets(ctx, k8sClient, testNamespace, n, storeRef, time.Hour)).To(Succeed())

			By("waiting for all to reach Ready")
			wallTime, err := perf.WaitAllReady(ctx, k8sClient, testNamespace, n, 30*time.Minute) // I need more than 45 minutes for N = 50k.
			Expect(err).ToNot(HaveOccurred())

			after := perf.Snapshot("externalsecret")
			_, errorsDelta, heapDelta, gcDelta := perf.DiffSnapshots(before, after)

			result := perf.PerfResult{
				Plan:               "burst",
				Scenario:           scenario,
				NumStores:          1,
				NumESPerStore:      n,
				Concurrency:        concurrency,
				WallTimeSec:        wallTime.Seconds(),
				ThroughputRPS:      float64(n) / wallTime.Seconds(),
				ReconcileP50ms:     perf.HistogramPercentile(after.ReconcileTime, 0.50) * 1000,
				ReconcileP90ms:     perf.HistogramPercentile(after.ReconcileTime, 0.90) * 1000,
				ReconcileP99ms:     perf.HistogramPercentile(after.ReconcileTime, 0.99) * 1000,
				QueueP50ms:         perf.HistogramPercentile(after.QueueDuration, 0.50) * 1000,
				QueueP90ms:         perf.HistogramPercentile(after.QueueDuration, 0.90) * 1000,
				HeapDeltaMB:        float64(heapDelta) / (1024 * 1024),
				NumGCDelta:         gcDelta,
				PauseTotalMs:       float64(after.PauseTotalNs-before.PauseTotalNs) / 1e6,
				ErrorsDelta:        errorsDelta,
				TotalObjects:       n + 1, // n ES + 1 Store
				HeapBytesPerObject: float64(heapDelta) / float64(n+1),
				GCsPerKObject:      float64(gcDelta) / float64(n+1) * 1000,
			}

			AddReportEntry("perf-result", result)
			allResults = append(allResults, result)

			GinkgoWriter.Printf(
				"\n[burst] n=%d concurrency=%d wall=%.2fs rps=%.1f p50=%.2fms p90=%.2fms p99=%.2fms heap/obj=%.0fB gc/kobj=%.2f errors=%.0f\n",
				n, concurrency, result.WallTimeSec, result.ThroughputRPS,
				result.ReconcileP50ms, result.ReconcileP90ms, result.ReconcileP99ms,
				result.HeapBytesPerObject, result.GCsPerKObject, errorsDelta,
			)
		},
		Entry("N=100", 100),
		Entry("N=500", 500),
		Entry("N=1000", 1000),
		Entry("N=5000", 5000),
		// Entry("N=10000", 10000),
		// Entry("N=50000", 50000), // This one seems off, commented by default. Needs analysis!
	)
})
