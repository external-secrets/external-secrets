//go:build perf

package perf

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// PerfResult holds the measurements from one perf scenario run.
type PerfResult struct {
	Plan           string  `json:"plan"`
	Scenario       string  `json:"scenario"`
	NumStores      int     `json:"num_stores"`
	NumESPerStore  int     `json:"num_es_per_store"`
	Concurrency    int     `json:"concurrency"`
	WallTimeSec    float64 `json:"wall_time_sec"`
	ThroughputRPS  float64 `json:"throughput_rps"`
	ReconcileP50ms float64 `json:"reconcile_p50_ms"`
	ReconcileP90ms float64 `json:"reconcile_p90_ms"`
	ReconcileP99ms float64 `json:"reconcile_p99_ms"`
	QueueP50ms     float64 `json:"queue_p50_ms"`
	QueueP90ms     float64 `json:"queue_p90_ms"`
	HeapDeltaMB    float64 `json:"heap_delta_mb"`
	NumGCDelta     uint32  `json:"num_gc_delta"`
	PauseTotalMs   float64 `json:"pause_total_ms"`
	ErrorsDelta    float64 `json:"errors_delta"`
	// Per-object metrics normalise heap and GC cost by the number of reconciled
	// objects so that results are comparable across different N values.
	TotalObjects       int     `json:"total_objects"`
	HeapBytesPerObject float64 `json:"heap_bytes_per_object"`
	GCsPerKObject      float64 `json:"gcs_per_k_object"`
}

// WriteResultsJSON writes results to perf-results-<RFC3339>.json in dir.
func WriteResultsJSON(results []PerfResult, dir string) error {
	name := fmt.Sprintf("perf-results-%s.json", time.Now().UTC().Format("20060102T150405Z"))
	path := filepath.Join(dir, name)
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// PrintResultsTable writes an ASCII table of results to w.
func PrintResultsTable(results []PerfResult, w io.Writer) {
	fmt.Fprintf(w, "\n%-30s %-8s %-8s %-8s %-10s %-10s %-10s %-10s %-10s %-10s %-12s %-10s\n",
		"Scenario", "Stores", "ES/Store", "Workers",
		"Wall(s)", "RPS", "P50(ms)", "P90(ms)", "P99(ms)", "HeapΔ(MB)", "Heap/Obj(B)", "GC/kObj")
	fmt.Fprintf(w, "%s\n", "-----------------------------------------------------------"+
		"-------------------------------------------------------------------")
	for _, r := range results {
		fmt.Fprintf(w, "%-30s %-8d %-8d %-8d %-10.2f %-10.1f %-10.2f %-10.2f %-10.2f %-10.2f %-12.0f %-10.2f\n",
			r.Scenario, r.NumStores, r.NumESPerStore, r.Concurrency,
			r.WallTimeSec, r.ThroughputRPS, r.ReconcileP50ms, r.ReconcileP90ms, r.ReconcileP99ms,
			r.HeapDeltaMB, r.HeapBytesPerObject, r.GCsPerKObject)
	}
	fmt.Fprintln(w)
}
