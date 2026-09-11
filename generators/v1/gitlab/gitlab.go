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

// Package gitlab provides a generator for GitLab project and group deploy tokens.
package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	defaultGitlabAPI = "https://gitlab.com"
	apiPath          = "/api/v4"

	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"

	// requestTimeout bounds a single Generate or Cleanup. Each makes exactly one
	// HTTP call, governed by this context deadline; the default client uses the
	// same value so a shorter transport timeout cannot preempt it and abandon an
	// in-flight create (which would orphan a deploy token GitLab already minted).
	requestTimeout = 30 * time.Second

	// minExpiresAfter is the floor for spec.expiresAfter. It mirrors the CEL rule
	// on GitlabDeployTokenSpec.ExpiresAfter (apis/generators/v1alpha1); keep the
	// two in sync. validateExpiry re-checks it in code so the invariant still
	// holds on clusters where CEL admission is not enforced.
	minExpiresAfter = 24 * time.Hour
)

// Generator implements GitLab deploy token generation.
type Generator struct {
	httpClient *http.Client
	// now returns the current time and is overridable in tests so the relative
	// expiresAfter expiry is deterministic. Defaults to time.Now.
	now func() time.Time
}

// deployTokenState is persisted as the generator state so that Cleanup can revoke
// the deploy token that Generate created. GitLab deploy tokens are persistent, so
// without revoking them every refresh would leave a dangling token behind.
type deployTokenState struct {
	URL       string `json:"url"`
	ProjectID string `json:"projectID,omitempty"`
	GroupID   string `json:"groupID,omitempty"`
	TokenID   int    `json:"tokenID"`
}

// createTokenResponse mirrors the fields returned by the GitLab deploy token API.
type createTokenResponse struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

// Generate creates a new GitLab deploy token and returns its username and token.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(ctx, jsonSpec, kube, namespace)
}

// Cleanup revokes the deploy token created during Generate. It is idempotent: a
// token that has already been deleted (HTTP 404) is treated as success.
func (g *Generator) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, kube client.Client, namespace string) error {
	return g.cleanup(ctx, jsonSpec, state, kube, namespace)
}

func (g *Generator) generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	spec, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}
	if err := validateExpiry(&spec.Spec); err != nil {
		return nil, nil, err
	}
	token, err := g.fetchAuthToken(ctx, kube, namespace, &spec.Spec)
	if err != nil {
		return nil, nil, err
	}

	payload := map[string]any{
		"name":   spec.Spec.Name,
		"scopes": spec.Spec.Scopes,
	}
	if spec.Spec.Username != "" {
		payload["username"] = spec.Spec.Username
	}
	if spec.Spec.ExpiresAt != nil {
		payload["expires_at"] = spec.Spec.ExpiresAt.UTC().Format(time.RFC3339)
	}
	// expiresAfter is a relative expiry resolved to an absolute expires_at at each
	// generation. It stays in the outbound request only; nothing is written back to
	// the GitlabDeployToken object, so a GitOps-managed spec does not drift.
	// validateExpiry above (and CEL admission) guarantee expiresAt and expiresAfter
	// are never both set, so this cannot clobber the expiresAt branch.
	if spec.Spec.ExpiresAfter != nil {
		expiry := g.clock().Add(spec.Spec.ExpiresAfter.Duration)
		payload["expires_at"] = expiry.UTC().Format(time.RFC3339)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshaling payload: %w", err)
	}

	endpoint, err := deployTokensURL(&spec.Spec)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client().Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error performing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("error generating deploy token: response code: %d, response: %s", resp.StatusCode, gitlabError(raw))
	}

	var parsed createTokenResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, nil, fmt.Errorf("error decoding response: %w", err)
	}
	if parsed.Token == "" {
		return nil, nil, errors.New("deploy token missing from GitLab response")
	}

	state, err := json.Marshal(deployTokenState{
		URL:       spec.Spec.URL,
		ProjectID: spec.Spec.ProjectID,
		GroupID:   spec.Spec.GroupID,
		TokenID:   parsed.ID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error marshaling state: %w", err)
	}

	return map[string][]byte{
		"username": []byte(parsed.Username),
		"token":    []byte(parsed.Token),
	}, &apiextensions.JSON{Raw: state}, nil
}

func (g *Generator) cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, rawState genv1alpha1.GeneratorProviderState, kube client.Client, namespace string) error {
	if jsonSpec == nil || rawState == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	spec, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return fmt.Errorf(errParseSpec, err)
	}
	var state deployTokenState
	if err := json.Unmarshal(rawState.Raw, &state); err != nil {
		return fmt.Errorf("error parsing generator state: %w", err)
	}
	if state.TokenID == 0 {
		return nil
	}

	authToken, err := g.fetchAuthToken(ctx, kube, namespace, &spec.Spec)
	if err != nil {
		return err
	}

	// Build the revoke endpoint from the persisted state, not the (possibly
	// changed) current spec, so cleanup always targets where the token was made.
	base, err := deployTokensURL(&genv1alpha1.GitlabDeployTokenSpec{
		URL:       state.URL,
		ProjectID: state.ProjectID,
		GroupID:   state.GroupID,
	})
	if err != nil {
		return err
	}
	endpoint := base + "/" + strconv.Itoa(state.TokenID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, http.NoBody)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", authToken)

	resp, err := g.client().Do(req)
	if err != nil {
		return fmt.Errorf("error performing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 204 No Content on success; 404 means the token is already gone.
	if resp.StatusCode == http.StatusNotFound || (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil
	}
	raw, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("error revoking deploy token: response code: %d, response: %s", resp.StatusCode, gitlabError(raw))
}

func (g *Generator) client() *http.Client {
	if g.httpClient != nil {
		return g.httpClient
	}
	return &http.Client{Timeout: requestTimeout}
}

// clock returns the current time, using the injected now func when set so tests
// can pin a deterministic value for the expiresAfter calculation.
func (g *Generator) clock() time.Time {
	if g.now != nil {
		return g.now()
	}
	return time.Now()
}

func (g *Generator) fetchAuthToken(ctx context.Context, kube client.Client, namespace string, spec *genv1alpha1.GitlabDeployTokenSpec) (string, error) {
	token, err := resolvers.SecretKeyRef(ctx, kube, resolvers.EmptyStoreKind, namespace, &spec.Auth.Token.SecretRef)
	if err != nil {
		return "", fmt.Errorf("error getting GitLab token from secret: %w", err)
	}
	return token, nil
}

// deployTokensURL builds the deploy-tokens collection URL for the configured
// project or group. Exactly one of projectID / groupID must be set.
func deployTokensURL(spec *genv1alpha1.GitlabDeployTokenSpec) (string, error) {
	base := spec.URL
	if base == "" {
		base = defaultGitlabAPI
	}
	// Trim trailing slashes so a user-supplied url such as
	// "https://gitlab.example.com/" does not yield a double slash before the
	// API path.
	base = strings.TrimRight(base, "/")
	switch {
	case spec.ProjectID != "" && spec.GroupID == "":
		return base + apiPath + "/projects/" + url.PathEscape(spec.ProjectID) + "/deploy_tokens", nil
	case spec.GroupID != "" && spec.ProjectID == "":
		return base + apiPath + "/groups/" + url.PathEscape(spec.GroupID) + "/deploy_tokens", nil
	default:
		return "", errors.New("exactly one of projectID or groupID must be set")
	}
}

// gitlabError extracts a human-readable message from a GitLab error body, which
// uses either a "message" or an "error" field.
func gitlabError(raw []byte) string {
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err == nil {
		if msg, ok := body["message"]; ok {
			return fmt.Sprintf("%v", msg)
		}
		if msg, ok := body["error"]; ok {
			return fmt.Sprintf("%v", msg)
		}
	}
	return string(raw)
}

// validateExpiry enforces, in code, the same invariants the CEL rules declare on
// GitlabDeployTokenSpec (apis/generators/v1alpha1/types_gitlab.go): expiresAt and
// expiresAfter are mutually exclusive, and expiresAfter has a minExpiresAfter
// floor. This is defense in depth for clusters where CEL admission is not
// enforced; the error strings match the CEL messages.
func validateExpiry(spec *genv1alpha1.GitlabDeployTokenSpec) error {
	if spec.ExpiresAt != nil && spec.ExpiresAfter != nil {
		return errors.New("expiresAt and expiresAfter are mutually exclusive")
	}
	if spec.ExpiresAfter != nil && spec.ExpiresAfter.Duration < minExpiresAfter {
		return errors.New("expiresAfter must be at least 24h")
	}
	return nil
}

func parseSpec(data []byte) (*genv1alpha1.GitlabDeployToken, error) {
	var spec genv1alpha1.GitlabDeployToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindGitlabDeployToken)
}
