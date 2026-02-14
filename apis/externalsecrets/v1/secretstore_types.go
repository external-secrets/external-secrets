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

	// Used to configure HTTP retries on failures.
	// +optional
	RetrySettings *SecretStoreRetrySettings `json:"retrySettings,omitempty"`

	// Used to configure store refresh interval in seconds. Empty or 0 will default to the controller config.
	// +optional
	RefreshInterval int `json:"refreshInterval,omitempty"`

	// Used to constrain a ClusterSecretStore to specific namespaces. Relevant only to ClusterSecretStore.
	// +optional
	Conditions []ClusterSecretStoreCondition `json:"conditions,omitempty"`
}

// ClusterSecretStoreCondition describes a condition by which to choose namespaces to process ExternalSecrets in
// for a ClusterSecretStore instance.
type ClusterSecretStoreCondition struct {
	// Choose namespace using a labelSelector
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Choose namespaces by name
	// +optional
	// +kubebuilder:validation:items:MinLength:=1
	// +kubebuilder:validation:items:MaxLength:=63
	// +kubebuilder:validation:items:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Namespaces []string `json:"namespaces,omitempty"`

	// Choose namespaces by using regex matching
	// +optional
	NamespaceRegexes []string `json:"namespaceRegexes,omitempty"`
}

// SecretStoreProvider contains the provider-specific configuration.
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

	// BitwardenSecretsManager configures this store to sync secrets using BitwardenSecretsManager provider
	// +optional
	BitwardenSecretsManager *BitwardenSecretsManagerProvider `json:"bitwardensecretsmanager,omitempty"`

	// Vault configures this store to sync secrets using the HashiCorp Vault provider.
	// +optional
	Vault *VaultProvider `json:"vault,omitempty"`

	// Ovh configures this store to sync secrets using the OVHcloud provider.
	// +optional
	Ovh *OvhProvider `json:"ovh,omitempty"`

	// GCPSM configures this store to sync secrets using Google Cloud Platform Secret Manager provider
	// +optional
	GCPSM *GCPSMProvider `json:"gcpsm,omitempty"`

	// Oracle configures this store to sync secrets using Oracle Vault provider
	// +optional
	Oracle *OracleProvider `json:"oracle,omitempty"`

	// IBM configures this store to sync secrets using IBM Cloud provider
	// +optional
	IBM *IBMProvider `json:"ibm,omitempty"`

	// YandexCertificateManager configures this store to sync secrets using Yandex Certificate Manager provider
	// +optional
	YandexCertificateManager *YandexCertificateManagerProvider `json:"yandexcertificatemanager,omitempty"`

	// YandexLockbox configures this store to sync secrets using Yandex Lockbox provider
	// +optional
	YandexLockbox *YandexLockboxProvider `json:"yandexlockbox,omitempty"`

	// Github configures this store to push GitHub Actions secrets using the GitHub API provider.
	// Note: This provider only supports write operations (PushSecret) and cannot fetch secrets from GitHub
	// +optional
	Github *GithubProvider `json:"github,omitempty"`

	// GitLab configures this store to sync secrets using GitLab Variables provider
	// +optional
	Gitlab *GitlabProvider `json:"gitlab,omitempty"`

	// OnePassword configures this store to sync secrets using the 1Password Cloud provider
	// +optional
	OnePassword *OnePasswordProvider `json:"onepassword,omitempty"`

	// OnePasswordSDK configures this store to use 1Password's new Go SDK to sync secrets.
	// +optional
	OnePasswordSDK *OnePasswordSDKProvider `json:"onepasswordSDK,omitempty"`

	// Webhook configures this store to sync secrets using a generic templated webhook
	// +optional
	Webhook *WebhookProvider `json:"webhook,omitempty"`

	// Kubernetes configures this store to sync secrets using a Kubernetes cluster provider
	// +optional
	Kubernetes *KubernetesProvider `json:"kubernetes,omitempty"`

	// Fake configures a store with static key/value pairs
	// +optional
	Fake *FakeProvider `json:"fake,omitempty"`

	// Senhasegura configures this store to sync secrets using senhasegura provider
	// +optional
	Senhasegura *SenhaseguraProvider `json:"senhasegura,omitempty"`

	// Scaleway configures this store to sync secrets using the Scaleway provider.
	// +optional
	Scaleway *ScalewayProvider `json:"scaleway,omitempty"`

	// Doppler configures this store to sync secrets using the Doppler provider
	// +optional
	Doppler *DopplerProvider `json:"doppler,omitempty"`

	// Previder configures this store to sync secrets using the Previder provider
	// +optional
	Previder *PreviderProvider `json:"previder,omitempty"`

	// Onboardbase configures this store to sync secrets using the Onboardbase provider
	// +optional
	Onboardbase *OnboardbaseProvider `json:"onboardbase,omitempty"`

	// KeeperSecurity configures this store to sync secrets using the KeeperSecurity provider
	// +optional
	KeeperSecurity *KeeperSecurityProvider `json:"keepersecurity,omitempty"`

	// Conjur configures this store to sync secrets using conjur provider
	// +optional
	Conjur *ConjurProvider `json:"conjur,omitempty"`

	// Delinea DevOps Secrets Vault
	// https://docs.delinea.com/online-help/products/devops-secrets-vault/current
	// +optional
	Delinea *DelineaProvider `json:"delinea,omitempty"`

	// SecretServer configures this store to sync secrets using SecretServer provider
	// https://docs.delinea.com/online-help/secret-server/start.htm
	// +optional
	SecretServer *SecretServerProvider `json:"secretserver,omitempty"`

	// Chef configures this store to sync secrets with chef server
	// +optional
	Chef *ChefProvider `json:"chef,omitempty"`

	// Pulumi configures this store to sync secrets using the Pulumi provider
	// +optional
	Pulumi *PulumiProvider `json:"pulumi,omitempty"`

	// Fortanix configures this store to sync secrets using the Fortanix provider
	// +optional
	Fortanix *FortanixProvider `json:"fortanix,omitempty"`

	// +optional
	PasswordDepot *PasswordDepotProvider `json:"passworddepot,omitempty"`

	// +optional
	Passbolt *PassboltProvider `json:"passbolt,omitempty"`

	// DVLS configures this store to sync secrets using Devolutions Server provider
	// +optional
	DVLS *DVLSProvider `json:"dvls,omitempty"`

	// Infisical configures this store to sync secrets using the Infisical provider
	// +optional
	Infisical *InfisicalProvider `json:"infisical,omitempty"`

	// Beyondtrust configures this store to sync secrets using Password Safe provider.
	// +optional
	Beyondtrust *BeyondtrustProvider `json:"beyondtrust,omitempty"`

	// CloudruSM configures this store to sync secrets using the Cloud.ru Secret Manager provider
	// +optional
	CloudruSM *CloudruSMProvider `json:"cloudrusm,omitempty"`

	// Volcengine configures this store to sync secrets using the Volcengine provider
	// +optional
	Volcengine *VolcengineProvider `json:"volcengine,omitempty"`

	// Ngrok configures this store to sync secrets using the ngrok provider.
	// +optional
	Ngrok *NgrokProvider `json:"ngrok,omitempty"`

	// Barbican configures this store to sync secrets using the OpenStack Barbican provider
	// +optional
	Barbican *BarbicanProvider `json:"barbican,omitempty"`
}

// CAProviderType defines the type of provider for certificate authority.
type CAProviderType string

// Supported CA provider types.
const (
	// CAProviderTypeSecret indicates that the CA certificate is stored in a Secret resource.
	CAProviderTypeSecret CAProviderType = "Secret"
	// CAProviderTypeConfigMap indicates that the CA certificate is stored in a ConfigMap resource.
	CAProviderTypeConfigMap CAProviderType = "ConfigMap"
)

// CAProvider provides a custom certificate authority for accessing the provider's store.
// The CAProvider points to a Secret or ConfigMap resource that contains a PEM-encoded certificate.
type CAProvider struct {
	// The type of provider to use such as "Secret", or "ConfigMap".
	// +kubebuilder:validation:Enum="Secret";"ConfigMap"
	Type CAProviderType `json:"type"`

	// The name of the object located at the provider type.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name"`

	// The key where the CA certificate can be found in the Secret or ConfigMap.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[-._a-zA-Z0-9]+$
	Key string `json:"key,omitempty"`

	// The namespace the Provider type is in.
	// Can only be defined when used in a ClusterSecretStore.
	// +optional
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Namespace *string `json:"namespace,omitempty"`
}

// SecretStoreRetrySettings defines the retry settings for accessing external secrets manager stores.
type SecretStoreRetrySettings struct {
	MaxRetries    *int32  `json:"maxRetries,omitempty"`
	RetryInterval *string `json:"retryInterval,omitempty"`
}

// SecretStoreConditionType represents the condition of the SecretStore.
type SecretStoreConditionType string

// These are valid conditions of a secret store.
const (
	// SecretStoreReady indicates that the store is ready and able to serve requests.
	SecretStoreReady SecretStoreConditionType = "Ready"

	ReasonInvalidStore          = "InvalidStoreConfiguration"
	ReasonInvalidProviderConfig = "InvalidProviderConfig"
	ReasonValidationFailed      = "ValidationFailed"
	ReasonValidationUnknown     = "ValidationUnknown"
	ReasonStoreValid            = "Valid"
	StoreUnmaintained           = "StoreUnmaintained"
	StoreDeprecated             = "StoreDeprecated"
)

// SecretStoreStatusCondition contains condition information for a SecretStore.
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

// SecretStoreCapabilities defines the possible operations a SecretStore can do.
type SecretStoreCapabilities string

// These are the valid capabilities of a secret store.
const (
	// SecretStoreReadOnly indicates that the store can only read secrets.
	SecretStoreReadOnly SecretStoreCapabilities = "ReadOnly"
	// SecretStoreWriteOnly indicates that the store can only write secrets.
	SecretStoreWriteOnly SecretStoreCapabilities = "WriteOnly"
	// SecretStoreReadWrite indicates that the store can both read and write secrets.
	SecretStoreReadWrite SecretStoreCapabilities = "ReadWrite"
)

// SecretStoreStatus defines the observed state of the SecretStore.
type SecretStoreStatus struct {
	// +optional
	Conditions []SecretStoreStatusCondition `json:"conditions,omitempty"`
	// +optional
	Capabilities SecretStoreCapabilities `json:"capabilities,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// SecretStore represents a secure external location for storing secrets, which can be referenced as part of `storeRef` fields.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Capabilities",type=string,JSONPath=`.status.capabilities`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets},shortName=ss
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
// +kubebuilder:storageversion

// ClusterSecretStore represents a secure external location for storing secrets, which can be referenced as part of `storeRef` fields.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Capabilities",type=string,JSONPath=`.status.capabilities`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Cluster,categories={external-secrets},shortName=css
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
