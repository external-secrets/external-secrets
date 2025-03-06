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

func (g *Client) envGetSecretFn(ctx context.Context, ref esv1beta1.PushSecretRemoteRef) (*github.Secret, *github.Response, error) {
	return g.baseClient.GetEnvSecret(ctx, int(g.repoID), g.provider.Environment, ref.GetRemoteKey())
}

func (g *Client) envGetPublicKeyFn(ctx context.Context) (*github.PublicKey, *github.Response, error) {
	return g.baseClient.GetEnvPublicKey(ctx, int(g.repoID), g.provider.Environment)
}

func (g *Client) envCreateOrUpdateSecret(ctx context.Context, encryptedSecret *github.EncryptedSecret) (*github.Response, error) {
	return g.baseClient.CreateOrUpdateEnvSecret(ctx, int(g.repoID), g.provider.Environment, encryptedSecret)
}

func (g *Client) envListSecretsFn(ctx context.Context) (*github.Secrets, *github.Response, error) {
	return g.baseClient.ListEnvSecrets(ctx, int(g.repoID), g.provider.Environment, &github.ListOptions{})
}

func (g *Client) envDeleteSecretsFn(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) (*github.Response, error) {
	return g.baseClient.DeleteEnvSecret(ctx, int(g.repoID), g.provider.Environment, remoteRef.GetRemoteKey())
}
