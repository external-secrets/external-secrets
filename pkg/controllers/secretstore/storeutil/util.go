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

// Package storeutil provides utility functions for SecretStore operations
package storeutil

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errSecretStoreNotReady = "%s %q is not ready"
)

// ShouldProcessStore returns true if the store should be processed.
func ShouldProcessStore(store esv1.GenericStore, class string) bool {
	if store == nil || store.GetSpec().Controller == "" || store.GetSpec().Controller == class {
		return true
	}

	return false
}

// AssertStoreIsUsable asserts that the store is ready to use.
func AssertStoreIsUsable(store esv1.GenericStore) error {
	if store == nil {
		return nil
	}
	condition := GetSecretStoreCondition(store.GetStatus(), esv1.SecretStoreReady)
	if condition == nil || condition.Status != v1.ConditionTrue {
		return fmt.Errorf(errSecretStoreNotReady, store.GetKind(), store.GetName())
	}
	return nil
}

// GetSecretStoreCondition returns the condition with the provided type.
func GetSecretStoreCondition(status esv1.SecretStoreStatus, condType esv1.SecretStoreConditionType) *esv1.SecretStoreStatusCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}
