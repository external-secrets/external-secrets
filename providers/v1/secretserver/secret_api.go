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

package secretserver

import (
	"github.com/DelineaXPM/tss-sdk-go/v3/server"
)

// secretAPI represents the subset of the Secret Server API
// which is supported by tss-sdk-go/v3.
type secretAPI interface {
	// Secret retrieves a secret by its ID.
	Secret(id int) (*server.Secret, error)
	// Secrets searches for secrets by text and field name.
	Secrets(searchText, field string) ([]server.Secret, error)
	// SecretByPath retrieves a secret using its folder path.
	SecretByPath(secretPath string) (*server.Secret, error)
	// CreateSecret creates a new secret in Secret Server.
	CreateSecret(secret server.Secret) (*server.Secret, error)
	// UpdateSecret updates an existing secret in Secret Server.
	UpdateSecret(secret server.Secret) (*server.Secret, error)
	// DeleteSecret deletes a secret by its ID.
	DeleteSecret(id int) error
	// SecretTemplate retrieves a secret template by its ID.
	SecretTemplate(id int) (*server.SecretTemplate, error)
}
