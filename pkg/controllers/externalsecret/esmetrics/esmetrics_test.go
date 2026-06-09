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

package esmetrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

func TestUpdateExternalSecretCondition(t *testing.T) {
	tmpConditionMetricLabels := metrics.ConditionMetricLabels
	defer func() {
		metrics.ConditionMetricLabels = tmpConditionMetricLabels
	}()
	metrics.ConditionMetricLabels = map[string]string{
		"name": "", "namespace": "", "condition": "", "status": "",
	}

	const (
		name      = "test-es"
		namespace = "test-ns"
	)

	tests := []struct {
		desc           string
		condition      *esv1.ExternalSecretStatusCondition
		value          float64
		expectedValues []struct {
			labels        prometheus.Labels
			expectedValue float64
		}
	}{
		{
			// Ready ES must emit exactly one series: {status=False}=0.0.
			desc: "Ready/ConditionTrue emits only status=False=0",
			condition: &esv1.ExternalSecretStatusCondition{
				Type:   esv1.ExternalSecretReady,
				Status: v1.ConditionTrue,
			},
			value: 1.0,
			expectedValues: []struct {
				labels        prometheus.Labels
				expectedValue float64
			}{
				{
					labels: prometheus.Labels{
						"namespace": namespace,
						"name":      name,
						"condition": "Ready",
						"status":    "False",
					},
					expectedValue: 0.0,
				},
			},
		},
		{
			// Not-ready ES must emit exactly one series: {status=False}=1.0.
			desc: "Ready/ConditionFalse emits only status=False=1",
			condition: &esv1.ExternalSecretStatusCondition{
				Type:   esv1.ExternalSecretReady,
				Status: v1.ConditionFalse,
			},
			value: 1.0,
			expectedValues: []struct {
				labels        prometheus.Labels
				expectedValue float64
			}{
				{
					labels: prometheus.Labels{
						"namespace": namespace,
						"name":      name,
						"condition": "Ready",
						"status":    "False",
					},
					expectedValue: 1.0,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}

			tmpGaugeVec := GetGaugeVec(ExternalSecretStatusConditionKey)
			defer func() {
				gaugeVecMetrics[ExternalSecretStatusConditionKey] = tmpGaugeVec
			}()

			gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Subsystem: "esmetrics_test",
				Name:      "update_external_secret_condition",
			}, []string{"name", "namespace", "condition", "status"})

			gaugeVecMetrics[ExternalSecretStatusConditionKey] = gaugeVec
			UpdateExternalSecretCondition(es, test.condition, test.value)

			if got := testutil.CollectAndCount(gaugeVec); got != len(test.expectedValues) {
				t.Fatalf("unexpected metric count: got %d, expected %d",
					got, len(test.expectedValues))
			}

			for i, expected := range test.expectedValues {
				if got := testutil.ToFloat64(gaugeVec.With(expected.labels)); got != expected.expectedValue {
					t.Fatalf("#%d unexpected gauge value: got %v, expected %v",
						i, got, expected.expectedValue)
				}
			}
		})
	}
}

// TestUpdateExternalSecretConditionDeletedCleanup verifies that updating
// with type=Deleted removes the condition=Ready series.
func TestUpdateExternalSecretConditionDeletedCleanup(t *testing.T) {
	tmpConditionMetricLabels := metrics.ConditionMetricLabels
	defer func() {
		metrics.ConditionMetricLabels = tmpConditionMetricLabels
	}()
	metrics.ConditionMetricLabels = map[string]string{
		"name": "", "namespace": "", "condition": "", "status": "",
	}

	const (
		name      = "test-es"
		namespace = "test-ns"
	)

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	tmpGaugeVec := GetGaugeVec(ExternalSecretStatusConditionKey)
	defer func() {
		gaugeVecMetrics[ExternalSecretStatusConditionKey] = tmpGaugeVec
	}()

	gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "esmetrics_test",
		Name:      "update_external_secret_condition_deleted_cleanup",
	}, []string{"name", "namespace", "condition", "status"})
	gaugeVecMetrics[ExternalSecretStatusConditionKey] = gaugeVec

	// Seed a Ready metric.
	UpdateExternalSecretCondition(es, &esv1.ExternalSecretStatusCondition{
		Type:   esv1.ExternalSecretReady,
		Status: v1.ConditionTrue,
	}, 1.0)

	if got := testutil.CollectAndCount(gaugeVec); got != 1 {
		t.Fatalf("pre-delete: expected 1 series, got %d", got)
	}

	// Deleted condition must remove the Ready series and emit only Deleted.
	UpdateExternalSecretCondition(es, &esv1.ExternalSecretStatusCondition{
		Type:   esv1.ExternalSecretDeleted,
		Status: v1.ConditionTrue,
	}, 1.0)

	deletedLabels := prometheus.Labels{
		"namespace": namespace,
		"name":      name,
		"condition": "Deleted",
		"status":    "True",
	}

	// Only the Deleted series should remain; count proves Ready was cleaned up.
	if got := testutil.CollectAndCount(gaugeVec); got != 1 {
		t.Fatalf("post-delete: expected 1 series, got %d", got)
	}
	if got := testutil.ToFloat64(gaugeVec.With(deletedLabels)); got != 1.0 {
		t.Fatalf("Deleted series: got %v, expected 1.0", got)
	}
}
