/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSumMetricValues(t *testing.T) {
	metrics := MetricsMap{
		"grpc_pool_connections_total": {
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-a"}, Value: 1},
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-a"}, Value: 2},
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-b"}, Value: 4},
		},
	}

	got := SumMetricValues(metrics, "grpc_pool_connections_total", map[string]string{"address": "provider-a"})
	if got != 3 {
		t.Fatalf("expected sum 3, got %v", got)
	}
}

func TestCountMetricSamples(t *testing.T) {
	metrics := MetricsMap{
		"grpc_pool_connections_total": {
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-a"}, Value: 1},
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-a"}, Value: 2},
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-b"}, Value: 4},
		},
	}

	got := CountMetricSamples(metrics, "grpc_pool_connections_total", map[string]string{"address": "provider-a"})
	if got != 2 {
		t.Fatalf("expected count 2, got %d", got)
	}
}

func TestGetMetricValueMatchesSubsetOfLabels(t *testing.T) {
	metrics := MetricsMap{
		"grpc_pool_connections_total": {
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-a", "state": "idle"}, Value: 2},
			{Name: "grpc_pool_connections_total", Labels: map[string]string{"address": "provider-b", "state": "active"}, Value: 4},
		},
	}

	got, found := GetMetricValue(metrics, "grpc_pool_connections_total", map[string]string{"address": "provider-a"})
	if !found {
		t.Fatalf("expected matching metric sample")
	}
	if got != 2 {
		t.Fatalf("expected value 2, got %v", got)
	}
}

func TestWaitForMetricHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WaitForMetric(ctx, func() (MetricsMap, error) {
		return MetricsMap{}, nil
	}, "grpc_pool_connections_total", map[string]string{"address": "provider-a"}, 1, 10*time.Second)
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
}

func TestParsePrometheusMetricsParsesEscapedLabels(t *testing.T) {
	body := "grpc_pool_connections_total{address=\"provider-\\\"a\\\"\",state=\"idle\\nstate\"} 2\n"

	metrics, err := parsePrometheusMetrics(body)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	got, found := GetMetricValue(metrics, "grpc_pool_connections_total", map[string]string{
		"address": "provider-\"a\"",
		"state":   "idle\nstate",
	})
	if !found {
		t.Fatalf("expected escaped metric labels to match parsed sample, got metrics %#v", metrics)
	}
	if got != 2 {
		t.Fatalf("expected value 2, got %v", got)
	}
}
