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

package fanout

import (
	"context"
	"fmt"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	perf "github.com/external-secrets/external-secrets/pkg/perf"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Fan-out: N stores * M ExternalSecrets", func() {
	It("reconciles all ESes across all stores", func() {
		ctx := context.Background()
		numStores := perf.EnvOrInt("PERF_NUM_STORES", 100)
		esPerStore := perf.EnvOrInt("PERF_ES_PER_STORE", 100)
		esConcurrency := perf.EnvOrInt("PERF_ES_CONCURRENCY", 16)
		storeConcurrency := perf.EnvOrInt("PERF_STORE_CONCURRENCY", 16)
		total := numStores * esPerStore
		scenario := fmt.Sprintf("fanout-s%d-e%d-sc%d-ec%d", numStores, esPerStore, storeConcurrency, esConcurrency)

		By(fmt.Sprintf("creating %d namespaces", numStores))
		namespaces, err := perf.CreateNamespaces(ctx, k8sClient, numStores, "perf-fanout")
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("creating %d SecretStores", numStores))
		Expect(perf.CreateSecretStores(ctx, k8sClient, namespaces, fakeProviderSpec())).To(Succeed())

		By("waiting for all stores to reach Ready before creating ExternalSecrets")
		_, err = perf.WaitAllStoresReady(ctx, k8sClient, namespaces, 15*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("creating %d ExternalSecrets (%d per store)", total, esPerStore))
		storeRef := esv1.SecretStoreRef{Name: "perf-store"}
		for _, ns := range namespaces {
			Expect(perf.CreateExternalSecrets(ctx, k8sClient, ns, esPerStore, storeRef, 2*time.Hour)).To(Succeed())
		}

		By("measuring time for all ExternalSecrets to reach Ready")
		before := perf.Snapshot("externalsecret")
		start := time.Now()

		_, err = perf.WaitAllReadyMultiNS(ctx, k8sClient, namespaces, esPerStore, 60*time.Minute)
		Expect(err).ToNot(HaveOccurred())
		wallTime := time.Since(start)

		after := perf.Snapshot("externalsecret")
		_, errorsDelta, heapDelta, gcDelta := perf.DiffSnapshots(before, after)

		// total ES + one Store per namespace
		totalObjects := total + numStores
		result := perf.PerfResult{
			Plan:               "fanout",
			Scenario:           scenario,
			NumStores:          numStores,
			NumESPerStore:      esPerStore,
			Concurrency:        esConcurrency,
			WallTimeSec:        wallTime.Seconds(),
			ThroughputRPS:      float64(total) / wallTime.Seconds(),
			ReconcileP50ms:     perf.HistogramPercentile(after.ReconcileTime, 0.50) * 1000,
			ReconcileP90ms:     perf.HistogramPercentile(after.ReconcileTime, 0.90) * 1000,
			ReconcileP99ms:     perf.HistogramPercentile(after.ReconcileTime, 0.99) * 1000,
			QueueP50ms:         perf.HistogramPercentile(after.QueueDuration, 0.50) * 1000,
			QueueP90ms:         perf.HistogramPercentile(after.QueueDuration, 0.90) * 1000,
			HeapDeltaMB:        float64(heapDelta) / (1024 * 1024),
			NumGCDelta:         gcDelta,
			PauseTotalMs:       float64(after.PauseTotalNs-before.PauseTotalNs) / 1e6,
			ErrorsDelta:        errorsDelta,
			TotalObjects:       totalObjects,
			HeapBytesPerObject: float64(heapDelta) / float64(totalObjects),
			GCsPerKObject:      float64(gcDelta) / float64(totalObjects) * 1000,
		}

		AddReportEntry("perf-result", result)
		allResults = append(allResults, result)

		GinkgoWriter.Printf(
			"\n[fanout] stores=%d es/store=%d total=%d wall=%.2fs rps=%.1f p50=%.2fms p90=%.2fms p99=%.2fms heapΔ=%.1fMB heap/obj=%.0fB gc/kobj=%.2f errors=%.0f\n",
			numStores, esPerStore, total,
			result.WallTimeSec, result.ThroughputRPS,
			result.ReconcileP50ms, result.ReconcileP90ms, result.ReconcileP99ms,
			result.HeapDeltaMB, result.HeapBytesPerObject, result.GCsPerKObject, errorsDelta,
		)
	})
})
