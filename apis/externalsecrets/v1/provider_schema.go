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
	"errors"
	"sync"
)

// Registry lookup hooks, set by runtime/provider to avoid circular imports.
// runtime/provider imports esv1; esv1 must NOT import runtime/provider.
var (
	hookMu                   sync.RWMutex
	getProviderHook          func(GenericStore) (Provider, error)
	getProviderByNameHook    func(string) (Provider, bool)
	listProvidersHook        func() map[string]Provider
	getMaintenanceStatusHook func(GenericStore) (MaintenanceStatus, error)
)

// SetRegistryHooks is called once by runtime/provider's init() to wire up lookups.
func SetRegistryHooks(
	getProvider func(GenericStore) (Provider, error),
	getProviderByName func(string) (Provider, bool),
	listProviders func() map[string]Provider,
	getMaintenanceStatus func(GenericStore) (MaintenanceStatus, error),
) {
	hookMu.Lock()
	defer hookMu.Unlock()
	getProviderHook = getProvider
	getProviderByNameHook = getProviderByName
	listProvidersHook = listProviders
	getMaintenanceStatusHook = getMaintenanceStatus
}

// GetProvider returns the provider from the generic store.
func GetProvider(s GenericStore) (Provider, error) {
	hookMu.RLock()
	fn := getProviderHook
	hookMu.RUnlock()
	if fn == nil {
		return nil, errors.New("provider registry not initialized — ensure runtime/provider is imported")
	}
	return fn(s)
}

// GetProviderByName returns the provider implementation by name.
func GetProviderByName(name string) (Provider, bool) {
	hookMu.RLock()
	fn := getProviderByNameHook
	hookMu.RUnlock()
	if fn == nil {
		return nil, false
	}
	return fn(name)
}

// List returns all registered provider implementations.
func List() map[string]Provider {
	hookMu.RLock()
	fn := listProvidersHook
	hookMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn()
}
