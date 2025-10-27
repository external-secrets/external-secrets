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

// SmopAuthSecretRef defines a reference to a secret containing credentials for the Smop provider.
type SmopAuthSecretRef struct {
	// The SmopToken is used for authentication.
	SmopToken esmeta.SecretKeySelector `json:"smopToken"`
}

// SmopAuth defines the authentication method for the Smop provider.
type SmopAuth struct {
	APIKey SmopAuthSecretRef `json:"apikey"`
}

// SmopServer defines configuration for connecting to Smop server.
type SmopServer struct {
	// +required
	APIURL string `json:"apiUrl"`
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
	// +optional
	SiteId string `json:"siteId,omitempty"`
}

// SmopProvider configures a store to sync secrets using the Smop provider.
// Project and Config are required if not using a Service Token.
type SmopProvider struct {
	// Auth configures how the Operator authenticates with the Smop API
	Auth *SmopAuth `json:"auth"`

	// Server configures the Smop server connection details
	// +optional
	Server *SmopServer `json:"server,omitempty"`

	// Smop folder path to retrieve secret from
	// +optional
	FolderPath string `json:"folderPath,omitempty"`
}
