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

// BeyondtrustWorkloadCredentialsDynamicSecretSpec defines the desired spec for BeyondtrustWorkloadCredentials dynamic generator.
// This generator enables obtaining temporary, short-lived credentials from BeyondTrust Workload Credentials.
// For more information, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api
type BeyondtrustWorkloadCredentialsDynamicSecretSpec struct {
	// Controller selects the controller that should handle this generator.
	// Leave empty to use the default controller.
	// +optional
	Controller string `json:"controller,omitempty"`

	// Provider contains the BeyondtrustWorkloadCredentials provider configuration including authentication,
	// server connection details, and the folder path to the dynamic secret definition.
	// The folderPath should point to a dynamic secret definition that has been created in
	// BeyondTrust Workload Credentials (e.g., "production/aws-temp").
	// For setup details, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api
	// +required
	Provider *esv1.BeyondtrustWorkloadCredentialsProvider `json:"provider"`

	// RetrySettings configures exponential backoff for failed API requests.
	// If not specified, uses the default retry settings.
	// +optional
	RetrySettings *esv1.SecretStoreRetrySettings `json:"retrySettings,omitempty"`
}

// BeyondtrustWorkloadCredentialsDynamicSecret represents a generator that requests dynamic credentials from BeyondTrust Workload Credentials.
// This generator calls the BeyondTrust Workload Credentials API to generate fresh, temporary credentials
// (such as AWS STS credentials) each time an ExternalSecret is refreshed.
// Dynamic secret definitions must be created in BeyondTrust Workload Credentials before they can be referenced.
// For complete documentation, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type BeyondtrustWorkloadCredentialsDynamicSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BeyondtrustWorkloadCredentialsDynamicSecretSpec `json:"spec,omitempty"`
}

// BeyondtrustWorkloadCredentialsDynamicSecretList contains a list of BeyondtrustWorkloadCredentialsDynamicSecret resources.
// +kubebuilder:object:root=true
type BeyondtrustWorkloadCredentialsDynamicSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BeyondtrustWorkloadCredentialsDynamicSecret `json:"items"`
}
