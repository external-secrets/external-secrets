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

package pulumi

import (
	"context"
	"errors"
	"fmt"
	"sync"

	esc "github.com/pulumi/esc-sdk/sdk/go"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/cache"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/feature"
)

// Provider implements the esv1.Provider interface for Pulumi ESC.
type Provider struct{}

var _ esv1.Provider = &Provider{}

var (
	oidcClientCache  *cache.Cache[esv1.SecretsClient]
	defaultCacheSize = 2 << 17
	cacheOnce        sync.Once
)

func initCache(cacheSize int) {
	if cacheSize > 0 {
		cacheOnce.Do(func() {
			oidcClientCache = cache.Must(cacheSize, func(_ esv1.SecretsClient) {
				// No cleanup is needed when evicting OIDC clients from cache
			})
		})
	}
}

// InitializeFlags registers Pulumi-specific flags with the feature system.
func InitializeFlags() *feature.Feature {
	var pulumiOIDCCacheSize int
	fs := pflag.NewFlagSet("pulumi", pflag.ExitOnError)
	fs.IntVar(&pulumiOIDCCacheSize, "pulumi-oidc-cache-size", defaultCacheSize,
		"Maximum size of Pulumi OIDC provider cache. Set to 0 to disable caching.")

	return &feature.Feature{
		Flags: fs,
		Initialize: func() {
			initCache(pulumiOIDCCacheSize)
		},
	}
}

const (
	errClusterStoreRequiresNamespace = "cluster store requires namespace"
	errCannotResolveSecretKeyRef     = "cannot resolve secret key ref: %w"
	errStoreIsNil                    = "store is nil"
	errNoStoreTypeOrWrongStoreType   = "no store type or wrong store type"
	errOrganizationIsRequired        = "organization is required"
	errEnvironmentIsRequired         = "environment is required"
	errProjectIsRequired             = "project is required"
	errSecretRefNameIsRequired       = "secretRef.name is required"
	errSecretRefKeyIsRequired        = "secretRef.key is required"
)

// NewClient creates a new Pulumi ESC client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	cfg, err := getConfig(store)
	if err != nil {
		return nil, err
	}
	storeKind := store.GetKind()
	if storeKind == esv1.ClusterSecretStoreKind && doesConfigDependOnNamespace(cfg) {
		return nil, errors.New(errClusterStoreRequiresNamespace)
	}

	// Check if we should use cache
	useCache := cfg.Auth != nil && cfg.Auth.OIDCConfig != nil && oidcClientCache != nil

	key := cache.Key{
		Name:      store.GetObjectMeta().Name,
		Namespace: namespace,
		Kind:      store.GetTypeMeta().Kind,
	}

	if useCache {
		if cachedClient, ok := oidcClientCache.Get(store.GetObjectMeta().ResourceVersion, key); ok {
			return cachedClient, nil
		}
	}

	accessToken, oidcManager, err := p.resolveAuthentication(ctx, cfg, store, kube, storeKind, namespace)
	if err != nil {
		return nil, err
	}

	client := p.createClient(cfg, accessToken, oidcManager)

	if useCache {
		oidcClientCache.Add(store.GetObjectMeta().ResourceVersion, key, client)
	}

	return client, nil
}

// resolveAuthentication determines the authentication method and returns the access token and optional OIDC manager.
func (p *Provider) resolveAuthentication(ctx context.Context, cfg *esv1.PulumiProvider, store esv1.GenericStore, kube kclient.Client, storeKind, namespace string) (string, *OIDCTokenManager, error) {
	// New auth structure with access token
	if cfg.Auth != nil && cfg.Auth.AccessToken != nil {
		token, err := loadAccessTokenSecret(ctx, cfg.Auth.AccessToken, kube, storeKind, namespace)
		return token, nil, err
	}

	// New auth structure with OIDC
	if cfg.Auth != nil && cfg.Auth.OIDCConfig != nil {
		return p.resolveOIDCAuthentication(ctx, cfg, store, namespace)
	}

	// Deprecated AccessToken field
	if cfg.AccessToken != nil {
		token, err := loadAccessTokenSecret(ctx, cfg.AccessToken, kube, storeKind, namespace)
		return token, nil, err
	}

	return "", nil, errors.New("no authentication method configured: either auth.accessToken, auth.oidcConfig, or accessToken must be specified")
}

// resolveOIDCAuthentication sets up OIDC authentication and returns the token and manager.
func (p *Provider) resolveOIDCAuthentication(ctx context.Context, cfg *esv1.PulumiProvider, store esv1.GenericStore, namespace string) (string, *OIDCTokenManager, error) {
	oidcManager, err := p.setupOIDCAuth(cfg, store, namespace)
	if err != nil {
		return "", nil, err
	}
	token, err := oidcManager.GetToken(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get OIDC token: %w", err)
	}
	return token, oidcManager, nil
}

// createClient creates a new Pulumi ESC client with the given configuration.
func (p *Provider) createClient(cfg *esv1.PulumiProvider, accessToken string, oidcManager *OIDCTokenManager) *client {
	configuration := esc.NewConfiguration()
	configuration.UserAgent = "external-secrets-operator"
	configuration.Servers = esc.ServerConfigurations{
		esc.ServerConfiguration{
			URL: cfg.APIURL,
		},
	}
	authCtx := esc.NewAuthContext(accessToken)
	escClient := esc.NewClient(configuration)

	return &client{
		escClient:    *escClient,
		authCtx:      authCtx,
		project:      cfg.Project,
		environment:  cfg.Environment,
		organization: cfg.Organization,
		oidcManager:  oidcManager,
		store:        cfg,
	}
}

func (p *Provider) setupOIDCAuth(cfg *esv1.PulumiProvider, store esv1.GenericStore, namespace string) (*OIDCTokenManager, error) {
	k8sConfig, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return NewOIDCTokenManager(
		clientset.CoreV1(),
		cfg,
		namespace,
		store.GetObjectKind().GroupVersionKind().Kind,
	), nil
}

func loadAccessTokenSecret(ctx context.Context, ref *esv1.PulumiProviderSecretRef, kube kclient.Client, storeKind, namespace string) (string, error) {
	acctoken, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, ref.SecretRef)
	if err != nil {
		return "", fmt.Errorf(errCannotResolveSecretKeyRef, err)
	}
	return acctoken, nil
}

func doesConfigDependOnNamespace(cfg *esv1.PulumiProvider) bool {
	// Check new auth structure
	if cfg.Auth != nil {
		if cfg.Auth.AccessToken != nil && cfg.Auth.AccessToken.SecretRef != nil && cfg.Auth.AccessToken.SecretRef.Namespace == nil {
			return true
		}
		// Check OIDC config
		if cfg.Auth.OIDCConfig != nil && cfg.Auth.OIDCConfig.ServiceAccountRef.Namespace == nil {
			return true
		}
	}
	// Check deprecated AccessToken field
	if cfg.AccessToken != nil && cfg.AccessToken.SecretRef != nil && cfg.AccessToken.SecretRef.Namespace == nil {
		return true
	}
	return false
}

func getConfig(store esv1.GenericStore) (*esv1.PulumiProvider, error) {
	if store == nil {
		return nil, errors.New(errStoreIsNil)
	}
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.Pulumi == nil {
		return nil, errors.New(errNoStoreTypeOrWrongStoreType)
	}
	cfg := spec.Provider.Pulumi

	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.pulumi.com/api/esc"
	}

	if cfg.Organization == "" {
		return nil, errors.New(errOrganizationIsRequired)
	}
	if cfg.Environment == "" {
		return nil, errors.New(errEnvironmentIsRequired)
	}
	if cfg.Project == "" {
		return nil, errors.New(errProjectIsRequired)
	}

	// Validate authentication configuration
	if err := validateAuth(store, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateAuth(store esv1.GenericStore, cfg *esv1.PulumiProvider) error {
	hasNewAccessToken := cfg.Auth != nil && cfg.Auth.AccessToken != nil
	hasOIDCConfig := cfg.Auth != nil && cfg.Auth.OIDCConfig != nil
	hasDeprecatedAuth := cfg.AccessToken != nil

	// Count how many auth methods are configured
	authMethodCount := 0
	if hasNewAccessToken {
		authMethodCount++
	}
	if hasOIDCConfig {
		authMethodCount++
	}
	if hasDeprecatedAuth {
		authMethodCount++
	}

	// Enforce mutual exclusivity
	if authMethodCount > 1 {
		return errors.New("only one authentication method may be configured: use either auth.accessToken, auth.oidcConfig, or the deprecated accessToken field")
	}

	if authMethodCount == 0 {
		return errors.New("no authentication method configured: either auth.accessToken, auth.oidcConfig, or accessToken must be specified")
	}

	// Validate the configured auth method
	if hasNewAccessToken {
		return validateStoreSecretRef(store, cfg.Auth.AccessToken)
	}
	if hasOIDCConfig {
		return validateOIDCConfig(cfg.Auth.OIDCConfig)
	}
	if hasDeprecatedAuth {
		return validateStoreSecretRef(store, cfg.AccessToken)
	}

	return nil
}

func validateOIDCConfig(oidcConfig *esv1.PulumiOIDCAuth) error {
	if oidcConfig.Organization == "" {
		return errors.New("oidcConfig.organization is required")
	}
	if oidcConfig.ServiceAccountRef.Name == "" {
		return errors.New("oidcConfig.serviceAccountRef.name is required")
	}
	return nil
}

func validateStoreSecretRef(store esv1.GenericStore, ref *esv1.PulumiProviderSecretRef) error {
	if ref != nil {
		if err := esutils.ValidateReferentSecretSelector(store, *ref.SecretRef); err != nil {
			return err
		}
	}
	return validateSecretRef(ref)
}

func validateSecretRef(ref *esv1.PulumiProviderSecretRef) error {
	if ref.SecretRef != nil {
		if ref.SecretRef.Name == "" {
			return errors.New(errSecretRefNameIsRequired)
		}
		if ref.SecretRef.Key == "" {
			return errors.New(errSecretRefKeyIsRequired)
		}
	}
	return nil
}

// ValidateStore validates the store's configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	_, err := getConfig(store)
	return nil, err
}

// Capabilities returns the provider's esv1.SecretStoreCapabilities.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Pulumi: &esv1.PulumiProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
