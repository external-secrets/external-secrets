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

package beyondtrustworkloadcredentials

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/beyondtrustworkloadcredentials/httpclient"
	btwcutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustworkloadcredentials/util"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// serviceAccountTokenExpirationSeconds is the lifetime requested for OIDC ServiceAccount tokens (the TokenRequest minimum).
// A fresh token is requested every time a client is constructed.
const serviceAccountTokenExpirationSeconds = 600

// serviceNamePattern matches the service names accepted by BeyondTrust Workload Credentials.
var serviceNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var (
	// ErrNoStore is returned when the BeyondtrustWorkloadCredentials SecretStore is missing or invalid.
	ErrNoStore = errors.New("missing or invalid BeyondtrustWorkloadCredentials SecretStore")
	// ErrNoAPIKey is returned when the API Token is missing or invalid.
	ErrNoAPIKey = errors.New("missing or invalid BeyondtrustWorkloadCredentials API Token in BeyondtrustWorkloadCredentials SecretStore")
	// ErrNoTokenName is returned when the API Token name is missing or invalid.
	ErrNoTokenName = errors.New("missing or invalid BeyondtrustWorkloadCredentials API Token name in BeyondtrustWorkloadCredentials SecretStore")
	// ErrNoTokenKey is returned when the API Token key is missing or invalid.
	ErrNoTokenKey = errors.New("missing or invalid BeyondtrustWorkloadCredentials API Token key in BeyondtrustWorkloadCredentials SecretStore")
	// ErrNoAuth is returned when the auth block does not set exactly one authentication method.
	ErrNoAuth = errors.New("exactly one of apikey or oidc must be set in BeyondtrustWorkloadCredentials SecretStore auth")
	// ErrNoServiceName is returned when the OIDC service name is missing.
	ErrNoServiceName = errors.New("missing BeyondtrustWorkloadCredentials OIDC service name in BeyondtrustWorkloadCredentials SecretStore")
	// ErrInvalidServiceName is returned when the OIDC service name has an invalid format.
	ErrInvalidServiceName = errors.New("invalid BeyondtrustWorkloadCredentials OIDC service name: use letters, digits, hyphens, and underscores only")
	// ErrNoServiceAccount is returned when the OIDC ServiceAccount name is missing.
	ErrNoServiceAccount = errors.New("missing BeyondtrustWorkloadCredentials OIDC service account name in BeyondtrustWorkloadCredentials SecretStore")
	// ErrNoServer is returned when the BeyondtrustWorkloadCredentials Server is missing or invalid.
	ErrNoServer = errors.New("missing or invalid BeyondtrustWorkloadCredentials Server in BeyondtrustWorkloadCredentials SecretStore")
	// ErrNoAPIURL is returned when the Server API URL is missing or invalid.
	ErrNoAPIURL = errors.New("missing or invalid BeyondtrustWorkloadCredentials Server API URL in BeyondtrustWorkloadCredentials SecretStore")
	// ErrNoSiteID is returned when the Server site ID is missing or invalid.
	ErrNoSiteID = errors.New("missing or invalid BeyondtrustWorkloadCredentials Server site ID in BeyondtrustWorkloadCredentials SecretStore")
)

// Provider is a BeyondtrustWorkloadCredentials provider implementing NewClient and ValidateStore for the esv1.Provider interface.
type Provider struct {
	// NewBeyondtrustWorkloadCredentialsClient is a function that returns a new BeyondTrust Secrets client.
	// This is used for testing to inject a fake client.
	NewBeyondtrustWorkloadCredentialsClient func(server, token, serviceName string) (btwcutil.Client, error)
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Client{}
var _ esv1.Provider = &Provider{}

// NewClient constructs a BeyondtrustWorkloadCredentials SecretsManager Provider.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.BeyondtrustWorkloadCredentials == nil {
		return nil, ErrNoStore
	}

	BeyondtrustWorkloadCredentialsStoreSpec := storeSpec.Provider.BeyondtrustWorkloadCredentials
	storeKind := store.GetKind()

	// fetch server values from spec
	serverURL, token, serviceName, err := fetchServerValuesFromSpec(ctx, BeyondtrustWorkloadCredentialsStoreSpec, kube, namespace, storeKind)
	if err != nil {
		return nil, err
	}

	// create BeyondtrustWorkloadCredentials client
	BeyondtrustWorkloadCredentialsClient, err := p.newClient(ctx, serverURL, token, serviceName, BeyondtrustWorkloadCredentialsStoreSpec, kube, namespace, storeKind)
	if err != nil {
		return nil, fmt.Errorf("failed to create BeyondtrustWorkloadCredentials client: %w", err)
	}

	client := &Client{
		beyondtrustWorkloadCredentialsClient: BeyondtrustWorkloadCredentialsClient,
		store:                                BeyondtrustWorkloadCredentialsStoreSpec,
	}

	return client, nil
}

// newClient is a shared helper creates the appropriate BeyondtrustWorkloadCredentials client based on the provided spec.
func (p *Provider) newClient(
	ctx context.Context,
	serverURL, token, serviceName string,
	btSpec *esv1.BeyondtrustWorkloadCredentialsProvider,
	kube kclient.Client,
	namespace, storeKind string,
) (btwcutil.Client, error) {
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
		return httpclient.NewBeyondtrustWorkloadCredentialsClientWithCustomCA(serverURL, token, serviceName, caCert)
	}
	if p.NewBeyondtrustWorkloadCredentialsClient != nil {
		return p.NewBeyondtrustWorkloadCredentialsClient(serverURL, token, serviceName)
	}
	return httpclient.NewBeyondtrustWorkloadCredentialsClient(serverURL, token, serviceName)
}

// NewGeneratorClient creates a new BeyondtrustWorkloadCredentials client for the generator controller.
func (p *Provider) NewGeneratorClient(ctx context.Context, kube kclient.Client, btSpec *esv1.BeyondtrustWorkloadCredentialsProvider, namespace string) (btwcutil.Client, error) {
	if btSpec == nil {
		return nil, ErrNoStore
	}

	serverURL, token, serviceName, err := fetchServerValuesFromSpec(ctx, btSpec, kube, namespace, "")
	if err != nil {
		return nil, err
	}

	client, err := p.newClient(ctx, serverURL, token, serviceName, btSpec, kube, namespace, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create BeyondtrustWorkloadCredentials client: %w", err)
	}

	return client, nil
}

// ValidateStore checks if the BeyondtrustWorkloadCredentials store is valid.
// The provider may return a warning and an error.
// The intended use of the warning to indicate a deprecation of behavior
// or other type of message that is NOT a validation failure but should be noticed by the user.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.BeyondtrustWorkloadCredentials == nil {
		return nil, ErrNoStore
	}

	BeyondtrustWorkloadCredentialsStoreSpec := storeSpec.Provider.BeyondtrustWorkloadCredentials

	// validate auth: exactly one method must be set
	auth := BeyondtrustWorkloadCredentialsStoreSpec.Auth
	if auth == nil || (auth.APIKey == nil) == (auth.OIDC == nil) {
		return nil, ErrNoAuth
	}
	if auth.APIKey != nil {
		tokenRef := auth.APIKey.Token
		if err := esutils.ValidateSecretSelector(store, tokenRef); err != nil {
			return nil, err
		}
		if tokenRef.Name == "" {
			return nil, ErrNoTokenName
		}
	} else {
		if err := validateServiceName(auth.OIDC.ServiceName); err != nil {
			return nil, err
		}
		saRef := auth.OIDC.ServiceAccountRef
		if err := esutils.ValidateServiceAccountSelector(store, saRef); err != nil {
			return nil, err
		}
		if saRef.Name == "" {
			return nil, ErrNoServiceAccount
		}
	}

	// validate server config is present and contains required fields
	if BeyondtrustWorkloadCredentialsStoreSpec.Server == nil {
		return nil, ErrNoServer
	}
	if BeyondtrustWorkloadCredentialsStoreSpec.Server.APIURL == "" {
		return nil, ErrNoAPIURL
	}

	// Validate APIURL format
	if err := validateAPIURL(BeyondtrustWorkloadCredentialsStoreSpec.Server.APIURL); err != nil {
		return nil, fmt.Errorf("invalid apiUrl: %w", err)
	}

	if BeyondtrustWorkloadCredentialsStoreSpec.Server.SiteID == "" {
		return nil, ErrNoSiteID
	}

	// Validate SiteID format (should be UUID)
	if err := validateSiteID(BeyondtrustWorkloadCredentialsStoreSpec.Server.SiteID); err != nil {
		return nil, fmt.Errorf("invalid siteId: %w", err)
	}

	return nil, nil
}

// Capabilities returns the BeyondtrustWorkloadCredentials provider Capabilities (Read, Write, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// loadCredentialsFromSpec returns the Bearer token and, for OIDC authentication, the service name to present.
func loadCredentialsFromSpec(ctx context.Context, spec *esv1.BeyondtrustWorkloadCredentialsProvider, kube kclient.Client, namespace, storeKind, siteID string) (string, string, error) {
	if spec == nil {
		return "", "", ErrNoStore
	}
	if spec.Auth == nil || (spec.Auth.APIKey == nil) == (spec.Auth.OIDC == nil) {
		return "", "", ErrNoAuth
	}

	if spec.Auth.OIDC != nil {
		oidc := spec.Auth.OIDC
		if err := validateServiceName(oidc.ServiceName); err != nil {
			return "", "", err
		}
		token, err := fetchServiceAccountToken(ctx, kube, storeKind, namespace, oidc.ServiceAccountRef, siteID)
		if err != nil {
			return "", "", err
		}
		return token, oidc.ServiceName, nil
	}

	tokenRef := spec.Auth.APIKey.Token
	if tokenRef.Name == "" {
		return "", "", ErrNoTokenName
	}
	if tokenRef.Key == "" {
		return "", "", ErrNoTokenKey
	}

	apiKey, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &tokenRef)
	if err != nil {
		return "", "", err
	}
	return apiKey, "", nil
}

// fetchServiceAccountToken requests a short-lived token for the referenced ServiceAccount via the TokenRequest API.
// The ServiceAccount lives in the ExternalSecret namespace; a ClusterSecretStore may override it.
// The token audience defaults to the site ID, which is what BeyondTrust Workload Credentials expects.
func fetchServiceAccountToken(ctx context.Context, kube kclient.Client, storeKind, namespace string, saRef esmeta.ServiceAccountSelector, siteID string) (string, error) {
	if saRef.Name == "" {
		return "", ErrNoServiceAccount
	}

	saNamespace := namespace
	if storeKind == esv1.ClusterSecretStoreKind && saRef.Namespace != nil {
		saNamespace = *saRef.Namespace
	}

	audiences := saRef.Audiences
	if len(audiences) == 0 {
		audiences = []string{siteID}
	}

	expirationSeconds := int64(serviceAccountTokenExpirationSeconds)
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}
	sa := &corev1.ServiceAccount{
		Name:      saRef.Name,
		Namespace: saNamespace,
	}
	if err := kube.SubResource("token").Create(ctx, sa, tokenRequest); err != nil {
		return "", fmt.Errorf("failed to create token for service account %s/%s: %w", saNamespace, saRef.Name, err)
	}

	return tokenRequest.Status.Token, nil
}

func loadURLFromSpec(spec *esv1.BeyondtrustWorkloadCredentialsProvider) (string, string, error) {
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

// fetchServerValuesFromSpec returns the site server URL, the Bearer token, and the service name (OIDC only).
func fetchServerValuesFromSpec(ctx context.Context, spec *esv1.BeyondtrustWorkloadCredentialsProvider, kube kclient.Client, namespace, storeKind string) (string, string, string, error) {
	if spec == nil {
		return "", "", "", ErrNoStore
	}

	// The site ID is loaded first because it is the default OIDC token audience.
	baseURL, siteID, err := loadURLFromSpec(spec)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to load server URL configuration: %w", err)
	}

	token, serviceName, err := loadCredentialsFromSpec(ctx, spec, kube, namespace, storeKind, siteID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to load credentials: %w", err)
	}

	return siteServerURL(baseURL, siteID), token, serviceName, nil
}

// siteServerURL is the root of the site's Workload Credentials API: {apiUrl}/{siteId}/wlc.
func siteServerURL(baseURL, siteID string) string {
	// Normalize baseURL by removing trailing slash to prevent double slashes
	return fmt.Sprintf("%s/%s/wlc", strings.TrimRight(baseURL, "/"), siteID)
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{
		NewBeyondtrustWorkloadCredentialsClient: httpclient.NewBeyondtrustWorkloadCredentialsClient,
	}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		BeyondtrustWorkloadCredentials: &esv1.BeyondtrustWorkloadCredentialsProvider{},
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

	// The site path is appended to apiUrl as a string, so anything after a "?" or "#" would
	// swallow it. Checked on the raw string: url.Parse drops a bare trailing "#".
	if strings.ContainsAny(apiURL, "?#") {
		return fmt.Errorf("apiUrl must not include a query string or fragment")
	}

	return nil
}

// validateSiteID validates the BeyondTrust site ID format (must be a valid UUID).
func validateSiteID(siteID string) error {
	if siteID == "" {
		return fmt.Errorf("siteId cannot be empty")
	}

	// Validate UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	// Where y is one of [8, 9, a, b] (RFC 4122 variant bits)
	uuidPattern := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
	matched, err := regexp.MatchString(uuidPattern, siteID)
	if err != nil {
		return fmt.Errorf("failed to validate siteId format: %w", err)
	}

	if !matched {
		return fmt.Errorf("siteId must be a valid UUID format, got %q", siteID)
	}

	return nil
}

// validateServiceName validates the OIDC service name format.
func validateServiceName(serviceName string) error {
	if serviceName == "" {
		return ErrNoServiceName
	}
	if !serviceNamePattern.MatchString(serviceName) {
		return ErrInvalidServiceName
	}
	return nil
}
