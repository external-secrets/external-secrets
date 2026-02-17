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

import "errors"

// MaintenanceStatus defines a type for different maintenance states of a provider schema.
type MaintenanceStatus string

// These are the defined maintenance states for a provider schema.
const (
	MaintenanceStatusMaintained    MaintenanceStatus = "Maintained"
	MaintenanceStatusNotMaintained MaintenanceStatus = "NotMaintained"
	MaintenanceStatusDeprecated    MaintenanceStatus = "Deprecated"
)

// GetMaintenanceStatus returns the maintenance status of the provider from the generic store.
// It delegates to the hook set by runtime/provider to avoid circular imports.
func GetMaintenanceStatus(s GenericStore) (MaintenanceStatus, error) {
	hookMu.RLock()
	fn := getMaintenanceStatusHook
	hookMu.RUnlock()
	if fn == nil {
		return MaintenanceStatusNotMaintained, errors.New("provider registry not initialized — ensure runtime/provider is imported")
	}
	return fn(s)
}
