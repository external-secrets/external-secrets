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

package externalsecret

import (
	"crypto/sha3"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"
)

// NewExternalSecretCondition a set of default options for creating an External Secret Condition.
func NewExternalSecretCondition(condType esv1.ExternalSecretConditionType, status v1.ConditionStatus, reason, message string) *esv1.ExternalSecretStatusCondition {
	return &esv1.ExternalSecretStatusCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetExternalSecretCondition returns the condition with the provided type.
func GetExternalSecretCondition(status esv1.ExternalSecretStatus, condType esv1.ExternalSecretConditionType) *esv1.ExternalSecretStatusCondition {
	for _, c := range status.Conditions {
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetExternalSecretCondition updates the external secret to include the provided
// condition.
func SetExternalSecretCondition(es *esv1.ExternalSecret, condition esv1.ExternalSecretStatusCondition) {
	currentCond := GetExternalSecretCondition(es.Status, condition.Type)

	if currentCond != nil && currentCond.Status == condition.Status &&
		currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
		esmetrics.UpdateExternalSecretCondition(es, &condition, 1.0)
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	es.Status.Conditions = append(filterOutCondition(es.Status.Conditions, condition.Type), condition)

	if currentCond != nil {
		esmetrics.UpdateExternalSecretCondition(es, currentCond, 0.0)
	}

	esmetrics.UpdateExternalSecretCondition(es, &condition, 1.0)
}

// filterOutCondition returns an empty set of conditions with the provided type.
func filterOutCondition(conditions []esv1.ExternalSecretStatusCondition, condType esv1.ExternalSecretConditionType) []esv1.ExternalSecretStatusCondition {
	newConditions := make([]esv1.ExternalSecretStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

func fqdnFor(name string) string {
	fqdn := fmt.Sprintf(fieldOwnerTemplate, name)
	// If secret name is just too big, use the SHA3 hash of the secret name
	// Done this way for backwards compatibility thus avoiding breaking changes
	if len(fqdn) > 63 {
		fqdn = fmt.Sprintf(fieldOwnerTemplateSha, sha3.Sum224([]byte(name)))
	}
	return fqdn
}
