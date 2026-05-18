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

// Package secretmanager implements a provider for MWS Secret Manager.
package secretmanager

import (
	"context"
	"errors"
	"fmt"

	mwssdk "go.mws.cloud/go-sdk/mws"
	mwsiam "go.mws.cloud/go-sdk/mws/iam"
	secretmanagersdk "go.mws.cloud/go-sdk/service/secretmanager/sdk"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// ProviderSpec returns the provider specification.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		MWSSecretManager: &esv1.MWSSecretManagerProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}

var _ esv1.Provider = (*Provider)(nil)

// Provider implements the External Secrets provider interface for MWS Secret Manager.
type Provider struct{}

// NewProvider creates a new Provider instance.
func NewProvider() *Provider {
	return &Provider{}
}

// NewClient creates a new Client instance.
func (p *Provider) NewClient(
	ctx context.Context,
	store esv1.GenericStore,
	kube kclient.Client,
	namespace string,
) (esv1.SecretsClient, error) {
	if _, err := p.ValidateStore(store); err != nil {
		return nil, fmt.Errorf("invalid store: %w", err)
	}

	spec := store.GetSpec()

	authorizedKeyData, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		&spec.Provider.MWSSecretManager.Auth.AuthorizedKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve authorized key reference: %w", err)
	}

	authorizedKey := mwsiam.ServiceAccountAuthorizedKey{}
	if err := authorizedKey.UnmarshalJSON([]byte(authorizedKeyData)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal authorized key data: %w", err)
	}

	sdk, err := mwssdk.Load(ctx, mwssdk.WithServiceAccountAuthorizedKey(authorizedKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create mws sdk: %w", err)
	}

	secretVersion, err := secretmanagersdk.NewSecretVersion(ctx, sdk)
	if err != nil {
		return nil, fmt.Errorf("failed to create mws secret manager sdk: %w", err)
	}

	return &Client{
		sdk:           sdk,
		secretVersion: secretVersion,
		project:       authorizedKey.ServiceAccount.Project,
	}, nil
}

// ValidateStore validates the store.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, errors.New("store is not provided")
	}

	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.MWSSecretManager == nil {
		return nil, errors.New("MWS Secret Manager spec is not provided")
	}

	providerSpec := spec.Provider.MWSSecretManager

	if providerSpec.Auth.AuthorizedKey.Name == "" {
		return nil, errors.New("invalid spec: auth.authorizedKeySecretRef is required")
	}

	err := esutils.ValidateReferentSecretSelector(store, providerSpec.Auth.AuthorizedKey)
	if err != nil {
		return nil, fmt.Errorf("invalid spec: auth.authorizedKeySecretRef: %w", err)
	}

	return nil, nil
}

// Capabilities returns the provider supported capabilities.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}
