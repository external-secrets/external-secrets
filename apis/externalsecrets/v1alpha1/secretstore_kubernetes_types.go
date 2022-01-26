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
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Configures a store to sync secrets with a Kubernetes instance.
type KubernetesProvider struct {
	// configures the Kubernetes server Address.
	// +kubebuilder:default= kubernetes.default
	// +optional
	Server string `json:"server,omitempty"`

	// Auth configures how secret-manager authenticates with a Kubernetes instance.
	// +optional
	Auth KubernetesAuth `json:"auth"`

	// +optional
	User string `json:"user"`

	//Remote namespace to fetch the secrets from
	// +kubebuilder:default= default
	// +optional
	RemoteNamespace string `json:"remoteNamespace"`
}

type KubernetesAuth struct {
	SecretRef KubernetesSecretRef `json:"secretRef"`
}

type KubernetesSecretRef struct {
	// +optional
	Certificate esmeta.SecretKeySelector `json:"certificate,omitempty"`
	// +optional
	Key         esmeta.SecretKeySelector `json:"key,omitempty"`
	CA          esmeta.SecretKeySelector `json:"ca,omitempty"`
	BearerToken esmeta.SecretKeySelector `json:"bearerToken,omitempty"`
}
