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

// Package senhasegura implements Senhasegura provider for External Secrets Operator
package senhasegura

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	senhaseguraAuth "github.com/external-secrets/external-secrets/providers/v1/senhasegura/auth"
	"github.com/external-secrets/external-secrets/providers/v1/senhasegura/dsm"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.Provider = &Provider{}

// Provider struct that satisfier ESO interface.
type Provider struct{}

const (
	errUnknownProviderService     = "unknown senhasegura Provider Service: %s"
	errNilStore                   = "nil store found"
	errMissingStoreSpec           = "store is missing spec"
	errMissingProvider            = "storeSpec is missing provider"
	errInvalidProvider            = "invalid provider spec. Missing senhasegura field in store %s"
	errInvalidSenhaseguraURL      = "invalid senhasegura URL"
	errInvalidSenhaseguraURLHTTPS = "invalid senhasegura URL, must be HTTPS for security reasons"
	errMissingClientID            = "missing senhasegura authentication Client ID"
)

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient construct a new secrets client based on provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	spec := store.GetSpec()
	provider := spec.Provider.Senhasegura

	isoSession, err := senhaseguraAuth.Authenticate(ctx, store, provider, kube, namespace)
	if err != nil {
		return nil, err
	}

	if provider.Module == esv1.SenhaseguraModuleDSM {
		return dsm.New(isoSession)
	}

	return nil, fmt.Errorf(errUnknownProviderService, provider.Module)
}

// ValidateStore validates store using Validating webhook during secret store creating
// Checks here are usually the best experience for the user, as the SecretStore will not be created until it is a 'valid' one.
// https://github.com/external-secrets/external-secrets/pull/830#discussion_r833278518
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	return nil, validateStore(store)
}

func validateStore(store esv1.GenericStore) error {
	if store == nil {
		return errors.New(errNilStore)
	}

	spec := store.GetSpec()
	if spec == nil {
		return errors.New(errMissingStoreSpec)
	}

	if spec.Provider == nil {
		return errors.New(errMissingProvider)
	}

	provider := spec.Provider.Senhasegura
	if provider == nil {
		return fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}

	url, err := url.Parse(provider.URL)
	if err != nil {
		return errors.New(errInvalidSenhaseguraURL)
	}

	// senhasegura doesn't accept requests without SSL/TLS layer for security reasons
	// DSM doesn't provides gRPC schema, only HTTPS
	if url.Scheme != "https" {
		return errors.New(errInvalidSenhaseguraURLHTTPS)
	}

	if url.Host == "" {
		return errors.New(errInvalidSenhaseguraURL)
	}

	if provider.Auth.ClientID == "" {
		return errors.New(errMissingClientID)
	}

	return nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Senhasegura: &esv1.SenhaseguraProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
