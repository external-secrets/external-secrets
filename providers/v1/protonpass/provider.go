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

package protonpass

import (
	"context"
	"errors"
	"fmt"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/cache"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// Provider is the Proton Pass External Secrets provider. It authenticates with a
// Personal Access Token and speaks directly to the Proton Pass HTTP API.
type Provider struct{}

var _ esv1.Provider = &Provider{}

// NewProvider returns a new Proton Pass provider.
func NewProvider() esv1.Provider { return &Provider{} }

// ProviderSpec returns the provider's slot on the SecretStore discriminator union.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{ProtonPass: &esv1.ProtonPassProvider{}}
}

// MaintenanceStatus reports whether the provider is actively maintained.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}

// Capabilities reports that the provider supports both reads and writes. Whether a
// given store can actually write is ultimately gated by the Personal Access Token's
// role (a viewer token yields read-only behavior even though the provider is RW).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// ValidateStore validates the static configuration of a Proton Pass store.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	spec, err := storeProvider(store)
	if err != nil {
		return nil, err
	}
	ref := spec.Auth.PersonalAccessTokenSecretRef
	if ref.Name == "" {
		return nil, errors.New("protonpass: auth.personalAccessTokenSecretRef.name is required")
	}
	if ref.Key == "" {
		return nil, errors.New("protonpass: auth.personalAccessTokenSecretRef.key is required")
	}
	// Enforce ClusterSecretStore vs SecretStore namespace scoping on the credential ref.
	if err := esutils.ValidateSecretSelector(store, ref); err != nil {
		return nil, err
	}
	return nil, nil
}

// NewClient constructs a SecretsClient for the given store: it resolves and parses
// the Personal Access Token and builds the Proton Pass API client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	spec, err := storeProvider(store)
	if err != nil {
		return nil, err
	}
	// Reuse this store's already-minted session when one is cached for the current
	// store version. Proton rate-limits logins per account, so minting once per
	// store (not once per reconcile/validation) is what keeps the provider usable
	// at more than a handful of resources. See sessionCache.
	key := cache.Key{
		Name:      store.GetObjectMeta().GetName(),
		Namespace: store.GetObjectMeta().GetNamespace(),
		Kind:      store.GetTypeMeta().Kind,
	}
	if api, ok := sessionCache.Get(store.GetObjectMeta().GetResourceVersion(), key); ok {
		return &client{api: api, vaults: spec.Vaults}, nil
	}
	patStr, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, &spec.Auth.PersonalAccessTokenSecretRef)
	if err != nil {
		return nil, fmt.Errorf("protonpass: resolve personal access token: %w", err)
	}
	pat, err := parsePAT(patStr)
	if err != nil {
		return nil, err
	}
	api := newAPIClient(pat, defaultHost)
	sessionCache.Add(store.GetObjectMeta().GetResourceVersion(), key, api)
	return &client{api: api, vaults: spec.Vaults}, nil
}

// storeProvider extracts the ProtonPass configuration from a generic store.
func storeProvider(store esv1.GenericStore) (*esv1.ProtonPassProvider, error) {
	if store == nil {
		return nil, errors.New("protonpass: nil store")
	}
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.ProtonPass == nil {
		return nil, errors.New("protonpass: missing or invalid ProtonPass store spec")
	}
	return spec.Provider.ProtonPass, nil
}
