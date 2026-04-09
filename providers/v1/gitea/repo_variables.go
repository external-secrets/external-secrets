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

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// repoGetVariableFn fetches a single repository-level Actions variable by name using the SDK.
func (g *Client) repoGetVariableFn(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (string, error) {
	owner := g.provider.Organization
	variable, resp, err := g.baseClient.GetRepoActionVariable(owner, g.provider.Repository, ref.Key)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return "", fmt.Errorf("variable %q not found in repo %s/%s", ref.Key, owner, g.provider.Repository)
		}
		return "", fmt.Errorf("failed to get repo variable: %w", err)
	}
	return variable.Value, nil
}

// repoListVariablesFn lists all repository-level Actions variables.
// The SDK has no list method for repo variables, so we use direct HTTP (shared helper from org_variables.go).
func (g *Client) repoListVariablesFn(ctx context.Context) (map[string][]byte, error) {
	token, err := g.getToken(ctx)
	if err != nil {
		return nil, err
	}
	return listVariablesHTTP(ctx, g.provider.URL, token,
		fmt.Sprintf("/api/v1/repos/%s/%s/actions/variables", g.provider.Organization, g.provider.Repository))
}
