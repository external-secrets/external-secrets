/*
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
	"strconv"
	"strings"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type Generator struct {
	httpClient *http.Client
}

type CloudsmithOIDCRequest struct {
	OIDCToken   string `json:"oidc_token"`
	ServiceSlug string `json:"service_slug"`
}

type CloudsmithOIDCResponse struct {
	Token string `json:"token"`
}

const (
	defaultCloudsmithAPIHost = "api.cloudsmith.io"

	errNoSpec            = "no config spec provided"
	errParseSpec         = "unable to parse spec: %w"
	errGetToken          = "unable to get authorization token: %w"
	errExchangeToken     = "unable to exchange OIDC token: %w"
	errMarshalRequest    = "failed to marshal request payload: %w"
	errCreateRequest     = "failed to create HTTP request: %w"
	errUnexpectedStatus  = "request failed due to unexpected status: %s"
	errReadResponse      = "failed to read response body: %w"
	errUnmarshalResponse = "failed to unmarshal response: %w"
	errTokenNotFound     = "token not found in response"
	errTokenTypeCast     = "error when typecasting token to string"

	httpClientTimeout = 30 * time.Second
)

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(
		ctx,
		jsonSpec,
		kube,
		namespace,
	)
}

func (g *Generator) Cleanup(_ context.Context, jsonSpec *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	_ client.Client,
	namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	// Fetch the service account token
	oidcToken, err := fetchServiceAccountToken(ctx, res.Spec.ServiceAccountRef, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch service account token: %w", err)
	}

	apiHost := res.Spec.APIHost
	if apiHost == "" {
		apiHost = defaultCloudsmithAPIHost
	}
	apiHost = strings.TrimPrefix(apiHost, "https://")

	accessToken, err := g.exchangeTokenWithCloudsmith(ctx, oidcToken, res.Spec.OrgSlug, res.Spec.ServiceSlug, apiHost)
	if err != nil {
		return nil, nil, fmt.Errorf(errExchangeToken, err)
	}

	exp, err := tokenExpiration(accessToken)
	if err != nil {
		return nil, nil, err
	}

	return map[string][]byte{
		"auth": []byte(b64.StdEncoding.EncodeToString([]byte("token:" + accessToken))),
		"expiry": []byte(exp),
	}, nil, nil
}

func (g *Generator) exchangeTokenWithCloudsmith(ctx context.Context, oidcToken, orgSlug, serviceSlug, apiHost string) (string, error) {
	klog.V(4).InfoS("Starting OIDC token exchange with Cloudsmith")

	requestPayload := CloudsmithOIDCRequest{
		OIDCToken:   oidcToken,
		ServiceSlug: serviceSlug,
	}

	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		klog.ErrorS(err, "Failed to marshal OIDC request payload")
		return "", fmt.Errorf(errMarshalRequest, err)
	}

	url := fmt.Sprintf("https://%s/openid/%s/", apiHost, orgSlug)
	klog.InfoS("Exchanging OIDC token with Cloudsmith",
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
		klog.ErrorS(err, "Failed to create HTTP request")
		return "", fmt.Errorf(errCreateRequest, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		klog.ErrorS(err, "Failed to execute HTTP request")
		return "", fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		klog.ErrorS(nil, "Unexpected HTTP status", "status", resp.Status, "statusCode", resp.StatusCode)
		return "", fmt.Errorf(errUnexpectedStatus, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.ErrorS(err, "Failed to read response body")
		return "", fmt.Errorf(errReadResponse, err)
	}

	var result CloudsmithOIDCResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		klog.ErrorS(err, "Failed to unmarshal response", "body", string(body))
		return "", fmt.Errorf(errUnmarshalResponse, err)
	}

	if result.Token == "" {
		klog.ErrorS(nil, "Token not found in response", "body", string(body))
		return "", errors.New(errTokenNotFound)
	}

	klog.V(4).InfoS("Successfully exchanged OIDC token for Cloudsmith access token")
	return result.Token, nil
}

func getClaims(tokenString string) (map[string]interface{}, error) {
	// Split the token into its three parts
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Decode the payload (the second part of the token)
	payload, err := b64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error decoding payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("error un-marshaling claims: %w", err)
	}
	return claims, nil
}

func tokenExpiration(tokenString string) (string, error) {
	claims, err := getClaims(tokenString)
	if err != nil {
		return "", fmt.Errorf("error getting claims: %w", err)
	}
	exp, ok := claims["exp"].(float64)
	if ok {
		return strconv.FormatFloat(exp, 'f', -1, 64), nil
	}

	return "", fmt.Errorf("exp claim not found or wrong type")
}

func fetchServiceAccountToken(ctx context.Context, saRef esmeta.ServiceAccountSelector, namespace string) (string, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return "", err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: saRef.Audiences,
		},
	}
	tokenResponse, err := kubeClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, saRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}
	return tokenResponse.Status.Token, nil
}

func parseSpec(data []byte) (*genv1alpha1.CloudsmithAccessToken, error) {
	var spec genv1alpha1.CloudsmithAccessToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.CloudsmithAccessTokenKind, &Generator{})
}
