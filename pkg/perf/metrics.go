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

package perf

import (
	"runtime"

	dto "github.com/prometheus/client_model/go"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// MetricSnapshot captures a point-in-time view of controller performance metrics.
type MetricSnapshot struct {
	// From controller-runtime Prometheus registry
	ReconcileTime   *dto.Histogram // controller_runtime_reconcile_time_seconds (highest-count result for given controller)
	QueueDuration   *dto.Histogram // workqueue_queue_duration_seconds
	WorkDuration    *dto.Histogram // workqueue_work_duration_seconds
	ReconcileTotal  float64        // controller_runtime_reconcile_total (all results)
	ReconcileErrors float64        // controller_runtime_reconcile_total{result="error"}
	// From runtime.ReadMemStats
	HeapAllocBytes uint64
	NumGC          uint32
	PauseTotalNs   uint64
}

// Snapshot gathers a MetricSnapshot for the named controller.
// controllerName must match the controller label used by controller-runtime
// (typically the lowercased Kind, e.g. "externalsecret" or "secretstore").
func Snapshot(controllerName string) MetricSnapshot {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	snap := MetricSnapshot{
		HeapAllocBytes: ms.HeapAlloc,
		NumGC:          ms.NumGC,
		PauseTotalNs:   ms.PauseTotalNs,
	}

	mfs, err := ctrlmetrics.Registry.Gather()
	if err != nil {
		return snap
	}

	for _, mf := range mfs {
		switch mf.GetName() {
		case "controller_runtime_reconcile_time_seconds":
			// Pick the histogram series with the most samples (usually the dominant result label).
			var best uint64
			for _, m := range mf.GetMetric() {
				if !hasLabel(m, "controller", controllerName) {
					continue
				}
				if h := m.GetHistogram(); h.GetSampleCount() > best {
					best = h.GetSampleCount()
					snap.ReconcileTime = h
				}
			}

		case "workqueue_queue_duration_seconds":
			for _, m := range mf.GetMetric() {
				if hasLabel(m, "name", controllerName) {
					snap.QueueDuration = m.GetHistogram()
				}
			}

		case "workqueue_work_duration_seconds":
			for _, m := range mf.GetMetric() {
				if hasLabel(m, "name", controllerName) {
					snap.WorkDuration = m.GetHistogram()
				}
			}

		case "controller_runtime_reconcile_total":
			for _, m := range mf.GetMetric() {
				if !hasLabel(m, "controller", controllerName) {
					continue
				}
				snap.ReconcileTotal += m.GetCounter().GetValue()
				if hasLabel(m, "result", "error") {
					snap.ReconcileErrors += m.GetCounter().GetValue()
				}
			}
		}
	}

	return snap
}

// HistogramPercentile returns the approximate p-th percentile value (p in [0,1]) from a dto.Histogram
// using linear interpolation between bucket boundaries. Returns 0 if h is nil or empty.
func HistogramPercentile(h *dto.Histogram, p float64) float64 {
	if h == nil || h.GetSampleCount() == 0 {
		return 0
	}
	target := p * float64(h.GetSampleCount())
	buckets := h.GetBucket()
	var prevCount float64
	var prevBound float64
	for _, b := range buckets {
		count := float64(b.GetCumulativeCount())
		if count >= target {
			if count == prevCount {
				return b.GetUpperBound()
			}
			fraction := (target - prevCount) / (count - prevCount)
			return prevBound + fraction*(b.GetUpperBound()-prevBound)
		}
		prevCount = count
		prevBound = b.GetUpperBound()
	}
	// All samples are in the +Inf bucket; return the mean as a best-effort.
	return h.GetSampleSum() / float64(h.GetSampleCount())
}

// DiffSnapshots returns the per-field deltas between two snapshots.
func DiffSnapshots(before, after MetricSnapshot) (reconcileDelta, errorsDelta float64, heapDeltaBytes uint64, gcDelta uint32) {
	reconcileDelta = after.ReconcileTotal - before.ReconcileTotal
	errorsDelta = after.ReconcileErrors - before.ReconcileErrors
	if after.HeapAllocBytes > before.HeapAllocBytes {
		heapDeltaBytes = after.HeapAllocBytes - before.HeapAllocBytes
	}
	gcDelta = after.NumGC - before.NumGC
	return
}

func hasLabel(m *dto.Metric, key, value string) bool {
	for _, lp := range m.GetLabel() {
		if lp.GetName() == key && lp.GetValue() == value {
			return true
		}
	}
	return false
}
