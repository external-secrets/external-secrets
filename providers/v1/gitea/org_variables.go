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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type giteaVariableResponse struct {
	Name  string `json:"name"`
	Value string `json:"data"`
}

type giteaVariablesListResponse struct {
	Data       []giteaVariableResponse `json:"data"`
	TotalCount int                     `json:"total_count"`
}

// httpClient is a package-level client used for direct Gitea REST calls.
// Using a shared client avoids spawning a new one per request.
var httpClient = &http.Client{}

// listVariablesHTTP is a shared helper that fetches variables from any Gitea API path.
// path must be the full path segment, e.g. "/api/v1/repos/{org}/{repo}/actions/variables".
func listVariablesHTTP(ctx context.Context, baseURL, token, path string) (map[string][]byte, error) {
	url := strings.TrimRight(baseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to build list request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read variables list response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d listing variables: %s", resp.StatusCode, body)
	}
	var list giteaVariablesListResponse
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("failed to decode variables list: %w", err)
	}
	out := make(map[string][]byte, len(list.Data))
	for _, v := range list.Data {
		out[v.Name] = []byte(v.Value)
	}
	return out, nil
}

// orgGetVariableFn fetches a single org-level Actions variable by name.
func (g *Client) orgGetVariableFn(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (string, error) {
	token, err := g.getToken(ctx)
	if err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("%s/api/v1/orgs/%s/actions/variables/%s",
		strings.TrimRight(g.provider.URL, "/"), url.PathEscape(g.provider.Organization), url.PathEscape(ref.Key))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get org variable: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", fmt.Errorf("failed to read org variable response: %w", readErr)
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("variable %q not found in org %s", ref.Key, g.provider.Organization)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d fetching org variable: %s", resp.StatusCode, body)
	}

	var v giteaVariableResponse
	if err := json.Unmarshal(body, &v); err != nil {
		return "", fmt.Errorf("failed to decode variable response: %w", err)
	}
	return v.Value, nil
}

// orgListVariablesFn lists all Actions variables for the organization.
func (g *Client) orgListVariablesFn(ctx context.Context) (map[string][]byte, error) {
	token, err := g.getToken(ctx)
	if err != nil {
		return nil, err
	}
	return listVariablesHTTP(ctx, g.provider.URL, token,
		fmt.Sprintf("/api/v1/orgs/%s/actions/variables", url.PathEscape(g.provider.Organization)))
}
