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

package generatorstate

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

// NewgeneratorstateCondition a set of default options for creating an GeneratorState Condition.
func NewGeneratorStateCondition(condType genv1alpha1.GeneratorStateConditionType, status v1.ConditionStatus, reason, message string) *genv1alpha1.GeneratorStateStatusCondition {
	return &genv1alpha1.GeneratorStateStatusCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetgeneratorstateCondition returns the condition with the provided type.
func GetGeneratorStateCondition(status genv1alpha1.GeneratorStateStatus, condType genv1alpha1.GeneratorStateConditionType) *genv1alpha1.GeneratorStateStatusCondition {
	for _, c := range status.Conditions {
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetGeneratorStateCondition updates the GeneratorState to include the provided
// condition.
func SetGeneratorStateCondition(gs *genv1alpha1.GeneratorState, condition genv1alpha1.GeneratorStateStatusCondition) {
	currentCond := GetGeneratorStateCondition(gs.Status, condition.Type)

	if currentCond != nil && currentCond.Status == condition.Status &&
		currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	gs.Status.Conditions = append(filterOutCondition(gs.Status.Conditions, condition.Type), condition)
}

// filterOutCondition returns an empty set of conditions with the provided type.
func filterOutCondition(conditions []genv1alpha1.GeneratorStateStatusCondition, condType genv1alpha1.GeneratorStateConditionType) []genv1alpha1.GeneratorStateStatusCondition {
	newConditions := make([]genv1alpha1.GeneratorStateStatusCondition, 0, len(conditions))
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
