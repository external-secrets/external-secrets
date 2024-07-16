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

package webhook

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type Spec struct {
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
	CAProvider *esv1beta1.CAProvider `json:"caProvider,omitempty"`
}

type Result struct {
	// Json path of return value
	// +optional
	JSONPath string `json:"jsonPath,omitempty"`
}

type Secret struct {
	// Name of this secret in templates
	Name string `json:"name"`

	// Secret ref to fill in credentials
	SecretRef SecretKeySelector `json:"secretRef"`
}

type SecretKeySelector struct {
	// The name of the Secret resource being referred to.
	Name string `json:"name,omitempty"`
	// The key where the token is found.
	Key string `json:"key,omitempty"`

	Namespace *string `json:"namespace,omitempty"`
}
