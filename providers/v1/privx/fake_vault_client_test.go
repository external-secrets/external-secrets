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

// fakeVaultClient is a test double for VaultClient.
type fakeVaultClient struct {
	statusFn                  func() (*response.ServiceStatus, error)
	getSecretsMetadataFn      func(name string) (*vault.Secret, error)
	getUsersSecretsMetadataFn func(userID, name string) (*vault.Secret, error)
	getUserSecretsFn          func(userID string, opts ...filters.Option) (*response.ResultSet[vault.Secret], error)
	createUserSecretFn        func(userID string, secret *vault.SecretRequest) (vault.SecretCreate, error)
	getUserSecretFn           func(userID, secretName string) (*vault.Secret, error)
	updateUserSecretFn        func(userID, secretName string, secret *vault.SecretRequest) error
	deleteUserSecretFn        func(userID, secretName string) error
	getSchemasFn              func() (*json.RawMessage, error)
	getSecretsFn              func(opts ...filters.Option) (*response.ResultSet[vault.Secret], error)
	createSecretFn            func(secret *vault.SecretRequest) (vault.SecretCreate, error)
	getSecretFn               func(secretName string) (*vault.Secret, error)
	updateSecretFn            func(secretName string, secret *vault.SecretRequest) error
	deleteSecretFn            func(secretName string) error
	searchSecretsFn           func(search vault.SecretSearch, opts ...filters.Option) (*response.ResultSet[vault.Secret], error)
}

// Status returns the configured fake result for Status.
func (f *fakeVaultClient) Status() (*response.ServiceStatus, error) {
	if f.statusFn != nil {
		return f.statusFn()
	}
	return nil, nil
}

// GetSecretsMetadata returns the configured fake result for GetSecretsMetadata.
func (f *fakeVaultClient) GetSecretsMetadata(name string) (*vault.Secret, error) {
	if f.getSecretsMetadataFn != nil {
		return f.getSecretsMetadataFn(name)
	}
	return nil, nil
}

// GetUsersSecretsMetadata returns the configured fake result for GetUsersSecretsMetadata.
func (f *fakeVaultClient) GetUsersSecretsMetadata(userID, name string) (*vault.Secret, error) {
	if f.getUsersSecretsMetadataFn != nil {
		return f.getUsersSecretsMetadataFn(userID, name)
	}
	return nil, nil
}

// GetUserSecrets returns the configured fake result for GetUserSecrets.
func (f *fakeVaultClient) GetUserSecrets(userID string, opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
	if f.getUserSecretsFn != nil {
		return f.getUserSecretsFn(userID, opts...)
	}
	return nil, nil
}

// CreateUserSecret returns the configured fake result for CreateUserSecret.
func (f *fakeVaultClient) CreateUserSecret(userID string, secret *vault.SecretRequest) (vault.SecretCreate, error) {
	if f.createUserSecretFn != nil {
		return f.createUserSecretFn(userID, secret)
	}
	return vault.SecretCreate{}, nil
}

// GetUserSecret returns the configured fake result for GetUserSecret.
func (f *fakeVaultClient) GetUserSecret(userID, secretName string) (*vault.Secret, error) {
	if f.getUserSecretFn != nil {
		return f.getUserSecretFn(userID, secretName)
	}
	return nil, nil
}

// UpdateUserSecret returns the configured fake result for UpdateUserSecret.
func (f *fakeVaultClient) UpdateUserSecret(userID, secretName string, secret *vault.SecretRequest) error {
	if f.updateUserSecretFn != nil {
		return f.updateUserSecretFn(userID, secretName, secret)
	}
	return nil
}

// DeleteUserSecret returns the configured fake result for DeleteUserSecret.
func (f *fakeVaultClient) DeleteUserSecret(userID, secretName string) error {
	if f.deleteUserSecretFn != nil {
		return f.deleteUserSecretFn(userID, secretName)
	}
	return nil
}

// GetSchemas returns the configured fake result for GetSchemas.
func (f *fakeVaultClient) GetSchemas() (*json.RawMessage, error) {
	if f.getSchemasFn != nil {
		return f.getSchemasFn()
	}
	return nil, nil
}

// GetSecrets returns the configured fake result for GetSecrets.
func (f *fakeVaultClient) GetSecrets(opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
	if f.getSecretsFn != nil {
		return f.getSecretsFn(opts...)
	}
	return nil, nil
}

// CreateSecret returns the configured fake result for CreateSecret.
func (f *fakeVaultClient) CreateSecret(secret *vault.SecretRequest) (vault.SecretCreate, error) {
	if f.createSecretFn != nil {
		return f.createSecretFn(secret)
	}
	return vault.SecretCreate{}, nil
}

// GetSecret returns the configured fake result for GetSecret.
func (f *fakeVaultClient) GetSecret(secretName string) (*vault.Secret, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(secretName)
	}
	return nil, nil
}

// UpdateSecret returns the configured fake result for UpdateSecret.
func (f *fakeVaultClient) UpdateSecret(secretName string, secret *vault.SecretRequest) error {
	if f.updateSecretFn != nil {
		return f.updateSecretFn(secretName, secret)
	}
	return nil
}

// DeleteSecret returns the configured fake result for DeleteSecret.
func (f *fakeVaultClient) DeleteSecret(secretName string) error {
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(secretName)
	}
	return nil
}

// SearchSecrets returns the configured fake result for SearchSecrets.
func (f *fakeVaultClient) SearchSecrets(search vault.SecretSearch, opts ...filters.Option) (*response.ResultSet[vault.Secret], error) {
	if f.searchSecretsFn != nil {
		return f.searchSecretsFn(search, opts...)
	}
	return nil, nil
}
