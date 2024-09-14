//Copyright External Secrets Inc. All Rights Reserved

package v1alpha1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// AlibabaAuth contains a secretRef for credentials.
type AlibabaAuth struct {
	// +optional
	SecretRef *AlibabaAuthSecretRef `json:"secretRef,omitempty"`
	// +optional
	RRSAAuth *AlibabaRRSAAuth `json:"rrsa,omitempty"`
}

// Authenticate against Alibaba using RRSA.
type AlibabaRRSAAuth struct {
	OIDCProviderARN   string `json:"oidcProviderArn"`
	OIDCTokenFilePath string `json:"oidcTokenFilePath"`
	RoleARN           string `json:"roleArn"`
	SessionName       string `json:"sessionName"`
}

// AlibabaAuthSecretRef holds secret references for Alibaba credentials.
type AlibabaAuthSecretRef struct {
	// The AccessKeyID is used for authentication
	AccessKeyID esmeta.SecretKeySelector `json:"accessKeyIDSecretRef"`
	// The AccessKeySecret is used for authentication
	AccessKeySecret esmeta.SecretKeySelector `json:"accessKeySecretSecretRef"`
}

// AlibabaProvider configures a store to sync secrets using the Alibaba Secret Manager provider.
type AlibabaProvider struct {
	Auth AlibabaAuth `json:"auth"`
	// Alibaba Region to be used for the provider
	RegionID string `json:"regionID"`
}
