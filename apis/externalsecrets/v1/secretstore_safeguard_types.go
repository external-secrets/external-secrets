/*
Copyright © The ESO Authors

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

// SafeguardProviderSecretRef references a value that can be specified directly or via a secret.
type SafeguardProviderSecretRef struct {
	// Value can be specified directly to set a value without using a secret.
	// +optional
	Value string `json:"value,omitempty"`

	// SecretRef references a key in a secret that will be used as value.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
}

// SafeguardA2AAuth configures Application-to-Application authentication with a client certificate.
type SafeguardA2AAuth struct {
	// Certificate is the PEM-encoded client certificate (leaf, chain, and optionally the private key).
	// +required
	Certificate SafeguardProviderSecretRef `json:"certificate"`

	// CertificateKey is the PEM-encoded private key when it is not included in Certificate.
	// +optional
	CertificateKey *SafeguardProviderSecretRef `json:"certificateKey,omitempty"`

	// CertificatePassword decrypts an encrypted PEM private key.
	// +optional
	CertificatePassword *SafeguardProviderSecretRef `json:"certificatePassword,omitempty"`
}

// SafeguardAuth configures how the operator authenticates to Safeguard.
// +kubebuilder:validation:MaxProperties=1
type SafeguardAuth struct {
	// A2A authenticates with a client certificate for Application-to-Application credential retrieval.
	// +optional
	A2A *SafeguardA2AAuth `json:"a2a,omitempty"`
}

// SafeguardProvider configures a store to sync secrets from One Identity Safeguard for Privileged Passwords.
type SafeguardProvider struct {
	// Appliance is the Safeguard appliance host name or URL. Must use the https scheme.
	// +required
	Appliance string `json:"appliance"`

	// Auth configures authentication to Safeguard.
	// +required
	Auth SafeguardAuth `json:"auth"`

	// APIVersion overrides the default Safeguard API version (v4).
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	// CABundle is a PEM-encoded CA bundle used to validate the appliance TLS certificate.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// CAProvider references a ConfigMap or Secret containing the CA bundle.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`
}
