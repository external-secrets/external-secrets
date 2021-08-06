package v1alpha1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// AlibabaAuth contains a secretRef for credentials.
type AlibabaAuth struct {
	SecretRef AlibabaAuthSecretRef `json:"secretRef"`
}

// AlibabaAuthSecretRef holds secret references for Alibaba credentials
type AlibabaAuthSecretRef struct {
	// The AccessKeyID is used for authentication
	AccessKeyID esmeta.SecretKeySelector `json:"accessKeyIDSecretRef"`
	// The AccessKeySecret is used for authentication
	AccessKeySecret esmeta.SecretKeySelector `json:"accessKeySecretSecretRef"`
}

//AlibabaProvider configures a store to sync secrets using the Alibaba Secret Manager provider.
type AlibabaProvider struct {
	Auth *AlibabaAuth `json:"auth"`
	// +optional
	Endpoint string `json:"endpoint"`
	// Alibaba Region to be used for the provider
	RegionID string `json:"regionID"`
}
