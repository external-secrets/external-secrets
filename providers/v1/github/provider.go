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

// Package github implements a provider for GitHub secrets, allowing
// External Secrets to write secrets to GitHub Actions or Dependabot.
package github

import (
	"context"
	"errors"
	"fmt"

	github "github.com/google/go-github/v56/github"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errUnexpectedStoreSpec = "unexpected store spec"
	errInvalidStoreSpec    = "invalid store spec"
	errInvalidStoreProv    = "invalid store provider"
	errInvalidGithubProv   = "invalid github provider"
	errInvalidStore        = "invalid store"
)

// Provider implements the GitHub provider for managing secrets through GitHub Actions or Dependabot.
type Provider struct {
}

var _ esv1.Provider = &Provider{}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreWriteOnly
}

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

func newClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	secretType, err := validateGithubProvider(provider)
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
	ghClient, err := g.AuthWithPrivateKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get private key: %w", err)
	}
	if err := g.configureSecretClient(ctx, ghClient, secretType); err != nil {
		return nil, err
	}

	return g, nil
}

func (g *Client) configureSecretClient(ctx context.Context, ghClient *github.Client, secretType esv1.GithubSecretType) error {
	if secretType == esv1.GithubSecretTypeDependabot {
		g.dependabotClient = *ghClient.Dependabot
		g.getSecretFn = g.dependabotOrgGetSecretFn
		g.getPublicKeyFn = g.dependabotOrgGetPublicKeyFn
		g.createOrUpdateFn = g.dependabotOrgCreateOrUpdateSecret
		g.listSecretsFn = g.dependabotOrgListSecretsFn
		g.deleteSecretFn = g.dependabotOrgDeleteSecretFn
		g.listSelectedReposFn = g.dependabotOrgListSelectedRepoIDs
		if g.provider.Repository != "" {
			g.getSecretFn = g.dependabotRepoGetSecretFn
			g.getPublicKeyFn = g.dependabotRepoGetPublicKeyFn
			g.createOrUpdateFn = g.dependabotRepoCreateOrUpdateSecret
			g.listSecretsFn = g.dependabotRepoListSecretsFn
			g.deleteSecretFn = g.dependabotRepoDeleteSecretFn
			g.listSelectedReposFn = nil
		}
		return nil
	}

	g.baseClient = *ghClient.Actions
	g.getSecretFn = g.orgGetSecretFn
	g.getPublicKeyFn = g.orgGetPublicKeyFn
	g.createOrUpdateFn = g.orgCreateOrUpdateSecret
	g.listSecretsFn = g.orgListSecretsFn
	g.deleteSecretFn = g.orgDeleteSecretsFn
	g.listSelectedReposFn = g.orgListSelectedRepoIDs
	if g.provider.Repository != "" {
		g.getSecretFn = g.repoGetSecretFn
		g.getPublicKeyFn = g.repoGetPublicKeyFn
		g.createOrUpdateFn = g.repoCreateOrUpdateSecret
		g.listSecretsFn = g.repoListSecretsFn
		g.deleteSecretFn = g.repoDeleteSecretsFn
		// Repo and env secrets have no "selected repositories" concept.
		g.listSelectedReposFn = nil
		if g.provider.Environment != "" {
			// For environment to work, we need the repository ID instead of its name.
			repo, _, err := ghClient.Repositories.Get(ctx, g.provider.Organization, g.provider.Repository)
			if err != nil {
				return fmt.Errorf("error fetching repository: %w", err)
			}
			g.repoID = repo.GetID()
			g.getSecretFn = g.envGetSecretFn
			g.getPublicKeyFn = g.envGetPublicKeyFn
			g.createOrUpdateFn = g.envCreateOrUpdateSecret
			g.listSecretsFn = g.envListSecretsFn
			g.deleteSecretFn = g.envDeleteSecretsFn
		}
	}
	return nil
}

func getProvider(store esv1.GenericStore) (*esv1.GithubProvider, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider.Github == nil {
		return nil, errors.New(errUnexpectedStoreSpec)
	}

	return spc.Provider.Github, nil
}

// ValidateStore validates the configuration of a GitHub secret store.
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
	prov := spc.Provider.Github
	if prov == nil {
		return nil, errors.New(errInvalidGithubProv)
	}
	if _, err := validateGithubProvider(prov); err != nil {
		return nil, err
	}

	return nil, nil
}

func validateGithubProvider(provider *esv1.GithubProvider) (esv1.GithubSecretType, error) {
	secretType := provider.SecretType
	if secretType == "" {
		secretType = esv1.GithubSecretTypeActions
	}
	if secretType != esv1.GithubSecretTypeActions && secretType != esv1.GithubSecretTypeDependabot {
		return "", fmt.Errorf("unsupported GitHub secret type %q", secretType)
	}
	if secretType == esv1.GithubSecretTypeDependabot && provider.Environment != "" {
		return "", errors.New("Dependabot secrets do not support environments")
	}
	return secretType, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Github: &esv1.GithubProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
