//Copyright External Secrets Inc. All Rights Reserved

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
)

const StatusConditionKey = "status_condition"

type GaugeVevGetter func(key string) *prometheus.GaugeVec

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
