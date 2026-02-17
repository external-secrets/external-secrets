/*
Copyright © 2026 ESO Maintainer Team

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

package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// RegistryEntry holds both the provider implementation and its metadata.
type RegistryEntry struct {
	Provider esv1.Provider
	Metadata Metadata
}

// Registry holds provider entries and provides thread-safe access.
type Registry struct {
	providers map[Name]RegistryEntry
	mu        sync.RWMutex
}

// NewRegistry creates a new empty registry for testing or isolated use.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[Name]RegistryEntry),
	}
}

func (r *Registry) store(name string, p esv1.Provider, md Metadata) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[Name(name)] = RegistryEntry{Provider: p, Metadata: md}
}

func (r *Registry) storeName(spec *esv1.SecretStoreProvider, p esv1.Provider, md Metadata) error {
	name, err := getProviderName(spec)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.providers[Name(name)]
	if exists {
		return fmt.Errorf("store %q already registered", name)
	}
	r.providers[Name(name)] = RegistryEntry{Provider: p, Metadata: md}
	return nil
}

func (r *Registry) forceName(spec *esv1.SecretStoreProvider, p esv1.Provider, md Metadata) error {
	name, err := getProviderName(spec)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[Name(name)] = RegistryEntry{Provider: p, Metadata: md}
	return nil
}

// Get returns the registry entry for a provider by name.
func (r *Registry) Get(providerName string) (RegistryEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.providers[Name(providerName)]
	return entry, ok
}

// GetBySpec looks up a provider by its SecretStoreProvider spec.
func (r *Registry) GetBySpec(spec *esv1.SecretStoreProvider) (esv1.Provider, error) {
	name, err := getProviderName(spec)
	if err != nil {
		return nil, err
	}
	r.mu.RLock()
	entry, ok := r.providers[Name(name)]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("failed to find registered store backend for type: %s", name)
	}
	return entry.Provider, nil
}

// List returns a copy of all registered entries.
func (r *Registry) List() map[Name]RegistryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[Name]RegistryEntry, len(r.providers))
	for name, entry := range r.providers {
		result[name] = entry
	}
	return result
}

// Global registry instance for production use.
var globalRegistry = NewRegistry()

func init() {
	// Wire up esv1 lookup stubs to delegate to this registry.
	// This avoids circular imports: runtime/provider imports esv1, not vice versa.
	esv1.SetRegistryHooks(
		func(store esv1.GenericStore) (esv1.Provider, error) {
			return GetProvider(store)
		},
		func(name string) (esv1.Provider, bool) {
			return GetProviderByName(name)
		},
		func() map[string]esv1.Provider {
			entries := globalRegistry.List()
			result := make(map[string]esv1.Provider, len(entries))
			for name, entry := range entries {
				result[string(name)] = entry.Provider
			}
			return result
		},
		func(store esv1.GenericStore) (esv1.MaintenanceStatus, error) {
			return GetMaintenanceStatus(store)
		},
	)
}

// Register stores provider + metadata in the single global registry.
// It panics if a provider with the same name is already registered.
func Register(name string, p Provider, spec *esv1.SecretStoreProvider) {
	md := p.Metadata()
	if err := globalRegistry.storeName(spec, p, md); err != nil {
		panic(fmt.Sprintf("store error registering provider: %s", err.Error()))
	}
}

// ForceRegister registers a provider, overwriting any existing entry.
// For use in tests where the provider implements the combined Provider interface.
func ForceRegister(name string, p Provider, spec *esv1.SecretStoreProvider) {
	md := p.Metadata()
	if err := globalRegistry.forceName(spec, p, md); err != nil {
		panic(fmt.Sprintf("store error force-registering provider: %s", err.Error()))
	}
}

// ForceRegisterProvider registers any esv1.Provider with zero Metadata.
// For use in tests where the test fake does not implement MetadataReporter.
func ForceRegisterProvider(p esv1.Provider, spec *esv1.SecretStoreProvider) {
	if err := globalRegistry.forceName(spec, p, Metadata{}); err != nil {
		panic(fmt.Sprintf("store error force-registering provider: %s", err.Error()))
	}
}

// Get returns the registry entry for a provider by name from the global registry.
func Get(providerName string) (RegistryEntry, bool) {
	return globalRegistry.Get(providerName)
}

// GetProvider returns the provider implementation for the given store.
func GetProvider(store esv1.GenericStore) (esv1.Provider, error) {
	if store == nil {
		return nil, nil
	}
	spec := store.GetSpec()
	if spec == nil {
		return nil, fmt.Errorf("no spec found in %#v", store)
	}
	p, err := globalRegistry.GetBySpec(spec.Provider)
	if err != nil {
		return nil, fmt.Errorf("store error for %s: %w", store.GetName(), err)
	}
	return p, nil
}

// GetProviderByName returns the provider implementation by name.
func GetProviderByName(name string) (esv1.Provider, bool) {
	entry, ok := globalRegistry.Get(name)
	if !ok {
		return nil, false
	}
	return entry.Provider, true
}

// GetMaintenanceStatus derives the maintenance status from the stored Metadata.
func GetMaintenanceStatus(store esv1.GenericStore) (esv1.MaintenanceStatus, error) {
	if store == nil {
		return esv1.MaintenanceStatusNotMaintained, nil
	}
	spec := store.GetSpec()
	if spec == nil {
		return esv1.MaintenanceStatusNotMaintained, fmt.Errorf("no spec found in %#v", store)
	}
	name, err := getProviderName(spec.Provider)
	if err != nil {
		return esv1.MaintenanceStatusNotMaintained, fmt.Errorf("store error for %s: %w", store.GetName(), err)
	}
	entry, ok := globalRegistry.Get(name)
	if !ok {
		return esv1.MaintenanceStatusNotMaintained, fmt.Errorf("failed to find registered store backend for type: %s, name: %s", name, store.GetName())
	}
	return entry.Metadata.MaintenanceStatus(), nil
}

// List returns all registered provider entries from the global registry.
func List() map[Name]RegistryEntry {
	return globalRegistry.List()
}

// getProviderName returns the name of the configured provider
// by marshaling the SecretStoreProvider spec to JSON and finding the single non-null key.
func getProviderName(storeSpec *esv1.SecretStoreProvider) (string, error) {
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

func GetCapabilities(providerName string) (esv1.SecretStoreCapabilities, bool) {
	entry, ok := globalRegistry.Get(providerName)
	if !ok {
		return esv1.SecretStoreReadOnly, false
	}
	return entry.Metadata.APICapabilities(), true
}
