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

// Package mysterybox contains the logic to work with Nebius MysteryBox API.
package mysterybox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	lru "github.com/hashicorp/golang-lru"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/auth"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/iam"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/mysterybox"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/feature"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

var (
	log                                   = ctrl.Log.WithName("provider").WithName("nebius").WithName("mysterybox")
	mysteryboxTokensCacheSize             int
	mysteryboxConnectionsCacheSize        int
	defaultTokenCacheSize                 = 2 << 11
	defaultMysteryboxConnectionsCacheSize = 2 << 6
)

// NewMysteryboxClient is a function that describes how to create a Nebius MysteryBox client to interact within.
type NewMysteryboxClient func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error)

// NewCoreV1Client creates a typed Kubernetes CoreV1 client.
type NewCoreV1Client func() (typedcorev1.CoreV1Interface, error)

// SecretsClientConfig holds configuration for interacting with.
type SecretsClientConfig struct {
	APIDomain           string
	ServiceAccountCreds *esmeta.SecretKeySelector
	Token               *esmeta.SecretKeySelector
	CACertificate       *esmeta.SecretKeySelector
	WorkloadIdentity    *esv1.NebiusWorkloadIdentity
}

// ClientCacheKey represents a unique key for identifying cached MysteryBox clients.
// It is composed of an API domain and a hash of the CA certificate.
type ClientCacheKey struct {
	APIDomain string
	CAHash    string
}

// Provider is a struct for managing MysteryBox clients.
type Provider struct {
	Logger                      logr.Logger
	NewMysteryboxClient         NewMysteryboxClient
	NewCoreV1Client             NewCoreV1Client
	TokenGetter                 TokenGetter
	coreV1Client                typedcorev1.CoreV1Interface
	mysteryboxClientsCache      *lru.Cache
	tokenInitMutex              sync.Mutex
	coreV1InitMutex             sync.Mutex
	cacheInitMutex              sync.Mutex
	mysteryboxClientsCacheMutex sync.Mutex
}

// Capabilities returns the capabilities of the secret store, indicating it is read-only.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient creates and returns a new SecretsClient for the specified SecretStore and namespace context.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	clientConfig, err := parseConfig(store)
	if err != nil {
		return nil, err
	}

	var caCert []byte
	if clientConfig.CACertificate != nil {
		caCert, err = p.getCaCert(ctx, clientConfig, store, kube, namespace)
		if err != nil {
			return nil, fmt.Errorf("read CA certificate %s/%s: %w", namespace, clientConfig.CACertificate.Name, err)
		}
	}

	// lazy initialization with a current flag value
	if err = p.initTokenGetter(); err != nil {
		return nil, fmt.Errorf("init token getter: %w", err)
	}

	iamToken, err := p.getIamToken(ctx, clientConfig, store, kube, namespace, caCert)
	if err != nil {
		p.Logger.Info("Could not get IAM token", "store", store.GetNamespacedName(), "err", err)
		return nil, err
	}

	mysteryboxGrpcClient, err := p.createOrGetMysteryboxClient(ctx, clientConfig.APIDomain, caCert)
	if err != nil {
		p.Logger.Info("Could not create or get MysteryBox Client", "store", store.GetNamespacedName(), "err", err)
		return nil, err
	}

	return &SecretsClient{
		mysteryboxClient: mysteryboxGrpcClient,
		token:            iamToken,
	}, nil
}

// getIamToken retrieves an IAM token based on the provided SecretsClientConfig and authentication options.
// It supports token retrieval from a predefined secret or via service account credentials with the TokenGetter.
func (p *Provider) getIamToken(ctx context.Context, config *SecretsClientConfig, store esv1.GenericStore, kube client.Client, namespace string, caCert []byte) (string, error) {
	if config.Token != nil && config.Token.Name != "" {
		creds, err := auth.GetTokenCredentials(ctx, config.Token, store, kube, namespace)
		if err != nil {
			return "", err
		}
		return creds.Token, nil
	}
	if config.ServiceAccountCreds != nil && config.ServiceAccountCreds.Name != "" {
		credentialsRequest, err := auth.NewServiceAccountCredentialsRequest(ctx, config.ServiceAccountCreds, store, kube, namespace)
		if err != nil {
			return "", err
		}
		token, err := p.TokenGetter.GetToken(ctx, config.APIDomain, credentialsRequest, caCert)
		if err != nil {
			return "", fmt.Errorf(errFailedToRetrieveToken, err)
		}
		return strings.TrimSpace(token), nil
	}
	if config.WorkloadIdentity != nil && config.WorkloadIdentity.ServiceAccountRef != nil && config.WorkloadIdentity.ServiceAccountRef.Name != "" && config.WorkloadIdentity.IAMServiceAccountID != "" {
		coreV1Client, err := p.getOrCreateCoreV1Client()
		if err != nil {
			return "", fmt.Errorf("initialize Kubernetes CoreV1 client: %w", err)
		}
		credentialsRequest, err := auth.NewFederatedAccountCredentialsRequest(
			*config.WorkloadIdentity.ServiceAccountRef,
			config.WorkloadIdentity.IAMServiceAccountID,
			store,
			coreV1Client,
			namespace,
		)
		if err != nil {
			return "", err
		}
		token, err := p.TokenGetter.GetToken(ctx, config.APIDomain, credentialsRequest, caCert)
		if err != nil {
			return "", fmt.Errorf(errFailedToRetrieveToken, err)
		}
		return strings.TrimSpace(token), nil
	}

	return "", errors.New(errMissingAuthOptions)
}

// createOrGetMysteryboxClient initializes or retrieves a cached MysteryBox client for a specified API domain and certificate.
func (p *Provider) createOrGetMysteryboxClient(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
	// lazy initialization with a current flag value
	if err := p.initMysteryboxClientsCache(); err != nil {
		return nil, err
	}

	cacheKey := ClientCacheKey{
		APIDomain: apiDomain,
		CAHash:    HashBytes(caCertificate),
	}

	// lock to avoid race and connections leaks during client creation for the same key
	p.mysteryboxClientsCacheMutex.Lock()
	defer p.mysteryboxClientsCacheMutex.Unlock()

	if value, ok := p.mysteryboxClientsCache.Get(cacheKey); ok {
		p.Logger.V(1).Info("Reusing cached MysteryBox client", "apiDomain", apiDomain)
		return value.(mysterybox.Client), nil
	}
	p.Logger.Info("Creating a new MysteryBox client", "apiDomain", apiDomain)
	mysteryboxClient, err := p.NewMysteryboxClient(ctx, apiDomain, caCertificate)
	if err != nil {
		return nil, err
	}
	p.mysteryboxClientsCache.Add(cacheKey, mysteryboxClient)
	return mysteryboxClient, nil
}

// getCaCert retrieves and returns the CA certificate as a byte slice for the specified secret in the given namespace.
func (p *Provider) getCaCert(ctx context.Context, config *SecretsClientConfig, store esv1.GenericStore, kube client.Client, namespace string) ([]byte, error) {
	caCert, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		config.CACertificate,
	)
	if err != nil {
		return nil, err
	}
	return []byte(strings.TrimSpace(caCert)), nil
}

func parseConfig(store esv1.GenericStore) (*SecretsClientConfig, error) {
	nebiusMysteryboxProvider, err := getNebiusMysteryboxProvider(store)
	if err != nil {
		return nil, err
	}

	if nebiusMysteryboxProvider.APIDomain == "" {
		return nil, errors.New(errMissingAPIDomain)
	}

	var token *esmeta.SecretKeySelector
	if nebiusMysteryboxProvider.Auth.Token.Name != "" {
		token = &nebiusMysteryboxProvider.Auth.Token
	}
	var caCertificate *esmeta.SecretKeySelector
	if nebiusMysteryboxProvider.CAProvider != nil {
		caCertificate = &nebiusMysteryboxProvider.CAProvider.Certificate
	}
	var serviceAccountCreds *esmeta.SecretKeySelector
	if nebiusMysteryboxProvider.Auth.ServiceAccountCreds.Name != "" {
		serviceAccountCreds = &nebiusMysteryboxProvider.Auth.ServiceAccountCreds
	}
	return &SecretsClientConfig{
		APIDomain:           strings.TrimSpace(nebiusMysteryboxProvider.APIDomain),
		ServiceAccountCreds: serviceAccountCreds,
		Token:               token,
		CACertificate:       caCertificate,
		WorkloadIdentity:    nebiusMysteryboxProvider.Auth.WorkloadIdentity,
	}, nil
}

func newMysteryboxClient(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
	return mysterybox.NewNebiusMysteryboxClientGrpc(ctx, apiDomain, caCertificate)
}

func (p *Provider) initMysteryboxClientsCache() error {
	p.cacheInitMutex.Lock()
	defer p.cacheInitMutex.Unlock()

	if p.mysteryboxClientsCache != nil {
		return nil
	}

	var err error
	var cache *lru.Cache
	cache, err = lru.NewWithEvict(
		mysteryboxConnectionsCacheSize,
		func(key, _ any) {
			p.Logger.V(1).Info("Evicting a Nebius MysteryBox client", "apiDomain", key.(ClientCacheKey).APIDomain)

			// We intentionally do not call Close() on the evicted client here.
			// This avoids "dial is closed" errors for active
			// reconciliation loops that might still be using this client instance
			// at the moment of eviction.
			//
			// If this approach leads to resource leaks in the future, we should consider
			// implementing a reference counter to safely close the client only when
			// it's no longer used by any active session.
		},
	)
	if err == nil {
		p.mysteryboxClientsCache = cache
		return nil
	}
	return fmt.Errorf("init clients cache: %w", err)
}

func (p *Provider) initTokenGetter() error {
	p.tokenInitMutex.Lock()
	defer p.tokenInitMutex.Unlock()

	if p.TokenGetter != nil {
		return nil
	}

	var err error
	c := clock.RealClock{}
	tokenExchangerLogger := ctrl.Log.WithName("provider").WithName("nebius").WithName("iam").WithName("grpctokenexchanger")
	tokenExchangeObserveFunction := func(err error) {
		metrics.ObserveAPICall(constants.ProviderNebiusMysterybox, constants.CallNebiusMysteryboxAuth, err)
	}
	var tokenGetter TokenGetter
	tokenGetter, err = NewCachedTokenGetter(
		mysteryboxTokensCacheSize,
		iam.NewGrpcTokenExchanger(
			tokenExchangerLogger,
			tokenExchangeObserveFunction,
		), c)
	if err == nil {
		p.TokenGetter = tokenGetter
	}

	return err
}

func (p *Provider) getOrCreateCoreV1Client() (typedcorev1.CoreV1Interface, error) {
	p.coreV1InitMutex.Lock()
	defer p.coreV1InitMutex.Unlock()

	if p.coreV1Client != nil {
		return p.coreV1Client, nil
	}

	newCoreV1Client := p.NewCoreV1Client
	if newCoreV1Client == nil {
		newCoreV1Client = newCoreV1ClientFromConfig
	}
	coreV1Client, err := newCoreV1Client()
	if err != nil {
		return nil, err
	}
	p.coreV1Client = coreV1Client
	return p.coreV1Client, nil
}

func newCoreV1ClientFromConfig() (typedcorev1.CoreV1Interface, error) {
	restConfig, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1(), nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{
		Logger:              log,
		NewMysteryboxClient: newMysteryboxClient,
		NewCoreV1Client:     newCoreV1ClientFromConfig,
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		NebiusMysterybox: &esv1.NebiusMysteryboxProvider{},
	}
}

func init() {
	fs := pflag.NewFlagSet("nebius", pflag.ExitOnError)
	fs.IntVar(
		&mysteryboxTokensCacheSize,
		"mysterybox-tokens-cache-size",
		defaultTokenCacheSize,
		"Size of Nebius MysteryBox token cache. "+
			"External secrets will reuse the Nebius IAM token without requesting a new one on each request.",
	)
	fs.IntVar(
		&mysteryboxConnectionsCacheSize,
		"mysterybox-connections-cache-size",
		defaultMysteryboxConnectionsCacheSize,
		"Size of Nebius MysteryBox grpc clients cache. External secrets will reuse the "+
			"connection to MysteryBox for the configuration without opening a new one on each request.",
	)
	feature.Register(feature.Feature{
		Flags: fs,
		Initialize: func() {
			if mysteryboxTokensCacheSize <= 0 {
				log.Error(nil, "invalid token cache size, use default",
					"got", mysteryboxTokensCacheSize,
					"default", defaultTokenCacheSize,
				)
				mysteryboxTokensCacheSize = defaultTokenCacheSize
			}
			if mysteryboxConnectionsCacheSize <= 0 {
				log.Error(nil, "invalid connections cache size, use default",
					"got", mysteryboxConnectionsCacheSize,
					"default", defaultMysteryboxConnectionsCacheSize,
				)
				mysteryboxConnectionsCacheSize = defaultMysteryboxConnectionsCacheSize
			}
			log.Info(
				"Registered Nebius MysteryBox provider",
				"token cache size", mysteryboxTokensCacheSize,
				"clients cache size", mysteryboxConnectionsCacheSize,
			)
		},
	})
}
