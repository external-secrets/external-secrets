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

package beyondtrustsecrets

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/httpclient"
	btsutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/util"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

var (
	// ErrNoStore is returned when the BeyondtrustSecrets SecretStore is missing or invalid.
	ErrNoStore = errors.New("missing or invalid BeyondtrustSecrets SecretStore")
	// ErrNoAPIKey is returned when the API Token is missing or invalid.
	ErrNoAPIKey = errors.New("missing or invalid BeyondtrustSecrets API Token in BeyondtrustSecrets SecretStore")
	// ErrNoTokenName is returned when the API Token name is missing or invalid.
	ErrNoTokenName = errors.New("missing or invalid BeyondtrustSecrets API Token name in BeyondtrustSecrets SecretStore")
	// ErrNoTokenKey is returned when the API Token key is missing or invalid.
	ErrNoTokenKey = errors.New("missing or invalid BeyondtrustSecrets API Token key in BeyondtrustSecrets SecretStore")
	// ErrNoServer is returned when the BeyondtrustSecrets Server is missing or invalid.
	ErrNoServer = errors.New("missing or invalid BeyondtrustSecrets Server in BeyondtrustSecrets SecretStore")
	// ErrNoAPIURL is returned when the Server API URL is missing or invalid.
	ErrNoAPIURL = errors.New("missing or invalid BeyondtrustSecrets Server API URL in BeyondtrustSecrets SecretStore")
	// ErrNoSiteID is returned when the Server site ID is missing or invalid.
	ErrNoSiteID = errors.New("missing or invalid BeyondtrustSecrets Server site ID in BeyondtrustSecrets SecretStore")
)

// Provider is a BeyondtrustSecrets provider implementing NewClient and ValidateStore for the esv1.Provider interface.
type Provider struct {
	// NewBeyondtrustSecretsClient is a function that returns a new BeyondtrustSecrets client.
	// This is used for testing to inject a fake client.
	NewBeyondtrustSecretsClient func(server, token string) (btsutil.Client, error)
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Client{}
var _ esv1.Provider = &Provider{}

// NewClient constructs a BeyondtrustSecrets SecretsManager Provider.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.BeyondtrustSecrets == nil {
		return nil, ErrNoStore
	}

	BeyondtrustSecretsStoreSpec := storeSpec.Provider.BeyondtrustSecrets
	storeKind := store.GetKind()

	// fetch server values from spec
	serverURL, apiKey, err := fetchServerValuesFromSpec(ctx, BeyondtrustSecretsStoreSpec, kube, namespace, storeKind)
	if err != nil {
		return nil, err
	}

	// create BeyondtrustSecrets client
	BeyondtrustSecretsClient, err := p.newClient(ctx, serverURL, apiKey, BeyondtrustSecretsStoreSpec, kube, namespace, storeKind)
	if err != nil {
		return nil, fmt.Errorf("failed to create BeyondtrustSecrets client: %w", err)
	}

	client := &Client{
		beyondtrustSecretsClient: BeyondtrustSecretsClient,
		store:                    BeyondtrustSecretsStoreSpec,
	}

	return client, nil
}

// newClient is a shared helper creates the appropriate BeyondtrustSecrets client based on the provided spec.
func (p *Provider) newClient(ctx context.Context, serverURL, apiKey string, btSpec *esv1.BeyondtrustSecretsProvider, kube kclient.Client, namespace, storeKind string) (btsutil.Client, error) {
	// Fetch CA from CABundle/CAProvider using ESO helper
	var caCert []byte
	var err error
	if btSpec != nil {
		caCert, err = esutils.FetchCACertFromSource(ctx, esutils.CreateCertOpts{
			StoreKind:  storeKind,
			Client:     kube,
			Namespace:  namespace,
			CABundle:   btSpec.CABundle,
			CAProvider: btSpec.CAProvider,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch CA certificate: %w", err)
		}
	}

	if len(caCert) > 0 {
		return httpclient.NewBeyondtrustSecretsClientWithCustomCA(serverURL, apiKey, caCert)
	}
	if p.NewBeyondtrustSecretsClient != nil {
		return p.NewBeyondtrustSecretsClient(serverURL, apiKey)
	}
	return httpclient.NewBeyondtrustSecretsClient(serverURL, apiKey)
}

// NewGeneratorClient creates a new BeyondtrustSecrets client for the generator controller.
func (p *Provider) NewGeneratorClient(ctx context.Context, kube kclient.Client, btSpec *esv1.BeyondtrustSecretsProvider, namespace string) (btsutil.Client, error) {
	if btSpec == nil {
		return nil, ErrNoStore
	}

	serverURL, apiKey, err := fetchServerValuesFromSpec(ctx, btSpec, kube, namespace, "")
	if err != nil {
		return nil, err
	}

	client, err := p.newClient(ctx, serverURL, apiKey, btSpec, kube, namespace, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create BeyondtrustSecrets client: %w", err)
	}

	return client, nil
}

// ValidateStore checks if the BeyondtrustSecrets store is valid.
// The provider may return a warning and an error.
// The intended use of the warning to indicate a deprecation of behavior
// or other type of message that is NOT a validation failure but should be noticed by the user.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.BeyondtrustSecrets == nil {
		return nil, ErrNoStore
	}

	BeyondtrustSecretsStoreSpec := storeSpec.Provider.BeyondtrustSecrets

	// validate token selector
	if BeyondtrustSecretsStoreSpec.Auth == nil {
		return nil, ErrNoAPIKey
	}
	tokenRef := BeyondtrustSecretsStoreSpec.Auth.APIKey.Token
	if err := esutils.ValidateSecretSelector(store, tokenRef); err != nil {
		return nil, err
	}
	if tokenRef.Name == "" {
		return nil, ErrNoTokenName
	}

	// validate server config is present and contains required fields
	if BeyondtrustSecretsStoreSpec.Server == nil {
		return nil, ErrNoServer
	}
	if BeyondtrustSecretsStoreSpec.Server.APIURL == "" {
		return nil, ErrNoAPIURL
	}

	// Validate APIURL format
	if err := validateAPIURL(BeyondtrustSecretsStoreSpec.Server.APIURL); err != nil {
		return nil, fmt.Errorf("invalid apiUrl: %w", err)
	}

	if BeyondtrustSecretsStoreSpec.Server.SiteID == "" {
		return nil, ErrNoSiteID
	}

	// Validate SiteID format (should be UUID)
	if err := validateSiteID(BeyondtrustSecretsStoreSpec.Server.SiteID); err != nil {
		return nil, fmt.Errorf("invalid siteId: %w", err)
	}

	return nil, nil
}

// Capabilities returns the BeyondtrustSecrets provider Capabilities (Read, Write, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

func loadAPIKeyFromSpec(ctx context.Context, spec *esv1.BeyondtrustSecretsProvider, kube kclient.Client, namespace, storeKind string) (string, error) {
	if spec == nil {
		return "", ErrNoStore
	}
	if spec.Auth == nil {
		return "", ErrNoAPIKey
	}

	tokenRef := spec.Auth.APIKey.Token
	if tokenRef.Name == "" {
		return "", ErrNoTokenName
	}
	if tokenRef.Key == "" {
		return "", ErrNoTokenKey
	}

	return resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &tokenRef)
}

func loadURLFromSpec(spec *esv1.BeyondtrustSecretsProvider) (string, string, error) {
	if spec == nil {
		return "", "", ErrNoStore
	}
	if spec.Server == nil {
		return "", "", ErrNoServer
	}

	if spec.Server.APIURL == "" {
		return "", "", ErrNoAPIURL
	}

	// Validate APIURL format
	if err := validateAPIURL(spec.Server.APIURL); err != nil {
		return "", "", fmt.Errorf("invalid apiUrl: %w", err)
	}

	if spec.Server.SiteID == "" {
		return "", "", ErrNoSiteID
	}

	// Validate SiteID format
	if err := validateSiteID(spec.Server.SiteID); err != nil {
		return "", "", fmt.Errorf("invalid siteId: %w", err)
	}

	return spec.Server.APIURL, spec.Server.SiteID, nil
}

func fetchServerValuesFromSpec(ctx context.Context, spec *esv1.BeyondtrustSecretsProvider, kube kclient.Client, namespace, storeKind string) (string, string, error) {
	if spec == nil {
		return "", "", ErrNoStore
	}

	apiKey, err := loadAPIKeyFromSpec(ctx, spec, kube, namespace, storeKind)
	if err != nil {
		return "", "", fmt.Errorf("failed to load credentials: %w", err)
	}

	baseURL, siteID, err := loadURLFromSpec(spec)
	if err != nil {
		return "", "", fmt.Errorf("failed to load server URL configuration: %w", err)
	}

	serverURL := fmt.Sprintf("%s/%s/secrets", baseURL, siteID)

	return serverURL, apiKey, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{
		NewBeyondtrustSecretsClient: httpclient.NewBeyondtrustSecretsClient,
	}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		BeyondtrustSecrets: &esv1.BeyondtrustSecretsProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}

// validateAPIURL validates the BeyondTrust API URL format.
func validateAPIURL(apiURL string) error {
	if apiURL == "" {
		return fmt.Errorf("apiUrl cannot be empty")
	}

	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return fmt.Errorf("failed to parse apiUrl: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("apiUrl must include a scheme (https)")
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("apiUrl must use https scheme, got %q", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("apiUrl must include a host")
	}

	return nil
}

// validateSiteID validates the BeyondTrust site ID format (must be a valid UUID).
func validateSiteID(siteID string) error {
	if siteID == "" {
		return fmt.Errorf("siteId cannot be empty")
	}

	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	// More lenient: accepts any UUID format
	uuidPattern := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	matched, err := regexp.MatchString(uuidPattern, siteID)
	if err != nil {
		return fmt.Errorf("failed to validate siteId format: %w", err)
	}

	if !matched {
		return fmt.Errorf("siteId must be a valid UUID format, got %q", siteID)
	}

	return nil
}
