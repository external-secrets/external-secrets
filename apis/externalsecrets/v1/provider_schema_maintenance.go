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

package v1

import (
	"fmt"
	"sync"
)

// MaintenanceStatus defines a type for different maintenance states of a provider schema.
type MaintenanceStatus string

// These are the defined maintenance states for a provider schema.
const (
	MaintenanceStatusMaintained    MaintenanceStatus = "Maintained"
	MaintenanceStatusNotMaintained MaintenanceStatus = "NotMaintained"
	MaintenanceStatusDeprecated    MaintenanceStatus = "Deprecated"
)

var maintenance map[string]MaintenanceStatus
var mlock sync.RWMutex

func init() {
	maintenance = make(map[string]MaintenanceStatus)
}

// RegisterMaintenanceStatus registers the maintenance status of the provider from the generic store.
// It panics if the provider is already registered or if there is an error getting the provider name.
func RegisterMaintenanceStatus(status MaintenanceStatus, storeSpec *SecretStoreProvider) {
	storeName, err := getProviderName(storeSpec)
	if err != nil {
		panic(fmt.Sprintf("store error registering schema: %s", err.Error()))
	}

	mlock.Lock()
	defer mlock.Unlock()
	_, exists := maintenance[storeName]
	if exists {
		panic(fmt.Sprintf("store %q already registered", storeName))
	}

	maintenance[storeName] = status
}

// ForceRegisterMaintenanceStatus registers the maintenance status of the provider from the generic store.
// It panics if there is an error getting the provider name, it overwrites existing provider status or
// stores new status for a provider if it exists.
func ForceRegisterMaintenanceStatus(status MaintenanceStatus, storeSpec *SecretStoreProvider) {
	storeName, err := getProviderName(storeSpec)
	if err != nil {
		panic(fmt.Sprintf("store error registering schema: %s", err.Error()))
	}

	mlock.Lock()
	defer mlock.Unlock()
	maintenance[storeName] = status
}

// GetMaintenanceStatus returns the maintenance status of the provider from the generic store.
func GetMaintenanceStatus(s GenericStore) (MaintenanceStatus, error) {
	if s == nil {
		return MaintenanceStatusNotMaintained, nil
	}
	spec := s.GetSpec()
	if spec == nil {
		// Note, this condition can never be reached, because
		// the Spec is not a pointer in Kubernetes. It will
		// always exist.
		return MaintenanceStatusNotMaintained, fmt.Errorf("no spec found in %#v", s)
	}
	storeName, err := getProviderName(spec.Provider)
	if err != nil {
		return MaintenanceStatusNotMaintained, fmt.Errorf("store error for %s: %w", s.GetName(), err)
	}

	mlock.RLock()
	status, ok := maintenance[storeName]
	mlock.RUnlock()

	if !ok {
		return MaintenanceStatusNotMaintained, fmt.Errorf("failed to find registered store backend for type: %s, name: %s", storeName, s.GetName())
	}

	return status, nil
}
