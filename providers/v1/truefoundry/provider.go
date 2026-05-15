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

// Package truefoundry implements an External Secrets provider that reads
// secrets from the TrueFoundry secret-management API.
package truefoundry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// Provider implements the External Secrets esv1.Provider interface for TrueFoundry.
type Provider struct{}

var _ esv1.Provider = &Provider{}

// NewProvider returns a new TrueFoundry Provider.
func NewProvider() esv1.Provider { return &Provider{} }

// ProviderSpec returns the SecretStoreProvider shell used to register this provider.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{TrueFoundry: &esv1.TrueFoundryProvider{}}
}

// MaintenanceStatus reports that the TrueFoundry provider is actively maintained.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}

// Capabilities reports that this provider supports read-only operations only.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// ValidateStore checks that the provided SecretStore configuration is well-formed.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, errors.New("invalid truefoundry store: nil store")
	}
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.TrueFoundry == nil {
		return nil, errors.New("invalid truefoundry store: missing provider config")
	}
	cfg := spec.Provider.TrueFoundry

	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("truefoundry.baseURL is required")
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("truefoundry.baseURL must be a valid http(s) URL: %q", cfg.BaseURL)
	}

	if strings.TrimSpace(cfg.Tenant) == "" {
		return nil, errors.New("truefoundry.tenant is required")
	}

	apiKey := cfg.Auth.SecretRef.APIKey
	if apiKey.Name == "" {
		return nil, errors.New("truefoundry.auth.secretRef.apiKey.name is required")
	}
	if apiKey.Key == "" {
		return nil, errors.New("truefoundry.auth.secretRef.apiKey.key is required")
	}
	if err := esutils.ValidateReferentSecretSelector(store, apiKey); err != nil {
		return nil, fmt.Errorf("truefoundry.auth.secretRef.apiKey: %w", err)
	}
	return nil, nil
}

// NewClient builds a SecretsClient for the given SecretStore. The TrueFoundry
// API key is resolved from the referenced Kubernetes Secret.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	if store == nil || store.GetSpec() == nil || store.GetSpec().Provider == nil || store.GetSpec().Provider.TrueFoundry == nil {
		return nil, errors.New("invalid truefoundry store: missing provider config")
	}
	cfg := store.GetSpec().Provider.TrueFoundry
	storeKind := store.GetObjectKind().GroupVersionKind().Kind

	apiKey, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &cfg.Auth.SecretRef.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve truefoundry api key: %w", err)
	}
	return newClient(cfg.BaseURL, cfg.Tenant, apiKey, http.DefaultClient), nil
}
