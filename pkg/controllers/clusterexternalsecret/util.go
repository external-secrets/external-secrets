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
