/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

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

// PluginProvider configures a store to sync secrets using external plugins via HashiCorp go-plugin.
// The plugins communicate via gRPC over Unix domain sockets or network connections.
type PluginProvider struct {
	// Endpoint specifies the plugin endpoint. Can be a Unix socket path (unix:///path/to/socket)
	// or a network address (tcp://host:port). The plugin must be running and listening on this endpoint.
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`

	// Timeout for plugin operations.
	// Default is 30 seconds.
	// +optional
	// +kubebuilder:default="30s"
	Timeout *string `json:"timeout,omitempty"`

	// Auth configures authentication for plugin communication.
	// +optional
	Auth *PluginAuth `json:"auth,omitempty"`
}

// PluginAuth configures authentication for plugin communication.
type PluginAuth struct {
	// SecretRef holds secret references for plugin authentication.
	// +optional
	SecretRef *PluginAuthSecretRef `json:"secretRef,omitempty"`
}

// PluginAuthSecretRef holds secret references for plugin authentication.
type PluginAuthSecretRef struct {
	// Token is a reference to a secret containing an authentication token for plugins.
	// +optional
	Token *esmeta.SecretKeySelector `json:"token,omitempty"`

	// Certificate is a reference to a secret containing TLS certificate for secure plugin communication.
	// +optional
	Certificate *esmeta.SecretKeySelector `json:"certificate,omitempty"`

	// PrivateKey is a reference to a secret containing TLS private key for secure plugin communication.
	// +optional
	PrivateKey *esmeta.SecretKeySelector `json:"privateKey,omitempty"`
}
