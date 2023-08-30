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

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Configures an store to sync secrets using a IBM Cloud Secrets Manager
// backend.
type IBMProvider struct {
	// Auth configures how secret-manager authenticates with the IBM secrets manager.
	Auth IBMAuth `json:"auth"`

	// ServiceURL is the Endpoint URL that is specific to the Secrets Manager service instance
	ServiceURL *string `json:"serviceUrl,omitempty"`
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type IBMAuth struct {
	SecretRef     *IBMAuthSecretRef     `json:"secretRef,omitempty"`
	ContainerAuth *IBMAuthContainerAuth `json:"containerAuth,omitempty"`
}

type IBMAuthSecretRef struct {
	// The SecretAccessKey is used for authentication
	SecretAPIKey esmeta.SecretKeySelector `json:"secretApiKeySecretRef,omitempty"`
}

// IBM Container-based auth with IAM Trusted Profile.
type IBMAuthContainerAuth struct {
	// the IBM Trusted Profile
	Profile string `json:"profile"`

	// Location the token is mounted on the pod
	TokenLocation string `json:"tokenLocation,omitempty"`

	IAMEndpoint string `json:"iamEndpoint,omitempty"`
}
