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
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// BarbicanProviderRef defines a reference to a secret containing credentials for the Barbican provider.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type BarbicanProviderRef struct {
	Value     string                    `json:"value,omitempty"`
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// BarbicanProvider setup a store to sync secrets with barbican.
type BarbicanProvider struct {
	AuthURL    string              `json:"authURL,omitempty"`
	TenantName string              `json:"tenantName,omitempty"`
	DomainName string              `json:"domainName,omitempty"`
	Region     string              `json:"region,omitempty"`
	Username   BarbicanProviderRef `json:"username"`
	Password   BarbicanProviderRef `json:"password"`
}
