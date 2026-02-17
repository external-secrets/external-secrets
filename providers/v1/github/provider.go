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

// Package github implements a provider for GitHub secrets, allowing
// External Secrets to write secrets to GitHub Actions.
package github

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/provider"
)

const (
	errUnexpectedStoreSpec = "unexpected store spec"
	errInvalidStoreSpec    = "invalid store spec"
	errInvalidStoreProv    = "invalid store provider"
	errInvalidGithubProv   = "invalid github provider"
	errInvalidStore        = "invalid store"
)

// Provider implements the GitHub provider for managing secrets through GitHub Actions.
type Provider struct {
}

var _ provider.Provider = &Provider{}

// Metadata returns the provider metadata.
func (p *Provider) Metadata() provider.Metadata {
	return Metadata()
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
	g := &Client{
		crClient:  kube,
		store:     store,
		namespace: namespace,
		provider:  provider,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}
	g.getSecretFn = g.orgGetSecretFn
	g.getPublicKeyFn = g.orgGetPublicKeyFn
	g.createOrUpdateFn = g.orgCreateOrUpdateSecret
	g.listSecretsFn = g.orgListSecretsFn
	g.deleteSecretFn = g.orgDeleteSecretsFn
	ghClient, err := g.AuthWithPrivateKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get private key: %w", err)
	}
	g.baseClient = *ghClient.Actions
	if provider.Repository != "" {
		g.getSecretFn = g.repoGetSecretFn
		g.getPublicKeyFn = g.repoGetPublicKeyFn
		g.createOrUpdateFn = g.repoCreateOrUpdateSecret
		g.listSecretsFn = g.repoListSecretsFn
		g.deleteSecretFn = g.repoDeleteSecretsFn
		if provider.Environment != "" {
			// For environment to work, we need the repository ID instead of its name.
			repo, _, err := ghClient.Repositories.Get(ctx, g.provider.Organization, g.provider.Repository)
			if err != nil {
				return nil, fmt.Errorf("error fetching repository: %w", err)
			}
			g.repoID = repo.GetID()
			g.getSecretFn = g.envGetSecretFn
			g.getPublicKeyFn = g.envGetPublicKeyFn
			g.createOrUpdateFn = g.envCreateOrUpdateSecret
			g.listSecretsFn = g.envListSecretsFn
			g.deleteSecretFn = g.envDeleteSecretsFn
		}
	}

	return g, nil
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

	return nil, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() provider.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Github: &esv1.GithubProvider{},
	}
}
