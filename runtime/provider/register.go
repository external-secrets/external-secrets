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

// Registry
var (
	registry     = make(map[Name]Metadata)
	registryLock sync.RWMutex
)

// Register stores provider metadata
func Register(providerName string, metadata Metadata) {
	registryLock.Lock()
	defer registryLock.Unlock()
	registry[Name(providerName)] = metadata
}

// Get returns metadata for a provider
func Get(providerName string) (Metadata, bool) {
	registryLock.RLock()
	defer registryLock.RUnlock()
	meta, ok := registry[Name(providerName)]
	return meta, ok
}

// List returns all registered provider metadata
func List() map[Name]Metadata {
	registryLock.RLock()
	defer registryLock.RUnlock()
	result := make(map[Name]Metadata, len(registry))
	for name, meta := range registry {
		result[name] = meta
	}
	return result
}

// HasCapability checks if a provider supports a capability
func HasCapability(providerName string, capability CapabilityName) bool {
	providerMeta, ok := Get(providerName)
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
