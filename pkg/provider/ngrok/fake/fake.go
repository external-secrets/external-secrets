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

package fake

import (
	"context"
	"maps"
	"math/rand"
	"slices"
	"strings"
	"time"

	"github.com/ngrok/ngrok-api-go/v7"
)

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano())) /* #nosec G404 */

	sb := strings.Builder{}
	sb.Grow(length)
	for i := 0; i < length; i++ {
		sb.WriteByte(charset[seededRand.Intn(len(charset))])
	}
	return sb.String()
}

type VaultClient struct {
	vaults    map[string]*ngrok.Vault
	createErr error
	listErr   error
}

func NewVaultClient(vaults ...*ngrok.Vault) *VaultClient {
	m := make(map[string]*ngrok.Vault)
	for _, v := range vaults {
		m[v.ID] = v
	}
	return &VaultClient{
		vaults: m,
	}
}

// WithCreateError sets an error to be returned when Create is called.
// This is useful for testing error handling in the client.
func (m *VaultClient) WithCreateError(err error) *VaultClient {
	m.createErr = err
	return m
}

// Create creates a new vault and returns it. If an error is set, it will return that error instead of the vault.
func (m *VaultClient) Create(_ context.Context, vault *ngrok.VaultCreate) (*ngrok.Vault, error) {
	ts := time.Now()
	newVault := &ngrok.Vault{
		ID:          "vault_" + GenerateRandomString(20),
		Name:        vault.Name,
		Description: vault.Description,
		Metadata:    vault.Metadata,
		CreatedAt:   ts.Format(time.RFC3339),
		UpdatedAt:   ts.Format(time.RFC3339),
	}
	m.vaults[newVault.ID] = newVault
	return newVault, m.createErr
}

// WithListError sets an error to be returned when List is called.
func (m *VaultClient) WithListError(err error) *VaultClient {
	m.listErr = err
	return m
}

// List returns an iterator over the vaults.
// If an error is set, it will return that error instead of the vaults.
func (m *VaultClient) List(paging *ngrok.Paging) ngrok.Iter[*ngrok.Vault] {
	items := slices.Collect(maps.Values(m.vaults))
	return NewIter(items, m.listErr)
}

// SecretsClient is a mock implementation of the SecretsClient interface.
// It allows you to create, update, delete, and list secrets.
// It can be used to test the client without needing a real ngrok API.
type SecretsClient struct {
	secrets   map[string]*ngrok.Secret
	createErr error
	updateErr error
	deleteErr error
	listErr   error
}

// NewSecretsClient creates a new SecretsClient with the given secrets.
// It initializes the secrets map with the provided secrets.
func NewSecretsClient(secrets ...*ngrok.Secret) *SecretsClient {
	m := make(map[string]*ngrok.Secret)
	for _, s := range secrets {
		m[s.ID] = s
	}
	return &SecretsClient{
		secrets: m,
	}
}

// WithCreateError sets an error to be returned when Create is called.
// This is useful for testing error handling in the client.
func (m *SecretsClient) WithCreateError(err error) *SecretsClient {
	m.createErr = err
	return m
}

// Create creates a new secret and returns it. If an error is set, it will return that error instead of the secret.
func (m *SecretsClient) Create(_ context.Context, secret *ngrok.SecretCreate) (*ngrok.Secret, error) {
	ts := time.Now()
	newSecret := &ngrok.Secret{
		ID: "secret_" + GenerateRandomString(20),
		Vault: ngrok.Ref{
			ID:  secret.VaultID,
			URI: "vaults/" + secret.VaultID,
		},
		Name:        secret.Name,
		Description: secret.Description,
		Metadata:    secret.Metadata,
		CreatedAt:   ts.Format(time.RFC3339),
		UpdatedAt:   ts.Format(time.RFC3339),
	}
	m.secrets[newSecret.ID] = newSecret
	return newSecret, m.createErr
}

// WithUpdateError sets an error to be returned when Update is called.
// This is useful for testing error handling in the client.
func (m *SecretsClient) WithUpdateError(err error) *SecretsClient {
	m.updateErr = err
	return m
}

// Update updates an existing secret and returns it. If an error is set, it will return that error instead of the secret.
func (m *SecretsClient) Update(_ context.Context, secret *ngrok.SecretUpdate) (*ngrok.Secret, error) {
	ts := time.Now()
	for i, s := range m.secrets {
		if s.ID != secret.ID {
			continue
		}

		s.UpdatedAt = ts.Format(time.RFC3339)
		if secret.Description != nil {
			s.Description = *secret.Description
		}
		if secret.Metadata != nil {
			s.Metadata = *secret.Metadata
		}

		return m.secrets[i], m.updateErr
	}
	return nil, &ngrok.Error{StatusCode: 404, Msg: "Secret not found"}
}

// WithDeleteError sets an error to be returned when Delete is called.
// This is useful for testing error handling in the client.
func (m *SecretsClient) WithDeleteError(err error) *SecretsClient {
	m.deleteErr = err
	return m
}

// Delete deletes a secret by its ID. If an error is set, it will return that error instead of deleting the secret.
// If the secret does not exist, it returns an error.
func (m *SecretsClient) Delete(_ context.Context, secretID string) error {
	_, ok := m.secrets[secretID]
	if !ok {
		return &ngrok.Error{StatusCode: 404, Msg: "Secret not found"}
	}
	delete(m.secrets, secretID)
	return m.deleteErr
}

// WithListError sets an error to be returned when List is called.
// This is useful for testing error handling in the client.
func (m *SecretsClient) WithListError(err error) *SecretsClient {
	m.listErr = err
	return m
}

// List returns an iterator over the secrets.
// If an error is set, it will return that error instead of the secrets.
func (m *SecretsClient) List(paging *ngrok.Paging) ngrok.Iter[*ngrok.Secret] {
	items := slices.Collect(maps.Values(m.secrets))
	return NewIter(items, m.listErr)
}

// Iter is a mock iterator that implements the ngrok.Iter[T] interface.
type Iter[T any] struct {
	items []T
	err   error
	n     int
}

func (m *Iter[T]) Next(_ context.Context) bool {
	// If there is an error, stop iteration
	if m.err != nil {
		return false
	}

	// Increment the index
	m.n++

	return m.n < len(m.items) && m.n >= 0
}

func (m *Iter[T]) Item() T {
	if m.n >= 0 && m.n < len(m.items) {
		return m.items[m.n]
	}
	return *new(T)
}

func (m *Iter[T]) Err() error {
	return m.err
}

func NewIter[T any](items []T, err error) *Iter[T] {
	return &Iter[T]{
		items: items,
		err:   err,
		n:     -1,
	}
}
