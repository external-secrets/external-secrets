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

// EtcdProvider configures a store to sync secrets using etcd.
type EtcdProvider struct {
	// Endpoints is a list of etcd endpoints to connect to.
	// +kubebuilder:validation:MinItems=1
	Endpoints []string `json:"endpoints"`

	// Prefix is the path prefix for all secrets stored in etcd.
	// Defaults to "/external-secrets/" if not specified.
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// Auth contains authentication configuration for etcd.
	// +optional
	Auth *EtcdAuth `json:"auth,omitempty"`

	// CAProvider is a reference to a Certificate Authority bundle to use when
	// verifying the etcd server's certificate.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`

	// RawMode when enabled, pushes secrets as raw values without wrapping them
	// in the standard external-secrets data/metadata structure.
	// This is useful when you need the exact value stored in etcd without any wrapper.
	// Note: When RawMode is enabled, the managed-by tracking is disabled.
	// +optional
	RawMode bool `json:"rawMode,omitempty"`
}

// EtcdAuth contains authentication configuration for etcd.
type EtcdAuth struct {
	// SecretRef contains username and password credentials for etcd authentication.
	// +optional
	SecretRef *EtcdAuthSecretRef `json:"secretRef,omitempty"`

	// TLS contains TLS client certificate authentication configuration.
	// +optional
	TLS *EtcdTLSAuth `json:"tls,omitempty"`
}

// EtcdAuthSecretRef contains username and password secret references.
type EtcdAuthSecretRef struct {
	// Username is a reference to a secret containing the etcd username.
	// +optional
	Username esmeta.SecretKeySelector `json:"username,omitempty"`

	// Password is a reference to a secret containing the etcd password.
	// +optional
	Password esmeta.SecretKeySelector `json:"password,omitempty"`
}

// EtcdTLSAuth contains TLS client certificate authentication configuration.
type EtcdTLSAuth struct {
	// ClientCert is a reference to a secret containing the TLS client certificate.
	ClientCert esmeta.SecretKeySelector `json:"clientCert"`

	// ClientKey is a reference to a secret containing the TLS client key.
	ClientKey esmeta.SecretKeySelector `json:"clientKey"`
}
