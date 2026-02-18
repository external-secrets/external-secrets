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
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Registry lookup hooks, set by runtime/provider to avoid circular imports.
// runtime/provider imports esv1; esv1 must NOT import runtime/provider.
var (
	hookMu                           sync.RWMutex
	getProviderByNameHook            func(string) (Provider, error)
	getProviderMaintenanceStatusHook func(string) (MaintenanceStatus, error)
)

// SetRegistryHooks is called once by runtime/provider's init() to wire up lookups.
func SetRegistryHooks(
	getProviderByName func(string) (Provider, error),
	getProviderMaintenanceStatus func(string) (MaintenanceStatus, error),
) {
	hookMu.Lock()
	defer hookMu.Unlock()
	getProviderByNameHook = getProviderByName
	getProviderMaintenanceStatusHook = getProviderMaintenanceStatus
}

// GetProviderByName interface removed because it was unused

// GetProvider returns the provider based the generic store by calling runtime registry.
func GetProvider(s GenericStore) (Provider, error) {
	if s == nil {
		return nil, nil
	}
	spec := s.GetSpec()
	if spec == nil {
		// Note, this condition can never be reached, because
		// the Spec is not a pointer in Kubernetes. It will
		// always exist.
		return nil, fmt.Errorf("no spec found in %#v", s)
	}
	providerName, err := GetProviderName(spec.Provider)
	if err != nil {
		return nil, err
	}
	hookMu.RLock()
	fn := getProviderByNameHook
	hookMu.RUnlock()
	if fn == nil {
		return nil, errors.New("provider registry not initialized — ensure runtime/provider is imported")
	}
	return fn(providerName)
}

// GetMaintenanceStatus returns the maintenance status of the provider from the generic store.
// It delegates to the hook set by runtime/provider to avoid circular imports.
func GetMaintenanceStatus(s GenericStore) (MaintenanceStatus, error) {
	if s == nil {
		return MaintenanceStatusNotMaintained, nil
	}
	spec := s.GetSpec()
	if spec == nil {
		return MaintenanceStatusNotMaintained, fmt.Errorf("no spec found in %#v", s)
	}
	name, err := GetProviderName(spec.Provider)
	if err != nil {
		return MaintenanceStatusNotMaintained, fmt.Errorf("store error for %s: %w", s.GetName(), err)
	}
	hookMu.RLock()
	fn := getProviderMaintenanceStatusHook
	hookMu.RUnlock()
	if fn == nil {
		return MaintenanceStatusNotMaintained, errors.New("provider registry not initialized — ensure runtime/provider is imported")
	}
	return fn(name)
}

// GetProviderName returns the name of the configured provider
// by marshaling the SecretStoreProvider spec to JSON and finding the single non-null key.
// Exposing a translation from SecretStoreProvider spec to a provider Name is convenient.
func GetProviderName(storeSpec *SecretStoreProvider) (string, error) {
	storeBytes, err := json.Marshal(storeSpec)
	if err != nil || storeBytes == nil {
		return "", fmt.Errorf("failed to marshal store spec: %w", err)
	}

	storeMap := make(map[string]any)
	err = json.Unmarshal(storeBytes, &storeMap)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal store spec: %w", err)
	}

	if len(storeMap) != 1 {
		return "", fmt.Errorf("secret stores must only have exactly one backend specified, found %d", len(storeMap))
	}

	for k := range storeMap {
		return k, nil
	}

	return "", errors.New("failed to find registered store backend")
}
