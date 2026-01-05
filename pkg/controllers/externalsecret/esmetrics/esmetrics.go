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

// Package esmetrics provides metrics functionality for the ExternalSecret controller
package esmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

const (
	// ExternalSecretSubsystem is the subsystem for the external-secret controller.
	ExternalSecretSubsystem = "externalsecret"
	// SyncCallsKey is the metric key for sync calls.
	SyncCallsKey = "sync_calls_total"
	// SyncCallsErrorKey is the metric key for sync call errors.
	SyncCallsErrorKey = "sync_calls_error"
	// ExternalSecretStatusConditionKey is the metric key for the external secret status condition.
	ExternalSecretStatusConditionKey = "status_condition"
	// ExternalSecretReconcileDurationKey is the metric key for the external secret reconcile duration.
	ExternalSecretReconcileDurationKey = "reconcile_duration"
)

var counterVecMetrics = map[string]*prometheus.CounterVec{}

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// SetUpMetrics is called at the root to set-up the metric logic using the
// config flags provided.
func SetUpMetrics() {
	// Obtain the prometheus metrics and register
	syncCallsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsKey,
		Help:      "Total number of the External Secret sync calls",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	syncCallsError := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsErrorKey,
		Help:      "Total number of the External Secret sync errors",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	externalSecretCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      ExternalSecretStatusConditionKey,
		Help:      "The status condition of a specific External Secret",
	}, ctrlmetrics.ConditionMetricLabelNames)

	externalSecretReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      ExternalSecretReconcileDurationKey,
		Help:      "The duration time to reconcile the External Secret",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	metrics.Registry.MustRegister(syncCallsTotal, syncCallsError, externalSecretCondition, externalSecretReconcileDuration)

	counterVecMetrics = map[string]*prometheus.CounterVec{
		SyncCallsKey:      syncCallsTotal,
		SyncCallsErrorKey: syncCallsError,
	}

	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		ExternalSecretStatusConditionKey:   externalSecretCondition,
		ExternalSecretReconcileDurationKey: externalSecretReconcileDuration,
	}
}

// UpdateExternalSecretCondition is a function that updates the condition of an external secret.
func UpdateExternalSecretCondition(es *esv1.ExternalSecret, condition *esv1.ExternalSecretStatusCondition, value float64) {
	esInfo := make(map[string]string)
	esInfo["name"] = es.Name
	esInfo["namespace"] = es.Namespace
	for k, v := range es.Labels {
		esInfo[k] = v
	}
	conditionLabels := ctrlmetrics.RefineConditionMetricLabels(esInfo)
	externalSecretCondition := GetGaugeVec(ExternalSecretStatusConditionKey)

	// this allows us to delete metrics even when other labels (like helm annotations) have changed
	baseLabels := prometheus.Labels{
		"name":      es.Name,
		"namespace": es.Namespace,
	}

	switch condition.Type {
	case esv1.ExternalSecretDeleted:
		// Remove condition=Ready metrics when the object gets deleted.
		baseLabels["condition"] = string(esv1.ExternalSecretReady)
		baseLabels["status"] = string(v1.ConditionFalse)
		externalSecretCondition.DeletePartialMatch(baseLabels)

		baseLabels["status"] = string(v1.ConditionTrue)
		externalSecretCondition.DeletePartialMatch(baseLabels)
		delete(baseLabels, "condition")
		delete(baseLabels, "status")

	case esv1.ExternalSecretReady:
		// Remove condition=Deleted metrics when the object gets ready.
		baseLabels["condition"] = string(esv1.ExternalSecretDeleted)
		baseLabels["status"] = string(v1.ConditionFalse)
		externalSecretCondition.DeletePartialMatch(baseLabels)

		baseLabels["status"] = string(v1.ConditionTrue)
		externalSecretCondition.DeletePartialMatch(baseLabels)
		delete(baseLabels, "condition")
		delete(baseLabels, "status")

		// Toggle opposite Status to 0, but first delete any stale metrics with old labels
		switch condition.Status {
		case v1.ConditionFalse:
			// delete any existing metrics with status True (regardless of other labels)
			// condition is fixed to ExternalSecretReady because other statuses were already handled above.
			baseLabels["condition"] = string(esv1.ExternalSecretReady)
			baseLabels["status"] = string(v1.ConditionTrue)
			externalSecretCondition.DeletePartialMatch(baseLabels)
			delete(baseLabels, "condition")
			delete(baseLabels, "status")

			// Set the metric with current labels
			externalSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esv1.ExternalSecretReady),
					"status":    string(v1.ConditionTrue),
				})).Set(0)
		case v1.ConditionTrue:
			// delete any existing metrics with status False (regardless of other labels)
			baseLabels["condition"] = string(esv1.ExternalSecretReady)
			baseLabels["status"] = string(v1.ConditionFalse)
			externalSecretCondition.DeletePartialMatch(baseLabels)
			delete(baseLabels, "condition")
			delete(baseLabels, "status")

			// finally, set the metric with current labels
			externalSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esv1.ExternalSecretReady),
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

	externalSecretCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
		map[string]string{
			"condition": string(condition.Type),
			"status":    string(condition.Status),
		})).Set(value)
}

// GetCounterVec returns the counter vec for the given key.
func GetCounterVec(key string) *prometheus.CounterVec {
	return counterVecMetrics[key]
}

// GetGaugeVec returns the gauge vec for the given key.
func GetGaugeVec(key string) *prometheus.GaugeVec {
	return gaugeVecMetrics[key]
}
