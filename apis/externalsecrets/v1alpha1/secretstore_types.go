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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretStoreSpec defines the desired state of SecretStore.
type SecretStoreSpec struct {
	// Used to select the correct ESO controller (think: ingress.ingressClassName)
	// The ESO controller is instantiated with a specific controller name and filters ES based on this property
	// +optional
	Controller string `json:"controller,omitempty"`

	// Used to configure the provider. Only one provider may be set
	Provider *SecretStoreProvider `json:"provider"`

	// Used to configure http retries if failed
	// +optional
	RetrySettings *SecretStoreRetrySettings `json:"retrySettings,omitempty"`
}

// SecretStoreProvider contains the provider-specific configration.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type SecretStoreProvider struct {
	// AWS configures this store to sync secrets using AWS Secret Manager provider
	// +optional
	AWS *AWSProvider `json:"aws,omitempty"`

	// AzureKV configures this store to sync secrets using Azure Key Vault provider
	// +optional
	AzureKV *AzureKVProvider `json:"azurekv,omitempty"`

	// Akeyless configures this store to sync secrets using Akeyless Vault provider
	// +optional
	Akeyless *AkeylessProvider `json:"akeyless,omitempty"`

	// Vault configures this store to sync secrets using Hashi provider
	// +optional
	Vault *VaultProvider `json:"vault,omitempty"`

	// GCPSM configures this store to sync secrets using Google Cloud Platform Secret Manager provider
	// +optional
	GCPSM *GCPSMProvider `json:"gcpsm,omitempty"`

	// Oracle configures this store to sync secrets using Oracle Vault provider
	// +optional
	Oracle *OracleProvider `json:"oracle,omitempty"`

	// IBM configures this store to sync secrets using IBM Cloud provider
	// +optional
	IBM *IBMProvider `json:"ibm,omitempty"`

	// YandexLockbox configures this store to sync secrets using Yandex Lockbox provider
	// +optional
	YandexLockbox *YandexLockboxProvider `json:"yandexlockbox,omitempty"`

	// GitLab configures this store to sync secrets using GitLab Variables provider
	// +optional
	Gitlab *GitlabProvider `json:"gitlab,omitempty"`

	// Alibaba configures this store to sync secrets using Alibaba Cloud provider
	// +optional
	Alibaba *AlibabaProvider `json:"alibaba,omitempty"`

	// Webhook configures this store to sync secrets using a generic templated webhook
	// +optional
	Webhook *WebhookProvider `json:"webhook,omitempty"`

	// Kubernetes configures this store to sync secrets using a Kubernetes cluster provider
	// +optional
	Kubernetes *KubernetesProvider `json:"kubernetes,omitempty"`

	PasswordDepot *PasswordDepotProvider `json:"passworddepot,omitempty"`

	// Fake configures a store with static key/value pairs
	// +optional
	Fake *FakeProvider `json:"fake,omitempty"`
}

type SecretStoreRetrySettings struct {
	MaxRetries    *int32  `json:"maxRetries,omitempty"`
	RetryInterval *string `json:"retryInterval,omitempty"`
}

type SecretStoreConditionType string

const (
	SecretStoreReady SecretStoreConditionType = "Ready"

	ReasonInvalidStore          = "InvalidStoreConfiguration"
	ReasonInvalidProviderConfig = "InvalidProviderConfig"
	ReasonValidationFailed      = "ValidationFailed"
	ReasonStoreValid            = "Valid"
)

type SecretStoreStatusCondition struct {
	Type   SecretStoreConditionType `json:"type"`
	Status corev1.ConditionStatus   `json:"status"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// SecretStoreStatus defines the observed state of the SecretStore.
type SecretStoreStatus struct {
	// +optional
	Conditions []SecretStoreStatusCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// SecretStore represents a secure external location for storing secrets, which can be referenced as part of `storeRef` fields.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion
// +kubebuilder:resource:scope=Namespaced,categories={externalsecrets},shortName=ss
type SecretStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretStoreSpec   `json:"spec,omitempty"`
	Status SecretStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretStoreList contains a list of SecretStore resources.
type SecretStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretStore `json:"items"`
}

// +kubebuilder:object:root=true

// ClusterSecretStore represents a secure external location for storing secrets, which can be referenced as part of `storeRef` fields.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:deprecatedversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={externalsecrets},shortName=css
type ClusterSecretStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretStoreSpec   `json:"spec,omitempty"`
	Status SecretStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterSecretStoreList contains a list of ClusterSecretStore resources.
type ClusterSecretStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterSecretStore `json:"items"`
}
