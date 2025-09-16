/*
Copyright © 2025 ESO Maintainer Team

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

package dex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/golang-jwt/jwt/v5"
	authv1 "k8s.io/api/authentication/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type UsernamePasswordGenerator struct {
	httpClient *http.Client
}

type TokenExchangeGenerator struct {
	httpClient *http.Client
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

type authRequest struct {
	tokenURL     string
	formData     url.Values
	clientID     string
	clientSecret string
	logMessage   string
}

const (
	defaultConnectorID = "kubernetes"

	grantTypePassword      = "password"
	grantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
	tokenTypeAccessToken   = "urn:ietf:params:oauth:token-type:access_token"
	tokenTypeIDToken       = "urn:ietf:params:oauth:token-type:id_token"

	errNoSpec              = "no config spec provided"
	errParseSpec           = "unable to parse spec: %w"
	errGetToken            = "unable to get Dex access token: %w"
	errCreateTokenRequest  = "failed to create token request: %w"
	errUnexpectedStatus    = "request failed due to unexpected status: %s"
	errReadResponse        = "failed to read response body: %w"
	errUnmarshalResponse   = "failed to unmarshal response: %w"
	errAccessTokenNotFound = "access_token not found in response"
	errFetchSecret         = "failed to fetch secret: %w"
	errFetchServiceAccount = "failed to fetch service account token: %w"

	httpClientTimeout = 30 * time.Second
)

func (g *UsernamePasswordGenerator) Generate(ctx context.Context, dexSpec *apiextensions.JSON, kubeClient client.Client, targetNamespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(ctx, dexSpec, kubeClient, targetNamespace)
}

func (g *UsernamePasswordGenerator) Cleanup(_ context.Context, dexSpec *apiextensions.JSON, providerState genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *TokenExchangeGenerator) Generate(ctx context.Context, dexSpec *apiextensions.JSON, kubeClient client.Client, targetNamespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(ctx, dexSpec, kubeClient, targetNamespace)
}

func (g *TokenExchangeGenerator) Cleanup(_ context.Context, dexSpec *apiextensions.JSON, providerState genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *UsernamePasswordGenerator) generate(
	ctx context.Context,
	dexSpec *apiextensions.JSON,
	kubeClient client.Client,
	targetNamespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if dexSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}

	res, err := parseUsernamePasswordSpec(dexSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	scopes := res.Spec.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}

	var clientSecret string
	if res.Spec.ClientSecretRef != nil {
		var err error
		clientSecret, err = resolvers.SecretKeyRef(ctx, kubeClient, resolvers.EmptyStoreKind, targetNamespace, &esmeta.SecretKeySelector{
			Namespace: &targetNamespace,
			Name:      res.Spec.ClientSecretRef.Name,
			Key:       res.Spec.ClientSecretRef.Key,
		})
		if err != nil {
			return nil, nil, fmt.Errorf(errFetchSecret, err)
		}
	}

	accessToken, err := g.authenticateWithPassword(ctx, res, clientSecret, scopes, kubeClient, targetNamespace)
	if err != nil {
		return nil, nil, fmt.Errorf(errGetToken, err)
	}

	exp, err := extractJWTExpiration(accessToken)
	if err != nil {
		return nil, nil, err
	}

	return map[string][]byte{
		"token":  []byte(accessToken),
		"expiry": []byte(exp),
	}, nil, nil
}

func (g *TokenExchangeGenerator) generate(
	ctx context.Context,
	dexSpec *apiextensions.JSON,
	kubeClient client.Client,
	targetNamespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if dexSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}

	res, err := parseTokenExchangeSpec(dexSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	connectorID := res.Spec.ConnectorID
	if connectorID == "" {
		connectorID = defaultConnectorID
	}

	scopes := res.Spec.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}

	var clientSecret string
	if res.Spec.ClientSecretRef != nil {
		var err error
		clientSecret, err = resolvers.SecretKeyRef(ctx, kubeClient, resolvers.EmptyStoreKind, targetNamespace, &esmeta.SecretKeySelector{
			Namespace: &targetNamespace,
			Name:      res.Spec.ClientSecretRef.Name,
			Key:       res.Spec.ClientSecretRef.Key,
		})
		if err != nil {
			return nil, nil, fmt.Errorf(errFetchSecret, err)
		}
	}

	accessToken, err := g.authenticateWithTokenExchange(ctx, res, clientSecret, connectorID, scopes, targetNamespace)
	if err != nil {
		return nil, nil, fmt.Errorf(errGetToken, err)
	}

	exp, err := extractJWTExpiration(accessToken)
	if err != nil {
		return nil, nil, err
	}

	return map[string][]byte{
		"token":  []byte(accessToken),
		"expiry": []byte(exp),
	}, nil, nil
}

func (g *UsernamePasswordGenerator) authenticateWithPassword(ctx context.Context, res *genv1alpha1.DexUsernamePassword, clientSecret string, scopes []string, kubeClient client.Client, targetNamespace string) (string, error) {
	log := logr.FromContextOrDiscard(ctx)

	username, err := resolvers.SecretKeyRef(ctx, kubeClient, resolvers.EmptyStoreKind, targetNamespace, &esmeta.SecretKeySelector{
		Namespace: &targetNamespace,
		Name:      res.Spec.UsernameRef.Name,
		Key:       res.Spec.UsernameRef.Key,
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch username: %w", err)
	}

	password, err := resolvers.SecretKeyRef(ctx, kubeClient, resolvers.EmptyStoreKind, targetNamespace, &esmeta.SecretKeySelector{
		Namespace: &targetNamespace,
		Name:      res.Spec.PasswordRef.Name,
		Key:       res.Spec.PasswordRef.Key,
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch password: %w", err)
	}

	log.V(1).Info("Authenticating with password mode", "username", username)

	formData := url.Values{}
	formData.Set("grant_type", grantTypePassword)
	formData.Set("scope", strings.Join(scopes, " "))
	formData.Set("username", username)
	formData.Set("password", password)

	request := authRequest{
		tokenURL:     buildTokenURL(res.Spec.DexURL),
		formData:     formData,
		clientID:     res.Spec.ClientID,
		clientSecret: clientSecret,
		logMessage:   "Authenticating with username/password to Dex",
	}

	accessToken, err := executeTokenRequest(ctx, g.httpClient, request)
	if err != nil {
		return "", err
	}

	log.V(1).Info("Successfully authenticated with username/password to Dex")
	return accessToken, nil
}

func (g *TokenExchangeGenerator) authenticateWithTokenExchange(ctx context.Context, res *genv1alpha1.DexTokenExchange, clientSecret, connectorID string, scopes []string, targetNamespace string) (string, error) {
	log := logr.FromContextOrDiscard(ctx)

	serviceAccountToken, err := fetchServiceAccountToken(ctx, res.Spec.ServiceAccountRef, targetNamespace)
	if err != nil {
		return "", fmt.Errorf(errFetchServiceAccount, err)
	}

	log.V(1).Info("Starting token exchange with Dex", "connectorId", connectorID, "clientId", res.Spec.ClientID)

	formData := url.Values{}
	formData.Set("grant_type", grantTypeTokenExchange)
	formData.Set("connector_id", connectorID)
	formData.Set("scope", strings.Join(scopes, " "))
	formData.Set("requested_token_type", tokenTypeAccessToken)
	formData.Set("subject_token", serviceAccountToken)
	formData.Set("subject_token_type", tokenTypeIDToken)

	request := authRequest{
		tokenURL:     buildTokenURL(res.Spec.DexURL),
		formData:     formData,
		clientID:     res.Spec.ClientID,
		clientSecret: clientSecret,
		logMessage:   "Exchanging service account token for Dex access token",
	}

	accessToken, err := executeTokenRequest(ctx, g.httpClient, request)
	if err != nil {
		return "", err
	}

	log.V(1).Info("Successfully exchanged service account token for Dex access token")
	return accessToken, nil
}

func executeTokenRequest(ctx context.Context, httpClient *http.Client, req authRequest) (string, error) {
	log := logr.FromContextOrDiscard(ctx)

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: httpClientTimeout,
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.tokenURL, strings.NewReader(req.formData.Encode()))
	if err != nil {
		return "", fmt.Errorf(errCreateTokenRequest, err)
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if req.clientID != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(req.clientID + ":" + req.clientSecret))
		httpReq.Header.Set("Authorization", "Basic "+auth)
	}

	log.Info(req.logMessage, "url", req.tokenURL, "clientId", req.clientID)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error(nil, "Authentication failed", "status", resp.Status, "body", string(body))
		return "", fmt.Errorf(errUnexpectedStatus, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf(errReadResponse, err)
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return "", fmt.Errorf(errUnmarshalResponse, err)
	}

	if tokenResp.AccessToken == "" {
		return "", errors.New(errAccessTokenNotFound)
	}

	return tokenResp.AccessToken, nil
}

func buildTokenURL(dexURL string) string {
	return strings.TrimSuffix(dexURL, "/") + "/token"
}

func parseUsernamePasswordSpec(specData []byte) (*genv1alpha1.DexUsernamePassword, error) {
	var spec genv1alpha1.DexUsernamePassword
	err := yaml.Unmarshal(specData, &spec)
	return &spec, err
}

func parseTokenExchangeSpec(specData []byte) (*genv1alpha1.DexTokenExchange, error) {
	var spec genv1alpha1.DexTokenExchange
	err := yaml.Unmarshal(specData, &spec)
	return &spec, err
}

func extractJWTExpiration(tokenString string) (string, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse JWT: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("failed to extract claims from JWT")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return "", fmt.Errorf("no expiration claim found in JWT")
	}

	expTime := time.Unix(int64(exp), 0)
	return expTime.Format(time.RFC3339), nil
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

	expirationSeconds := int64(600)
	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences:         saRef.Audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}
	tokenResponse, err := kubeClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, saRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}
	return tokenResponse.Status.Token, nil
}

func init() {
	genv1alpha1.Register(genv1alpha1.DexUsernamePasswordKind, &UsernamePasswordGenerator{})
	genv1alpha1.Register(genv1alpha1.DexTokenExchangeKind, &TokenExchangeGenerator{})
}
