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
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

const (
	ClusterPushSecretSubsystem            = "clusterpushsecret"
	ClusterPushSecretReconcileDurationKey = "reconcile_duration"
	ClusterPushSecretStatusConditionKey   = "status_condition"
)

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// SetUpMetrics is called at the root to set-up the metric logic using the
// config flags provided.
func SetUpMetrics() {
	ClusterPushSecretReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ClusterPushSecretSubsystem,
		Name:      ClusterPushSecretReconcileDurationKey,
		Help:      "The duration time to reconcile the Cluster Push Secret",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	ClusterPushSecretCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ClusterPushSecretSubsystem,
		Name:      ClusterPushSecretStatusConditionKey,
		Help:      "The status condition of a specific Cluster Push Secret",
	}, ctrlmetrics.ConditionMetricLabelNames)

	metrics.Registry.MustRegister(ClusterPushSecretReconcileDuration, ClusterPushSecretCondition)

	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		ClusterPushSecretStatusConditionKey:   ClusterPushSecretCondition,
		ClusterPushSecretReconcileDurationKey: ClusterPushSecretReconcileDuration,
	}
}

func GetGaugeVec(key string) *prometheus.GaugeVec {
	return gaugeVecMetrics[key]
}

func UpdateClusterPushSecretCondition(ces *v1alpha1.ClusterPushSecret, condition *v1alpha1.PushSecretStatusCondition) {
	if condition.Status != v1.ConditionTrue {
		// This should not happen
		return
	}

	cesInfo := make(map[string]string)
	cesInfo["name"] = ces.Name
	for k, v := range ces.Labels {
		cesInfo[k] = v
	}
	conditionLabels := ctrlmetrics.RefineConditionMetricLabels(cesInfo)
	ClusterPushSecretCondition := GetGaugeVec(ClusterPushSecretStatusConditionKey)

	theOtherStatus := v1.ConditionFalse
	if condition.Status == v1.ConditionFalse {
		theOtherStatus = v1.ConditionTrue
	}

	ClusterPushSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
		map[string]string{
			"condition": string(condition.Type),
			"status":    string(condition.Status),
		})).Set(1)
	ClusterPushSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
		map[string]string{
			"condition": string(condition.Type),
			"status":    string(theOtherStatus),
		})).Set(0)
}

// RemoveMetrics deletes all metrics published by the resource.
func RemoveMetrics(namespace, name string) {
	for _, gaugeVecMetric := range gaugeVecMetrics {
		gaugeVecMetric.DeletePartialMatch(
			map[string]string{
				"namespace": namespace,
				"name":      name,
			},
		)
	}
}
