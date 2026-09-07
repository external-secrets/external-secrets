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

// Package ovh implements a provider that enables synchronization with OVHcloud's Secret Manager.
package ovh

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/ovh/okms-sdk-go"
	"github.com/ovh/okms-sdk-go/types"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	emptyTokenSecretRef            = "ovh store auth.token.tokenSecretRef cannot be empty"
	emptyKeySecretRef              = "ovh store auth.mtls.keySecretRef cannot be empty"
	emptyCertSecretRef             = "ovh store auth.mtls.certSecretRef cannot be empty"
	createOvhProviderError         = "failed to create new ovh provider client"
	createOkmsClientError          = "failed to create new okms client"
	configureTokenOkmsClientError  = "failed to configure token okms client"
	configureMtlsOkmsClientError   = "failed to configure mtls okms client"
	emptyClientIDSecretRef         = "ovh store auth.oauth2.clientIDSecretRef cannot be empty"
	emptyClientSecretSecretRef     = "ovh store auth.oauth2.clientSecretSecretRef cannot be empty"
	configureOAuth2OkmsClientError = "failed to configure oauth2 okms client"
	// defaultOAuth2TokenURL is OVHcloud's European token endpoint. Canada and the US have
	// their own, and a store may name either through auth.oauth2.tokenURL.
	defaultOAuth2TokenURL = "https://www.ovh.com/auth/oauth2/token"
	invalidTokenURLError  = "invalid auth.oauth2.tokenURL"
)

// oauth2TokenURLHosts are the hosts OVHcloud publishes token endpoints on, one per region.
var oauth2TokenURLHosts = []string{"www.ovh.com", "ca.ovh.com", "us.ovhcloud.com"}

// Provider implements the ESO Provider interface for OVHcloud.
type Provider struct {
	secretKeyResolver SecretKeyResolver
}

// OkmsClient defines an interface for interacting with the OVH OKMS service.
// It allows for both real API calls and mocking for unit tests.
type OkmsClient interface {
	GetSecretV2(ctx context.Context, okmsID uuid.UUID, path string, version *uint32, includeData *bool) (*types.GetSecretV2Response, error)
	ListSecretV2(ctx context.Context, okmsID uuid.UUID, pageSize *uint32, pageCursor *string) (*types.ListSecretV2ResponseWithPagination, error)
	PostSecretV2(ctx context.Context, okmsID uuid.UUID, body types.PostSecretV2Request) (*types.PostSecretV2Response, error)
	PutSecretV2(ctx context.Context, okmsID uuid.UUID, path string, cas *uint32, body types.PutSecretV2Request) (*types.PutSecretV2Response, error)
	DeleteSecretV2(ctx context.Context, okmsID uuid.UUID, path string) error
	WithCustomHeader(key, value string) *okms.Client
	GetSecretsMetadata(ctx context.Context, okmsID uuid.UUID, path string, list bool) (*types.GetMetadataResponse, error)
}

// SecretKeyResolver resolves the value of a key from a Kubernetes Secret.
// It is defined as an interface to allow different implementations, including mocks for testing.
type SecretKeyResolver interface {
	Resolve(ctx context.Context, kube kclient.Client, ovhStoreKind string, ovhStoreNameSpace string, secretRef v1.SecretKeySelector) (string, error)
}

// DefaultSecretKeyResolver is the default implementation for resolving keys from Kubernetes Secrets.
type DefaultSecretKeyResolver struct{}

type ovhClient struct {
	ovhStoreNameSpace string
	ovhStoreKind      string
	kube              kclient.Client
	okmsID            uuid.UUID
	cas               bool
	okmsTimeout       time.Duration
	okmsClient        OkmsClient
}

var _ esv1.SecretsClient = &ovhClient{}

// Resolve returns the value of the referenced key from a Kubernetes Secret.
func (r DefaultSecretKeyResolver) Resolve(ctx context.Context, kube kclient.Client, ovhStoreKind, ovhStoreNameSpace string, secretRef v1.SecretKeySelector) (string, error) {
	return resolvers.SecretKeyRef(ctx, kube, ovhStoreKind, ovhStoreNameSpace, &secretRef)
}

// NewClient creates a new Provider client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	// Validate Store before creating a client from it.
	_, err := p.ValidateStore(store)
	if err != nil {
		return nil, fmt.Errorf("%s: store validation failed: %w", createOvhProviderError, err)
	}

	if kube == nil {
		return nil, fmt.Errorf("%s: controller-runtime client is nil", createOvhProviderError)
	}

	ovhStore := store.GetSpec().Provider.OVHcloud
	// ovhClient configuration.
	okmsID, err := uuid.Parse(ovhStore.OkmsID)
	if err != nil {
		return nil, fmt.Errorf("%s: could not parse okmsID: %w", createOvhProviderError, err)
	}

	cas := false
	if ovhStore.CasRequired != nil {
		cas = *ovhStore.CasRequired
	}

	okmsTimeout := 30 * time.Second
	if ovhStore.OkmsTimeout != nil {
		okmsTimeout = time.Duration(*ovhStore.OkmsTimeout) * time.Second
	}
	cl := &ovhClient{
		ovhStoreNameSpace: namespace,
		ovhStoreKind:      store.GetKind(),
		kube:              kube,
		okmsID:            okmsID,
		cas:               cas,
		okmsTimeout:       okmsTimeout,
	}

	// Authentication configuration: token, mTLS or OAuth2.
	if ovhStore.Auth.ClientToken != nil {
		err = configureHTTPTokenClient(ctx, p, cl,
			ovhStore.Server, ovhStore.Auth.ClientToken)
	} else if ovhStore.Auth.ClientMTLS != nil {
		err = configureHTTPMTLSClient(ctx, p, cl,
			ovhStore.Server, ovhStore.Auth.ClientMTLS)
	} else if ovhStore.Auth.ClientOAuth2 != nil {
		err = configureHTTPOAuth2Client(ctx, p, cl,
			ovhStore.Server, ovhStore.Auth.ClientOAuth2)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", createOvhProviderError, err)
	}
	return cl, nil
}

// configureHTTPTokenClient clientConfigure the client to use the provided token for HTTP requests.
func configureHTTPTokenClient(ctx context.Context, p *Provider, cl *ovhClient, server string, clientToken *esv1.OvhClientToken) error {
	token, err := getToken(ctx, p, cl, clientToken)
	if err != nil {
		return fmt.Errorf("%s: could not retrieve token: %w", configureTokenOkmsClientError, err)
	}
	bearerToken := fmt.Sprintf("Bearer %s", token)

	// Request a new OKMS client from the OVH SDK.
	httpClient := &http.Client{
		Timeout: cl.okmsTimeout,
	}
	cl.okmsClient, err = okms.NewRestAPIClientWithHttp(server, httpClient)
	if err != nil {
		return fmt.Errorf("%s: %s: %w", configureTokenOkmsClientError, createOkmsClientError, err)
	}
	if cl.okmsClient == nil {
		return fmt.Errorf("%s: okms client is nil", configureTokenOkmsClientError)
	}

	// Add a custom header.
	cl.okmsClient.WithCustomHeader("Authorization", bearerToken)
	cl.okmsClient.WithCustomHeader("Content-type", "application/json")

	return nil
}

// getToken retrieves the token value from the Kubernetes secret.
func getToken(ctx context.Context, p *Provider, cl *ovhClient, clientToken *esv1.OvhClientToken) (string, error) {
	// ClienTokenSecret refers to the Kubernetes secret that stores the token.
	tokenSecretRef := clientToken.ClientTokenSecret

	// Retrieve the token value.
	token, err := p.secretKeyResolver.Resolve(ctx, cl.kube,
		cl.ovhStoreKind, cl.ovhStoreNameSpace, tokenSecretRef)
	if err != nil {
		return "", fmt.Errorf("failed to resolve token secret ref: %w", err)
	}
	if token == "" {
		return "", errors.New(emptyTokenSecretRef)
	}

	return token, nil
}

// configureHTTPOAuth2Client configures the client to authenticate with an OVHcloud service account.
//
// A service account yields an OAuth2 client id and client secret, which are exchanged for a short
// lived access token. The exchange and every renewal are handled by the transport returned by
// clientcredentials, so nothing here has to watch an expiry.
func configureHTTPOAuth2Client(ctx context.Context, p *Provider, cl *ovhClient, server string, clientOAuth2 *esv1.OvhClientOAuth2) error {
	clientID, err := p.secretKeyResolver.Resolve(ctx, cl.kube,
		cl.ovhStoreKind, cl.ovhStoreNameSpace, clientOAuth2.ClientID)
	if err != nil {
		return fmt.Errorf("%s: could not retrieve client id: %w", configureOAuth2OkmsClientError, err)
	}
	if clientID == "" {
		return errors.New(emptyClientIDSecretRef)
	}

	clientSecret, err := p.secretKeyResolver.Resolve(ctx, cl.kube,
		cl.ovhStoreKind, cl.ovhStoreNameSpace, clientOAuth2.ClientSecret)
	if err != nil {
		return fmt.Errorf("%s: could not retrieve client secret: %w", configureOAuth2OkmsClientError, err)
	}
	if clientSecret == "" {
		return errors.New(emptyClientSecretSecretRef)
	}

	tokenURL, err := resolveOAuth2TokenURL(clientOAuth2.TokenURL)
	if err != nil {
		return fmt.Errorf("%s: %w", configureOAuth2OkmsClientError, err)
	}

	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		AuthStyle:    oauth2.AuthStyleInHeader,
	}

	// The timeout applies to the token exchange as well as to the KMS calls: without this the
	// exchange would use the default client and ignore okmsTimeout entirely.
	//
	// CheckRedirect refuses a redirect that leaves HTTPS. The client credentials travel in the
	// exchange request, so a token endpoint answering a redirect to plain HTTP would put them on
	// the wire in clear.
	exchangeClient := &http.Client{
		Timeout: cl.okmsTimeout,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if req.URL.Scheme != "https" {
				return fmt.Errorf("refusing redirect to a non-https token endpoint: %s", req.URL.Scheme)
			}
			return nil
		},
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, exchangeClient)
	httpClient := config.Client(ctx)
	httpClient.Timeout = cl.okmsTimeout

	cl.okmsClient, err = okms.NewRestAPIClientWithHttp(server, httpClient)
	if err != nil {
		return fmt.Errorf("%s: %s: %w", configureOAuth2OkmsClientError, createOkmsClientError, err)
	}
	if cl.okmsClient == nil {
		return fmt.Errorf("%s: okms client is nil", configureOAuth2OkmsClientError)
	}

	// No Authorization header is set here: the transport adds one and renews it.
	cl.okmsClient.WithCustomHeader("Content-type", "application/json")

	return nil
}

// resolveOAuth2TokenURL returns the token endpoint to use, and refuses anything that is not one
// of OVHcloud's.
//
// The value comes from a SecretStore, so whoever can edit one would otherwise choose where the
// client credentials are sent. Restricting it to OVHcloud's own hosts, over HTTPS, keeps that
// choice to picking a region.
func resolveOAuth2TokenURL(raw string) (string, error) {
	if raw == "" {
		return defaultOAuth2TokenURL, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%s: %w", invalidTokenURLError, err)
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("%s: scheme must be https, got %q", invalidTokenURLError, parsed.Scheme)
	}
	if !slices.Contains(oauth2TokenURLHosts, parsed.Host) {
		return "", fmt.Errorf("%s: host must be one of %v, got %q",
			invalidTokenURLError, oauth2TokenURLHosts, parsed.Host)
	}
	return raw, nil
}

// configureHTTPMTLSClient configures the client to use mTLS for HTTP requests.
func configureHTTPMTLSClient(ctx context.Context, p *Provider, cl *ovhClient, server string, clientMTLS *esv1.OvhClientMTLS) error {
	httpClient, err := newHTTPClientWithMTLS(ctx, p, cl, clientMTLS)
	if err != nil {
		return fmt.Errorf("%s: could not create http client:%w", configureMtlsOkmsClientError, err)
	}

	// Request a new OKMS client from the OVH SDK (mTLS configured).
	cl.okmsClient, err = okms.NewRestAPIClientWithHttp(server, httpClient)
	if err != nil {
		return fmt.Errorf("%s: %s: %w", configureMtlsOkmsClientError, createOkmsClientError, err)
	}
	if cl.okmsClient == nil {
		return fmt.Errorf("%s: okms client is nil", configureMtlsOkmsClientError)
	}

	return nil
}

// getClientConfig creates an HTTP client configured for MTLS using the provided
// client certificate and key, and optionally adds a custom CA from CAProvider or CABundle.
func newHTTPClientWithMTLS(ctx context.Context, p *Provider, cl *ovhClient, clientMTLS *esv1.OvhClientMTLS) (*http.Client, error) {
	cert, err := buildX509Certificate(ctx, cl, p, clientMTLS)
	if err != nil {
		return nil, fmt.Errorf("failed to build x509 certificate: %w", err)
	}

	// Create an HTTP transport for mTLS, enforcing TLS 1.2+ and using the client certificate.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	// Configure custom CA for the TLS client if provided via CAProvider or CABundle.
	if clientMTLS.CAProvider != nil || len(clientMTLS.CABundle) != 0 {
		caCertPool := x509.NewCertPool()
		ca, err := esutils.FetchCACertFromSource(ctx, esutils.CreateCertOpts{
			CABundle:   clientMTLS.CABundle,
			CAProvider: clientMTLS.CAProvider,
			StoreKind:  cl.ovhStoreKind,
			Namespace:  cl.ovhStoreNameSpace,
			Client:     cl.kube,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch CA cert: %w", err)
		}
		if !caCertPool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to append CA")
		}
		transport.TLSClientConfig.RootCAs = caCertPool
	}
	// Build the HTTP client with configured transport and timeout.
	httpClient := http.Client{
		Timeout:   cl.okmsTimeout,
		Transport: transport,
	}
	return &httpClient, nil
}

// buildX509Certificate retrieves client certificate and key to build X509 Certificate.
func buildX509Certificate(ctx context.Context, cl *ovhClient, p *Provider, clientMTLS *esv1.OvhClientMTLS) (tls.Certificate, error) {
	clientKey, err := resolveSecretValue(ctx, cl, p, clientMTLS.ClientKey, emptyKeySecretRef)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to resolve client key: %w", err)
	}
	clientCert, err := resolveSecretValue(ctx, cl, p, clientMTLS.ClientCertificate, emptyCertSecretRef)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to resolve client certificate: %w", err)
	}
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create x509 key pair: %w", err)
	}
	return cert, nil
}

// resolveSecret retrieves the value of the client certificate and key.
func resolveSecretValue(ctx context.Context, cl *ovhClient, p *Provider, ref v1.SecretKeySelector, errMsg string) (string, error) {
	// ref refers to the Kubernetes secret object.
	// Retrieve the value of ref.
	secret, err := p.secretKeyResolver.Resolve(ctx, cl.kube,
		cl.ovhStoreKind, cl.ovhStoreNameSpace, ref)
	if err != nil {
		return "", fmt.Errorf("failed to resolve secret value: %w", err)
	}
	if secret == "" {
		return "", errors.New(errMsg)
	}
	return secret, nil
}

// ValidateStore statically validate the Secret Store specification.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	// Nil checks.
	if store == nil || reflect.ValueOf(store).IsNil() {
		return nil, errors.New("store is nil")
	}
	spec := store.GetSpec()
	if spec == nil {
		return nil, errors.New("store spec is nil")
	}
	provider := spec.Provider
	if provider == nil {
		return nil, errors.New("store provider is nil")
	}
	if provider.OVHcloud == nil {
		return nil, errors.New("ovh store provider is nil")
	}
	if provider.OVHcloud.OkmsTimeout != nil && *provider.OVHcloud.OkmsTimeout <= 0 {
		return nil, errors.New("ovh store okmsTimeout must be greater than 0")
	}
	if provider.OVHcloud.Server == "" {
		return nil, errors.New("ovh store server is required")
	}
	if _, err := url.Parse(provider.OVHcloud.Server); err != nil {
		return nil, fmt.Errorf("ovh store server must contain a valid url: %w", err)
	}
	if provider.OVHcloud.OkmsID == "" {
		return nil, errors.New("ovh store okmsID is required")
	}

	// Validate the provider's authentication method.
	return p.validateAuth(provider)
}

func (p *Provider) validateAuth(provider *esv1.SecretStoreProvider) (admission.Warnings, error) {
	auth := provider.OVHcloud.Auth
	configured := 0
	for _, set := range []bool{auth.ClientMTLS != nil, auth.ClientToken != nil, auth.ClientOAuth2 != nil} {
		if set {
			configured++
		}
	}
	if configured == 0 {
		return nil, errors.New("missing authentication method")
	} else if configured > 1 {
		return nil, errors.New("only one authentication method allowed (mtls | token | oauth2)")
	}
	if auth.ClientToken != nil && auth.ClientToken.ClientTokenSecret == (v1.SecretKeySelector{}) {
		return nil, errors.New("missing token secret for token authentication")
	}
	if auth.ClientMTLS != nil && (auth.ClientMTLS.ClientCertificate == (v1.SecretKeySelector{}) || auth.ClientMTLS.ClientKey == (v1.SecretKeySelector{})) {
		return nil, errors.New("missing tls certificate or key for mtls authentication")
	}
	if auth.ClientOAuth2 != nil && (auth.ClientOAuth2.ClientID == (v1.SecretKeySelector{}) || auth.ClientOAuth2.ClientSecret == (v1.SecretKeySelector{})) {
		return nil, errors.New("missing client id or client secret for oauth2 authentication")
	}
	return nil, nil
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{
		secretKeyResolver: DefaultSecretKeyResolver{},
	}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		OVHcloud: &esv1.OvhProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
