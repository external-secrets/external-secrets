/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cpsmetrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

func TestUpdateClusterPushSecretCondition(t *testing.T) {
	// Evacuate the original condition metric labels
	tmpConditionMetricLabels := metrics.ConditionMetricLabels
	defer func() {
		metrics.ConditionMetricLabels = tmpConditionMetricLabels
	}()
	metrics.ConditionMetricLabels = map[string]string{"name": "", "namespace": "", "condition": "", "status": ""}

	name := "test"

	tests := []struct {
		desc           string
		condition      *v1alpha1.PushSecretStatusCondition
		expectedCount  int
		expectedValues []struct {
			labels        prometheus.Labels
			expectedValue float64
		}
	}{
		{
			desc: "ConditionTrue",
			condition: &v1alpha1.PushSecretStatusCondition{
				Type:   v1alpha1.PushSecretReady,
				Status: v1.ConditionTrue,
			},
			expectedValues: []struct {
				labels        prometheus.Labels
				expectedValue float64
			}{
				{
					labels: prometheus.Labels{
						"namespace": "",
						"name":      name,
						"condition": "Ready",
						"status":    "True",
					},
					expectedValue: 1.0,
				},
				{
					labels: prometheus.Labels{
						"namespace": "",
						"name":      name,
						"condition": "Ready",
						"status":    "False",
					},
					expectedValue: 0.0,
				},
			},
		},
		{
			desc: "ConditionFalse",
			condition: &v1alpha1.PushSecretStatusCondition{
				Type:   v1alpha1.PushSecretReady,
				Status: v1.ConditionFalse,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ces := &v1alpha1.ClusterPushSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			}

			// Evacuate the original gauge vec
			tmpGaugeVec := GetGaugeVec(ClusterPushSecretStatusConditionKey)
			defer func() {
				gaugeVecMetrics[ClusterPushSecretStatusConditionKey] = tmpGaugeVec
			}()

			gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Subsystem: "psmetrics",
				Name:      "TestUpdateClusterPushSecretCondition",
			}, []string{"name", "namespace", "condition", "status"})

			gaugeVecMetrics[ClusterPushSecretStatusConditionKey] = gaugeVec
			UpdateClusterPushSecretCondition(ces, test.condition)

			if got := testutil.CollectAndCount(gaugeVec); got != len(test.expectedValues) {
				t.Fatalf("unexpected number of calls: got: %d, expected: %d", got, len(test.expectedValues))
			}

			for i, expected := range test.expectedValues {
				if got := testutil.ToFloat64(gaugeVec.With(expected.labels)); got != expected.expectedValue {
					t.Fatalf("#%d received unexpected gauge value: got: %v, expected: %v", i, got, expected.expectedValue)
				}
			}
		})
	}
}
