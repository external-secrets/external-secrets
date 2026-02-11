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
	"sync"
)

// Registry holds provider metadata and provides thread-safe access
type Registry struct {
	providers map[Name]Metadata
	mu        sync.RWMutex
}

// NewRegistry creates a new empty registry for testing or isolated use
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[Name]Metadata),
	}
}

// Register stores provider metadata in the registry
func (r *Registry) Register(providerName string, metadata Metadata) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[Name(providerName)] = metadata
}

// Get returns metadata for a provider from the registry
func (r *Registry) Get(providerName string) (Metadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	meta, ok := r.providers[Name(providerName)]
	return meta, ok
}

// List returns all registered provider metadata from the registry
func (r *Registry) List() map[Name]Metadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[Name]Metadata, len(r.providers))
	for name, meta := range r.providers {
		result[name] = meta
	}
	return result
}

// HasCapability checks if a provider supports a capability in the registry
func (r *Registry) HasCapability(providerName string, capability CapabilityName) bool {
	providerMeta, ok := r.Get(providerName)
	if !ok {
		return false
	}
	for _, currentCapability := range providerMeta.Capabilities {
		if currentCapability.Name == capability {
			return true
		}
	}
	return false
}

// Global registry instance for production use
var globalRegistry = NewRegistry()

// Register stores provider metadata in the global registry
func Register(providerName string, metadata Metadata) {
	globalRegistry.Register(providerName, metadata)
}

// Get returns metadata for a provider from the global registry
func Get(providerName string) (Metadata, bool) {
	return globalRegistry.Get(providerName)
}

// List returns all registered provider metadata from the global registry
func List() map[Name]Metadata {
	return globalRegistry.List()
}

// HasCapability checks if a provider supports a capability in the global registry
func HasCapability(providerName string, capability CapabilityName) bool {
	return globalRegistry.HasCapability(providerName, capability)
}
