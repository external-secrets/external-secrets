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

// Package webhook provides functionality for interacting with external webhook services
// to fetch and push secret data.
package webhook

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Spec defines the configuration for a webhook provider.
type Spec struct {
	// Webhook Method
	// +optional, default GET
	Method string `json:"method,omitempty"`

	// Webhook url to call
	URL string `json:"url"`

	// Headers
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// Auth specifies a authorization protocol. Only one protocol may be set.
	// +optional
	Auth *AuthorizationProtocol `json:"auth,omitempty"`

	// Body
	// +optional
	Body string `json:"body,omitempty"`

	// Timeout
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Result formatting
	Result Result `json:"result"`

	// Secrets to fill in templates
	// These secrets will be passed to the templating function as key value pairs under the given name
	// +optional
	Secrets []Secret `json:"secrets,omitempty"`

	// PEM encoded CA bundle used to validate webhook server certificate. Only used
	// if the Server URL is using HTTPS protocol. This parameter is ignored for
	// plain HTTP protocol connection. If not set the system root certificates
	// are used to validate the TLS connection.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// The provider for the CA bundle to use to validate webhook server certificate.
	// +optional
	CAProvider *esv1.CAProvider `json:"caProvider,omitempty"`
}

// AuthorizationProtocol contains the protocol-specific configuration
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type AuthorizationProtocol struct {
	// NTLMProtocol configures the store to use NTLM for auth
	// +optional
	NTLM *NTLMProtocol `json:"ntlm,omitempty"`

	// Define other protocols here
}

// NTLMProtocol contains the NTLM-specific configuration.
type NTLMProtocol struct {
	UserName esmeta.SecretKeySelector `json:"usernameSecret"`
	Password esmeta.SecretKeySelector `json:"passwordSecret"`
}

// Result defines how to process and extract data from webhook responses.
type Result struct {
	// Json path of return value
	// +optional
	JSONPath string `json:"jsonPath,omitempty"`
}

// Secret defines a secret that can be used in webhook templates.
type Secret struct {
	// Name of this secret in templates
	Name string `json:"name"`

	// Secret ref to fill in credentials
	SecretRef esmeta.SecretKeySelector `json:"secretRef"`
}
