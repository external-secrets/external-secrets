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

package fake

import (
	"context"
	"fmt"
	"maps"
	"math/rand"
	"net/http"
	"slices"
	"strings"
	"sync"
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

func VaultNameEmpty() *ngrok.Error {
	return &ngrok.Error{
		ErrorCode:  "ERR_NGROK_23001",
		StatusCode: http.StatusBadRequest,
		Msg:        "The vault name cannot be empty.",
	}
}

func VaultNamesMustBeUniqueWithinAccount() *ngrok.Error {
	return &ngrok.Error{
		ErrorCode:  "ERR_NGROK_23004",
		StatusCode: http.StatusBadRequest,
		Msg:        "Vault names must be unique within an account.",
	}
}

func VaultNameInvalid(name string) *ngrok.Error {
	return &ngrok.Error{
		ErrorCode:  "ERR_NGROK_23002",
		StatusCode: http.StatusBadRequest,
		Msg:        fmt.Sprintf("The vault name %q is invalid. Must only contain the characters \"a-zA-Z0-9_/.\".", name),
	}
}

func SecretNameEmpty() *ngrok.Error {
	return &ngrok.Error{
		ErrorCode:  "ERR_NGROK_24001",
		StatusCode: http.StatusBadRequest,
		Msg:        "The secret name cannot be empty.",
	}
}

func SecretValueEmpty() *ngrok.Error {
	return &ngrok.Error{
		ErrorCode:  "ERR_NGROK_24003",
		StatusCode: http.StatusBadRequest,
		Msg:        "The secret value cannot be empty.",
	}
}

func SecretNameMustBeUniqueWithinVault() *ngrok.Error {
	return &ngrok.Error{
		ErrorCode:  "ERR_NGROK_24005",
		StatusCode: http.StatusBadRequest,
		Msg:        "Secret names must be unique within a vault.",
	}
}

func SecretVaultNotFound(id string) *ngrok.Error {
	return &ngrok.Error{
		ErrorCode:  "ERR_NGROK_24006",
		StatusCode: http.StatusNotFound,
		Msg:        fmt.Sprintf("Vault with ID %s not found.", id),
	}
}

func NotFound(id string) *ngrok.Error {
	return &ngrok.Error{
		StatusCode: http.StatusNotFound,
		Msg:        fmt.Sprintf("Resource with ID %s not found.", id),
	}
}

func VaultNotEmpty() *ngrok.Error {
	return &ngrok.Error{
		ErrorCode:  "ERR_NGROK_23003",
		StatusCode: http.StatusBadRequest,
		Msg:        "A Vault must be empty before it can be deleted. Please remove all secrets from the vault and try again.",
	}
}

type vault struct {
	vault *ngrok.Vault

	mu          sync.RWMutex
	secretsByID map[string]*ngrok.Secret
}

// newVault creates a new vault instance with an empty secrets map.
// given the ngrok.Vault to wrap.
func newVault(v *ngrok.Vault) *vault {
	return &vault{
		vault:       v,
		secretsByID: make(map[string]*ngrok.Secret),
	}
}

func (v *vault) setSecret(id string, secret *ngrok.Secret) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.secretsByID[id] = secret
}

func (v *vault) getSecret(id string) (*ngrok.Secret, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	val, ok := v.secretsByID[id]
	return val, ok
}

func (v *vault) deleteSecret(id string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.secretsByID, id)
}

// CreateSecret creates a new secret in the vault.
func (v *vault) CreateSecret(s *ngrok.SecretCreate) (*ngrok.Secret, error) {
	if s.Name == "" {
		return nil, SecretNameEmpty()
	}

	if s.Value == "" {
		return nil, SecretValueEmpty()
	}

	existing := v.GetSecretByName(s.Name)
	if existing != nil {
		return nil, SecretNameMustBeUniqueWithinVault()
	}

	ts := time.Now()
	newSecret := &ngrok.Secret{
		ID: "secret_" + GenerateRandomString(20),
		Vault: ngrok.Ref{
			ID:  v.vault.ID,
			URI: v.vault.URI,
		},
		Name:        s.Name,
		Description: s.Description,
		Metadata:    s.Metadata,
		CreatedAt:   ts.Format(time.RFC3339),
		UpdatedAt:   ts.Format(time.RFC3339),
	}

	v.setSecret(newSecret.ID, newSecret)
	return newSecret, nil
}

// DeleteSecret deletes a secret from the vault by ID.
func (v *vault) DeleteSecret(id string) error {
	_, exists := v.getSecret(id)

	if exists {
		v.deleteSecret(id)
		return nil
	}

	return NotFound(id)
}

// ListSecrets returns all secrets in the vault.
func (v *vault) ListSecrets() []*ngrok.Secret {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return slices.Collect(maps.Values(v.secretsByID))
}

// GetSecretByID returns the secret with the given ID, or nil if not found.
func (v *vault) GetSecretByID(id string) *ngrok.Secret {
	val, _ := v.getSecret(id)
	return val
}

// GetSecretByName returns the secret with the given name, or nil if not found.
func (v *vault) GetSecretByName(name string) *ngrok.Secret {
	for _, secret := range v.ListSecrets() {
		if secret.Name == name {
			return secret
		}
	}
	return nil
}

// UpdateSecret updates an existing secret in the vault.
func (v *vault) UpdateSecret(s *ngrok.SecretUpdate) (*ngrok.Secret, error) {
	secret := v.GetSecretByID(s.ID)
	if secret == nil {
		return nil, NotFound(s.ID)
	}

	if s.Name != nil {
		if *s.Name == "" {
			return nil, SecretNameEmpty()
		}

		existing := v.GetSecretByName(*s.Name)
		if existing != nil && existing.ID != s.ID {
			return nil, SecretNameMustBeUniqueWithinVault()
		}
	}

	if s.Value != nil {
		if *s.Value == "" {
			return nil, SecretValueEmpty()
		}
	}

	ts := time.Now()
	secret.UpdatedAt = ts.Format(time.RFC3339)
	if s.Name != nil {
		secret.Name = *s.Name
	}
	if s.Description != nil {
		secret.Description = *s.Description
	}
	if s.Metadata != nil {
		secret.Metadata = *s.Metadata
	}

	return secret, nil
}

type Store struct {
	mu         sync.RWMutex
	vaultsByID map[string]*vault
}

func NewStore() *Store {
	return &Store{
		vaultsByID: make(map[string]*vault),
	}
}

func (s *Store) setVault(id string, v *vault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vaultsByID[id] = v
}

func (s *Store) getVault(id string) (*vault, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.vaultsByID[id]
	return val, ok
}

func (s *Store) deleteVault(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.vaultsByID, id)
}

// CreateVault creates a new vault in the store.
func (s *Store) CreateVault(v *ngrok.VaultCreate) (*ngrok.Vault, error) {
	if v.Name == "" {
		return nil, VaultNameEmpty()
	}

	for _, vault := range s.ListVaults() {
		if vault.Name == v.Name {
			return nil, VaultNamesMustBeUniqueWithinAccount()
		}
	}

	ts := time.Now()
	ngrokVault := &ngrok.Vault{
		ID:          "vault_" + GenerateRandomString(20),
		Name:        v.Name,
		Description: v.Description,
		Metadata:    v.Metadata,
		CreatedAt:   ts.Format(time.RFC3339),
		UpdatedAt:   ts.Format(time.RFC3339),
	}

	s.setVault(ngrokVault.ID, newVault(ngrokVault))
	return ngrokVault, nil
}

func (s *Store) CreateSecret(secret *ngrok.SecretCreate) (*ngrok.Secret, error) {
	v, _ := s.getVault(secret.VaultID)

	if v == nil {
		return nil, SecretVaultNotFound(secret.VaultID)
	}
	return v.CreateSecret(secret)
}

// DeleteVault deletes a vault from the store by ID.
func (s *Store) DeleteVault(id string) error {
	v, _ := s.getVault(id)

	if v == nil {
		return NotFound(id)
	}

	if len(v.ListSecrets()) > 0 {
		return VaultNotEmpty()
	}

	s.deleteVault(id)
	return nil
}

// GetVaultByID returns the vault with the given ID, or nil if not found.
func (s *Store) GetVaultByID(id string) (*ngrok.Vault, error) {
	v, _ := s.getVault(id)
	if v == nil {
		return nil, NotFound(id)
	}
	return v.vault, nil
}

func (s *Store) GetVaultByName(name string) *ngrok.Vault {
	for _, v := range s.ListVaults() {
		if v.Name == name {
			return v
		}
	}
	return nil
}

// ListSecrets returns all secrets in the store.
func (s *Store) ListSecrets() []*ngrok.Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()

	secrets := []*ngrok.Secret{}
	for _, v := range s.vaultsByID {
		secrets = append(secrets, v.ListSecrets()...)
	}
	return secrets
}

// ListVaults returns all vaults in the store.
func (s *Store) ListVaults() []*ngrok.Vault {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vaults := make([]*ngrok.Vault, 0, len(s.vaultsByID))
	for _, v := range s.vaultsByID {
		vaults = append(vaults, v.vault)
	}
	return vaults
}

func (s *Store) ListVaultSecrets(vaultID string) ([]*ngrok.Secret, error) {
	v, _ := s.getVault(vaultID)

	if v == nil {
		return nil, NotFound(vaultID)
	}

	return v.ListSecrets(), nil
}

func (s *Store) UpdateSecret(secret *ngrok.SecretUpdate) (*ngrok.Secret, error) {
	var found *ngrok.Secret

	for _, sec := range s.ListSecrets() {
		if sec.ID == secret.ID {
			found = sec
			break
		}
	}

	if found == nil {
		return nil, NotFound(secret.ID)
	}

	v, ok := s.getVault(found.Vault.ID)
	if !ok {
		return nil, SecretVaultNotFound(found.Vault.ID)
	}

	return v.UpdateSecret(secret)
}

func (s *Store) DeleteSecret(secretID string) error {
	secret, vault, err := s.GetSecretAndVaultByID(secretID)
	if err != nil {
		return err
	}
	if secret == nil || vault == nil {
		return NotFound(secretID)
	}
	return vault.DeleteSecret(secretID)
}

func (s *Store) GetSecretAndVaultByID(secretID string) (*ngrok.Secret, *vault, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.vaultsByID {
		if sec := v.GetSecretByID(secretID); sec != nil {
			return sec, v, nil
		}
	}
	return nil, nil, NotFound(secretID)
}

func (s *Store) VaultClient() *VaultClient {
	return &VaultClient{
		store: s,
	}
}

func (s *Store) SecretsClient() *SecretsClient {
	return &SecretsClient{
		store: s,
	}
}

// VaultClient is a mock implementation which implements the ngrok.VaultsClient interface.
type VaultClient struct {
	store     *Store
	createErr error
	listErr   error
}

// WithCreateError sets an error to be returned when Create is called.
// This is useful for testing error handling in the client.
func (m *VaultClient) WithCreateError(err error) *VaultClient {
	m.createErr = err
	return m
}

// Create creates a new vault and returns it. If an error is set, it will return that error instead of the vault.
func (m *VaultClient) Create(_ context.Context, vault *ngrok.VaultCreate) (*ngrok.Vault, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.store.CreateVault(vault)
}

// Get retrieves a vault by its ID. If the vault does not exist, it returns an error.
func (m *VaultClient) Get(_ context.Context, vaultID string) (*ngrok.Vault, error) {
	return m.store.GetVaultByID(vaultID)
}

func (m *VaultClient) GetSecretsByVault(id string, paging *ngrok.Paging) ngrok.Iter[*ngrok.Secret] {
	secrets, err := m.store.ListVaultSecrets(id)
	return NewIter(secrets, err)
}

// WithListError sets an error to be returned when List is called.
func (m *VaultClient) WithListError(err error) *VaultClient {
	m.listErr = err
	return m
}

// List returns an iterator over the vaults.
// If an error is set, it will return that error instead of the vaults.
func (m *VaultClient) List(paging *ngrok.Paging) ngrok.Iter[*ngrok.Vault] {
	return NewIter(m.store.ListVaults(), m.listErr)
}

// SecretsClient is a mock implementation of the SecretsClient interface.
// It allows you to create, update, delete, and list secrets.
// It can be used to test the client without needing a real ngrok API.
type SecretsClient struct {
	store     *Store
	createErr error
	updateErr error
	deleteErr error
	listErr   error
}

// WithCreateError sets an error to be returned when Create is called.
// This is useful for testing error handling in the client.
func (m *SecretsClient) WithCreateError(err error) *SecretsClient {
	m.createErr = err
	return m
}

// Create creates a new secret and returns it. If an error is set, it will return that error instead of the secret.
func (m *SecretsClient) Create(_ context.Context, secret *ngrok.SecretCreate) (*ngrok.Secret, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	return m.store.CreateSecret(secret)
}

// WithUpdateError sets an error to be returned when Update is called.
// This is useful for testing error handling in the client.
func (m *SecretsClient) WithUpdateError(err error) *SecretsClient {
	m.updateErr = err
	return m
}

// Update updates an existing secret and returns it. If an error is set, it will return that error instead of the secret.
func (m *SecretsClient) Update(_ context.Context, secret *ngrok.SecretUpdate) (*ngrok.Secret, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}

	return m.store.UpdateSecret(secret)
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
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return m.store.DeleteSecret(secretID)
}

// Get retrieves a secret by its ID. If the secret does not exist, it returns an error.
func (m *SecretsClient) Get(_ context.Context, secretID string) (*ngrok.Secret, error) {
	s, _, err := m.store.GetSecretAndVaultByID(secretID) // to check existence
	return s, err
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
	return NewIter(m.store.ListSecrets(), m.listErr)
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
