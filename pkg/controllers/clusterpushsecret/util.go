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

package clusterpushsecret

import (
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/clusterpushsecret/cpsmetrics"
)

func NewClusterPushSecretCondition(failedNamespaces map[string]error) *v1alpha1.PushSecretStatusCondition {
	if len(failedNamespaces) == 0 {
		return &v1alpha1.PushSecretStatusCondition{
			Type:   v1alpha1.PushSecretReady,
			Status: v1.ConditionTrue,
		}
	}

	condition := &v1alpha1.PushSecretStatusCondition{
		Type:    v1alpha1.PushSecretReady,
		Status:  v1.ConditionFalse,
		Message: errNamespacesFailed,
	}

	return condition
}

func SetClusterPushSecretCondition(ces *v1alpha1.ClusterPushSecret, condition v1alpha1.PushSecretStatusCondition) {
	ces.Status.Conditions = append(filterOutCondition(ces.Status.Conditions, condition.Type), condition)
	cpsmetrics.UpdateClusterPushSecretCondition(ces, &condition)
}

// filterOutCondition returns an empty set of conditions with the provided type.
func filterOutCondition(conditions []v1alpha1.PushSecretStatusCondition, condType v1alpha1.PushSecretConditionType) []v1alpha1.PushSecretStatusCondition {
	newConditions := make([]v1alpha1.PushSecretStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
