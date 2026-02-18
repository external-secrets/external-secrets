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
	"fmt"
	"sync"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// RegistryEntry holds both the provider Information and its metadata.
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

func (r *Registry) Add(spec *esv1.SecretStoreProvider, p *esv1.Provider, md *Metadata) error {
	return r.createOrUpdateStore(spec, p, md, true)
}

func (r *Registry) Replace(spec *esv1.SecretStoreProvider, p *esv1.Provider, md *Metadata) error {
	return r.createOrUpdateStore(spec, p, md, false)
}

func (r *Registry) createOrUpdateStore(spec *esv1.SecretStoreProvider, p *esv1.Provider, md *Metadata, createOnly bool) error {
	name, err := esv1.GetProviderName(spec)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.providers[Name(name)]
	if exists && createOnly {
		return fmt.Errorf("store %q already registered", name)
	}
	if md == nil {
		return fmt.Errorf("store %q does not have a metadata", name)
	}
	r.providers[Name(name)] = RegistryEntry{Provider: *p, Metadata: *md}
	return nil
}

//
//// GetProviderByName returns the registry entry for a provider and whether it was found.
//func (r *Registry) GetProviderByName(providerName string) (esv1.Provider, error) {
//	provider, found := r.fetch(providerName)
//	if !found {
//		return nil, fmt.Errorf("provider %q not found", providerName)
//	}
//	return provider, nil
//}
//
//// GetProviderBySpec looks up a provider by its SecretStoreProvider spec.
//func (r *Registry) GetProviderBySpec(spec *esv1.SecretStoreProvider) (esv1.Provider, error) {
//	name, err := getProviderName(spec)
//	if err != nil {
//		return nil, err
//	}
//	entry, found := r.fetch(name)
//	if !found {
//		return nil, fmt.Errorf("failed to find registered store backend for type: %s", name)
//	}
//	return entry, nil
//}

func (r *Registry) Fetch(name string) (esv1.Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.providers[Name(name)]
	return entry.Provider, ok
}

func (r *Registry) FetchMetadata(name string) (Metadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.providers[Name(name)]
	return entry.Metadata, ok
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
		func(providerName string) (esv1.Provider, error) {
			return GetProviderFromRegistry(providerName, globalRegistry)
		},
		func(providerName string) (esv1.MaintenanceStatus, error) {
			return GetMaintenanceStatusFromRegistry(providerName, globalRegistry)
		},
	)
}

// GetProviderFromRegistry returns the provider implementation for the given store from the given registry.
func GetProviderFromRegistry(providerName string, registry *Registry) (esv1.Provider, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	p, found := registry.Fetch(providerName)
	if !found {
		return nil, fmt.Errorf("provider %s not found in the registry", providerName)
	}
	return p, nil
}

// GetMaintenanceStatusFromRegistry derives the maintenance status from the stored Metadata.
func GetMaintenanceStatusFromRegistry(providerName string, registry *Registry) (esv1.MaintenanceStatus, error) {
	if registry == nil {
		return esv1.MaintenanceStatusUnknown, fmt.Errorf("registry is nil")
	}
	p, found := registry.FetchMetadata(providerName)
	if !found {
		return esv1.MaintenanceStatusUnknown, fmt.Errorf("provider %s not found in the registry", providerName)
	}
	return p.MaintenanceStatus(), nil
}

// Register stores provider + metadata in the single global registry.
// It panics if a provider with the same name is already registered.
func Register(p *esv1.Provider, spec *esv1.SecretStoreProvider, md *Metadata, registry *Registry) {
	if err := registry.Add(spec, p, md); err != nil {
		panic(fmt.Sprintf("store error registering provider: %s", err.Error()))
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

// GetProviderByName returns the provider implementation by name.
func GetProviderByName(name string) (esv1.Provider, bool) {
	entry, ok := globalRegistry.Get(name)
	if !ok {
		return nil, false
	}
	return entry.Provider, true
}

// List returns all registered provider entries from the global registry.
func List() map[Name]RegistryEntry {
	return globalRegistry.List()
}

func GetCapabilities(providerName string) (esv1.SecretStoreCapabilities, bool) {
	entry, ok := globalRegistry.Get(providerName)
	if !ok {
		return esv1.SecretStoreReadOnly, false
	}
	return entry.Metadata.APICapabilities(), true
}
