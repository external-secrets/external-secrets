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

package openbao

import (
	"context"
	"errors"
	"fmt"
	"time"

	bao "github.com/hashicorp/vault/api"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	baoutil "github.com/external-secrets/external-secrets/providers/v1/openbao/util"
	"github.com/external-secrets/external-secrets/runtime/cache"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/feature"
)

var (
	_           esv1.Provider = &Provider{}
	enableCache bool
	logger      = ctrl.Log.WithName("provider").WithName("openbao")
	clientCache *cache.Cache[baoutil.Client]
)

const (
	errOpenBaoStore  = "received invalid OpenBao SecretStore resource: %w"
	errOpenBaoClient = "cannot setup new OpenBao client: %w"
	errOpenBaoCert   = "cannot set OpenBao CA certificate: %w"
	errClientTLSAuth = "error from Client TLS Auth: %q"
	errCANamespace   = "missing namespace on caProvider secret"
)

const (
	defaultCacheSize = 2 << 17
)

// Provider implements the ESO Provider interface for OpenBao.
type Provider struct {
	// NewOpenBaoClient is a function that returns a new OpenBao client.
	// This is used for testing to inject a fake client.
	NewOpenBaoClient func(config *bao.Config) (baoutil.Client, error)
}

// NewOpenBaoClient returns a new OpenBao client.
func NewOpenBaoClient(config *bao.Config) (baoutil.Client, error) {
	baoClient, err := bao.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &baoutil.OpenBaoClient{
		SetTokenFunc:     baoClient.SetToken,
		TokenFunc:        baoClient.Token,
		ClearTokenFunc:   baoClient.ClearToken,
		AuthField:        baoClient.Auth(),
		AuthTokenField:   baoClient.Auth().Token(),
		LogicalField:     baoClient.Logical(),
		NamespaceFunc:    baoClient.Namespace,
		SetNamespaceFunc: baoClient.SetNamespace,
		AddHeaderFunc:    baoClient.AddHeader,
	}, nil
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient implements the Client interface.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	// controller-runtime/client does not support TokenRequest or other subresource APIs
	// so we need to construct our own client and use it to fetch tokens
	// (for Kubernetes service account token auth)
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	return p.newClient(ctx, store, kube, clientset.CoreV1(), namespace)
}

// NewGeneratorClient creates a new OpenBao client for the generator controller.
func (p *Provider) NewGeneratorClient(
	ctx context.Context,
	kube kclient.Client,
	corev1 typedcorev1.CoreV1Interface,
	baoSpec *esv1.OpenBaoProvider,
	namespace string,
	retrySettings *esv1.SecretStoreRetrySettings,
) (baoutil.Client, error) {
	vStore, cfg, err := p.prepareConfig(ctx, kube, corev1, baoSpec, retrySettings, namespace, resolvers.EmptyStoreKind)
	if err != nil {
		return nil, err
	}

	client, err := p.NewOpenBaoClient(cfg)
	if err != nil {
		return nil, err
	}

	_, err = p.initClient(ctx, vStore, client, cfg, baoSpec)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (p *Provider) newClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, corev1 typedcorev1.CoreV1Interface, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.OpenBao == nil {
		return nil, errors.New(errOpenBaoStore)
	}
	baoSpec := storeSpec.Provider.OpenBao

	vStore, cfg, err := p.prepareConfig(
		ctx,
		kube,
		corev1,
		baoSpec,
		storeSpec.RetrySettings,
		namespace,
		store.GetObjectKind().GroupVersionKind().Kind)
	if err != nil {
		return nil, err
	}

	client, err := getOpenBaoClient(p, store, cfg, namespace)
	if err != nil {
		return nil, fmt.Errorf(errOpenBaoClient, err)
	}

	return p.initClient(ctx, vStore, client, cfg, baoSpec)
}

func (p *Provider) initClient(ctx context.Context, c *client, client baoutil.Client, cfg *bao.Config, baoSpec *esv1.OpenBaoProvider) (esv1.SecretsClient, error) {
	if baoSpec.Namespace != nil {
		client.SetNamespace(*baoSpec.Namespace)
	}

	if baoSpec.Headers != nil {
		for hKey, hValue := range baoSpec.Headers {
			client.AddHeader(hKey, hValue)
		}
	}

	c.client = client
	c.auth = client.Auth()
	c.logical = client.Logical()
	c.token = client.AuthToken()

	// allow SecretStore controller validation to pass
	// when using referent namespace.
	if c.storeKind == esv1.ClusterSecretStoreKind && c.namespace == "" && isReferentSpec(baoSpec) {
		return c, nil
	}
	// set auth also sets the token expiry value
	if err := c.setAuth(ctx, cfg); err != nil {
		return nil, err
	}

	return c, nil
}

func (p *Provider) prepareConfig(
	ctx context.Context,
	kube kclient.Client,
	corev1 typedcorev1.CoreV1Interface,
	baoSpec *esv1.OpenBaoProvider,
	retrySettings *esv1.SecretStoreRetrySettings,
	namespace, storeKind string,
) (*client, *bao.Config, error) {
	c := &client{
		kube:      kube,
		corev1:    corev1,
		store:     baoSpec,
		log:       logger,
		namespace: namespace,
		storeKind: storeKind,
	}

	cfg, err := c.newConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Setup retry options if present
	if retrySettings != nil {
		if retrySettings.MaxRetries != nil {
			cfg.MaxRetries = int(*retrySettings.MaxRetries)
		} else {
			// By default we rely only on the reconciliation process for retrying
			cfg.MaxRetries = 0
		}

		if retrySettings.RetryInterval != nil {
			retryWait, err := time.ParseDuration(*retrySettings.RetryInterval)
			if err != nil {
				return nil, nil, err
			}
			cfg.MinRetryWait = retryWait
			cfg.MaxRetryWait = retryWait
		}
	}

	return c, cfg, nil
}

func getOpenBaoClient(p *Provider, store esv1.GenericStore, cfg *bao.Config, namespace string) (baoutil.Client, error) {
	baoProvider := store.GetSpec().Provider.OpenBao
	auth := baoProvider.Auth
	isStaticToken := auth != nil && auth.TokenSecretRef != nil
	useCache := enableCache && !isStaticToken

	keyNamespace := store.GetObjectMeta().Namespace
	// A single ClusterSecretStore may need to spawn separate OpenBao clients for each namespace.
	if store.GetTypeMeta().Kind == esv1.ClusterSecretStoreKind && namespace != "" && isReferentSpec(baoProvider) {
		keyNamespace = namespace
	}

	key := cache.Key{
		Name:      store.GetObjectMeta().Name,
		Namespace: keyNamespace,
		Kind:      store.GetTypeMeta().Kind,
	}
	if useCache {
		client, ok := clientCache.Get(store.GetObjectMeta().ResourceVersion, key)
		if ok {
			return client, nil
		}
	}

	client, err := p.NewOpenBaoClient(cfg)
	if err != nil {
		return nil, fmt.Errorf(errOpenBaoClient, err)
	}

	if useCache && !clientCache.Contains(key) {
		clientCache.Add(store.GetObjectMeta().ResourceVersion, key, client)
	}
	return client, nil
}

func isReferentSpec(prov *esv1.OpenBaoProvider) bool {
	if prov.Auth == nil {
		return false
	}

	if prov.Auth.TokenSecretRef != nil && prov.Auth.TokenSecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.AppRole != nil && prov.Auth.AppRole.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.Kubernetes != nil && prov.Auth.Kubernetes.SecretRef != nil && prov.Auth.Kubernetes.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.Kubernetes != nil && prov.Auth.Kubernetes.ServiceAccountRef != nil && prov.Auth.Kubernetes.ServiceAccountRef.Namespace == nil {
		return true
	}
	if prov.Auth.Ldap != nil && prov.Auth.Ldap.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.UserPass != nil && prov.Auth.UserPass.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.Jwt != nil && prov.Auth.Jwt.SecretRef != nil && prov.Auth.Jwt.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.Jwt != nil && prov.Auth.Jwt.ServiceAccountRef != nil && prov.Auth.Jwt.ServiceAccountRef.Namespace == nil {
		return true
	}
	if prov.Auth.Cert != nil && prov.Auth.Cert.SecretRef.Namespace == nil {
		return true
	}
	if prov.Auth.Iam != nil && prov.Auth.Iam.JWTAuth != nil && prov.Auth.Iam.JWTAuth.ServiceAccountRef != nil && prov.Auth.Iam.JWTAuth.ServiceAccountRef.Namespace == nil {
		return true
	}
	if prov.Auth.Iam != nil && prov.Auth.Iam.SecretRef != nil &&
		(prov.Auth.Iam.SecretRef.AccessKeyID.Namespace == nil ||
			prov.Auth.Iam.SecretRef.SecretAccessKey.Namespace == nil ||
			(prov.Auth.Iam.SecretRef.SessionToken != nil && prov.Auth.Iam.SecretRef.SessionToken.Namespace == nil)) {
		return true
	}
	return false
}

func initCache(size int) {
	logger.Info("initializing OpenBao cache", "size", size)
	clientCache = cache.Must(size, func(client baoutil.Client) {
		err := revokeTokenIfValid(context.Background(), client)
		if err != nil {
			logger.Error(err, "unable to revoke cached token on eviction")
		}
	})
}

func init() {
	var (
		baoTokenCacheSize int
	)

	fs := pflag.NewFlagSet("openbao", pflag.ExitOnError)
	fs.BoolVar(
		&enableCache,
		"enable-openbao-token-cache",
		false,
		"Enable OpenBao token cache. External secrets will reuse the OpenBao token without creating a new one on each request.",
	)
	// max. 265k OpenBao leases with 30bytes each ~= 7MB
	fs.IntVar(
		&baoTokenCacheSize,
		"openbao-token-cache-size",
		defaultCacheSize,
		"Maximum size of OpenBao token cache. Only used if --enable-openbao-token-cache is set.",
	)
	feature.Register(feature.Feature{
		Flags: fs,
		Initialize: func() {
			initCache(baoTokenCacheSize)
		},
	})
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{
		NewOpenBaoClient: NewOpenBaoClient,
	}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		OpenBao: &esv1.OpenBaoProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
