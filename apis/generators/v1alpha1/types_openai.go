// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenAISpec controls the behavior of the openAI generator.
type OpenAISpec struct {
	// ProjectID is the id of the project the account will related to.
	ProjectID string `json:"projectId"`
	// Host is the server where the openAI api is hosted.
	// Default: "https://api.openai.com/v1"
	// +kubebuilder:default="https://api.openai.com/v1"
	Host string `json:"host"`
	// OpenAiAdminKey contains the Admin API Key used to authenticate against the OpenAI server.
	OpenAiAdminKey esmeta.SecretKeySelector `json:"openAiAdminKey"`
	// ServiceAccountNamePrefix define a prefix to add before the generated name for the service account
	// +optional
	ServiceAccountNamePrefix *string `json:"serviceAccountNamePrefix,omitempty"`
	// ServiceAccountNameSize define the size of the generated name for the service account
	// Default: 12
	// +optional
	// +kubebuilder:default=12
	ServiceAccountNameSize *int `json:"serviceAccountNameSize,omitempty"`
	// CleanupPolicy controls the behavior of the cleanup process
	// +optional
	CleanupPolicy CleanupPolicy `json:"cleanupPolicy,omitempty"`
}

// OpenAiServiceAccount represents an OpenAI service account.
type OpenAiServiceAccount struct {
	// Object defines the type of this OpenAI resource.
	// Example: "organization.project.service_account"
	Object string `json:"object"`
	// ID is the unique identifier of the service account.
	ID string `json:"id"`
	// Name is the display name of the service account.
	Name string `json:"name"`
	// Role defines the role assigned to this service account (e.g., "member").
	Role string `json:"role"`
	// CreatedAt is the Unix timestamp representing creation time.
	CreatedAt int64 `json:"created_at"`
	// APIKey contains the API key associated with this service account.
	APIKey OpenAiAPIKey `json:"api_key"`
}

// OpenAiAPIKey represents an OpenAI API key.
type OpenAiAPIKey struct {
	// Object defines the type of this OpenAI API key resource.
	// Example: "organization.project.service_account.api_key"
	Object string `json:"object"`
	// Value is the actual secret API key (e.g., "sk-...").
	Value string `json:"value"`
	// Name is the display name of the API key.
	Name string `json:"name"`
	// CreatedAt is the Unix timestamp representing creation time.
	CreatedAt int64 `json:"created_at"`
	// LastUsedAt is the Unix timestamp representing the last time the API key was used.
	LastUsedAt int64 `json:"last_used_at"`
	// ID is the unique identifier of the API key.
	ID string `json:"id"`
}

// OpenAiServiceAccountState represents the state of an OpenAI service account.
type OpenAiServiceAccountState struct {
	ServiceAccountID string `json:"serviceAccountId,omitempty"`
	APIKeyID         string `json:"apiKeyId,omitempty"`
}

// OpenAI generates an OpenAI service account based on the configuration parameters in spec.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type OpenAI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenAISpec      `json:"spec,omitempty"`
	Status GeneratorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OpenAIList contains a list of OpenAI resources.
type OpenAIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenAI `json:"items"`
}
