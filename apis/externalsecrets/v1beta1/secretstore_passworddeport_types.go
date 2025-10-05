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

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// PasswordDepotProvider configures a store to sync secrets with a Password Depot instance.
type PasswordDepotProvider struct {
	// URL configures the Password Depot instance URL.
	Host string `json:"host"`

	// Database to use as source
	Database string `json:"database"`

	// Auth configures how secret-manager authenticates with a Password Depot instance.
	Auth PasswordDepotAuth `json:"auth"`
}

// PasswordDepotAuth defines the authentication method for the Password Depot provider.
type PasswordDepotAuth struct {
	SecretRef PasswordDepotSecretRef `json:"secretRef"`
}

// PasswordDepotSecretRef defines a reference to a secret containing credentials for the Password Depot provider.
type PasswordDepotSecretRef struct {
	// Username / Password is used for authentication.
	// +optional
	Credentials esmeta.SecretKeySelector `json:"credentials,omitempty"`
}
