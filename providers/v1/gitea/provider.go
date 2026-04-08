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

package gitea

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errUnexpectedStoreSpec = "unexpected store spec"
	errInvalidStoreSpec    = "invalid store spec"
	errInvalidStoreProv    = "invalid store provider"
	errInvalidStore        = "invalid store"
)

// Provider implements esv1.Provider for Gitea Actions secrets.
type Provider struct{}

var _ esv1.Provider = &Provider{}

// Capabilities returns the provider capabilities — read and write.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient creates a new Gitea secrets client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

func newClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	g := &Client{
		crClient:  kube,
		store:     store,
		namespace: namespace,
		provider:  provider,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}
	g.createOrUpdateFn = g.orgCreateOrUpdateSecret
	g.listSecretsFn = g.orgListSecretsFn
	g.deleteSecretFn = g.orgDeleteSecretsFn
	g.getSecretFn = g.orgGetSecretFn

	giteaClient, err := g.newGiteaClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create gitea client: %w", err)
	}
	g.baseClient = giteaClient

	// Wire variable (read) functions — default to org scope.
	g.getVariableFn = g.orgGetVariableFn
	g.listVariablesFn = g.orgListVariablesFn

	if provider.Repository != "" {
		g.createOrUpdateFn = g.repoCreateOrUpdateSecret
		g.listSecretsFn = g.repoListSecretsFn
		g.deleteSecretFn = g.repoDeleteSecretsFn
		g.getSecretFn = g.repoGetSecretFn
		g.getVariableFn = g.repoGetVariableFn
		g.listVariablesFn = g.repoListVariablesFn
	}

	return g, nil
}

func getProvider(store esv1.GenericStore) (*esv1.GiteaProvider, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.Gitea == nil {
		return nil, errors.New(errUnexpectedStoreSpec)
	}
	return spc.Provider.Gitea, nil
}

// ValidateStore validates the store spec.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, errors.New(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, errors.New(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return nil, errors.New(errInvalidStoreProv)
	}
	prov := spc.Provider.Gitea
	if prov == nil {
		return nil, errors.New("invalid gitea provider")
	}
	if prov.URL == "" {
		return nil, errors.New("gitea provider URL is required")
	}
	if prov.Auth.SecretRef.Name == "" {
		return nil, errors.New("gitea provider auth.secretRef.name is required")
	}
	if prov.Organization == "" {
		return nil, errors.New("gitea provider organization is required")
	}
	return nil, nil
}

// NewProvider returns a new Gitea Provider.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns a SecretStoreProvider configured for Gitea.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Gitea: &esv1.GiteaProvider{},
	}
}

// MaintenanceStatus returns the maintenance status for this provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
