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

// Package onepasswordsdk implements a provider for 1Password using the official SDK.
// It allows fetching and managing secrets stored in 1Password using their official Go SDK.
package onepasswordsdk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errOnePasswordSdkStore                              = "received invalid 1PasswordSdk SecretStore resource: %w"
	errOnePasswordSdkStoreNilSpec                       = "nil spec"
	errOnePasswordSdkStoreNilSpecProvider               = "nil spec.provider"
	errOnePasswordSdkStoreNilSpecProviderOnePasswordSdk = "nil spec.provider.onepasswordsdk"
	errOnePasswordSdkStoreMissingRefName                = "missing: spec.provider.onepasswordsdk.auth.secretRef.serviceAccountTokenSecretRef.name"
	errOnePasswordSdkStoreMissingRefKey                 = "missing: spec.provider.onepasswordsdk.auth.secretRef.serviceAccountTokenSecretRef.key"
	errOnePasswordSdkStoreMissingVaultKey               = "missing: spec.provider.onepasswordsdk.vault"
	errVersionNotImplemented                            = "'remoteRef.version' is not implemented in the 1Password SDK provider"
	errNotImplemented                                   = "not implemented"
)

// Provider implements the External Secrets provider interface for 1Password SDK.
type Provider struct {
	client      *onepassword.Client
	vaultPrefix string
	vaultID     string
	cache       *expirable.LRU[string, []byte] // nil if caching is disabled
}

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	config := store.GetSpec().Provider.OnePasswordSDK
	serviceAccountToken, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		&config.Auth.ServiceAccountSecretRef,
	)
	if err != nil {
		return nil, err
	}

	if config.IntegrationInfo == nil {
		config.IntegrationInfo = &esv1.IntegrationInfo{
			Name:    "1Password SDK",
			Version: "v1.0.0",
		}
	}

	c, err := onepassword.NewClient(
		ctx,
		onepassword.WithServiceAccountToken(serviceAccountToken),
		onepassword.WithIntegrationInfo(config.IntegrationInfo.Name, config.IntegrationInfo.Version),
	)
	if err != nil {
		return nil, err
	}

	provider := &Provider{
		client:      c,
		vaultPrefix: "op://" + config.Vault + "/",
	}

	vaultID, err := provider.GetVault(ctx, config.Vault)
	if err != nil {
		return nil, fmt.Errorf("failed to get store ID: %w", err)
	}
	provider.vaultID = vaultID

	if config.Cache != nil {
		ttl := 5 * time.Minute
		if config.Cache.TTL.Duration > 0 {
			ttl = config.Cache.TTL.Duration
		}

		maxSize := 100
		if config.Cache.MaxSize > 0 {
			maxSize = config.Cache.MaxSize
		}

		provider.cache = expirable.NewLRU[string, []byte](maxSize, nil, ttl)
	}

	return provider, nil
}

// ValidateStore validates the 1Password SDK SecretStore resource configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreNilSpec))
	}
	if storeSpec.Provider == nil {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreNilSpecProvider))
	}
	if storeSpec.Provider.OnePasswordSDK == nil {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreNilSpecProviderOnePasswordSdk))
	}

	config := storeSpec.Provider.OnePasswordSDK
	if config.Auth.ServiceAccountSecretRef.Name == "" {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreMissingRefName))
	}
	if config.Auth.ServiceAccountSecretRef.Key == "" {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreMissingRefKey))
	}

	if config.Vault == "" {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreMissingVaultKey))
	}

	// check namespace compared to kind
	if err := esutils.ValidateSecretSelector(store, config.Auth.ServiceAccountSecretRef); err != nil {
		return nil, fmt.Errorf(errOnePasswordSdkStore, err)
	}

	return nil, nil
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		OnePasswordSDK: &esv1.OnePasswordSDKProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
