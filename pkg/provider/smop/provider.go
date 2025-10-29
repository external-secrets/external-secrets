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

package smop

import (
	"context"
	"errors"
	"fmt"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/esutils"
	"github.com/external-secrets/external-secrets/pkg/esutils/resolvers"
	"github.com/external-secrets/external-secrets/pkg/provider/smop/smopclient"
)

var (
	ErrNoStore = errors.New("missing or invalid Smop SecretStore")
	ErrNoApiKey = errors.New("missing or invalid Smop API Token in Smop SecretStore")
	ErrNoTokenName = errors.New("missing or invalid Smop API Token name in Smop SecretStore")
	ErrNoTokenKey = errors.New("missing or invalid Smop API Token key in Smop SecretStore")
	ErrNoServer = errors.New("missing or invalid Smop Server in Smop SecretStore")
	ErrNoApiUrl = errors.New("missing or invalid Smop Server API URL in Smop SecretStore")
	ErrNoSiteId = errors.New("missing or invalid Smop Server site ID in Smop SecretStore")
)

// Provider is a Doppler secrets provider implementing NewClient and ValidateStore for the esv1.Provider interface.
type Provider struct{}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Client{}
var _ esv1.Provider = &Provider{}

func init() {
	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		Smop: &esv1.SmopProvider{},
	}, esv1.MaintenanceStatusMaintained)
}

// NewClient constructs a Smop SecretsManager Provider.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Smop == nil {
		return nil, ErrNoStore
	}
	
	smopStoreSpec := storeSpec.Provider.Smop

	storeKind := store.GetKind()
	apiKey, err := loadApiKeyFromSpec(ctx, smopStoreSpec, kube, namespace, storeKind)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	baseURL, siteID, err := loadUrlFromSpec(smopStoreSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to load server URL configuration: %w", err)
	}
	
	smopServerURL := fmt.Sprintf("%s/%s/secrets", baseURL, siteID)

	smopClient, err := smopclient.NewSMOPClient(smopServerURL, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMOP client: %w", err)
	}

	err = smopClient.SetBaseURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to set base URL for SMOP client: %w", err)
	}

	client := &Client{
		smopClient: smopClient,
		store:     smopStoreSpec,
	}

	return client, nil
}

// ValidateStore checks if the Smop store is valid.
// The provider may return a warning and an error.
// The intended use of the warning to indicate a deprecation of behavior
// or other type of message that is NOT a validation failure but should be noticed by the user.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	smopStoreSpec := storeSpec.Provider.Smop
	smopTokenSecretRef := smopStoreSpec.Auth.APIKey.SmopToken
	if err := esutils.ValidateSecretSelector(store, smopTokenSecretRef); err != nil {
		return nil, err
	}

	if smopTokenSecretRef.Name == "" {
		return nil, ErrNoServer
	}

	return nil, nil
}

// Capabilities returns the Smop provider Capabilities (Read, Write, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

func loadApiKeyFromSpec(ctx context.Context, spec *esv1.SmopProvider, kube kclient.Client, namespace, storeKind string) (string, error) {
	if spec.Auth == nil {
		return "", ErrNoApiKey
	}

	tokenRef := spec.Auth.APIKey.SmopToken
	if tokenRef.Name == "" {
		return "", ErrNoTokenName
	}
	if tokenRef.Key == "" {
		return "", ErrNoTokenKey
	}

	return resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &tokenRef)
}

func loadUrlFromSpec(spec *esv1.SmopProvider) (string, string, error) {
	if spec.Server == nil {
		return "", "", ErrNoServer
	}

	if spec.Server.APIURL == "" {
		return "", "", ErrNoApiUrl
	}

	if spec.Server.SiteId == "" {
		return "", "", ErrNoSiteId
	}

	return spec.Server.APIURL, spec.Server.SiteId, nil
}
