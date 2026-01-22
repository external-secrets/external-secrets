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

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// OvhProvider holds the configuration to synchronize secrets with OVHcloud's Secret Manager.
// +optional
type OvhProvider struct {
	// specifies the OKMS server endpoint.
	// +required
	Server string `json:"server"`
	// specifies the OKMS ID.
	// +required
	OkmsID string `json:"okmsid"`
	// Enables or disables check-and-set (CAS) (default: false).
	// +optional
	CasRequired *bool `json:"casRequired,omitempty"`
	// Setup a timeout in seconds when requests to the KMS are made (default: 30).
	// +optional
	// +kubebuilder:default=30
	OkmsTimeout *uint32 `json:"okmsTimeout,omitempty"`
	// Authentication method (mtls or token).
	// +required
	Auth OvhAuth `json:"auth"`
}

// OvhAuth tells the controller how to authenticate to OVHcloud's Secret Manager, either using mTLS or a token.
// +required
type OvhAuth struct {
	// +optional
	ClientMTLS *OvhClientMTLS `json:"mtls,omitempty"`
	// +optional
	ClientToken *OvhClientToken `json:"token,omitempty"`
}

// OvhClientMTLS defines the configuration required to authenticate to OVHcloud's Secret Manager using mTLS.
// +optional
type OvhClientMTLS struct {
	// +required
	ClientCertificate *esmeta.SecretKeySelector `json:"certSecretRef,omitempty"`
	// +required
	ClientKey *esmeta.SecretKeySelector `json:"keySecretRef,omitempty"`
}

// OvhClientToken defines the configuration required to authenticate to OVHcloud's Secret Manager using a token.
// +optional
type OvhClientToken struct {
	// +required
	ClientTokenSecret *esmeta.SecretKeySelector `json:"tokenSecretRef,omitempty"`
}
