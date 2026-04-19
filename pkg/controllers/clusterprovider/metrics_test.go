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

package clusterprovider

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestUpdateStatusCondition(t *testing.T) {
	tmpGaugeVecMetrics := gaugeVecMetrics
	defer func() {
		gaugeVecMetrics = tmpGaugeVecMetrics
	}()

	conditionGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "clusterprovider",
		Name:      "status_condition_test",
	}, []string{"name", "condition", "status"})
	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		StatusConditionKey: conditionGauge,
	}

	UpdateStatusCondition("aws-shared", "Ready", "False")

	if got := testutil.CollectAndCount(conditionGauge); got != 2 {
		t.Fatalf("unexpected number of condition samples: got %d, want 2", got)
	}
	if got := testutil.ToFloat64(conditionGauge.With(prometheus.Labels{
		"name":      "aws-shared",
		"condition": "Ready",
		"status":    "True",
	})); got != 0 {
		t.Fatalf("unexpected Ready=True value: got %v, want 0", got)
	}
	if got := testutil.ToFloat64(conditionGauge.With(prometheus.Labels{
		"name":      "aws-shared",
		"condition": "Ready",
		"status":    "False",
	})); got != 1 {
		t.Fatalf("unexpected Ready=False value: got %v, want 1", got)
	}
}

func TestRecordReconcileDuration(t *testing.T) {
	tmpGaugeVecMetrics := gaugeVecMetrics
	defer func() {
		gaugeVecMetrics = tmpGaugeVecMetrics
	}()

	durationGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "clusterprovider",
		Name:      "reconcile_duration_test",
	}, []string{"name"})
	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		ClusterProviderReconcileDurationKey: durationGauge,
	}

	RecordReconcileDuration("aws-shared", 1.75)

	if got := testutil.ToFloat64(durationGauge.With(prometheus.Labels{
		"name": "aws-shared",
	})); got != 1.75 {
		t.Fatalf("unexpected reconcile duration: got %v, want 1.75", got)
	}
}
