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
	"sync"

	"github.com/external-secrets/external-secrets/runtime/feature"
)

// ProviderCapability represents a specific operation a provider can perform.
type ProviderCapability string

const (
	// Read operations
	CapabilityGetSecret     ProviderCapability = "GetSecret"
	CapabilityGetSecretMap  ProviderCapability = "GetSecretMap"
	CapabilityGetAllSecrets ProviderCapability = "GetAllSecrets"

	// Write operations
	CapabilityPushSecret   ProviderCapability = "PushSecret"
	CapabilityDeleteSecret ProviderCapability = "DeleteSecret"
	CapabilitySecretExists ProviderCapability = "SecretExists"

	// Validation operations
	CapabilityValidate      ProviderCapability = "Validate"
	CapabilityValidateStore ProviderCapability = "ValidateStore"
)

// CapabilityInfo describes a single capability with its maturity and safety level.
type CapabilityInfo struct {
	Capability ProviderCapability
	Maturity   feature.Maturity
	Safety     feature.Safety
	// Optional: Additional metadata
	Notes string
}

// ProviderCapabilities represents the complete capability set for a provider.
type ProviderCapabilities struct {
	// Overall capabilities (ReadOnly, WriteOnly, ReadWrite)
	StoreCapabilities SecretStoreCapabilities

	// Detailed capability matrix
	Capabilities []CapabilityInfo
}

// ProviderMetadata combines all provider registration information.
type ProviderMetadata struct {
	Name              string
	Provider          Provider
	ProviderSpec      *SecretStoreProvider
	MaintenanceStatus MaintenanceStatus
	Capabilities      ProviderCapabilities
}

var (
	providerMetadata = make(map[string]*ProviderMetadata)
	metadataLock     sync.RWMutex
)
