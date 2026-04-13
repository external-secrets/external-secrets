/*
Copyright © The ESO Authors

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

// NPWSProvider configures a store to sync secrets using the Netwrix Password Secure provider.
type NPWSProvider struct {
	// Auth defines the authentication credentials for Netwrix Password Secure.
	Auth NPWSAuth `json:"auth"`
	// Host defines the base URL of the Netwrix Password Secure API.
	// Example: "https://npws.example.com"
	Host string `json:"host"`
	// DeletionPolicyWholeEntry controls the behavior of deletionPolicy: Delete.
	// If false (default) and a property is specified, only that field is removed from the entry.
	// If true or no property is specified, the entire entry is deleted.
	// +optional
	DeletionPolicyWholeEntry bool `json:"deletionPolicyWholeEntry,omitempty"`
}

// NPWSAuth contains the authentication configuration for Netwrix Password Secure.
type NPWSAuth struct {
	// SecretRef references a Kubernetes Secret containing the API key.
	SecretRef *NPWSAuthSecretRef `json:"secretRef"`
}

// NPWSAuthSecretRef holds the reference to the Kubernetes Secret containing the NPWS API key.
type NPWSAuthSecretRef struct {
	// APIKey is a reference to a Kubernetes Secret key that contains the Netwrix Password Secure API key.
	APIKey esmeta.SecretKeySelector `json:"apiKey"`
}
