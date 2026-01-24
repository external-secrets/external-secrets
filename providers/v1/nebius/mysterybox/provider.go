// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package mysterybox contains the logic to work with Nebius Mysterybox API.
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
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/iam"
	mysterybox2 "github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/mysterybox"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/feature"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

var (
	log                                   = ctrl.Log.WithName("provider").WithName("nebius").WithName("mysterybox")
	mysteryboxTokensCacheSize             int
	mysteryboxClientsCacheSize            int
	defaultTokenCacheSize                 = 2 << 11
	defaultMysteryboxConnectionsCacheSize = 2 << 6
)

// NewMysteryboxClient is a function that describes how to create a Nebius Mysterybox client to interact within.
type NewMysteryboxClient func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox2.Client, error)

// SecretsClientConfig holds configuration for interacting with.
type SecretsClientConfig struct {
	APIDomain           string
	ServiceAccountCreds *esmeta.SecretKeySelector
	Token               *esmeta.SecretKeySelector
	CACertificate       *esmeta.SecretKeySelector
}

// ClientCacheKey represents a unique key for identifying cached Mysterybox clients.
// It is composed of an API domain and a hash of the CA certificate.
type ClientCacheKey struct {
	APIDomain string
	CAHash    string
}

// Provider is a struct for managing Mysterybox clients.
type Provider struct {
	Logger                      logr.Logger
	NewMysteryboxClient         NewMysteryboxClient
	TokenService                TokenService
	mysteryboxClientsCache      *lru.Cache
	tokenOnce                   sync.Once
	cacheOnce                   sync.Once
	mysteryboxClientsCacheMutex sync.Mutex
}

// Capabilities returns the capabilities of the secret store, indicating it is read-only.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient creates and returns a new SecretsClient for the specified SecretStore and namespace context.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	// lazy initialization with a current flag value
	if p.TokenService == nil {
		if err := p.initTokenService(); err != nil {
			return nil, fmt.Errorf("init token service: %w", err)
		}
	}

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
	iamToken, err := p.getIamToken(ctx, clientConfig, store, kube, namespace, caCert)
	if err != nil {
		p.Logger.Info("Could not get IAM token", "store", store.GetNamespacedName(), "err", err)
		return nil, err
	}

	mysteryboxGrpcClient, err := p.createOrGetMysteryboxClient(ctx, clientConfig.APIDomain, caCert)
	if err != nil {
		p.Logger.Info("Could not create or get Mysterybox Client", "store", store.GetNamespacedName(), "err", err)
		return nil, err
	}

	return &SecretsClient{
		mysteryboxClient: mysteryboxGrpcClient,
		token:            iamToken,
	}, nil
}

// getIamToken retrieves an IAM token based on the provided SecretsClientConfig and authentication options.
// It supports token retrieval from a predefined secret or via service account credentials with the TokenService.
func (p *Provider) getIamToken(ctx context.Context, config *SecretsClientConfig, store esv1.GenericStore, kube client.Client, namespace string, caCert []byte) (string, error) {
	if config.Token.Name != "" {
		iamToken, err := resolvers.SecretKeyRef(
			ctx,
			kube,
			store.GetKind(),
			namespace,
			config.Token,
		)
		if err != nil {
			return "", fmt.Errorf("read token secret %s/%s: %w", namespace, config.Token.Name, err)
		}
		return strings.TrimSpace(iamToken), nil
	}
	if config.ServiceAccountCreds.Name != "" {
		subjectCreds, err := resolvers.SecretKeyRef(
			ctx,
			kube,
			store.GetKind(),
			namespace,
			config.ServiceAccountCreds,
		)
		if err != nil {
			return "", fmt.Errorf("read service account creds %s/%s: %w", namespace, config.ServiceAccountCreds.Name, err)
		}
		token, err := p.TokenService.GetToken(ctx, config.APIDomain, subjectCreds, caCert)
		if err != nil {
			return "", fmt.Errorf(errFailedToRetrieveToken, err)
		}
		return strings.TrimSpace(token), nil
	}

	return "", errors.New(errMissingAuthOptions)
}

// createOrGetMysteryboxClient initializes or retrieves a cached Mysterybox client for a specified API domain and certificate.
func (p *Provider) createOrGetMysteryboxClient(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox2.Client, error) {
	// lazy initialization with a current flag value
	if p.mysteryboxClientsCache == nil {
		err := p.initMysteryboxClientsCache()
		if err != nil {
			return nil, err
		}
	}

	cacheKey := ClientCacheKey{
		APIDomain: apiDomain,
		CAHash:    HashBytes(caCertificate),
	}

	// lock to avoid race and connections leaks during client creation for the same key
	p.mysteryboxClientsCacheMutex.Lock()
	defer p.mysteryboxClientsCacheMutex.Unlock()

	if value, ok := p.mysteryboxClientsCache.Get(cacheKey); ok {
		p.Logger.V(1).Info("Reusing cached Mysterybox client", "apiDomain", apiDomain)
		return value.(mysterybox2.Client), nil
	}
	p.Logger.Info("Creating a new Mysterybox client", "apiDomain", apiDomain)
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

	var caCertificate *esmeta.SecretKeySelector
	if nebiusMysteryboxProvider.CAProvider != nil {
		caCertificate = &nebiusMysteryboxProvider.CAProvider.Certificate
	}
	return &SecretsClientConfig{
		APIDomain:           strings.TrimSpace(nebiusMysteryboxProvider.APIDomain),
		ServiceAccountCreds: &nebiusMysteryboxProvider.Auth.ServiceAccountCreds,
		Token:               &nebiusMysteryboxProvider.Auth.Token,
		CACertificate:       caCertificate,
	}, nil
}

func newMysteryboxClient(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox2.Client, error) {
	return mysterybox2.NewNebiusMysteryboxClientGrpc(ctx, apiDomain, caCertificate)
}

func (p *Provider) initMysteryboxClientsCache() error {
	var err error
	p.cacheOnce.Do(func() {
		var cache *lru.Cache
		cache, err = lru.NewWithEvict(
			mysteryboxClientsCacheSize,
			func(key, value interface{}) {
				p.Logger.V(1).Info("Evicting a Nebius Mysterybox client", "apiDomain", key.(ClientCacheKey).APIDomain)
				err := value.(mysterybox2.Client).Close()
				if err != nil {
					p.Logger.Error(err, "Failed to close Nebius Mysterybox client")
				}
			})
		if err == nil {
			p.mysteryboxClientsCache = cache
		}
	})
	return err
}

func (p *Provider) initTokenService() error {
	var err error
	p.tokenOnce.Do(func() {
		c := clock.RealClock{}
		tokenExchangerLogger := ctrl.Log.WithName("provider").WithName("nebius").WithName("iam").WithName("grpctokenexchanger")
		tokenExchangeObserveFunction := func(err error) {
			metrics.ObserveAPICall(constants.ProviderNebiusMysterybox, constants.CallNebiusMysteryboxAuth, err)
		}
		var tokenService TokenService
		tokenService, err = NewTokenCacheService(
			mysteryboxTokensCacheSize,
			iam.NewGrpcTokenExchangerClient(
				tokenExchangerLogger,
				tokenExchangeObserveFunction,
			), c)
		if err == nil {
			p.TokenService = tokenService
		}
	})
	return err
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{
		Logger:              log,
		NewMysteryboxClient: newMysteryboxClient,
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
		"Size of Nebius Mysterybox token cache. "+
			"External secrets will reuse the Nebius IAM token without requesting a new one on each request.",
	)
	fs.IntVar(
		&mysteryboxClientsCacheSize,
		"mysterybox-connections-cache-size",
		defaultMysteryboxConnectionsCacheSize,
		"Size of Nebius Mysterybox grpc clients cache. External secrets will reuse the "+
			"connection to mysterybox for the configuration without opening a new one on each request.",
	)
	feature.Register(feature.Feature{
		Flags: fs,
		Initialize: func() {
			log.Info(
				"Registered Nebius Mysterybox provider",
				"token cache size", mysteryboxTokensCacheSize,
				"clients cache size", mysteryboxClientsCacheSize,
			)
		},
	})
}
