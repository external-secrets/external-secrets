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

package ngrok

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/ngrok/ngrok-api-go/v7"
	"github.com/ngrok/ngrok-api-go/v7/secrets"
	"github.com/ngrok/ngrok-api-go/v7/vaults"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

var (
	defaultAPIURL = "https://api.ngrok.com"
	userAgent     = "external-secrets"

	errClusterStoreRequiresNamespace = errors.New("cluster store requires namespace")
	errInvalidStore                  = errors.New("invalid store")
	errInvalidStoreSpec              = errors.New("invalid store spec")
	errInvalidStoreProv              = errors.New("invalid store provider")
	errInvalidNgrokProv              = errors.New("invalid ngrok provider")
	errInvalidAuthAPIKeyRequired     = errors.New("ngrok provider auth APIKey is required")
	errInvalidAPIURL                 = errors.New("invalid API URL")
	errMissingVaultName              = errors.New("ngrok provider vault name is required")
)

type vaultClientFactory func(cfg *ngrok.ClientConfig) VaultClient
type secretsClientFactory func(cfg *ngrok.ClientConfig) SecretsClient

var getVaultsClient vaultClientFactory = func(cfg *ngrok.ClientConfig) VaultClient {
	return vaults.NewClient(cfg)
}

var getSecretsClient secretsClientFactory = func(cfg *ngrok.ClientConfig) SecretsClient {
	return secrets.NewClient(cfg)
}

// Provider implements the ngrok provider for External Secrets Operator.
type Provider struct{}

// Capabilities returns the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite). Currently,
// ngrok only supports WriteOnly capabilities.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreWriteOnly
}

// NewClient implements the Client interface.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kubeClient kubeClient.Client, namespace string) (esv1.SecretsClient, error) {
	cfg, err := getConfig(store)
	if err != nil {
		return nil, err
	}

	if store.GetKind() == esv1.ClusterSecretStoreKind && doesConfigDependOnNamespace(cfg) {
		return nil, errClusterStoreRequiresNamespace
	}

	apiKey, err := loadAPIKeySecret(ctx, cfg.Auth.APIKey, kubeClient, store.GetKind(), namespace)
	if err != nil {
		return nil, err
	}

	clientConfig := ngrok.NewClientConfig(
		apiKey,
		ngrok.WithBaseURL(cfg.APIURL),
		ngrok.WithUserAgent(userAgent),
	)

	vaultClient := getVaultsClient(clientConfig)
	secretsClient := getSecretsClient(clientConfig)

	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	var vault *ngrok.Vault
	vaultIter := vaultClient.List(nil)
	for vaultIter.Next(listCtx) {
		if vaultIter.Item().Name == cfg.Vault.Name {
			vault = vaultIter.Item()
			break
		}
	}

	if err := vaultIter.Err(); err != nil {
		return nil, fmt.Errorf("error listing vaults: %w", err)
	}

	if vault == nil {
		return nil, fmt.Errorf("vault %q not found", cfg.Vault.Name)
	}

	return &client{
		vaultClient:   vaultClient,
		secretsClient: secretsClient,
		vaultName:     cfg.Vault.Name,
		vaultID:       vault.ID,
	}, nil
}

// ValidateStore validates the store configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	_, err := getConfig(store)
	return nil, err
}

func loadAPIKeySecret(ctx context.Context, ref *esv1.NgrokProviderSecretRef, kube kubeClient.Client, storeKind, namespace string) (string, error) {
	return resolvers.SecretKeyRef(
		ctx,
		kube,
		storeKind,
		namespace,
		ref.SecretRef,
	)
}

func doesConfigDependOnNamespace(cfg *esv1.NgrokProvider) bool {
	ref := cfg.Auth.APIKey
	return ref != nil && ref.SecretRef != nil && ref.SecretRef.Namespace == nil
}

func getConfig(store esv1.GenericStore) (*esv1.NgrokProvider, error) {
	if store == nil {
		return nil, errInvalidStore
	}

	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return nil, errInvalidStoreSpec
	}

	if storeSpec.Provider == nil {
		return nil, errInvalidStoreProv
	}

	cfg := storeSpec.Provider.Ngrok
	if cfg == nil {
		return nil, errInvalidNgrokProv
	}

	if cfg.APIURL == "" {
		cfg.APIURL = defaultAPIURL
	} else if _, err := url.Parse(cfg.APIURL); err != nil {
		return nil, fmt.Errorf("%q: %w", cfg.APIURL, errInvalidAPIURL)
	}

	if cfg.Auth.APIKey == nil {
		return nil, errInvalidAuthAPIKeyRequired
	}

	if cfg.Vault.Name == "" {
		return nil, errMissingVaultName
	}

	return cfg, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Ngrok: &esv1.NgrokProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
