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

// Package sapcredentialstore implements an ESO provider for SAP Credential Store on BTP.
package sapcredentialstore

import (
	"context"
	"fmt"

	"golang.org/x/oauth2/clientcredentials"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/providers/v1/sapcredentialstore/api"
)

var _ esv1.Provider = &Provider{}

// Provider implements esv1.Provider for SAP Credential Store.
type Provider struct{}

// Capabilities returns ReadWrite because the provider supports both ExternalSecret and PushSecret.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// ValidateStore checks that the SecretStore configuration is complete and self-consistent.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.SAPCredentialStore == nil {
		return nil, fmt.Errorf("sapCredentialStore: missing provider spec")
	}

	s := spec.Provider.SAPCredentialStore

	if s.ServiceURL == "" {
		return nil, fmt.Errorf("sapCredentialStore: serviceURL is required")
	}

	if s.Namespace == "" {
		return nil, fmt.Errorf("sapCredentialStore: namespace is required")
	}

	if s.Auth.OAuth2 == nil && s.Auth.MTLS == nil {
		return nil, fmt.Errorf("sapCredentialStore: exactly one of auth.oauth2 or auth.mtls must be set")
	}

	if s.Auth.OAuth2 != nil && s.Auth.MTLS != nil {
		return nil, fmt.Errorf("sapCredentialStore: only one of auth.oauth2 or auth.mtls may be set")
	}

	if s.Auth.OAuth2 != nil {
		if err := validateOAuth2(s.Auth.OAuth2); err != nil {
			return nil, err
		}
	}

	if s.Auth.MTLS != nil {
		if err := validateMTLS(s.Auth.MTLS); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func validateOAuth2(o *esv1.SAPCSOAuth2Auth) error {
	if o.TokenURL == "" {
		return fmt.Errorf("sapCredentialStore: auth.oauth2.tokenURL is required")
	}

	if o.ClientID.Name == "" {
		return fmt.Errorf("sapCredentialStore: auth.oauth2.clientId.name is required")
	}

	if o.ClientSecret.Name == "" {
		return fmt.Errorf("sapCredentialStore: auth.oauth2.clientSecret.name is required")
	}

	return nil
}

func validateMTLS(m *esv1.SAPCSMTLSAuth) error {
	if m.Certificate.Name == "" {
		return fmt.Errorf("sapCredentialStore: auth.mtls.certificate.name is required")
	}

	if m.PrivateKey.Name == "" {
		return fmt.Errorf("sapCredentialStore: auth.mtls.privateKey.name is required")
	}

	return nil
}

// NewClient constructs a SecretsClient from the store spec and resolves any referenced Kubernetes Secrets.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.SAPCredentialStore == nil {
		return nil, fmt.Errorf("sapCredentialStore: missing provider spec")
	}

	s := spec.Provider.SAPCredentialStore
	storeKind := store.GetObjectKind().GroupVersionKind().Kind

	var sapClient api.SAPCSClientInterface

	switch {
	case s.Auth.OAuth2 != nil:
		clientID, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &s.Auth.OAuth2.ClientID)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: resolving clientId: %w", err)
		}

		clientSecret, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &s.Auth.OAuth2.ClientSecret)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: resolving clientSecret: %w", err)
		}

		cfg := clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     s.Auth.OAuth2.TokenURL,
		}
		transport := cfg.Client(ctx).Transport
		sapClient = api.NewOAuth2Client(s.ServiceURL, transport)

	case s.Auth.MTLS != nil:
		certPEM, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &s.Auth.MTLS.Certificate)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: resolving certificate: %w", err)
		}

		keyPEM, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &s.Auth.MTLS.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: resolving privateKey: %w", err)
		}

		var buildErr error
		sapClient, buildErr = api.NewMTLSClient(s.ServiceURL, []byte(certPEM), []byte(keyPEM))
		if buildErr != nil {
			return nil, fmt.Errorf("sapCredentialStore: building mTLS client: %w", buildErr)
		}

	default:
		return nil, fmt.Errorf("sapCredentialStore: no auth mode configured")
	}

	return &Client{
		sapClient: sapClient,
		namespace: s.Namespace,
	}, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns a sentinel SecretStoreProvider for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		SAPCredentialStore: &esv1.SAPCredentialStoreProvider{},
	}
}

// MaintenanceStatus returns the maintenance status for the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
