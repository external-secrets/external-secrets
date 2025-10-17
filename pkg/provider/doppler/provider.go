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

package doppler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/cache"
	"github.com/external-secrets/external-secrets/pkg/esutils"
	"github.com/external-secrets/external-secrets/pkg/feature"
	dclient "github.com/external-secrets/external-secrets/pkg/provider/doppler/client"
)

const (
	errNewClient    = "unable to create DopplerClient : %s"
	errInvalidStore = "invalid store: %s"
	errDopplerStore = "missing or invalid Doppler SecretStore"
)

// Provider is a Doppler secrets provider implementing NewClient and ValidateStore for the esv1.Provider interface.
type Provider struct{}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Client{}
var _ esv1.Provider = &Provider{}

var (
	oidcClientCache  *cache.Cache[esv1.SecretsClient]
	defaultCacheSize = 2 << 17
)

func initCache(cacheSize int) {
	if oidcClientCache == nil && cacheSize > 0 {
		oidcClientCache = cache.Must(cacheSize, func(client esv1.SecretsClient) {})
	}
}

func init() {
	var dopplerOIDCCacheSize int
	fs := pflag.NewFlagSet("doppler", pflag.ExitOnError)
	fs.IntVar(&dopplerOIDCCacheSize, "doppler-oidc-cache-size", defaultCacheSize,
		"Maximum size of Doppler OIDC provider cache. Set to 0 to disable caching.")

	feature.Register(feature.Feature{
		Flags: fs,
		Initialize: func() {
			initCache(dopplerOIDCCacheSize)
		},
	})

	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		Doppler: &esv1.DopplerProvider{},
	}, esv1.MaintenanceStatusMaintained)
}

// Capabilities returns the provider's supported capabilities.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient creates a new Doppler client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Doppler == nil {
		return nil, errors.New(errDopplerStore)
	}

	dopplerStoreSpec := storeSpec.Provider.Doppler

	useCache := dopplerStoreSpec.Auth.OIDCConfig != nil && oidcClientCache != nil

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

	client := &Client{
		kube:      kube,
		store:     dopplerStoreSpec,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	if dopplerStoreSpec.Auth.SecretRef != nil {
		if dopplerStoreSpec.Auth.SecretRef.DopplerToken.Key == "" {
			dopplerStoreSpec.Auth.SecretRef.DopplerToken.Key = "dopplerToken"
		}
	} else if dopplerStoreSpec.Auth.OIDCConfig != nil {
		cfg, err := config.GetConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
		}

		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
		}

		client.corev1 = clientset.CoreV1()
		client.oidcManager = NewOIDCTokenManager(
			client.corev1,
			dopplerStoreSpec,
			namespace,
			store.GetObjectKind().GroupVersionKind().Kind,
			store.GetObjectMeta().Name,
		)
	}

	if err := client.setAuth(ctx); err != nil {
		return nil, err
	}

	doppler, err := dclient.NewDopplerClient(client.dopplerToken)
	if err != nil {
		return nil, fmt.Errorf(errNewClient, err)
	}

	if customBaseURL, found := os.LookupEnv(customBaseURLEnvVar); found {
		if err := doppler.SetBaseURL(customBaseURL); err != nil {
			return nil, fmt.Errorf(errNewClient, err)
		}
	}

	if customVerifyTLS, found := os.LookupEnv(verifyTLSOverrideEnvVar); found {
		customVerifyTLS, err := strconv.ParseBool(customVerifyTLS)
		if err == nil {
			doppler.VerifyTLS = customVerifyTLS
		}
	}

	client.doppler = doppler
	client.project = client.store.Project
	client.config = client.store.Config
	client.nameTransformer = client.store.NameTransformer
	client.format = client.store.Format

	if useCache {
		oidcClientCache.Add(store.GetObjectMeta().ResourceVersion, key, client)
	}

	return client, nil
}

// ValidateStore validates the Doppler provider configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	dopplerStoreSpec := storeSpec.Provider.Doppler

	if dopplerStoreSpec.Auth.SecretRef != nil {
		dopplerTokenSecretRef := dopplerStoreSpec.Auth.SecretRef.DopplerToken
		if err := esutils.ValidateSecretSelector(store, dopplerTokenSecretRef); err != nil {
			return nil, fmt.Errorf(errInvalidStore, err)
		}

		if dopplerTokenSecretRef.Name == "" {
			return nil, fmt.Errorf(errInvalidStore, "dopplerToken.name cannot be empty")
		}
	} else if dopplerStoreSpec.Auth.OIDCConfig != nil {
		oidcAuth := dopplerStoreSpec.Auth.OIDCConfig

		if oidcAuth.Identity == "" {
			return nil, fmt.Errorf(errInvalidStore, "oidcConfig.identity cannot be empty")
		}

		if oidcAuth.ServiceAccountRef.Name == "" {
			return nil, fmt.Errorf(errInvalidStore, "oidcConfig.serviceAccountRef.name cannot be empty")
		}

		if err := esutils.ValidateServiceAccountSelector(store, oidcAuth.ServiceAccountRef); err != nil {
			return nil, fmt.Errorf(errInvalidStore, err)
		}
	} else {
		return nil, fmt.Errorf(errInvalidStore, "either auth.secretRef or auth.oidcConfig must be specified")
	}

	return nil, nil
}
