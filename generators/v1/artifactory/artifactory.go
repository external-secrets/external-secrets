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

// Package artifactory implements a generator for JFrog Artifactory access tokens.
package artifactory

import (
	"bytes"
	"context"
	b64 "encoding/base64"
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
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	defaultProviderType = "GenericOidc"
	oidcTokenPath       = "/access/api/v1/oidc/token"
	createTokenPath     = "/access/api/v1/tokens"

	grantTypeTokenExchange  = "urn:ietf:params:oauth:grant-type:token-exchange"
	subjectTokenTypeIDToken = "urn:ietf:params:oauth:token-type:id_token"

	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
	errNoAuth    = "either auth.oidc or auth.referenceToken must be set"

	httpClientTimeout = 30 * time.Second
)

// Generator implements Artifactory access token generation.
type Generator struct {
	httpClient *http.Client
}

type oidcExchangeRequest struct {
	GrantType             string `json:"grant_type"`
	SubjectTokenType      string `json:"subject_token_type"`
	SubjectToken          string `json:"subject_token"`
	ProviderName          string `json:"provider_name"`
	ProviderType          string `json:"provider_type,omitempty"`
	ApplicationKey        string `json:"application_key,omitempty"`
	ProjectKey            string `json:"project_key,omitempty"`
	IdentityMappingName   string `json:"identity_mapping_name,omitempty"`
	IncludeReferenceToken bool   `json:"include_reference_token,omitempty"`
}

type oidcExchangeResponse struct {
	AccessToken    string `json:"access_token"`
	TokenType      string `json:"token_type"`
	ExpiresIn      int64  `json:"expires_in"`
	Username       string `json:"username"`
	ReferenceToken string `json:"reference_token"`
}

type createTokenRequest struct {
	Scope                 string `json:"scope"`
	Description           string `json:"description,omitempty"`
	ExpiresIn             int64  `json:"expires_in,omitempty"`
	IncludeReferenceToken bool   `json:"include_reference_token,omitempty"`
	Refreshable           bool   `json:"refreshable,omitempty"`
	ProjectKey            string `json:"project_key,omitempty"`
}

type createTokenResponse struct {
	AccessToken    string `json:"access_token"`
	ReferenceToken string `json:"reference_token"`
	ExpiresIn      int64  `json:"expires_in"`
	Scope          string `json:"scope"`
	TokenType      string `json:"token_type"`
}

// Generate creates an Artifactory access token from the provided spec.
func (g *Generator) Generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	kubeClient client.Client,
	targetNamespace string,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}

	spec, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	switch {
	case spec.Spec.Auth.OIDC != nil:
		return g.generateOIDC(ctx, spec, kubeClient, targetNamespace)
	case spec.Spec.Auth.ReferenceToken != nil:
		return g.generateReferenceToken(ctx, spec, kubeClient, targetNamespace)
	default:
		return nil, nil, errors.New(errNoAuth)
	}
}

// Cleanup is a no-op for the Artifactory generator.
func (g *Generator) Cleanup(
	_ context.Context,
	_ *apiextensions.JSON,
	_ genv1alpha1.GeneratorProviderState,
	_ client.Client,
	_ string,
) error {
	return nil
}

func (g *Generator) generateOIDC(
	ctx context.Context,
	spec *genv1alpha1.ArtifactoryAccessToken,
	_ client.Client,
	targetNamespace string,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	auth := spec.Spec.Auth.OIDC
	saToken, err := esutils.FetchServiceAccountToken(ctx, auth.ServiceAccountRef, targetNamespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch service account token: %w", err)
	}

	resp, err := g.exchangeOIDCToken(ctx, spec.Spec.URL, auth, saToken)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to exchange OIDC token: %w", err)
	}

	return buildOutput(spec, tokenResult{
		accessToken:    resp.AccessToken,
		referenceToken: resp.ReferenceToken,
		username:       resp.Username,
		expiresIn:      resp.ExpiresIn,
	})
}

func (g *Generator) generateReferenceToken(
	ctx context.Context,
	spec *genv1alpha1.ArtifactoryAccessToken,
	kubeClient client.Client,
	targetNamespace string,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	auth := spec.Spec.Auth.ReferenceToken
	bootstrapToken, err := resolvers.SecretKeyRef(
		ctx, kubeClient, resolvers.EmptyStoreKind, targetNamespace, &auth.Token,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve bootstrap token: %w", err)
	}

	resp, err := g.createScopedToken(ctx, spec.Spec.URL, auth, bootstrapToken)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create scoped token: %w", err)
	}

	username := usernameFromToken(resp.AccessToken)
	return buildOutput(spec, tokenResult{
		accessToken:    resp.AccessToken,
		referenceToken: resp.ReferenceToken,
		username:       username,
		expiresIn:      resp.ExpiresIn,
	})
}

func (g *Generator) exchangeOIDCToken(
	ctx context.Context,
	platformURL string,
	auth *genv1alpha1.ArtifactoryOIDCAuth,
	saToken string,
) (*oidcExchangeResponse, error) {
	providerType := auth.ProviderType
	if providerType == "" {
		providerType = defaultProviderType
	}

	payload := oidcExchangeRequest{
		GrantType:             grantTypeTokenExchange,
		SubjectTokenType:      subjectTokenTypeIDToken,
		SubjectToken:          saToken,
		ProviderName:          auth.ProviderName,
		ProviderType:          providerType,
		ApplicationKey:        auth.ApplicationKey,
		ProjectKey:            auth.ProjectKey,
		IdentityMappingName:   auth.IdentityMappingName,
		IncludeReferenceToken: auth.IncludeReferenceToken,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OIDC request: %w", err)
	}

	return postJSON[oidcExchangeResponse](
		ctx, g.httpClient, platformURL+oidcTokenPath, body, "",
	)
}

func (g *Generator) createScopedToken(
	ctx context.Context,
	platformURL string,
	auth *genv1alpha1.ArtifactoryReferenceTokenAuth,
	bootstrapToken string,
) (*createTokenResponse, error) {
	payload := createTokenRequest{
		Scope:                 auth.Scope,
		Description:           auth.Description,
		ExpiresIn:             auth.ExpiresIn,
		IncludeReferenceToken: auth.IncludeReferenceToken,
		Refreshable:           auth.Refreshable,
		ProjectKey:            auth.ProjectKey,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create token request: %w", err)
	}

	return postJSON[createTokenResponse](
		ctx, g.httpClient, platformURL+createTokenPath, body, bootstrapToken,
	)
}

// postJSON sends a JSON POST request to the specified endpoint using the provided HTTP client
// (or a default client with a timeout if nil), optionally sets a Bearer authorization header,
// and unmarshals a successful (2xx) response body into a value of type T.
// It returns a pointer to the unmarshaled value or an error if request creation, execution,
// a non-2xx response status, reading the response body, or JSON unmarshaling fails.
func postJSON[T any](
	ctx context.Context,
	hc *http.Client,
	endpoint string,
	body []byte,
	bearerToken string,
) (*T, error) {
	client := hc
	if client == nil {
		client = &http.Client{Timeout: httpClientTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request failed due to unexpected status: %s", resp.Status)
	}

	var result T
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

type tokenResult struct {
	accessToken    string
	referenceToken string
	username       string
	expiresIn      int64
}

// buildOutput assembles the generator output map for an Artifactory access token request.
//
// The returned map always contains the `registry` entry. It will include `access_token`,
// `reference_token`, and `username` when those values are present in the provided tokenResult.
// If `username` is present, `auth` is added as the base64 encoding of `username:password`,
// where `password` prefers the reference token and falls back to the access token.
// The `expiry` entry is computed from the tokenResult (either from expires_in or the token's JWT).
//
// Returns an error if neither access nor reference tokens are present or if registry resolution
// or expiry computation fails. The generator provider state is always nil.
func buildOutput(
	spec *genv1alpha1.ArtifactoryAccessToken,
	result tokenResult,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if result.accessToken == "" && result.referenceToken == "" {
		return nil, nil, errors.New("token not found in response")
	}

	registry, err := registryHost(spec)
	if err != nil {
		return nil, nil, err
	}

	out := map[string][]byte{
		"registry": []byte(registry),
	}

	if result.accessToken != "" {
		out["access_token"] = []byte(result.accessToken)
	}
	if result.referenceToken != "" {
		out["reference_token"] = []byte(result.referenceToken)
	}
	if result.username != "" {
		out["username"] = []byte(result.username)
	}

	password := result.referenceToken
	if password == "" {
		password = result.accessToken
	}
	if result.username != "" {
		out["auth"] = []byte(b64.StdEncoding.EncodeToString([]byte(result.username + ":" + password)))
	}

	expiry, err := computeExpiry(result)
	if err != nil {
		return nil, nil, err
	}
	out["expiry"] = []byte(expiry)

	return out, nil, nil
}

// registryHost returns the registry host for the given ArtifactoryAccessToken spec.
// If Spec.Registry is set, it strips a leading "https://" and returns that value.
// Otherwise it parses Spec.URL and returns the URL's host. An error is returned
// if Spec.URL is invalid or does not contain a hostname.
func registryHost(spec *genv1alpha1.ArtifactoryAccessToken) (string, error) {
	if spec.Spec.Registry != "" {
		return strings.TrimPrefix(spec.Spec.Registry, "https://"), nil
	}

	parsed, err := url.Parse(spec.Spec.URL)
	if err != nil {
		return "", fmt.Errorf("invalid platform URL: %w", err)
	}
	if parsed.Host == "" {
		return "", errors.New("platform URL must include a hostname")
	}
	return parsed.Host, nil
}

// computeExpiry computes the expiration timestamp (Unix seconds) as a base-10 string for the provided tokenResult.
// If ExpiresIn is greater than zero, it returns the current Unix time plus ExpiresIn.
// Otherwise it attempts to extract the `exp` claim from the access token, falling back to the reference token.
// Returns an error if neither expiry source is available or extraction fails.
func computeExpiry(result tokenResult) (string, error) {
	if result.expiresIn > 0 {
		return strconv.FormatInt(time.Now().Unix()+result.expiresIn, 10), nil
	}

	token := result.accessToken
	if token == "" {
		token = result.referenceToken
	}
	if token == "" {
		return "", errors.New("unable to determine token expiry")
	}

	return esutils.ExtractJWTExpiration(token)
}

// usernameFromToken extracts the `sub` claim from a JWT and returns it as the username.
// If the token cannot be parsed or the `sub` claim is missing or not a string, an empty string is returned.
func usernameFromToken(token string) string {
	claims, err := esutils.ParseJWTClaims(token)
	if err != nil {
		return ""
	}
	sub, ok := claims["sub"].(string)
	if !ok {
		return ""
	}
	return sub
}

// parseSpec parses the provided YAML or JSON bytes into an ArtifactoryAccessToken and returns the populated struct and any unmarshalling error.
func parseSpec(data []byte) (*genv1alpha1.ArtifactoryAccessToken, error) {
	var spec genv1alpha1.ArtifactoryAccessToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// NewGenerator creates a new Generator with an uninitialized HTTP client (nil), suitable for callers to use as-is or configure before use.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindArtifactoryAccessToken)
}
