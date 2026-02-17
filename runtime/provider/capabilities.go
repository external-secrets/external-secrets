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

// CapabilityName represents a specific operation a provider can perform
type CapabilityName string

// A series of capability Names for standard implementations.
// This can _later_ be moved to more interfaces in this package, guaranteeing an easy observation
const (
	CapabilityGetSecret              CapabilityName = "GetSecret"
	CapabilityGetSecretMap           CapabilityName = "GetSecretMap"
	CapabilityGetAllSecrets          CapabilityName = "GetAllSecrets"
	CapabilityPushSecret             CapabilityName = "PushSecret"
	CapabilityDeleteSecret           CapabilityName = "DeleteSecret"
	CapabilitySecretExists           CapabilityName = "SecretExists"
	CapabilityValidate               CapabilityName = "Validate"
	CapabilityValidateStore          CapabilityName = "ValidateStore"
	CapabilityFindByName             CapabilityName = "FindByName"
	CapabilityFindByTag              CapabilityName = "FindByTag"
	CapabilityMetadataPolicyFetch    CapabilityName = "MetadataPolicyFetch"
	CapabilityReferentAuthentication CapabilityName = "ReferentAuthentication"
	CapabilityDeletionPolicy         CapabilityName = "DeletionPolicy"
)

// Capability describes a capability with optional notes
type Capability struct {
	Name  CapabilityName `json:"name"`
	Notes string         `json:"notes,omitempty"`
}
