//go:build vaultwarden || all_providers

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

// Package vaultwarden implements a provider for syncing secrets from a self-hosted Vaultwarden instance.
package vaultwarden

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const errUnexpectedStoreSpec = "unexpected store spec"

// Provider implements esv1.Provider for Vaultwarden.
type Provider struct{}

var _ esv1.Provider = &Provider{}

// Capabilities returns the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(_ context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	prov, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if len(prov.CABundle) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(prov.CABundle) {
			return nil, fmt.Errorf("vaultwarden: failed to parse CABundle")
		}
		tlsConfig.RootCAs = pool
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	return &Client{
		httpClient: httpClient,
		provider:   prov,
		crClient:   kube,
		namespace:  namespace,
		store:      store,
	}, nil
}

// ValidateStore validates the configuration of a Vaultwarden secret store.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, errors.New("invalid store")
	}
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.Vaultwarden == nil {
		return nil, errors.New(errUnexpectedStoreSpec)
	}
	return nil, nil
}

// NewProvider returns a new Vaultwarden provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the SecretStoreProvider spec for Vaultwarden registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Vaultwarden: &esv1.VaultwardenProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
