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

package barbican

import (
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errGeneric         = "barbican provider error: %w"
	errMissingField    = "barbican provider missing required field: %w"
	errAuthFailed      = "barbican provider authentication failed: %w"
	errClientInit      = "barbican provider client initialization failed: %w"
	errUnsupportedAuth = "barbican provider unsupported auth type: %s"
)

var _ esv1.Provider = &Provider{}

var (
	authenticatedClient = openstack.AuthenticatedClient
	newKeyManagerV1     = openstack.NewKeyManagerV1
)

// Provider implements the Barbican provider.
type Provider struct{}

// Capabilities returns the capabilities of the Barbican provider.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// ValidateStore validates the Barbican store configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, fmt.Errorf(errGeneric, errors.New("store is nil"))
	}

	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}

	if provider.AuthURL == "" {
		return nil, fmt.Errorf(errMissingField, errors.New("authURL is required"))
	}

	authType := esv1.BarbicanAuthTypePassword
	if provider.Auth.AuthType != nil {
		authType = *provider.Auth.AuthType
	}

	switch authType {
	case esv1.BarbicanAuthTypePassword:
		return nil, validatePasswordAuth(provider.Auth)
	case esv1.BarbicanAuthTypeApplicationCredential:
		return nil, validateAppCredAuth(provider.Auth)
	default:
		return nil, fmt.Errorf(errUnsupportedAuth, authType)
	}
}

func validatePasswordAuth(auth esv1.BarbicanAuth) error {
	if auth.Username.Value == "" && auth.Username.SecretRef == nil {
		return fmt.Errorf(errMissingField, errors.New("username must specify either value or secretRef"))
	}
	if auth.Password.SecretRef == nil {
		return fmt.Errorf(errMissingField, errors.New("password secretRef is required"))
	}
	return nil
}

func validateAppCredAuth(auth esv1.BarbicanAuth) error {
	if auth.ApplicationCredentialID == nil {
		return fmt.Errorf(errMissingField, errors.New("applicationCredentialID is required for applicationCredential auth"))
	}
	if auth.ApplicationCredentialID.Value == "" && auth.ApplicationCredentialID.SecretRef == nil {
		return fmt.Errorf(errMissingField, errors.New("applicationCredentialID must specify either value or secretRef"))
	}
	if auth.ApplicationCredentialSecret == nil || auth.ApplicationCredentialSecret.SecretRef == nil {
		return fmt.Errorf(errMissingField, errors.New("applicationCredentialSecret secretRef is required for applicationCredential auth"))
	}
	return nil
}

// NewClient creates a new Barbican client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

// getProvider retrieves the Barbican provider configuration from the store.
func getProvider(store esv1.GenericStore) (*esv1.BarbicanProvider, error) {
	spec := store.GetSpec()
	if spec.Provider == nil || spec.Provider.Barbican == nil {
		return nil, fmt.Errorf(errMissingField, errors.New("provider barbican is nil"))
	}
	return spec.Provider.Barbican, nil
}

func newClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	if provider.AuthURL == "" {
		return nil, fmt.Errorf(errMissingField, errors.New("authURL is required"))
	}

	authType := esv1.BarbicanAuthTypePassword
	if provider.Auth.AuthType != nil {
		authType = *provider.Auth.AuthType
	}

	var authopts gophercloud.AuthOptions
	switch authType {
	case esv1.BarbicanAuthTypePassword:
		authopts, err = buildPasswordAuthOpts(ctx, store, kube, namespace, provider)
	case esv1.BarbicanAuthTypeApplicationCredential:
		authopts, err = buildAppCredAuthOpts(ctx, store, kube, namespace, provider)
	default:
		return nil, fmt.Errorf(errUnsupportedAuth, authType)
	}
	if err != nil {
		return nil, err
	}

	auth, err := authenticatedClient(ctx, authopts)
	if err != nil {
		return nil, fmt.Errorf(errAuthFailed, err)
	}

	barbicanClient, err := newKeyManagerV1(auth, gophercloud.EndpointOpts{
		Region: provider.Region,
	})
	if err != nil {
		return nil, fmt.Errorf(errClientInit, err)
	}

	return &Client{keyManager: barbicanClient}, nil
}

func buildPasswordAuthOpts(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string, provider *esv1.BarbicanProvider) (gophercloud.AuthOptions, error) {
	username := provider.Auth.Username.Value
	var err error

	if username == "" {
		if provider.Auth.Username.SecretRef == nil {
			return gophercloud.AuthOptions{}, fmt.Errorf(errMissingField, errors.New("username.secretRef is required when value is empty"))
		}
		username, err = resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, provider.Auth.Username.SecretRef)
		if err != nil {
			return gophercloud.AuthOptions{}, fmt.Errorf(errMissingField, err)
		}
	}
    if provider.Auth.Password.SecretRef == nil {
		return gophercloud.AuthOptions{}, fmt.Errorf(errMissingField, errors.New("password.secretRef is required"))
	}
	password, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, provider.Auth.Password.SecretRef)
	if err != nil {
		return gophercloud.AuthOptions{}, fmt.Errorf(errMissingField, err)
	}

	return gophercloud.AuthOptions{
		IdentityEndpoint: provider.AuthURL,
		TenantName:       provider.TenantName,
		DomainName:       provider.DomainName,
		Username:         username,
		Password:         password,
	}, nil
}

func buildAppCredAuthOpts(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string, provider *esv1.BarbicanProvider) (gophercloud.AuthOptions, error) {
	if provider.Auth.ApplicationCredentialID == nil {
		return gophercloud.AuthOptions{}, fmt.Errorf(errMissingField, errors.New("applicationCredentialID is required"))
	}

	if provider.Auth.ApplicationCredentialSecret == nil {
		return gophercloud.AuthOptions{}, fmt.Errorf(errMissingField, errors.New("applicationCredentialSecret is required"))
	}

	appCredID := provider.Auth.ApplicationCredentialID.Value
	var err error

	if appCredID == "" {
		if provider.Auth.ApplicationCredentialID.SecretRef == nil {
			return gophercloud.AuthOptions{}, fmt.Errorf(errMissingField, errors.New("applicationCredentialID.secretRef is required when value is empty"))
		}
		appCredID, err = resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, provider.Auth.ApplicationCredentialID.SecretRef)
		if err != nil {
			return gophercloud.AuthOptions{}, fmt.Errorf(errMissingField, err)
		}
	}

	if provider.Auth.ApplicationCredentialSecret.SecretRef == nil {
		return gophercloud.AuthOptions{}, fmt.Errorf(errMissingField, errors.New("applicationCredentialSecret.secretRef is required"))
	}
	appCredSecret, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, provider.Auth.ApplicationCredentialSecret.SecretRef)
	if err != nil {
		return gophercloud.AuthOptions{}, fmt.Errorf(errMissingField, err)
	}

	return gophercloud.AuthOptions{
		IdentityEndpoint:            provider.AuthURL,
		ApplicationCredentialSecret: appCredSecret,
		ApplicationCredentialID:     appCredID,
	}, nil
}

// NewProvider constructs a new Barbican provider.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns a sample Barbican provider spec.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Barbican: &esv1.BarbicanProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the Barbican provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
