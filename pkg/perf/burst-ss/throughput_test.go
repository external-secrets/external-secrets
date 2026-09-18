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

package burst_ss

import (
	"context"
	"fmt"
	"time"

	perf "github.com/external-secrets/external-secrets/pkg/perf"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SecretStore reconciler throughput", func() {
	It("reconciles PERF_NUM_STORES SecretStores across as many namespaces", func() {
		ctx := context.Background()
		numStores := perf.EnvOrInt("PERF_NUM_STORES", 10_000)
		concurrency := perf.EnvOrInt("PERF_STORE_CONCURRENCY", 16)
		scenario := fmt.Sprintf("stores-n%d-c%d", numStores, concurrency)

		By(fmt.Sprintf("creating %d namespaces", numStores))
		namespaces, err := perf.CreateNamespaces(ctx, k8sClient, numStores, "perf-store")
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("creating %d SecretStores", numStores))
		before := perf.Snapshot("secretstore")
		start := time.Now()
		Expect(perf.CreateSecretStores(ctx, k8sClient, namespaces, fakeProviderSpec())).To(Succeed())

		By("waiting for all stores to reach Ready")
		_, err = perf.WaitAllStoresReady(ctx, k8sClient, namespaces, 30*time.Minute)
		Expect(err).ToNot(HaveOccurred())
		wallTime := time.Since(start)

		after := perf.Snapshot("secretstore")
		_, errorsDelta, heapDelta, gcDelta := perf.DiffSnapshots(before, after)

		result := perf.PerfResult{
			Plan:               "stores",
			Scenario:           scenario,
			NumStores:          numStores,
			NumESPerStore:      0,
			Concurrency:        concurrency,
			WallTimeSec:        wallTime.Seconds(),
			ThroughputRPS:      float64(numStores) / wallTime.Seconds(),
			ReconcileP50ms:     perf.HistogramPercentile(after.ReconcileTime, 0.50) * 1000,
			ReconcileP90ms:     perf.HistogramPercentile(after.ReconcileTime, 0.90) * 1000,
			ReconcileP99ms:     perf.HistogramPercentile(after.ReconcileTime, 0.99) * 1000,
			QueueP50ms:         perf.HistogramPercentile(after.QueueDuration, 0.50) * 1000,
			QueueP90ms:         perf.HistogramPercentile(after.QueueDuration, 0.90) * 1000,
			HeapDeltaMB:        float64(heapDelta) / (1024 * 1024),
			NumGCDelta:         gcDelta,
			PauseTotalMs:       float64(after.PauseTotalNs-before.PauseTotalNs) / 1e6,
			ErrorsDelta:        errorsDelta,
			TotalObjects:       numStores,
			HeapBytesPerObject: float64(heapDelta) / float64(numStores),
			GCsPerKObject:      float64(gcDelta) / float64(numStores) * 1000,
		}

		AddReportEntry("perf-result", result)
		allResults = append(allResults, result)

		GinkgoWriter.Printf(
			"\n[stores] n=%d concurrency=%d wall=%.2fs rps=%.1f p50=%.2fms p90=%.2fms p99=%.2fms heap/obj=%.0fB gc/kobj=%.2f errors=%.0f\n",
			numStores, concurrency, result.WallTimeSec, result.ThroughputRPS,
			result.ReconcileP50ms, result.ReconcileP90ms, result.ReconcileP99ms,
			result.HeapBytesPerObject, result.GCsPerKObject, errorsDelta,
		)
	})
})
