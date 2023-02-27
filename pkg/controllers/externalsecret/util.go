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
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// NewExternalSecretCondition a set of default options for creating an External Secret Condition.
func NewExternalSecretCondition(condType esv1beta1.ExternalSecretConditionType, status v1.ConditionStatus, reason, message string) *esv1beta1.ExternalSecretStatusCondition {
	return &esv1beta1.ExternalSecretStatusCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetExternalSecretCondition returns the condition with the provided type.
func GetExternalSecretCondition(status esv1beta1.ExternalSecretStatus, condType esv1beta1.ExternalSecretConditionType) *esv1beta1.ExternalSecretStatusCondition {
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
func SetExternalSecretCondition(es *esv1beta1.ExternalSecret, condition esv1beta1.ExternalSecretStatusCondition) {
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

// Refine the given Prometheus Labels with values from a map `newLabels`
// Only overwrite a value if the corresponding key is present in the
// Prometheus' Labels already to avoid adding label names which are
// not defined in a metric's description. Note that non-alphanumeric
// characters from keys of `newLabels` are replaced by an underscore
// because Promtheus does not accept non-alphanumeric, non-underscore
// characters in label names.
func RefineLabels(promLabels prometheus.Labels, newLabels map[string]string) prometheus.Labels {
	var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

	for k, v := range newLabels {
		cleanKey := nonAlphanumericRegex.ReplaceAllString(k, "_")
		if _, ok := promLabels[cleanKey]; ok {
			promLabels[cleanKey] = v
		}
	}

	return promLabels
}

// Apply RefineLabels with sevrel maps in a row.
func RefineLabelsWithMaps(promLabels prometheus.Labels, maps ...map[string]string) prometheus.Labels {
	for _, m := range maps {
		RefineLabels(promLabels, m)
	}

	return promLabels
}

// filterOutCondition returns an empty set of conditions with the provided type.
func filterOutCondition(conditions []esv1beta1.ExternalSecretStatusCondition, condType esv1beta1.ExternalSecretConditionType) []esv1beta1.ExternalSecretStatusCondition {
	newConditions := make([]esv1beta1.ExternalSecretStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
