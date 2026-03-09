/*
Copyright © 2025 ESO Maintainer Team

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

// Package psmetrics provides metrics for PushSecret controller.
package psmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

const (
	// PushSecretSubsystem is the subsystem name for PushSecret metrics.
	PushSecretSubsystem = "pushsecret"

	// PushSecretReconcileDurationKey is the key for the reconcile duration metric.
	PushSecretReconcileDurationKey = "reconcile_duration"

	// PushSecretStatusConditionKey is the key for the status condition metric.
	PushSecretStatusConditionKey = "status_condition"
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

// UpdatePushSecretCondition updates the condition metrics for a PushSecret.
func UpdatePushSecretCondition(ps *esapi.PushSecret, condition *esapi.PushSecretStatusCondition, value float64) {
	psInfo := make(map[string]string)
	psInfo["name"] = ps.Name
	psInfo["namespace"] = ps.Namespace
	for k, v := range ps.Labels {
		psInfo[k] = v
	}
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

// GetGaugeVec returns a GaugeVec for the given metric key.
func GetGaugeVec(key string) *prometheus.GaugeVec {
	return gaugeVecMetrics[key]
}
