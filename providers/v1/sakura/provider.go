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

package sakura

import (
	"context"
	"fmt"

	sakuraclient "github.com/sacloud/api-client-go"
	"github.com/sacloud/iaas-api-go"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

type Provider struct{}

// Check if the Provider satisfies the esv1.Provider interface.
// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.Provider = &Provider{}

// Capabilities returns the provider's esv1.SecretStoreCapabilities.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient creates a new Sakura Cloud Secret Manager client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	provider, err := getSakuraProvider(store)
	if err != nil {
		return nil, err
	}

	// Set the zone to is1a
	// 	The zone doesn't matter for Sakura Cloud Secret Manager as it is a global resource
	baseURL := fmt.Sprintf("https://secure.sakura.ad.jp/cloud/zone/%s/api/cloud/1.1/secretmanager/vaults/%s/secrets", "is1a", provider.VaultResourceID)

	opt, err := sakuraclient.DefaultOption()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve DefaultOption: %w", err)
	}

	accessToken, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, &provider.Auth.SecretRef.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve auth.secretRef.accessToken: %w", err)
	}
	accessTokenSecret, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, &provider.Auth.SecretRef.AccessTokenSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve auth.secretRef.accessTokenSecret: %w", err)
	}
	opt = sakuraclient.MergeOptions(opt, &sakuraclient.Options{
		AccessToken:       accessToken,
		AccessTokenSecret: accessTokenSecret,
	})

	c := iaas.NewClientWithOptions(opt)

	return &Client{
		baseURL: baseURL,
		client:  c,
	}, nil
}

// ValidateStore validates the store's configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	prov, err := getSakuraProvider(store)
	if err != nil {
		return nil, err
	}

	if err := esutils.ValidateReferentSecretSelector(store, prov.Auth.SecretRef.AccessToken); err != nil {
		return nil, fmt.Errorf("invalid Auth.SecretRef.AccessToken: %w", err)
	}
	if err := esutils.ValidateReferentSecretSelector(store, prov.Auth.SecretRef.AccessTokenSecret); err != nil {
		return nil, fmt.Errorf("invalid Auth.SecretRef.AccessTokenSecret: %w", err)
	}

	return nil, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Sakura: &esv1.SakuraProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
