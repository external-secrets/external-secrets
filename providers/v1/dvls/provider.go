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

package dvls

import (
	"context"
	"fmt"
	"net/url"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

var _ esv1.Provider = &Provider{}

// Provider implements the external-secrets Provider interface for DVLS.
type Provider struct{}

// NewClient creates a new DVLS SecretsClient.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	dvlsProvider, err := getDVLSProvider(store)
	if err != nil {
		return nil, err
	}

	storeKind := store.GetObjectKind().GroupVersionKind().Kind

	dvlsClient, err := NewDVLSClient(ctx, kube, storeKind, namespace, dvlsProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create DVLS client: %w", err)
	}

	return NewClient(dvlsClient), nil
}

// ValidateStore validates the SecretStore configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	dvlsProvider, err := getDVLSProvider(store)
	if err != nil {
		return nil, err
	}

	if dvlsProvider.ServerURL == "" {
		return nil, fmt.Errorf("serverUrl is required")
	}

	parsedURL, err := url.Parse(dvlsProvider.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("serverUrl must be a valid URL: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("serverUrl must be a valid URL with scheme and host")
	}

	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return nil, fmt.Errorf("serverUrl scheme must be http or https, got %q", parsedURL.Scheme)
	}

	if parsedURL.Scheme == "http" && !dvlsProvider.Insecure {
		return nil, fmt.Errorf("http URLs require 'insecure: true' to be set explicitly")
	}

	// Validate auth configuration
	if err := validateAuthSecretRef(store, &dvlsProvider.Auth.SecretRef); err != nil {
		return nil, err
	}

	return nil, nil
}

// Capabilities returns the provider's capabilities.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// validateAuthSecretRef validates the authentication secret references.
func validateAuthSecretRef(store esv1.GenericStore, ref *esv1.DVLSAuthSecretRef) error {
	if err := requireSecretSelector(ref.AppID, "appId"); err != nil {
		return err
	}
	if err := esutils.ValidateSecretSelector(store, ref.AppID); err != nil {
		return fmt.Errorf("invalid appId: %w", err)
	}

	if err := requireSecretSelector(ref.AppSecret, "appSecret"); err != nil {
		return err
	}
	if err := esutils.ValidateSecretSelector(store, ref.AppSecret); err != nil {
		return fmt.Errorf("invalid appSecret: %w", err)
	}
	return nil
}

func requireSecretSelector(sel esmeta.SecretKeySelector, field string) error {
	if sel.Name == "" {
		return fmt.Errorf("%s secret name is required", field)
	}

	if sel.Key == "" {
		return fmt.Errorf("%s secret key is required", field)
	}

	return nil
}

// getDVLSProvider extracts the DVLS provider configuration from the store.
func getDVLSProvider(store esv1.GenericStore) (*esv1.DVLSProvider, error) {
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.DVLS == nil {
		return nil, fmt.Errorf("DVLS provider configuration is missing")
	}
	return spec.Provider.DVLS, nil
}

// NewProvider creates a new DVLS Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		DVLS: &esv1.DVLSProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
