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

// Package metrics provides metrics for SecretStore controllers.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

// StatusConditionKey is the key for the status condition metric.
const StatusConditionKey = "status_condition"

// GaugeVevGetter is a function type that retrieves a Prometheus GaugeVec based on a provided key.
type GaugeVevGetter func(key string) *prometheus.GaugeVec

// UpdateStatusCondition updates the condition metrics for a SecretStore.
func UpdateStatusCondition(ss esapi.GenericStore, condition esapi.SecretStoreStatusCondition, gaugeVecGetter GaugeVevGetter) {
	ssInfo := make(map[string]string)
	ssInfo["name"] = ss.GetName()
	ssInfo["namespace"] = ss.GetNamespace()
	for k, v := range ss.GetLabels() {
		ssInfo[k] = v
	}
	conditionLabels := ctrlmetrics.RefineConditionMetricLabels(ssInfo)
	secretStoreCondition := gaugeVecGetter(StatusConditionKey)

	if condition.Type == esapi.SecretStoreReady {
		switch condition.Status {
		case v1.ConditionFalse:
			secretStoreCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esapi.SecretStoreReady),
					"status":    string(v1.ConditionTrue),
				})).Set(0)
		case v1.ConditionTrue:
			secretStoreCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
				map[string]string{
					"condition": string(esapi.SecretStoreReady),
					"status":    string(v1.ConditionFalse),
				})).Set(0)
		case v1.ConditionUnknown:
			break
		}
	}

	secretStoreCondition.With(ctrlmetrics.RefineLabels(conditionLabels,
		map[string]string{
			"condition": string(condition.Type),
			"status":    string(condition.Status),
		})).Set(1)
}
