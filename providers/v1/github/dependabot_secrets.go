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

package github

import (
	"context"

	github "github.com/google/go-github/v56/github"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func adaptDependabotEncryptedSecret(secret *github.EncryptedSecret) *github.DependabotEncryptedSecret {
	return &github.DependabotEncryptedSecret{
		Name:                  secret.Name,
		KeyID:                 secret.KeyID,
		EncryptedValue:        secret.EncryptedValue,
		Visibility:            secret.Visibility,
		SelectedRepositoryIDs: github.DependabotSecretsSelectedRepoIDs(secret.SelectedRepositoryIDs),
	}
}

func (g *Client) dependabotOrgGetSecretFn(ctx context.Context, ref esv1.PushSecretRemoteRef) (*github.Secret, *github.Response, error) {
	return g.dependabotClient.GetOrgSecret(ctx, g.provider.Organization, ref.GetRemoteKey())
}

func (g *Client) dependabotOrgGetPublicKeyFn(ctx context.Context) (*github.PublicKey, *github.Response, error) {
	return g.dependabotClient.GetOrgPublicKey(ctx, g.provider.Organization)
}

func (g *Client) dependabotOrgCreateOrUpdateSecret(ctx context.Context, secret *github.EncryptedSecret) (*github.Response, error) {
	return g.dependabotClient.CreateOrUpdateOrgSecret(ctx, g.provider.Organization, adaptDependabotEncryptedSecret(secret))
}

func (g *Client) dependabotOrgListSecretsFn(ctx context.Context) (*github.Secrets, *github.Response, error) {
	return g.dependabotClient.ListOrgSecrets(ctx, g.provider.Organization, &github.ListOptions{})
}

func (g *Client) dependabotOrgDeleteSecretFn(ctx context.Context, ref esv1.PushSecretRemoteRef) (*github.Response, error) {
	return g.dependabotClient.DeleteOrgSecret(ctx, g.provider.Organization, ref.GetRemoteKey())
}

func (g *Client) dependabotOrgListSelectedRepoIDs(ctx context.Context, name string) (github.SelectedRepoIDs, error) {
	ids := github.SelectedRepoIDs{}
	opts := &github.ListOptions{PerPage: 100}
	for {
		repos, resp, err := g.dependabotClient.ListSelectedReposForOrgSecret(ctx, g.provider.Organization, name, opts)
		if err != nil {
			return nil, err
		}
		for _, repo := range repos.Repositories {
			ids = append(ids, repo.GetID())
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return ids, nil
}

func (g *Client) dependabotRepoGetSecretFn(ctx context.Context, ref esv1.PushSecretRemoteRef) (*github.Secret, *github.Response, error) {
	return g.dependabotClient.GetRepoSecret(ctx, g.provider.Organization, g.provider.Repository, ref.GetRemoteKey())
}

func (g *Client) dependabotRepoGetPublicKeyFn(ctx context.Context) (*github.PublicKey, *github.Response, error) {
	return g.dependabotClient.GetRepoPublicKey(ctx, g.provider.Organization, g.provider.Repository)
}

func (g *Client) dependabotRepoCreateOrUpdateSecret(ctx context.Context, secret *github.EncryptedSecret) (*github.Response, error) {
	return g.dependabotClient.CreateOrUpdateRepoSecret(ctx, g.provider.Organization, g.provider.Repository, adaptDependabotEncryptedSecret(secret))
}

func (g *Client) dependabotRepoListSecretsFn(ctx context.Context) (*github.Secrets, *github.Response, error) {
	return g.dependabotClient.ListRepoSecrets(ctx, g.provider.Organization, g.provider.Repository, &github.ListOptions{})
}

func (g *Client) dependabotRepoDeleteSecretFn(ctx context.Context, ref esv1.PushSecretRemoteRef) (*github.Response, error) {
	return g.dependabotClient.DeleteRepoSecret(ctx, g.provider.Organization, g.provider.Repository, ref.GetRemoteKey())
}
