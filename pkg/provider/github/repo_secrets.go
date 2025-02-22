// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package github

import (
	"context"

	github "github.com/google/go-github/v56/github"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

func (g *Client) repoGetSecretFn(ctx context.Context, ref esv1beta1.PushSecretRemoteRef) (*github.Secret, *github.Response, error) {
	return g.baseClient.GetRepoSecret(ctx, g.provider.Organization, g.provider.Repository, ref.GetRemoteKey())
}

func (g *Client) repoGetPublicKeyFn(ctx context.Context) (*github.PublicKey, *github.Response, error) {
	return g.baseClient.GetRepoPublicKey(ctx, g.provider.Organization, g.provider.Repository)
}

func (g *Client) repoCreateOrUpdateSecret(ctx context.Context, encryptedSecret *github.EncryptedSecret) (*github.Response, error) {
	return g.baseClient.CreateOrUpdateRepoSecret(ctx, g.provider.Organization, g.provider.Repository, encryptedSecret)
}

func (g *Client) repoListSecretsFn(ctx context.Context) (*github.Secrets, *github.Response, error) {
	return g.baseClient.ListRepoSecrets(ctx, g.provider.Organization, g.provider.Repository, &github.ListOptions{})
}

func (g *Client) repoDeleteSecretsFn(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) (*github.Response, error) {
	return g.baseClient.DeleteRepoSecret(ctx, g.provider.Organization, g.provider.Environment, remoteRef.GetRemoteKey())
}
