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

package psmetrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	v1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

func TestRemoveMetrics(t *testing.T) {
	tmpConditionMetricLabels := metrics.ConditionMetricLabels
	tmpGaugeVecMetrics := gaugeVecMetrics
	defer func() {
		metrics.ConditionMetricLabels = tmpConditionMetricLabels
		gaugeVecMetrics = tmpGaugeVecMetrics
	}()
	metrics.ConditionMetricLabels = map[string]string{"name": "", "namespace": "", "condition": "", "status": ""}

	condition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "psmetrics",
		Name:      "TestRemoveMetricsCondition",
	}, []string{"name", "namespace", "condition", "status"})
	duration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "psmetrics",
		Name:      "TestRemoveMetricsDuration",
	}, []string{"name", "namespace"})
	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		PushSecretStatusConditionKey:   condition,
		PushSecretReconcileDurationKey: duration,
	}

	notReady := &esapi.PushSecretStatusCondition{Type: esapi.PushSecretReady, Status: v1.ConditionFalse}
	for _, name := range []string{"deleted", "kept"} {
		ps := &esapi.PushSecret{Name: name, Namespace: "default"}
		UpdatePushSecretCondition(ps, notReady, 1.0)
		duration.With(prometheus.Labels{"name": name, "namespace": "default"}).Set(1)
	}

	RemoveMetrics("default", "deleted")

	if got := testutil.CollectAndCount(condition); got != 2 {
		t.Fatalf("condition series: got %d, expected the 2 of the kept PushSecret", got)
	}
	if got := testutil.CollectAndCount(duration); got != 1 {
		t.Fatalf("duration series: got %d, expected the 1 of the kept PushSecret", got)
	}
	kept := prometheus.Labels{"name": "kept", "namespace": "default", "condition": "Ready", "status": "False"}
	if got := testutil.ToFloat64(condition.With(kept)); got != 1 {
		t.Fatalf("kept PushSecret's Ready=False: got %v, expected 1", got)
	}
}
