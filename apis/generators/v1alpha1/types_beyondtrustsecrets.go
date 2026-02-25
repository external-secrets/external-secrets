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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// BeyondtrustSecretsDynamicSecretSpec defines the desired spec for BeyondtrustSecrets dynamic generator.
type BeyondtrustSecretsDynamicSecretSpec struct {
	// Controller selects the controller that should handle this generator
	// +optional
	Controller string `json:"controller,omitempty"`

	// Provider contains the BeyondtrustSecrets provider configuration (apiUrl, siteId, auth)
	Provider *esv1.BeyondtrustSecretsProvider `json:"provider"`

	// Used to configure http retries if failed
	// +optional
	RetrySettings *esv1.SecretStoreRetrySettings `json:"retrySettings,omitempty"`
}

// BeyondtrustSecretsDynamicSecret represents a generator that requests dynamic credentials from BeyondtrustSecrets
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type BeyondtrustSecretsDynamicSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BeyondtrustSecretsDynamicSecretSpec `json:"spec,omitempty"`
}

// BeyondtrustSecretsDynamicSecretList contains a list of BeyondtrustSecretsDynamicSecret resources.
// +kubebuilder:object:root=true
type BeyondtrustSecretsDynamicSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BeyondtrustSecretsDynamicSecret `json:"items"`
}
