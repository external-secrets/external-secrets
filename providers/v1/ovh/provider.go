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

// Package ovh implements a provider that enables synchronization with OVHcloud's Secret Manager.
package ovh

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/ovh/okms-sdk-go"
	"github.com/ovh/okms-sdk-go/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	emptyTokenSecretRef    = "ovh store auth.token.tokenSecretRef cannot be empty"
	emptyKeySecretRef      = "ovh store auth.mtls.keySecretRef cannot be empty"
	emptyCertSecretRef     = "ovh store auth.mtls.certSecretRef cannot be empty"
	createOvhProviderError = "failed to create new ovh provider client"
	createOkmsClientError  = "failed to create new okms client"
)

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
	Resolve(ctx context.Context, kube kclient.Client, ovhStoreKind string, ovhStoreNameSpace string, secretRef *v1.SecretKeySelector) (string, error)
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
func (r DefaultSecretKeyResolver) Resolve(ctx context.Context, kube kclient.Client, ovhStoreKind, ovhStoreNameSpace string, secretRef *v1.SecretKeySelector) (string, error) {
	return resolvers.SecretKeyRef(ctx, kube, ovhStoreKind, ovhStoreNameSpace, secretRef)
}

// NewClient creates a new Provider client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	// Validate Store before creating a client from it.
	_, err := p.ValidateStore(store)
	if err != nil {
		return nil, err
	}

	if kube == nil {
		return nil, fmt.Errorf("%s: controller-runtime client is nil", createOvhProviderError)
	}

	ovhStore := store.GetSpec().Provider.Ovh
	// ovhClient configuration.
	okmsID, err := uuid.Parse(ovhStore.OkmsID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", createOvhProviderError, err)
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

	// Authentication configuration: token or mTLS.
	if p.secretKeyResolver == nil {
		p.secretKeyResolver = DefaultSecretKeyResolver{}
	}
	if ovhStore.Auth.ClientToken != nil {
		err = configureHTTPTokenClient(ctx, p, cl,
			ovhStore.Server, ovhStore.Auth.ClientToken)
	} else if ovhStore.Auth.ClientMTLS != nil {
		err = configureHTTPMTLSClient(ctx, p, cl,
			ovhStore.Server, ovhStore.Auth.ClientMTLS)
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
		return err
	}
	bearerToken := fmt.Sprintf("Bearer %s", token)

	// Request a new OKMS client from the OVH SDK.
	httpClient := &http.Client{
		Timeout: cl.okmsTimeout,
	}
	cl.okmsClient, err = okms.NewRestAPIClientWithHttp(server, httpClient)
	if err != nil {
		return err
	}
	if cl.okmsClient == nil {
		return errors.New(createOkmsClientError)
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
	if tokenSecretRef == nil {
		return "", errors.New(emptyTokenSecretRef)
	}

	// Retrieve the token value.
	token, err := p.secretKeyResolver.Resolve(ctx, cl.kube,
		cl.ovhStoreKind, cl.ovhStoreNameSpace, tokenSecretRef)
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", errors.New(emptyTokenSecretRef)
	}

	return token, nil
}

// configureHTTPMTLSClient configures the client to use mTLS for HTTP requests.
func configureHTTPMTLSClient(ctx context.Context, p *Provider, cl *ovhClient, server string, clientMTLS *esv1.OvhClientMTLS) error {
	httpClient, err := newHTTPClientWithMTLS(ctx, p, cl, clientMTLS)
	if err != nil {
		return err
	}

	// Request a new OKMS client from the OVH SDK (mTLS configured).
	cl.okmsClient, err = okms.NewRestAPIClientWithHttp(server, httpClient)
	if err != nil {
		return err
	}
	if cl.okmsClient == nil {
		return errors.New(createOkmsClientError)
	}

	return err
}

// getClientConfig creates an HTTP client configured for MTLS using the provided
// client certificate and key, and optionally adds a custom CA from CAProvider or CABundle.
func newHTTPClientWithMTLS(ctx context.Context, p *Provider, cl *ovhClient, clientMTLS *esv1.OvhClientMTLS) (*http.Client, error) {
	cert, err := buildX509Certificate(ctx, cl, p, clientMTLS)
	if err != nil {
		return nil, err
	}

	// Create an HTTP transport for mTLS, enforcing TLS 1.2+ and using the client certificate.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		},
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
			return nil, err
		}
		if ok := caCertPool.AppendCertsFromPEM(ca); !ok {
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
		return tls.Certificate{}, err
	}
	clientCert, err := resolveSecretValue(ctx, cl, p, clientMTLS.ClientCertificate, emptyCertSecretRef)
	if err != nil {
		return tls.Certificate{}, err
	}
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))

	return cert, err
}

// resolveSecret retrieves the value of the client certificate and key.
func resolveSecretValue(ctx context.Context, cl *ovhClient, p *Provider, ref *v1.SecretKeySelector, errMsg string) (string, error) {
	// ref refers to the Kubernetes secret object.
	if ref == nil {
		return "", errors.New(errMsg)
	}
	// Retrieve the value of ref.
	secret, err := p.secretKeyResolver.Resolve(ctx, cl.kube,
		cl.ovhStoreKind, cl.ovhStoreNameSpace, ref)
	if err != nil {
		return "", err
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
	if provider.Ovh == nil {
		return nil, errors.New("ovh store provider is nil")
	}
	if provider.Ovh.Server == "" {
		return nil, errors.New("ovh store server is required")
	}
	if provider.Ovh.OkmsID == "" {
		return nil, errors.New("ovh store okmsID is required")
	}

	// Validate the provider's authentication method.
	auth := provider.Ovh.Auth
	if auth.ClientMTLS == nil && auth.ClientToken == nil {
		return nil, errors.New("missing authentication method")
	} else if auth.ClientMTLS != nil && auth.ClientToken != nil {
		return nil, errors.New("only one authentication method allowed (mtls | token)")
	} else if auth.ClientMTLS != nil &&
		(auth.ClientMTLS.ClientCertificate == nil ||
			auth.ClientMTLS.ClientKey == nil) {
		return nil, errors.New("missing tls certificate or key")
	}

	return nil, nil
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Ovh: &esv1.OvhProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
