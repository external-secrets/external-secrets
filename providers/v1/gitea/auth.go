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

// Package gitea implements the External Secrets provider for Gitea Actions secrets and variables.
package gitea

import (
	"context"
	"fmt"

	giteasdk "code.gitea.io/sdk/gitea"

	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// newGiteaClient creates an authenticated Gitea SDK client using the PAT token
// referenced in the provider's auth.secretRef.
func (g *Client) newGiteaClient(ctx context.Context) (*giteasdk.Client, error) {
	token, err := g.getToken(ctx)
	if err != nil {
		return nil, err
	}

	client, err := giteasdk.NewClient(g.provider.URL, giteasdk.SetToken(token))
	if err != nil {
		return nil, fmt.Errorf("failed to create gitea client: %w", err)
	}

	return client, nil
}

// getToken retrieves the PAT token from the Kubernetes secret referenced by the provider.
func (g *Client) getToken(ctx context.Context) (string, error) {
	token, err := resolvers.SecretKeyRef(ctx, g.crClient, g.storeKind, g.namespace, &g.provider.Auth.SecretRef)
	if err != nil {
		return "", fmt.Errorf("failed to get gitea token from secret: %w", err)
	}
	return token, nil
}
