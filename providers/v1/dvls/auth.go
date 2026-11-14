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

// Package dvls implements the external-secrets provider for Devolutions Server (DVLS).
package dvls

import (
	"context"
	"fmt"

	"github.com/Devolutions/go-dvls"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

type realVaultClient struct {
	vaults *dvls.Vaults
}

func (r *realVaultClient) GetByName(ctx context.Context, name string) (dvls.Vault, error) {
	return r.vaults.GetByNameWithContext(ctx, name)
}

func (r *realVaultClient) Get(ctx context.Context, id string) (dvls.Vault, error) {
	return r.vaults.GetWithContext(ctx, id)
}

func (r *realVaultClient) List(ctx context.Context) ([]dvls.Vault, error) {
	return r.vaults.ListWithContext(ctx)
}

// storeClients holds the service clients and resolved vault for a DVLS store.
type storeClients struct {
	cred        credentialClient
	vaults      vaultGetter
	vaultID     string
	pinnedVault bool
}

// newStoreClients creates a new authenticated DVLS client and resolves the vault.
func newStoreClients(ctx context.Context, kube kclient.Client, storeKind, namespace string, provider *esv1.DVLSProvider) (*storeClients, error) {
	if provider == nil {
		return nil, fmt.Errorf("missing provider configuration")
	}

	appID, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &provider.Auth.SecretRef.AppID)
	if err != nil {
		return nil, fmt.Errorf("failed to get appId: %w", err)
	}

	appSecret, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &provider.Auth.SecretRef.AppSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get appSecret: %w", err)
	}

	client, err := dvls.NewClient(appID, appSecret, provider.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create DVLS client: %w", err)
	}

	clients := &storeClients{
		cred:   &realCredentialClient{cred: client.Entries.Credential},
		vaults: &realVaultClient{vaults: client.Vaults},
	}

	// When vault is empty, each secret key names its own vault (<vault>/<entry>).
	if provider.Vault == "" {
		return clients, nil
	}

	vaultID, err := resolveVaultRef(ctx, provider.Vault, clients.vaults)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve vault: %w", err)
	}
	clients.vaultID = vaultID
	clients.pinnedVault = true

	return clients, nil
}
