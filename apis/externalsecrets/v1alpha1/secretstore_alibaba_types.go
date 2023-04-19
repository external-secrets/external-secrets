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
