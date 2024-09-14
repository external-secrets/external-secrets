//Copyright External Secrets Inc. All Rights Reserved

package clusterexternalsecret

import (
	v1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/clusterexternalsecret/cesmetrics"
)

func NewClusterExternalSecretCondition(failedNamespaces map[string]error) *esv1beta1.ClusterExternalSecretStatusCondition {
	if len(failedNamespaces) == 0 {
		return &esv1beta1.ClusterExternalSecretStatusCondition{
			Type:   esv1beta1.ClusterExternalSecretReady,
			Status: v1.ConditionTrue,
		}
	}

	condition := &esv1beta1.ClusterExternalSecretStatusCondition{
		Type:    esv1beta1.ClusterExternalSecretReady,
		Status:  v1.ConditionFalse,
		Message: errNamespacesFailed,
	}

	return condition
}

func SetClusterExternalSecretCondition(ces *esv1beta1.ClusterExternalSecret, condition esv1beta1.ClusterExternalSecretStatusCondition) {
	ces.Status.Conditions = append(filterOutCondition(ces.Status.Conditions, condition.Type), condition)
	cesmetrics.UpdateClusterExternalSecretCondition(ces, &condition)
}

// filterOutCondition returns an empty set of conditions with the provided type.
func filterOutCondition(conditions []esv1beta1.ClusterExternalSecretStatusCondition, condType esv1beta1.ClusterExternalSecretConditionType) []esv1beta1.ClusterExternalSecretStatusCondition {
	newConditions := make([]esv1beta1.ClusterExternalSecretStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
