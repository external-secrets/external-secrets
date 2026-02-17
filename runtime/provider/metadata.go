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

import v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

// Name contains the name of the provider in the Provider (feature) registry
type Name string

// Metadata represents the complete view of provider supportability state. It must be implemented in each provider.
type Metadata struct {
	Stability    Stability    `json:"stability"`
	Capabilities []Capability `json:"capabilities,omitempty"`
	Comment      string       `json:"comment,omitempty"`
}

// Stability represents the maturity level of a provider
type Stability string

const (
	StabilityAlpha        Stability = "Alpha"
	StabilityBeta         Stability = "Beta"
	StabilityStable       Stability = "Stable"
	StabilityUnmaintained Stability = "Unmaintained"
	StabilityDeprecated   Stability = "Deprecated"
)

// MaintenanceStatus derives the MaintenanceStatus reporter to the user based on Provider Metadata, see https://github.com/external-secrets/external-secrets/issues/5494
func (m Metadata) MaintenanceStatus() v1.MaintenanceStatus {
	if m.Stability == StabilityUnmaintained {
		return v1.MaintenanceStatusNotMaintained
	}
	if m.Stability == StabilityDeprecated {
		return v1.MaintenanceStatusDeprecated
	}
	return v1.MaintenanceStatusMaintained
}

// APICapabilities derives the SecretStore capabilities (ReadOnly/WriteOnly/ReadWrite)
// from the provider's full capability list in their metadata, to be used in API.
func (m Metadata) APICapabilities() v1.SecretStoreCapabilities {
	var canRead, canWrite bool

	for _, capability := range m.Capabilities {
		switch capability.Name {
		case CapabilityGetSecret, CapabilityGetSecretMap, CapabilityGetAllSecrets:
			canRead = true
		case CapabilityPushSecret, CapabilityDeleteSecret:
			canWrite = true
		}
	}

	if canRead && canWrite {
		return v1.SecretStoreReadWrite
	}
	if canWrite {
		return v1.SecretStoreWriteOnly
	}
	return v1.SecretStoreReadOnly
}

// MetadataReporter is a way to guarantee each provider will report their metadata.
type MetadataReporter interface {
	Metadata() Metadata
}
