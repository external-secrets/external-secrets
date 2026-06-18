/*
Copyright © 2026 SSH Communications

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

package privx

import (
	"encoding/json"

	"github.com/SSHcom/privx-sdk-go/v2/api/filters"
	"github.com/SSHcom/privx-sdk-go/v2/api/response"
	"github.com/SSHcom/privx-sdk-go/v2/api/vault"
)

// VaultClient defines an interface to PrivX Vault.
type VaultClient interface {
	Status() (*response.ServiceStatus, error)

	GetSecretsMetadata(name string) (*vault.Secret, error)
	GetUsersSecretsMetadata(userID, name string) (*vault.Secret, error)

	GetUserSecrets(userID string, opts ...filters.Option) (*response.ResultSet[vault.Secret], error)
	CreateUserSecret(userID string, secret *vault.SecretRequest) (vault.SecretCreate, error)
	GetUserSecret(userID, secretName string) (*vault.Secret, error)
	UpdateUserSecret(userID, secretName string, secret *vault.SecretRequest) error
	DeleteUserSecret(userID, secretName string) error

	GetSchemas() (*json.RawMessage, error)

	GetSecrets(opts ...filters.Option) (*response.ResultSet[vault.Secret], error)
	CreateSecret(secret *vault.SecretRequest) (vault.SecretCreate, error)
	GetSecret(secretName string) (*vault.Secret, error)
	UpdateSecret(secretName string, secret *vault.SecretRequest) error
	DeleteSecret(secretName string) error
	SearchSecrets(search vault.SecretSearch, opts ...filters.Option) (*response.ResultSet[vault.Secret], error)
}

// sdkVaultClient is an adapter over the PrivX SDK vault client.
type sdkVaultClient struct {
	v *vault.Vault
}

// Status returns vault service status.
func (c *sdkVaultClient) Status() (*response.ServiceStatus, error) {
	return c.v.Status()
}

// GetSecretsMetadata returns metadata for a secret.
func (c *sdkVaultClient) GetSecretsMetadata(name string) (*vault.Secret, error) {
	return c.v.GetSecretsMetadata(name)
}

// GetUsersSecretsMetadata returns user secret metadata.
func (c *sdkVaultClient) GetUsersSecretsMetadata(userID, name string) (*vault.Secret, error) {
	return c.v.GetUsersSecretsMetadata(userID, name)
}

// GetUserSecrets returns user secrets.
func (c *sdkVaultClient) GetUserSecrets(userID string, opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
	return c.v.GetUserSecrets(userID, opts...)
}

// CreateUserSecret creates a user secret.
func (c *sdkVaultClient) CreateUserSecret(userID string, secret *vault.SecretRequest) (vault.SecretCreate, error) {
	return c.v.CreateUserSecret(userID, secret)
}

// GetUserSecret returns a user secret by name.
func (c *sdkVaultClient) GetUserSecret(userID, secretName string) (*vault.Secret, error) {
	return c.v.GetUserSecret(userID, secretName)
}

// UpdateUserSecret updates a user secret.
func (c *sdkVaultClient) UpdateUserSecret(userID, secretName string, secret *vault.SecretRequest) error {
	return c.v.UpdateUserSecret(userID, secretName, secret)
}

// DeleteUserSecret deletes a user secret.
func (c *sdkVaultClient) DeleteUserSecret(userID, secretName string) error {
	return c.v.DeleteUserSecret(userID, secretName)
}

// GetSchemas returns vault schemas.
func (c *sdkVaultClient) GetSchemas() (*json.RawMessage, error) {
	return c.v.GetSchemas()
}

// GetSecrets returns secrets.
func (c *sdkVaultClient) GetSecrets(opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
	return c.v.GetSecrets(opts...)
}

// CreateSecret creates a secret.
func (c *sdkVaultClient) CreateSecret(secret *vault.SecretRequest) (vault.SecretCreate, error) {
	return c.v.CreateSecret(secret)
}

// GetSecret returns a secret by name.
func (c *sdkVaultClient) GetSecret(secretName string) (*vault.Secret, error) {
	return c.v.GetSecret(secretName)
}

// UpdateSecret updates a secret.
func (c *sdkVaultClient) UpdateSecret(secretName string, secret *vault.SecretRequest) error {
	return c.v.UpdateSecret(secretName, secret)
}

// DeleteSecret deletes a secret.
func (c *sdkVaultClient) DeleteSecret(secretName string) error {
	return c.v.DeleteSecret(secretName)
}

// SearchSecrets searches secrets.
func (c *sdkVaultClient) SearchSecrets(search vault.SecretSearch, opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
	return c.v.SearchSecrets(search, opts...)
}
