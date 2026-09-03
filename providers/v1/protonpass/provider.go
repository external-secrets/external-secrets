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

// Package protonpass provides a read-only provider that syncs secrets from Proton Pass.
package protonpass

import (
	"context"
	"errors"
	"fmt"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/protonpass/internal/client"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errGeneric      = "protonpass provider error: %w"
	errMissingRef   = "protonpass provider missing required field: %w"
	errClientInit   = "protonpass provider client initialization failed: %w"
	errStoreInvalid = "protonpass provider invalid store: %w"
)

var _ esv1.Provider = &Provider{}

// Provider implements the Proton Pass provider.
type Provider struct{}

// Capabilities returns the capabilities of the Proton Pass provider.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// ValidateStore validates the Proton Pass store configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, fmt.Errorf(errGeneric, errors.New("store is nil"))
	}
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	if err := esutils.ValidateSecretSelector(store, provider.Auth.PersonalAccessTokenSecretRef); err != nil {
		return nil, fmt.Errorf(errStoreInvalid, err)
	}
	if provider.Auth.PersonalAccessTokenSecretRef.Name == "" {
		return nil, fmt.Errorf(errMissingRef, errors.New("auth.personalAccessTokenSecretRef.name cannot be empty"))
	}
	return nil, nil
}

// NewClient creates a new Proton Pass client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}

	pat, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, &provider.Auth.PersonalAccessTokenSecretRef)
	if err != nil {
		return nil, fmt.Errorf(errMissingRef, err)
	}
	if pat == "" {
		return nil, fmt.Errorf(errMissingRef, errors.New("personal access token secret value is empty"))
	}

	ppClient, err := client.NewClient(pat)
	if err != nil {
		return nil, fmt.Errorf(errClientInit, err)
	}

	return &Client{client: ppClient}, nil
}

// getProvider retrieves the Proton Pass provider configuration from the store.
func getProvider(store esv1.GenericStore) (*esv1.ProtonPassProvider, error) {
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.ProtonPass == nil {
		return nil, fmt.Errorf(errMissingRef, errors.New("provider protonpass is nil"))
	}
	return spec.Provider.ProtonPass, nil
}

// NewProvider constructs a new Proton Pass provider.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns a sample Proton Pass provider spec.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		ProtonPass: &esv1.ProtonPassProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the Proton Pass provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
