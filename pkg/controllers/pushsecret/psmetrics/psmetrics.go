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

package psmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

const (
	PushSecretSubsystem            = "pushsecret"
	PushSecretReconcileDurationKey = "reconcile_duration"
	PushSecretStatusConditionKey   = "status_condition"
)

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// SetUpMetrics is called at the root to set-up the metric logic using the
// config flags provided.
func SetUpMetrics() {
	pushSecretCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: PushSecretSubsystem,
		Name:      PushSecretStatusConditionKey,
		Help:      "The status condition of a specific Push Secret",
	}, ctrlmetrics.ConditionMetricLabelNames)

	pushSecretReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: PushSecretSubsystem,
		Name:      PushSecretReconcileDurationKey,
		Help:      "The duration time to reconcile the Push Secret",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	metrics.Registry.MustRegister(pushSecretReconcileDuration, pushSecretCondition)

	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		PushSecretStatusConditionKey:   pushSecretCondition,
		PushSecretReconcileDurationKey: pushSecretReconcileDuration,
	}
}

func UpdatePushSecretCondition(ps *esapi.PushSecret, condition *esapi.PushSecretStatusCondition, value float64) {
	psInfo := make(map[string]string)
	psInfo["name"] = ps.Name
	psInfo["namespace"] = ps.Namespace
	for k, v := range ps.Labels {
		psInfo[k] = v
	}
	conditionLabels := ctrlmetrics.RefineConditionMetricLabels(psInfo)
	pushSecretCondition := GetGaugeVec(PushSecretStatusConditionKey)

	switch condition.Type {
	case esapi.PushSecretReady:
		// Toggle opposite Status to 0
		switch condition.Status {
		case v1.ConditionFalse:
			pushSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esapi.PushSecretReady),
					"status":    string(v1.ConditionTrue),
				})).Set(0)
		case v1.ConditionTrue:
			pushSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esapi.PushSecretReady),
					"status":    string(v1.ConditionFalse),
				})).Set(0)
		case v1.ConditionUnknown:
			break
		default:
			break
		}

	default:
		break
	}

	pushSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
		map[string]string{
			"condition": string(condition.Type),
			"status":    string(condition.Status),
		})).Set(value)
}

func GetGaugeVec(key string) *prometheus.GaugeVec {
	return gaugeVecMetrics[key]
}
