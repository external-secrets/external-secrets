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

package cesmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

const (
	ClusterExternalSecretSubsystem            = "clusterexternalsecret"
	ClusterExternalSecretReconcileDurationKey = "reconcile_duration"
	ClusterExternalSecretStatusConditionKey   = "status_condition"
)

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// SetUpMetrics is called at the root to set-up the metric logic using the
// config flags provided.
func SetUpMetrics() {
	clusterExternalSecretReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ClusterExternalSecretSubsystem,
		Name:      ClusterExternalSecretReconcileDurationKey,
		Help:      "The duration time to reconcile the Cluster External Secret",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	clusterExternalSecretCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ClusterExternalSecretSubsystem,
		Name:      ClusterExternalSecretStatusConditionKey,
		Help:      "The status condition of a specific Cluster External Secret",
	}, ctrlmetrics.ConditionMetricLabelNames)

	metrics.Registry.MustRegister(clusterExternalSecretReconcileDuration, clusterExternalSecretCondition)

	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		ClusterExternalSecretStatusConditionKey:   clusterExternalSecretCondition,
		ClusterExternalSecretReconcileDurationKey: clusterExternalSecretReconcileDuration,
	}
}

func GetGaugeVec(key string) *prometheus.GaugeVec {
	return gaugeVecMetrics[key]
}

func UpdateClusterExternalSecretCondition(ces *esv1beta1.ClusterExternalSecret, condition *esv1beta1.ClusterExternalSecretStatusCondition) {
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
	clusterExternalSecretCondition := GetGaugeVec(ClusterExternalSecretStatusConditionKey)

	targetConditionType := condition.Type
	var theOtherConditionTypes []esv1beta1.ClusterExternalSecretConditionType
	for _, ct := range []esv1beta1.ClusterExternalSecretConditionType{
		esv1beta1.ClusterExternalSecretReady,
		esv1beta1.ClusterExternalSecretPartiallyReady,
		esv1beta1.ClusterExternalSecretNotReady,
	} {
		if ct != targetConditionType {
			theOtherConditionTypes = append(theOtherConditionTypes, ct)
		}
	}

	// Set the target condition metric
	clusterExternalSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
		map[string]string{
			"condition": string(targetConditionType),
			"status":    string(v1.ConditionTrue),
		})).Set(1)
	clusterExternalSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
		map[string]string{
			"condition": string(targetConditionType),
			"status":    string(v1.ConditionFalse),
		})).Set(0)

	// Remove the other condition metrics
	for _, ct := range theOtherConditionTypes {
		clusterExternalSecretCondition.Delete(ctrlmetrics.RefineLabels(conditionLabels,
			map[string]string{
				"condition": string(ct),
				"status":    string(v1.ConditionFalse),
			}))
		clusterExternalSecretCondition.Delete(ctrlmetrics.RefineLabels(conditionLabels,
			map[string]string{
				"condition": string(ct),
				"status":    string(v1.ConditionTrue),
			}))
	}
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
