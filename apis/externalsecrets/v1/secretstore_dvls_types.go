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

// DVLSProvider configures a store to sync secrets using Devolutions Server.
type DVLSProvider struct {
	// ServerURL is the DVLS instance URL (e.g., https://dvls.example.com).
	// +kubebuilder:validation:Required
	ServerURL string `json:"serverUrl"`

	// Insecure allows connecting to DVLS over plain HTTP.
	// This is NOT RECOMMENDED for production use.
	// Set to true only if you understand the security implications.
	// +optional
	Insecure bool `json:"insecure,omitempty"`

	// Auth defines the authentication method to use.
	// +kubebuilder:validation:Required
	Auth DVLSAuth `json:"auth"`
}

// DVLSAuth defines the authentication method for the DVLS provider.
type DVLSAuth struct {
	// SecretRef contains the Application ID and Application Secret for authentication.
	// +kubebuilder:validation:Required
	SecretRef DVLSAuthSecretRef `json:"secretRef"`
}

// DVLSAuthSecretRef defines the secret references for DVLS authentication credentials.
type DVLSAuthSecretRef struct {
	// AppID is the reference to the secret containing the Application ID.
	// +kubebuilder:validation:Required
	AppID esmeta.SecretKeySelector `json:"appId"`

	// AppSecret is the reference to the secret containing the Application Secret.
	// +kubebuilder:validation:Required
	AppSecret esmeta.SecretKeySelector `json:"appSecret"`
}
