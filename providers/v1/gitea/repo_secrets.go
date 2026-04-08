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

	giteasdk "code.gitea.io/sdk/gitea"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// repoCreateOrUpdateSecret creates or updates a repository Actions secret.
// The Gitea SDK embeds the secret name inside CreateSecretOption.
func (g *Client) repoCreateOrUpdateSecret(_ context.Context, name, value string) error {
	_, err := g.baseClient.CreateRepoActionSecret(g.provider.Organization, g.provider.Repository, giteasdk.CreateSecretOption{
		Name: name,
		Data: value,
	})
	return err
}

// repoListSecretsFn lists all Actions secrets for a repository.
// Note: the SDK method is ListRepoActionSecret (singular).
func (g *Client) repoListSecretsFn(_ context.Context) ([]*giteasdk.Secret, error) {
	secrets, _, err := g.baseClient.ListRepoActionSecret(g.provider.Organization, g.provider.Repository, giteasdk.ListRepoActionSecretOption{})
	return secrets, err
}

func (g *Client) repoDeleteSecretsFn(_ context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	_, err := g.baseClient.DeleteRepoActionSecret(g.provider.Organization, g.provider.Repository, remoteRef.GetRemoteKey())
	return err
}

func (g *Client) repoGetSecretFn(_ context.Context, ref esv1.PushSecretRemoteRef) (*giteasdk.Secret, error) {
	secrets, _, err := g.baseClient.ListRepoActionSecret(g.provider.Organization, g.provider.Repository, giteasdk.ListRepoActionSecretOption{})
	if err != nil {
		return nil, err
	}
	for _, s := range secrets {
		if s.Name == ref.GetRemoteKey() {
			return s, nil
		}
	}
	return nil, nil
}
