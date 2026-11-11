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
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

func TestRemoveMetrics(t *testing.T) {
	ctrlmetrics.SetUpLabelNames(false)

	tmpGaugeVecMetrics := gaugeVecMetrics
	defer func() { gaugeVecMetrics = tmpGaugeVecMetrics }()

	condition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: PushSecretSubsystem,
		Name:      PushSecretStatusConditionKey,
	}, ctrlmetrics.ConditionMetricLabelNames)
	duration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: PushSecretSubsystem,
		Name:      PushSecretReconcileDurationKey,
	}, ctrlmetrics.NonConditionMetricLabelNames)
	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		PushSecretStatusConditionKey:   condition,
		PushSecretReconcileDurationKey: duration,
	}

	deleted := &esapi.PushSecret{ObjectMeta: metav1.ObjectMeta{Name: "deleted", Namespace: "ns"}}
	kept := &esapi.PushSecret{ObjectMeta: metav1.ObjectMeta{Name: "kept", Namespace: "ns"}}
	for _, ps := range []*esapi.PushSecret{deleted, kept} {
		UpdatePushSecretCondition(ps, &esapi.PushSecretStatusCondition{
			Type:   esapi.PushSecretReady,
			Status: v1.ConditionFalse,
		}, 1.0)
		duration.With(prometheus.Labels{"name": ps.Name, "namespace": ps.Namespace}).Set(1)
	}

	if got := testutil.CollectAndCount(condition); got != 4 {
		t.Fatalf("condition series before removal = %d, want 4", got)
	}
	if got := testutil.CollectAndCount(duration); got != 2 {
		t.Fatalf("duration series before removal = %d, want 2", got)
	}

	RemoveMetrics(deleted.Namespace, deleted.Name)

	if got := testutil.CollectAndCount(condition); got != 2 {
		t.Errorf("condition series after removal = %d, want 2", got)
	}
	if got := testutil.CollectAndCount(duration); got != 1 {
		t.Errorf("duration series after removal = %d, want 1", got)
	}
	if got := testutil.ToFloat64(condition.With(prometheus.Labels{
		"name": "kept", "namespace": "ns", "condition": "Ready", "status": "False",
	})); got != 1 {
		t.Errorf("kept PushSecret Ready=False = %v, want 1", got)
	}
}
