/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package cloudsmith implements a generator for Cloudsmith access tokens using OIDC.
package cloudsmith

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

// Generator implements the Cloudsmith access token generator.
type Generator struct {
	httpClient *http.Client
}

// OIDCRequest represents the payload sent to Cloudsmith for OIDC token exchange.
type OIDCRequest struct {
	OIDCToken   string `json:"oidc_token"`
	ServiceSlug string `json:"service_slug"`
}

// OIDCResponse represents the response from Cloudsmith containing the access token.
type OIDCResponse struct {
	Token string `json:"token"`
}

const (
	defaultCloudsmithAPIURL = "https://api.cloudsmith.io"

	errNoSpec            = "no config spec provided"
	errParseSpec         = "unable to parse spec: %w"
	errExchangeToken     = "unable to exchange OIDC token: %w"
	errMarshalRequest    = "failed to marshal request payload: %w"
	errCreateRequest     = "failed to create HTTP request: %w"
	errUnexpectedStatus  = "request failed due to unexpected status: %s"
	errReadResponse      = "failed to read response body: %w"
	errUnmarshalResponse = "failed to unmarshal response: %w"
	errTokenNotFound     = "token not found in response"

	httpClientTimeout = 30 * time.Second
)

// Generate generates a Cloudsmith access token using the provided cloudsmith JSON spec.
func (g *Generator) Generate(ctx context.Context, cloudsmithSpec *apiextensions.JSON, kubeClient client.Client, targetNamespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(
		ctx,
		cloudsmithSpec,
		kubeClient,
		targetNamespace,
	)
}

// Cleanup is a no-op for the Cloudsmith generator.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

// generate performs the main logic of the Cloudsmith generator.
func (g *Generator) generate(ctx context.Context, cloudsmithSpec *apiextensions.JSON, _ client.Client, targetNamespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if cloudsmithSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(cloudsmithSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	// Fetch the service account token
	oidcToken, err := esutils.FetchServiceAccountToken(ctx, res.Spec.ServiceAccountRef, targetNamespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch service account token: %w", err)
	}

	apiURL := res.Spec.APIURL
	if apiURL == "" {
		apiURL = defaultCloudsmithAPIURL
	}

	accessToken, err := g.exchangeTokenWithCloudsmith(ctx, oidcToken, res.Spec.OrgSlug, res.Spec.ServiceSlug, apiURL)
	if err != nil {
		return nil, nil, fmt.Errorf(errExchangeToken, err)
	}

	exp, err := esutils.ExtractJWTExpiration(accessToken)
	if err != nil {
		return nil, nil, err
	}

	return map[string][]byte{
		"auth":   []byte(b64.StdEncoding.EncodeToString([]byte("token:" + accessToken))),
		"expiry": []byte(exp),
	}, nil, nil
}

func (g *Generator) exchangeTokenWithCloudsmith(ctx context.Context, oidcToken, orgSlug, serviceSlug, apiURL string) (string, error) {
	log := logr.FromContextOrDiscard(ctx)
	log.V(4).Info("Starting OIDC token exchange with Cloudsmith")

	requestPayload := OIDCRequest{
		OIDCToken:   oidcToken,
		ServiceSlug: serviceSlug,
	}

	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf(errMarshalRequest, err)
	}

	url := fmt.Sprintf("%s/openid/%s/", strings.TrimSuffix(apiURL, "/"), orgSlug)
	log.Info("Exchanging OIDC token with Cloudsmith",
		"url", url,
		"serviceSlug", serviceSlug,
		"orgSlug", orgSlug)

	httpClient := g.httpClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: httpClientTimeout,
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf(errCreateRequest, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf(errUnexpectedStatus, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf(errReadResponse, err)
	}

	var result OIDCResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf(errUnmarshalResponse, err)
	}

	if result.Token == "" {
		return "", errors.New(errTokenNotFound)
	}

	log.V(4).Info("Successfully exchanged OIDC token for Cloudsmith access token")
	return result.Token, nil
}

func parseSpec(specData []byte) (*genv1alpha1.CloudsmithAccessToken, error) {
	var spec genv1alpha1.CloudsmithAccessToken
	err := yaml.Unmarshal(specData, &spec)
	return &spec, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindCloudsmithAccessToken)
}
