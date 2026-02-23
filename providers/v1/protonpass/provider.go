/*
Copyright © 2026 ESO Maintainer Team

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

// Package protonpass implements a provider for Proton Pass using the pass-cli binary.
// It allows fetching secrets stored in Proton Pass vaults.
package protonpass

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/cache"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/feature"
)

var (
	enableCache bool
	logger      = ctrl.Log.WithName("provider").WithName("protonpass")
	clientCache *cache.Cache[*provider]

	// cliRegistry keeps a single CLI instance per session directory so
	// that concurrent reconcilers for the same store share one mutex
	// and one on-disk session instead of racing against each other.
	cliRegistry   = make(map[string]*cli)
	cliRegistryMu sync.Mutex
)

const (
	defaultCacheSize = 100
	sessionBaseDir   = "/tmp/protonpass-sessions"
)

const (
	errProtonPassStore                          = "received invalid ProtonPass SecretStore resource: %w"
	errProtonPassStoreNilSpec                   = "nil spec"
	errProtonPassStoreNilSpecProvider           = "nil spec.provider"
	errProtonPassStoreNilSpecProviderProtonPass = "nil spec.provider.protonpass"
	errProtonPassStoreNilAuth                   = "nil spec.provider.protonpass.auth"
	errProtonPassStoreMissingUsername           = "missing: spec.provider.protonpass.username"
	errProtonPassStoreMissingPasswordRefName    = "missing: spec.provider.protonpass.auth.secretRef.password.name"
	errProtonPassStoreMissingPasswordRefKey     = "missing: spec.provider.protonpass.auth.secretRef.password.key"
)

// provider implements the External Secrets provider interface for Proton Pass.
type provider struct {
	cli     passCLI
	vault   string
	homeDir string
}

// NewClient constructs a new secrets client based on the provided store.
func (p *provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	key := cache.Key{
		Name:      store.GetObjectMeta().Name,
		Namespace: store.GetObjectMeta().Namespace,
		Kind:      store.GetTypeMeta().Kind,
	}

	if enableCache {
		cached, ok := clientCache.Get(store.GetObjectMeta().ResourceVersion, key)
		if ok {
			logger.V(1).Info("reusing cached ProtonPass session", "key", key)
			return cached, nil
		}
	}

	config := store.GetSpec().Provider.ProtonPass

	// Resolve password
	password, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		&config.Auth.SecretRef.Password,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve password: %w", err)
	}

	// Resolve optional TOTP secret
	var totpSecret string
	if config.Auth.SecretRef.TOTP != nil {
		totpSecret, err = resolvers.SecretKeyRef(
			ctx,
			kube,
			store.GetKind(),
			namespace,
			config.Auth.SecretRef.TOTP,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve TOTP secret: %w", err)
		}
	}

	// Resolve optional extra password
	var extraPassword string
	if config.Auth.SecretRef.ExtraPassword != nil {
		extraPassword, err = resolvers.SecretKeyRef(
			ctx,
			kube,
			store.GetKind(),
			namespace,
			config.Auth.SecretRef.ExtraPassword,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve extra password: %w", err)
		}
	}

	// Use a deterministic directory per store so that concurrent
	// reconciliations for the same store share a single CLI instance
	// and on-disk session, avoiding TOTP conflicts.
	homeDir := sessionHomeDir(key)
	if err := os.MkdirAll(homeDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session home dir: %w", err)
	}

	cliRegistryMu.Lock()
	cli, exists := cliRegistry[homeDir]
	if !exists {
		cli = newCLI(config.Username, password, totpSecret, extraPassword, config.Vault, homeDir)
		cliRegistry[homeDir] = cli
	}
	cliRegistryMu.Unlock()

	// Clear the item cache so each reconciliation sees fresh data from
	// Proton Pass. The login session (the expensive part) is preserved.
	cli.InvalidateCache()

	if err := cli.ensureLoggedIn(ctx); err != nil {
		return nil, fmt.Errorf("failed to login to Proton Pass: %w", err)
	}

	provider := &provider{
		cli:     cli,
		vault:   config.Vault,
		homeDir: homeDir,
	}

	if enableCache {
		clientCache.Add(store.GetObjectMeta().ResourceVersion, key, provider)
	}

	return provider, nil
}

func sessionHomeDir(key cache.Key) string {
	h := sha256.Sum256([]byte(key.Name + "/" + key.Namespace + "/" + key.Kind))
	return filepath.Join(sessionBaseDir, hex.EncodeToString(h[:8]))
}

func initCache(size int) {
	logger.Info("initializing protonpass session cache", "size", size)
	clientCache = cache.Must(size, func(p *provider) {
		if err := p.cli.Logout(context.Background()); err != nil {
			logger.Error(err, "unable to logout cached ProtonPass session on eviction")
		}
		if p.homeDir != "" {
			cliRegistryMu.Lock()
			delete(cliRegistry, p.homeDir)
			cliRegistryMu.Unlock()

			if err := os.RemoveAll(p.homeDir); err != nil {
				logger.Error(err, "unable to remove session home dir on eviction", "dir", p.homeDir)
			}
		}
	})
}

func init() {
	var sessionCacheSize int
	fs := pflag.NewFlagSet("protonpass", pflag.ExitOnError)
	fs.BoolVar(
		&enableCache,
		"experimental-enable-protonpass-session-cache",
		false,
		"Enable experimental ProtonPass session cache. External secrets will reuse the ProtonPass session without creating a new one on each request.",
	)
	fs.IntVar(
		&sessionCacheSize,
		"experimental-protonpass-session-cache-size",
		defaultCacheSize,
		"Maximum size of ProtonPass session cache. Only used if --experimental-enable-protonpass-session-cache is set.",
	)
	feature.Register(feature.Feature{
		Flags:      fs,
		Initialize: func() { initCache(sessionCacheSize) },
	})
}

// ValidateStore validates the Proton Pass SecretStore resource configuration.
func (p *provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return nil, fmt.Errorf(errProtonPassStore, errors.New(errProtonPassStoreNilSpec))
	}
	if storeSpec.Provider == nil {
		return nil, fmt.Errorf(errProtonPassStore, errors.New(errProtonPassStoreNilSpecProvider))
	}
	if storeSpec.Provider.ProtonPass == nil {
		return nil, fmt.Errorf(errProtonPassStore, errors.New(errProtonPassStoreNilSpecProviderProtonPass))
	}

	config := storeSpec.Provider.ProtonPass
	if config.Auth == nil {
		return nil, fmt.Errorf(errProtonPassStore, errors.New(errProtonPassStoreNilAuth))
	}
	if config.Username == "" {
		return nil, fmt.Errorf(errProtonPassStore, errors.New(errProtonPassStoreMissingUsername))
	}
	if config.Auth.SecretRef.Password.Name == "" {
		return nil, fmt.Errorf(errProtonPassStore, errors.New(errProtonPassStoreMissingPasswordRefName))
	}
	if config.Auth.SecretRef.Password.Key == "" {
		return nil, fmt.Errorf(errProtonPassStore, errors.New(errProtonPassStoreMissingPasswordRefKey))
	}

	// Validate secret selectors
	if err := esutils.ValidateSecretSelector(store, config.Auth.SecretRef.Password); err != nil {
		return nil, fmt.Errorf(errProtonPassStore, err)
	}
	if config.Auth.SecretRef.TOTP != nil {
		if err := esutils.ValidateSecretSelector(store, *config.Auth.SecretRef.TOTP); err != nil {
			return nil, fmt.Errorf(errProtonPassStore, err)
		}
	}
	if config.Auth.SecretRef.ExtraPassword != nil {
		if err := esutils.ValidateSecretSelector(store, *config.Auth.SecretRef.ExtraPassword); err != nil {
			return nil, fmt.Errorf(errProtonPassStore, err)
		}
	}

	return nil, nil
}

// Capabilities returns the provider supported capabilities (ReadOnly).
func (p *provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		ProtonPass: &esv1.ProtonPassProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
