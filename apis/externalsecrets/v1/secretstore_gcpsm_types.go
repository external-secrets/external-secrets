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
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type SecretVersionSelectionPolicy string

const (
	// SecretVersionSelectionPolicyLatestOrFail means the provider always uses "latest", or fails if that version is disabled/destroyed.
	SecretVersionSelectionPolicyLatestOrFail SecretVersionSelectionPolicy = "LatestOrFail"

	// SecretVersionSelectionPolicyLatestOrFetch behaves like SecretVersionSelectionPolicyLatestOrFail but falls back to fetching the latest version if the version is DESTROYED or DISABLED.
	SecretVersionSelectionPolicyLatestOrFetch SecretVersionSelectionPolicy = "LatestOrFetch"
)

type GCPSMAuth struct {
	// +optional
	SecretRef *GCPSMAuthSecretRef `json:"secretRef,omitempty"`
	// +optional
	WorkloadIdentity *GCPWorkloadIdentity `json:"workloadIdentity,omitempty"`
	// +optional
	WorkloadIdentityFederation *GCPWorkloadIdentityFederation `json:"workloadIdentityFederation,omitempty"`
}

type GCPSMAuthSecretRef struct {
	// The SecretAccessKey is used for authentication
	// +optional
	SecretAccessKey esmeta.SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`
}

type GCPWorkloadIdentity struct {
	// +kubebuilder:validation:Required
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef"`
	// ClusterLocation is the location of the cluster
	// If not specified, it fetches information from the metadata server
	// +optional
	ClusterLocation string `json:"clusterLocation,omitempty"`
	// ClusterName is the name of the cluster
	// If not specified, it fetches information from the metadata server
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// ClusterProjectID is the project ID of the cluster
	// If not specified, it fetches information from the metadata server
	// +optional
	ClusterProjectID string `json:"clusterProjectID,omitempty"`
}

// GCPSMProvider Configures a store to sync secrets using the GCP Secret Manager provider.
type GCPSMProvider struct {
	// Auth defines the information necessary to authenticate against GCP
	// +optional
	Auth GCPSMAuth `json:"auth,omitempty"`

	// ProjectID project where secret is located
	ProjectID string `json:"projectID,omitempty"`

	// Location optionally defines a location for a secret
	Location string `json:"location,omitempty"`

	// SecretVersionSelectionPolicy specifies how the provider selects a secret version
	// when "latest" is disabled or destroyed.
	// Possible values are:
	// - LatestOrFail: the provider always uses "latest", or fails if that version is disabled/destroyed.
	// - LatestOrFetch: the provider falls back to fetching the latest version if the version is DESTROYED or DISABLED
	// +optional
	// +kubebuilder:default=LatestOrFail
	SecretVersionSelectionPolicy SecretVersionSelectionPolicy `json:"secretVersionSelectionPolicy,omitempty"`
}

// GCPWorkloadIdentityFederation holds the configurations required for generating federated access tokens.
type GCPWorkloadIdentityFederation struct {
	// credConfig holds the configmap reference containing the GCP external account credential configuration in JSON format and the key name containing the json data.
	// For using Kubernetes cluster as the identity provider, use serviceAccountRef instead. Operators mounted serviceaccount token cannot be used as the token source, instead
	// serviceAccountRef must be used by providing operators service account details.
	// +kubebuilder:validation:Optional
	CredConfig *ConfigMapReference `json:"credConfig,omitempty"`

	// serviceAccountRef is the reference to the kubernetes ServiceAccount to be used for obtaining the tokens,
	// when Kubernetes is configured as provider in workload identity pool.
	// +kubebuilder:validation:Optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`

	// awsSecurityCredentials is for configuring AWS region and credentials to use for obtaining the access token,
	// when using the AWS metadata server is not an option.
	// +kubebuilder:validation:Optional
	AwsSecurityCredentials *AwsCredentialsConfig `json:"awsSecurityCredentials,omitempty"`

	// audience is the Secure Token Service (STS) audience which contains the resource name for the workload identity pool and the provider identifier in that pool.
	// If specified, Audience found in the external account credential config will be overridden with the configured value.
	// audience must be provided when serviceAccountRef or awsSecurityCredentials is configured.
	// +kubebuilder:validation:Optional
	Audience string `json:"audience,omitempty"`

	// externalTokenEndpoint is the endpoint explicitly set up to provide tokens, which will be matched against the
	// credential_source.url in the provided credConfig. This field is merely to double-check the external token source
	// URL is having the expected value.
	// +kubebuilder:validation:Optional
	ExternalTokenEndpoint string `json:"externalTokenEndpoint,omitempty"`
}

// ConfigMapReference holds the details of a configmap.
type ConfigMapReference struct {
	// name of the configmap.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// namespace in which the configmap exists. If empty, configmap will looked up in local namespace.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`

	// key name holding the external account credential config.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[-._a-zA-Z0-9]+$
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// AwsCredentialsConfig holds the region and the Secret reference which contains the AWS credentials.
type AwsCredentialsConfig struct {
	// region is for configuring the AWS region to be used.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=50
	// +kubebuilder:validation:Pattern:=`^[a-z0-9-]+$`
	// +kubebuilder:example:="ap-south-1"
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// awsCredentialsSecretRef is the reference to the secret which holds the AWS credentials.
	// Secret should be created with below names for keys
	// - aws_access_key_id: Access Key ID, which is the unique identifier for the AWS account or the IAM user.
	// - aws_secret_access_key: Secret Access Key, which is used to authenticate requests made to AWS services.
	// - aws_session_token: Session Token, is the short-lived token to authenticate requests made to AWS services.
	// +kubebuilder:validation:Required
	AwsCredentialsSecretRef *SecretReference `json:"awsCredentialsSecretRef"`
}

// SecretReference holds the details of a secret.
type SecretReference struct {
	// name of the secret.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// namespace in which the secret exists. If empty, secret will looked up in local namespace.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}
