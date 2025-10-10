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
)

// ClusterGeneratorSpec defines the desired state of a ClusterGenerator.
type ClusterGeneratorSpec struct {
	// Kind the kind of this generator.
	Kind GeneratorKind `json:"kind"`

	// Generator the spec for this generator, must match the kind.
	Generator GeneratorSpec `json:"generator"`
}

// GeneratorKind represents a kind of generator.
// +kubebuilder:validation:Enum=ACRAccessToken;CloudsmithAccessToken;ECRAuthorizationToken;Fake;GCRAccessToken;GithubAccessToken;QuayAccessToken;Password;SSHKey;STSSessionToken;UUID;VaultDynamicSecret;Webhook;Grafana
type GeneratorKind string

const (
	// GeneratorKindACRAccessToken represents an Azure Container Registry access token generator.
	GeneratorKindACRAccessToken GeneratorKind = "ACRAccessToken"
	// GeneratorKindECRAuthorizationToken represents an AWS ECR authorization token generator.
	GeneratorKindECRAuthorizationToken GeneratorKind = "ECRAuthorizationToken"
	// GeneratorKindFake represents a fake generator for testing purposes.
	GeneratorKindFake GeneratorKind = "Fake"
	// GeneratorKindGCRAccessToken represents a Google Container Registry access token generator.
	GeneratorKindGCRAccessToken GeneratorKind = "GCRAccessToken"
	// GeneratorKindGithubAccessToken represents a GitHub access token generator.
	GeneratorKindGithubAccessToken GeneratorKind = "GithubAccessToken"
	// GeneratorKindQuayAccessToken represents a Quay access token generator.
	GeneratorKindQuayAccessToken GeneratorKind = "QuayAccessToken"
	// GeneratorKindPassword represents a password generator.
	GeneratorKindPassword GeneratorKind = "Password"
	// GeneratorKindSSHKey represents an SSH key generator.
	GeneratorKindSSHKey GeneratorKind = "SSHKey"
	// GeneratorKindSTSSessionToken represents an AWS STS session token generator.
	GeneratorKindSTSSessionToken GeneratorKind = "STSSessionToken"
	// GeneratorKindUUID represents a UUID generator.
	GeneratorKindUUID GeneratorKind = "UUID"
	// GeneratorKindVaultDynamicSecret represents a HashiCorp Vault dynamic secret generator.
	GeneratorKindVaultDynamicSecret GeneratorKind = "VaultDynamicSecret"
	// GeneratorKindWebhook represents a webhook-based generator.
	GeneratorKindWebhook GeneratorKind = "Webhook"
	// GeneratorKindGrafana represents a Grafana token generator.
	GeneratorKindGrafana GeneratorKind = "Grafana"
	// GeneratorKindMFA represents a Multi-Factor Authentication generator.
	GeneratorKindMFA GeneratorKind = "MFA"
	// GeneratorKindCloudsmithAccessToken represents a Cloudsmith access token generator.
	GeneratorKindCloudsmithAccessToken GeneratorKind = "CloudsmithAccessToken"
)

// GeneratorSpec defines the configuration for various supported generator types.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type GeneratorSpec struct {
	ACRAccessTokenSpec        *ACRAccessTokenSpec        `json:"acrAccessTokenSpec,omitempty"`
	CloudsmithAccessTokenSpec *CloudsmithAccessTokenSpec `json:"cloudsmithAccessTokenSpec,omitempty"`
	ECRAuthorizationTokenSpec *ECRAuthorizationTokenSpec `json:"ecrAuthorizationTokenSpec,omitempty"`
	FakeSpec                  *FakeSpec                  `json:"fakeSpec,omitempty"`
	GCRAccessTokenSpec        *GCRAccessTokenSpec        `json:"gcrAccessTokenSpec,omitempty"`
	GithubAccessTokenSpec     *GithubAccessTokenSpec     `json:"githubAccessTokenSpec,omitempty"`
	QuayAccessTokenSpec       *QuayAccessTokenSpec       `json:"quayAccessTokenSpec,omitempty"`
	PasswordSpec              *PasswordSpec              `json:"passwordSpec,omitempty"`
	SSHKeySpec                *SSHKeySpec                `json:"sshKeySpec,omitempty"`
	STSSessionTokenSpec       *STSSessionTokenSpec       `json:"stsSessionTokenSpec,omitempty"`
	UUIDSpec                  *UUIDSpec                  `json:"uuidSpec,omitempty"`
	VaultDynamicSecretSpec    *VaultDynamicSecretSpec    `json:"vaultDynamicSecretSpec,omitempty"`
	WebhookSpec               *WebhookSpec               `json:"webhookSpec,omitempty"`
	GrafanaSpec               *GrafanaSpec               `json:"grafanaSpec,omitempty"`
	MFASpec                   *MFASpec                   `json:"mfaSpec,omitempty"`
}

// ClusterGenerator represents a cluster-wide generator which can be referenced as part of `generatorRef` fields.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Cluster,categories={external-secrets, external-secrets-generators}
type ClusterGenerator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterGeneratorSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterGeneratorList contains a list of ClusterGenerator resources.
type ClusterGeneratorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterGenerator `json:"items"`
}
