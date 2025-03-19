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

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// CSMAuth contains a secretRef for credentials.
type CSMAuth struct {
	// +optional
	SecretRef *CSMAuthSecretRef `json:"secretRef,omitempty"`
}

// CSMAuthSecretRef holds secret references for Cloud.ru credentials.
type CSMAuthSecretRef struct {
	// The AccessKeyID is used for authentication
	AccessKeyID esmeta.SecretKeySelector `json:"accessKeyIDSecretRef"`
	// The AccessKeySecret is used for authentication
	AccessKeySecret esmeta.SecretKeySelector `json:"accessKeySecretSecretRef"`
}

// CloudruSMProvider configures a store to sync secrets using the Cloud.ru Secret Manager provider.
type CloudruSMProvider struct {
	Auth CSMAuth `json:"auth"`

	// ProjectID is the project, which the secrets are stored in.
	ProjectID string `json:"projectID,omitempty"`
}
