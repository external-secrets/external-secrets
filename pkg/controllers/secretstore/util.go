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

package secretstore

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/metrics"
)

// NewSecretStoreCondition a set of default options for creating an External Secret Condition.
func NewSecretStoreCondition(condType esapi.SecretStoreConditionType, status v1.ConditionStatus, reason, message string) *esapi.SecretStoreStatusCondition {
	return &esapi.SecretStoreStatusCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetSecretStoreCondition returns the condition with the provided type.
func GetSecretStoreCondition(status esapi.SecretStoreStatus, condType esapi.SecretStoreConditionType) *esapi.SecretStoreStatusCondition {
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
func SetExternalSecretCondition(gs esapi.GenericStore, condition esapi.SecretStoreStatusCondition, gaugeVecGetter metrics.GaugeVevGetter) {
	metrics.UpdateStatusCondition(gs, condition, gaugeVecGetter)

	status := gs.GetStatus()
	currentCond := GetSecretStoreCondition(status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status &&
		currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	status.Conditions = append(filterOutCondition(status.Conditions, condition.Type), condition)
	gs.SetStatus(status)
}

// filterOutCondition returns an empty set of conditions with the provided type.
func filterOutCondition(conditions []esapi.SecretStoreStatusCondition, condType esapi.SecretStoreConditionType) []esapi.SecretStoreStatusCondition {
	newConditions := make([]esapi.SecretStoreStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
