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
	"fmt"
	"net/http"
	"strings"

	giteasdk "code.gitea.io/sdk/gitea"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// orgCreateOrUpdateSecret creates or updates an organization Actions secret.
// The Gitea SDK embeds the secret name inside CreateSecretOption.
func (g *Client) orgCreateOrUpdateSecret(_ context.Context, name, value string) error {
	_, err := g.baseClient.CreateOrgActionSecret(g.provider.Organization, giteasdk.CreateSecretOption{
		Name: name,
		Data: value,
	})
	if err != nil {
		return fmt.Errorf("CreateOrgActionSecret failed: %w", err)
	}
	return nil
}

// orgListSecretsFn lists all Actions secrets for the organization, paginating through all pages.
func (g *Client) orgListSecretsFn(_ context.Context) ([]*giteasdk.Secret, error) {
	var all []*giteasdk.Secret
	opts := giteasdk.ListOrgActionSecretOption{ListOptions: giteasdk.ListOptions{Page: 1, PageSize: 50}}
	for {
		page, resp, err := g.baseClient.ListOrgActionSecret(g.provider.Organization, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if resp.LastPage <= opts.Page {
			break
		}
		opts.Page++
	}
	return all, nil
}

// orgDeleteSecretsFn deletes an organization Actions secret.
// The Gitea SDK v0.20.0 does not expose DeleteOrgActionSecret, so we call the
// Gitea REST API directly using the token stored in the provider configuration.
// Endpoint: DELETE /api/v1/orgs/{org}/actions/secrets/{secretname}.
func (g *Client) orgDeleteSecretsFn(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	baseURL := strings.TrimRight(g.provider.URL, "/")
	apiURL := fmt.Sprintf("%s/api/v1/orgs/%s/actions/secrets/%s", baseURL, g.provider.Organization, remoteRef.GetRemoteKey())

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, apiURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to build delete request: %w", err)
	}

	// Retrieve the PAT token from the secret referenced in the provider.
	// We re-read it here because the client was already constructed; the token
	// itself does not change during the request lifetime.
	token, err := g.getToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token for delete: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE org action secret request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code deleting org action secret: %d", resp.StatusCode)
	}
	return nil
}

func (g *Client) orgGetSecretFn(ctx context.Context, ref esv1.PushSecretRemoteRef) (*giteasdk.Secret, error) {
	secrets, err := g.orgListSecretsFn(ctx)
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
