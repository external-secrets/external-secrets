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

package provider

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

func TestUpdateStatusCondition(t *testing.T) {
	tmpConditionMetricLabels := ctrlmetrics.ConditionMetricLabels
	defer func() {
		ctrlmetrics.ConditionMetricLabels = tmpConditionMetricLabels
	}()
	ctrlmetrics.ConditionMetricLabels = map[string]string{"name": "", "namespace": "", "condition": "", "status": ""}

	tmpGaugeVecMetrics := gaugeVecMetrics
	defer func() {
		gaugeVecMetrics = tmpGaugeVecMetrics
	}()

	conditionGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "provider",
		Name:      "status_condition_test",
	}, []string{"name", "namespace", "condition", "status"})
	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		StatusConditionKey: conditionGauge,
	}

	UpdateStatusCondition("aws-prod", "tenant-a", map[string]string{"team": "platform"}, "Ready", "True")

	if got := testutil.CollectAndCount(conditionGauge); got != 2 {
		t.Fatalf("unexpected number of condition samples: got %d, want 2", got)
	}
	if got := testutil.ToFloat64(conditionGauge.With(prometheus.Labels{
		"name":      "aws-prod",
		"namespace": "tenant-a",
		"condition": "Ready",
		"status":    "True",
	})); got != 1 {
		t.Fatalf("unexpected Ready=True value: got %v, want 1", got)
	}
	if got := testutil.ToFloat64(conditionGauge.With(prometheus.Labels{
		"name":      "aws-prod",
		"namespace": "tenant-a",
		"condition": "Ready",
		"status":    "False",
	})); got != 0 {
		t.Fatalf("unexpected Ready=False value: got %v, want 0", got)
	}
}

func TestRecordReconcileDuration(t *testing.T) {
	tmpNonConditionMetricLabels := ctrlmetrics.NonConditionMetricLabels
	defer func() {
		ctrlmetrics.NonConditionMetricLabels = tmpNonConditionMetricLabels
	}()
	ctrlmetrics.NonConditionMetricLabels = map[string]string{"name": "", "namespace": ""}

	tmpGaugeVecMetrics := gaugeVecMetrics
	defer func() {
		gaugeVecMetrics = tmpGaugeVecMetrics
	}()

	durationGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "provider",
		Name:      "reconcile_duration_test",
	}, []string{"name", "namespace"})
	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		ProviderReconcileDurationKey: durationGauge,
	}

	RecordReconcileDuration("aws-prod", "tenant-a", map[string]string{"team": "platform"}, 2.5)

	if got := testutil.ToFloat64(durationGauge.With(prometheus.Labels{
		"name":      "aws-prod",
		"namespace": "tenant-a",
	})); got != 2.5 {
		t.Fatalf("unexpected reconcile duration: got %v, want 2.5", got)
	}
}
