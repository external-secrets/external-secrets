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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

func TestUpdatePushSecretCondition(t *testing.T) {
	tmpConditionMetricLabels := metrics.ConditionMetricLabels
	defer func() {
		metrics.ConditionMetricLabels = tmpConditionMetricLabels
	}()
	metrics.ConditionMetricLabels = map[string]string{
		"name": "", "namespace": "", "condition": "", "status": "",
	}

	const (
		name      = "test-ps"
		namespace = "test-ns"
	)

	tests := []struct {
		desc           string
		condition      *esapi.PushSecretStatusCondition
		value          float64
		expectedValues []struct {
			labels        prometheus.Labels
			expectedValue float64
		}
	}{
		{
			// Ready PS must emit exactly one series: {status=False}=0.0.
			desc: "Ready/ConditionTrue emits only status=False=0",
			condition: &esapi.PushSecretStatusCondition{
				Type:   esapi.PushSecretReady,
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
			// Not-ready PS must emit exactly one series: {status=False}=1.0.
			desc: "Ready/ConditionFalse emits only status=False=1",
			condition: &esapi.PushSecretStatusCondition{
				Type:   esapi.PushSecretReady,
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
			ps := &esapi.PushSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}

			tmpGaugeVec := GetGaugeVec(PushSecretStatusConditionKey)
			defer func() {
				gaugeVecMetrics[PushSecretStatusConditionKey] = tmpGaugeVec
			}()

			gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Subsystem: "psmetrics_test",
				Name:      "update_push_secret_condition",
			}, []string{"name", "namespace", "condition", "status"})

			gaugeVecMetrics[PushSecretStatusConditionKey] = gaugeVec
			UpdatePushSecretCondition(ps, test.condition, test.value)

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
