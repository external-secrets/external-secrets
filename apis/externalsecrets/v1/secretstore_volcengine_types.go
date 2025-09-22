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

// VolcengineProvider defines the configuration for the Volcengine provider.
type VolcengineProvider struct {
	// Region specifies the Volcengine region to connect to.
	Region string `json:"region"`

	// Auth defines the authentication method to use.
	// If not specified, the provider will try to use IRSA (IAM Role for Service Account).
	// +optional
	Auth *VolcengineAuth `json:"auth,omitempty"`
}

// VolcengineAuth defines the authentication method for the Volcengine provider.
// Only one of the fields should be set.
type VolcengineAuth struct {
	// SecretRef defines the static credentials to use for authentication.
	// If not set, IRSA is used.
	// +optional
	SecretRef *VolcengineAuthSecretRef `json:"secretRef,omitempty"`
}

// VolcengineAuthSecretRef defines the secret reference for static credentials.
type VolcengineAuthSecretRef struct {
	// AccessKeyID is the reference to the secret containing the Access Key ID.
	AccessKeyID esmeta.SecretKeySelector `json:"accessKeyID"`

	// SecretAccessKey is the reference to the secret containing the Secret Access Key.
	SecretAccessKey esmeta.SecretKeySelector `json:"secretAccessKey"`

	// Token is the reference to the secret containing the STS(Security Token Service) Token.
	// +optional
	Token *esmeta.SecretKeySelector `json:"token,omitempty"`
}
