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
	"maps"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

// updatePushSecretConditionDeprecated emits the legacy dual status_condition
// series for the Ready condition: both {status="True"} and {status="False"} are
// set, as exact inverses. It is reached only when the operator runs with
// --use-deprecated-status-condition, as a migration fallback for dashboards and
// alert rules not yet moved to the single {status="False"} series.
//
// Deprecated: this whole file is slated for removal in v3 together with the flag
// and the dispatch branch in UpdatePushSecretCondition.
func updatePushSecretConditionDeprecated(ps *esapi.PushSecret, condition *esapi.PushSecretStatusCondition, value float64) {
	psInfo := make(map[string]string)
	psInfo["name"] = ps.Name
	psInfo["namespace"] = ps.Namespace
	maps.Copy(psInfo, ps.Labels)
	conditionLabels := ctrlmetrics.RefineConditionMetricLabels(psInfo)
	pushSecretCondition := GetGaugeVec(PushSecretStatusConditionKey)

	// This allows us to delete metrics even when other labels (like helm annotations) have changed
	baseLabels := prometheus.Labels{
		"name":      ps.Name,
		"namespace": ps.Namespace,
	}

	switch condition.Type {
	case esapi.PushSecretReady:
		// Toggle opposite Status to 0, but first delete any stale metrics with old labels
		switch condition.Status {
		case v1.ConditionFalse:
			// delete any existing metrics with status True (regardless of other labels)
			baseLabels["condition"] = string(esapi.PushSecretReady)
			baseLabels["status"] = string(v1.ConditionTrue)
			pushSecretCondition.DeletePartialMatch(baseLabels)
			delete(baseLabels, "condition")
			delete(baseLabels, "status")

			// Set the metric with current labels
			pushSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esapi.PushSecretReady),
					"status":    string(v1.ConditionTrue),
				})).Set(0)
		case v1.ConditionTrue:
			// delete any existing metrics with status False (regardless of other labels)
			baseLabels["condition"] = string(esapi.PushSecretReady)
			baseLabels["status"] = string(v1.ConditionFalse)
			pushSecretCondition.DeletePartialMatch(baseLabels)
			delete(baseLabels, "condition")
			delete(baseLabels, "status")

			// finally, set the metric with current labels
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
