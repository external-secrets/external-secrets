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

// Package previder implements a secret store provider for Previder Vault.
package previder

import (
	"context"
	"errors"
	"fmt"
	previderclient "github.com/previder/vault-cli/pkg"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errNotImplemented = "not implemented"
)

var _ esv1.Provider = &SecretManager{}

// SecretManager implements the esv1.Provider interface for Previder Vault.
type SecretManager struct {
	VaultClient previderclient.PreviderVaultClient
	TokenType   string
}

// NewClient creates a new Previder Vault client.
func (s *SecretManager) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	if store == nil {
		return nil, fmt.Errorf("secret store not found: %v", "nil store")
	}
	storeSpec := store.GetSpec().Provider.Previder

	storeKind := store.GetObjectKind().GroupVersionKind().Kind
	accessToken, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &storeSpec.Auth.SecretRef.AccessToken)
	if err != nil {
		return nil, fmt.Errorf(accessToken, err)
	}

	s.VaultClient, err = previderclient.NewVaultClient(storeSpec.BaseURI, accessToken)

	if err != nil {
		return nil, err
	}

	tokenInfo, err := s.VaultClient.GetTokenInfo()
	if err != nil {
		return nil, err
	}
	s.TokenType = tokenInfo.TokenType

	return s, nil
}

// ValidateStore validates the Previder Vault store configuration.
func (s *SecretManager) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	previderSpec := storeSpec.Provider.Previder
	if previderSpec == nil {
		return nil, errors.New("missing Previder spec")
	}
	if previderSpec.Auth.SecretRef == nil {
		return nil, errors.New("missing Previder Auth SecretRef")
	}
	accessToken := previderSpec.Auth.SecretRef.AccessToken

	if accessToken.Name == "" {
		return nil, errors.New("missing Previder accessToken name")
	}
	if accessToken.Key == "" {
		return nil, errors.New("missing Previder accessToken key")
	}

	return nil, nil
}

// Capabilities returns the capabilities of the Previder Vault provider.
func (s *SecretManager) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// GetSecret retrieves a secret from Previder Vault.
func (s *SecretManager) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := s.VaultClient.DecryptSecret(ref.Key)
	if err != nil {
		return nil, err
	}
	return []byte(secret.Secret), nil
}

// PushSecret is not implemented for Previder Vault.
func (s *SecretManager) PushSecret(context.Context, *corev1.Secret, esv1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

// DeleteSecret is not implemented for Previder Vault.
func (s *SecretManager) DeleteSecret(context.Context, esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

// SecretExists is not implemented for Previder Vault.
func (s *SecretManager) SecretExists(context.Context, esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// Validate checks if the Vault client can connect and retrieve secrets.
func (s *SecretManager) Validate() (esv1.ValidationResult, error) {
	_, err := s.VaultClient.GetTokenInfo()
	if err != nil {
		return esv1.ValidationResultError, err
	}
	return esv1.ValidationResultReady, nil
}

// GetSecretMap retrieves a secret and returns it as a map with a single key-value pair.
func (s *SecretManager) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secrets, err := s.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	secretData := make(map[string][]byte)
	secretData[ref.Key] = secrets
	return secretData, nil
}

// GetAllSecrets is not implemented for Previder Vault.
func (s *SecretManager) GetAllSecrets(context.Context, esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// Close cleans up any resources held by the client.
func (s *SecretManager) Close(context.Context) error {
	return nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &SecretManager{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Previder: &esv1.PreviderProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
