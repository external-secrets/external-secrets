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

package fake

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/ngrok/ngrok-api-go/v9"
)

// errFnNotConfigured is returned by a fake method when it is called without a
// corresponding function being set. Failing loudly is safer than returning a
// nil result, which the real ngrok API never does and which callers may deref.
var errFnNotConfigured = errors.New("fake: method called but no function configured")

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano())) /* #nosec G404 */

	sb := strings.Builder{}
	sb.Grow(length)
	for range length {
		sb.WriteByte(charset[seededRand.Intn(len(charset))])
	}
	return sb.String()
}

// VaultClient is a mock implementation which implements the ngrok.VaultsClient interface.
type VaultClient struct {
	CreateFn            func(context.Context, *ngrok.VaultCreate) (*ngrok.Vault, error)
	GetFn               func(context.Context, string) (*ngrok.Vault, error)
	GetSecretsByVaultFn func(string, *ngrok.Paging) ngrok.Iter[*ngrok.Secret]
	ListFn              func(*ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault]
}

func (m *VaultClient) Create(ctx context.Context, vault *ngrok.VaultCreate) (*ngrok.Vault, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, vault)
	}
	return nil, fmt.Errorf("VaultClient.Create: %w", errFnNotConfigured)
}

func (m *VaultClient) Get(ctx context.Context, vaultID string) (*ngrok.Vault, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, vaultID)
	}
	return nil, fmt.Errorf("VaultClient.Get: %w", errFnNotConfigured)
}

func (m *VaultClient) GetSecretsByVault(id string, paging *ngrok.Paging) ngrok.Iter[*ngrok.Secret] {
	if m.GetSecretsByVaultFn != nil {
		return m.GetSecretsByVaultFn(id, paging)
	}
	return NewIter[*ngrok.Secret](nil, fmt.Errorf("VaultClient.GetSecretsByVault: %w", errFnNotConfigured))
}

func (m *VaultClient) List(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
	if m.ListFn != nil {
		return m.ListFn(paging)
	}
	return NewIter[*ngrok.Vault](nil, fmt.Errorf("VaultClient.List: %w", errFnNotConfigured))
}

// SecretsClient is a mock implementation of the SecretsClient interface.
type SecretsClient struct {
	CreateFn func(context.Context, *ngrok.SecretCreate) (*ngrok.Secret, error)
	DeleteFn func(context.Context, string) error
	GetFn    func(context.Context, string) (*ngrok.Secret, error)
	ListFn   func(*ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret]
	UpdateFn func(context.Context, *ngrok.SecretUpdate) (*ngrok.Secret, error)
}

func (m *SecretsClient) Create(ctx context.Context, secret *ngrok.SecretCreate) (*ngrok.Secret, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, secret)
	}
	return nil, fmt.Errorf("SecretsClient.Create: %w", errFnNotConfigured)
}

func (m *SecretsClient) Delete(ctx context.Context, secretID string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, secretID)
	}
	return fmt.Errorf("SecretsClient.Delete: %w", errFnNotConfigured)
}

func (m *SecretsClient) Get(ctx context.Context, secretID string) (*ngrok.Secret, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, secretID)
	}
	return nil, fmt.Errorf("SecretsClient.Get: %w", errFnNotConfigured)
}

func (m *SecretsClient) List(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
	if m.ListFn != nil {
		return m.ListFn(paging)
	}
	return NewIter[*ngrok.Secret](nil, fmt.Errorf("SecretsClient.List: %w", errFnNotConfigured))
}

func (m *SecretsClient) Update(ctx context.Context, secret *ngrok.SecretUpdate) (*ngrok.Secret, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, secret)
	}
	return nil, fmt.Errorf("SecretsClient.Update: %w", errFnNotConfigured)
}

// Iter is a mock iterator that implements the ngrok.Iter[T] interface.
type Iter[T any] struct {
	items []T
	err   error
	n     int
}

func (m *Iter[T]) Next(_ context.Context) bool {
	if m.err != nil {
		return false
	}
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
