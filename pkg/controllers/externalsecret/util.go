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
package externalsecret

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1alpha2 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha2"
)

// NewExternalSecretCondition a set of default options for creating an External Secret Condition.
func NewExternalSecretCondition(condType esv1alpha2.ExternalSecretConditionType, status v1.ConditionStatus, reason, message string) *esv1alpha2.ExternalSecretStatusCondition {
	return &esv1alpha2.ExternalSecretStatusCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetExternalSecretCondition returns the condition with the provided type.
func GetExternalSecretCondition(status esv1alpha2.ExternalSecretStatus, condType esv1alpha2.ExternalSecretConditionType) *esv1alpha2.ExternalSecretStatusCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetExternalSecretCondition updates the external secret to include the provided
// condition.
func SetExternalSecretCondition(es *esv1alpha2.ExternalSecret, condition esv1alpha2.ExternalSecretStatusCondition) {
	currentCond := GetExternalSecretCondition(es.Status, condition.Type)

	if currentCond != nil && currentCond.Status == condition.Status &&
		currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
		updateExternalSecretCondition(es, &condition, 1.0)
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	es.Status.Conditions = append(filterOutCondition(es.Status.Conditions, condition.Type), condition)

	if currentCond != nil {
		updateExternalSecretCondition(es, currentCond, 0.0)
	}

	updateExternalSecretCondition(es, &condition, 1.0)
}

// filterOutCondition returns an empty set of conditions with the provided type.
func filterOutCondition(conditions []esv1alpha2.ExternalSecretStatusCondition, condType esv1alpha2.ExternalSecretConditionType) []esv1alpha2.ExternalSecretStatusCondition {
	newConditions := make([]esv1alpha2.ExternalSecretStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
