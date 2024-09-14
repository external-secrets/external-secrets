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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// AkeylessProvider Configures an store to sync secrets using Akeyless KV.
type WebhookProvider struct {
	// Webhook Method
	// +optional, default GET
	Method string `json:"method,omitempty"`

	// Webhook url to call
	URL string `json:"url"`

	// Headers
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// Body
	// +optional
	Body string `json:"body,omitempty"`

	// Timeout
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Result formatting
	Result WebhookResult `json:"result"`

	// Secrets to fill in templates
	// These secrets will be passed to the templating function as key value pairs under the given name
	// +optional
	Secrets []WebhookSecret `json:"secrets,omitempty"`

	// PEM encoded CA bundle used to validate webhook server certificate. Only used
	// if the Server URL is using HTTPS protocol. This parameter is ignored for
	// plain HTTP protocol connection. If not set the system root certificates
	// are used to validate the TLS connection.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// The provider for the CA bundle to use to validate webhook server certificate.
	// +optional
	CAProvider *WebhookCAProvider `json:"caProvider,omitempty"`
}

type WebhookCAProviderType string

const (
	WebhookCAProviderTypeSecret    WebhookCAProviderType = "Secret"
	WebhookCAProviderTypeConfigMap WebhookCAProviderType = "ConfigMap"
)

// Defines a location to fetch the cert for the webhook provider from.
type WebhookCAProvider struct {
	// The type of provider to use such as "Secret", or "ConfigMap".
	// +kubebuilder:validation:Enum="Secret";"ConfigMap"
	Type WebhookCAProviderType `json:"type"`

	// The name of the object located at the provider type.
	Name string `json:"name"`

	// The key the value inside of the provider type to use, only used with "Secret" type
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`

	// The namespace the Provider type is in.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

type WebhookResult struct {
	// Json path of return value
	// +optional
	JSONPath string `json:"jsonPath,omitempty"`
}

type WebhookSecret struct {
	// Name of this secret in templates
	Name string `json:"name"`

	// Secret ref to fill in credentials
	SecretRef esmeta.SecretKeySelector `json:"secretRef"`
}
