/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliec.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ngrok

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
	"github.com/ngrok/ngrok-api-go/v7"
	"github.com/ngrok/ngrok-api-go/v7/secrets"
	"github.com/ngrok/ngrok-api-go/v7/vaults"
)

var (
	defaultAPIURL = "https://api.ngrok.com"
	userAgent     = "external-secrets"

	errInvalidStore              = errors.New("invalid store")
	errInvalidStoreSpec          = errors.New("invalid store spec")
	errInvalidStoreProv          = errors.New("invalid store provider")
	errInvalidNgrokProv          = errors.New("invalid ngrok provider")
	errInvalidAuthAPIKeyRequired = errors.New("ngrok provider auth APIKey is required")
	errInvalidAPIURL             = errors.New("invalid API URL")
)

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
		return nil, errors.New("referent authentication isn't implemented in this provider")
	}

	if cfg.Auth.APIKey == nil {
		return nil, errInvalidAuthAPIKeyRequired
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

	vaultClient := vaults.NewClient(clientConfig)
	secretsClient := secrets.NewClient(clientConfig)

	return &client{
		vaultClient:   vaultClient,
		secretsClient: secretsClient,
		vaultName:     cfg.Vault.Name,
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

	return cfg, nil
}

func init() {
	esv1.Register(
		&Provider{},
		&esv1.SecretStoreProvider{
			Ngrok: &esv1.NgrokProvider{},
		},
		esv1.MaintenanceStatusMaintained,
	)
}
