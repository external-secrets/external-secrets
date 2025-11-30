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

package secretstore

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
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

// storeMatchesRef checks if a given store matches a store reference (PushSecretStoreRef).
// This helper function should be shared to avoid code duplication.
// A match can be by name or by label selector, respecting the Kind.
func storeMatchesRef(store esapi.GenericStore, ref esv1alpha1.PushSecretStoreRef) bool {
	storeKind := store.GetKind()
	storeName := store.GetName()

	// Check if the Kind of the reference is compatible with the store's Kind.
	// A reference with an empty Kind is compatible with both SecretStore and ClusterSecretStore.
	kindMatches := (ref.Kind == storeKind) || (ref.Kind == "" && (storeKind == esapi.SecretStoreKind || storeKind == esapi.ClusterSecretStoreKind))
	if !kindMatches {
		return false
	}

	// Check for a name match.
	if ref.Name == storeName {
		return true
	}

	// Check for a label selector match.
	if ref.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(ref.LabelSelector)
		// Skips invalid selectors.
		if err != nil {
			return false
		}
		if selector.Matches(labels.Set(store.GetLabels())) {
			return true
		}
	}

	return false
}

// shouldReconcileSecretStoreForPushSecret determines if a SecretStore should be reconciled
// when a PushSecret changes, based on whether the PushSecret references this store.
func shouldReconcileSecretStoreForPushSecret(store esapi.GenericStore, ps *esv1alpha1.PushSecret) bool {
	// Check if this PushSecret has pushed to this store
	storeKey := fmt.Sprintf("%s/%s", store.GetKind(), store.GetName())
	if _, hasPushed := ps.Status.SyncedPushSecrets[storeKey]; hasPushed {
		return true
	}
	// Also check if the PushSecret references this store in its spec
	for _, storeRef := range ps.Spec.SecretStoreRefs {
		if storeMatchesRef(store, storeRef) {
			return true
		}
	}

	return false
}
