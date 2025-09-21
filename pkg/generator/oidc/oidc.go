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

package oidc

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

type Generator struct {
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
	endpoint          string
	formData          url.Values
	clientID          string
	clientSecret      string
	additionalHeaders map[string]string
	logMessage        string
}

const (
	grantTypePassword      = "password"
	grantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"

	errNoSpec              = "no config spec provided"
	errParseSpec           = "unable to parse spec: %w"
	errCreateTokenRequest  = "failed to create token request: %w"
	errUnexpectedStatus    = "request failed due to unexpected status: %s"
	errReadResponse        = "failed to read response body: %w"
	errUnmarshalResponse   = "failed to unmarshal response: %w"
	errFetchSecret         = "failed to fetch secret: %w"
	errFetchServiceAccount = "failed to fetch service account token: %w"
	errInvalidGrantType    = "no valid grant type specified"

	httpClientTimeout = 30 * time.Second
)

func (g *Generator) Generate(ctx context.Context, oidcSpec *apiextensions.JSON, kubeClient client.Client, targetNamespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if g.httpClient == nil {
		g.httpClient = &http.Client{Timeout: httpClientTimeout}
	}
	return g.generate(ctx, oidcSpec, kubeClient, targetNamespace)
}

func (g *Generator) Cleanup(_ context.Context, oidcSpec *apiextensions.JSON, providerState genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(
	ctx context.Context,
	oidcSpec *apiextensions.JSON,
	kubeClient client.Client,
	targetNamespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if oidcSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}

	spec, err := parseOIDCSpec(oidcSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	scopes := g.getScopes(spec)
	clientSecret, err := g.getClientSecret(ctx, kubeClient, targetNamespace, spec)
	if err != nil {
		return nil, nil, err
	}

	tokenResp, err := g.authenticateWithGrant(ctx, spec, clientSecret, scopes, kubeClient, targetNamespace)
	if err != nil {
		return nil, nil, err
	}

	result := g.buildTokenResult(tokenResp)
	return result, nil, nil
}

func (g *Generator) getScopes(spec *genv1alpha1.OIDC) []string {
	if len(spec.Spec.Scopes) == 0 {
		return []string{"openid"}
	}
	return spec.Spec.Scopes
}

func (g *Generator) getClientSecret(ctx context.Context, kubeClient client.Client, targetNamespace string, spec *genv1alpha1.OIDC) (string, error) {
	if spec.Spec.ClientSecretRef == nil {
		return "", nil
	}

	clientSecret, err := resolvers.SecretKeyRef(ctx, kubeClient, resolvers.EmptyStoreKind, targetNamespace, &esmeta.SecretKeySelector{
		Namespace: &targetNamespace,
		Name:      spec.Spec.ClientSecretRef.Name,
		Key:       spec.Spec.ClientSecretRef.Key,
	})
	if err != nil {
		return "", fmt.Errorf(errFetchSecret, err)
	}
	return clientSecret, nil
}

func (g *Generator) authenticateWithGrant(ctx context.Context, spec *genv1alpha1.OIDC, clientSecret string, scopes []string, kubeClient client.Client, targetNamespace string) (*TokenResponse, error) {
	switch {
	case spec.Spec.Grant.Password != nil:
		tokenResp, err := g.authenticateWithPassword(ctx, spec, clientSecret, scopes, kubeClient, targetNamespace)
		if err != nil {
			return nil, fmt.Errorf("password grant authentication failed: %w", err)
		}
		return tokenResp, nil
	case spec.Spec.Grant.TokenExchange != nil:
		tokenResp, err := g.authenticateWithTokenExchange(ctx, spec, clientSecret, scopes, kubeClient, targetNamespace)
		if err != nil {
			return nil, fmt.Errorf("token exchange grant authentication failed: %w", err)
		}
		return tokenResp, nil
	default:
		return nil, errors.New(errInvalidGrantType)
	}
}

func (g *Generator) buildTokenResult(tokenResp *TokenResponse) map[string][]byte {
	result := make(map[string][]byte)

	if tokenResp.AccessToken != "" {
		result["access_token"] = []byte(tokenResp.AccessToken)
		result["token"] = []byte(tokenResp.AccessToken)

		if exp, err := extractJWTExpiration(tokenResp.AccessToken); err == nil {
			result["expiry"] = []byte(exp)
		}
	}

	if tokenResp.TokenType != "" {
		result["token_type"] = []byte(tokenResp.TokenType)
	}

	if tokenResp.ExpiresIn > 0 {
		result["expires_in"] = []byte(fmt.Sprintf("%d", tokenResp.ExpiresIn))
	}

	if tokenResp.RefreshToken != "" {
		result["refresh_token"] = []byte(tokenResp.RefreshToken)
	}

	if tokenResp.IDToken != "" {
		result["id_token"] = []byte(tokenResp.IDToken)
	}

	return result
}

func (g *Generator) authenticateWithPassword(ctx context.Context, spec *genv1alpha1.OIDC, clientSecret string, scopes []string, kubeClient client.Client, targetNamespace string) (*TokenResponse, error) {
	log := logr.FromContextOrDiscard(ctx)

	username, err := resolvers.SecretKeyRef(ctx, kubeClient, resolvers.EmptyStoreKind, targetNamespace, &esmeta.SecretKeySelector{
		Namespace: &targetNamespace,
		Name:      spec.Spec.Grant.Password.UsernameRef.Name,
		Key:       spec.Spec.Grant.Password.UsernameRef.Key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch username: %w", err)
	}

	password, err := resolvers.SecretKeyRef(ctx, kubeClient, resolvers.EmptyStoreKind, targetNamespace, &esmeta.SecretKeySelector{
		Namespace: &targetNamespace,
		Name:      spec.Spec.Grant.Password.PasswordRef.Name,
		Key:       spec.Spec.Grant.Password.PasswordRef.Key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch password: %w", err)
	}

	tokenURL := spec.Spec.TokenURL

	log.V(1).Info("Authenticating with password grant", "username", username, "tokenUrl", spec.Spec.TokenURL)

	formData := url.Values{}
	formData.Set("grant_type", grantTypePassword)
	formData.Set("scope", strings.Join(scopes, " "))
	formData.Set("username", username)
	formData.Set("password", password)

	for key, value := range spec.Spec.AdditionalParameters {
		formData.Set(key, value)
	}

	additionalHeaders := make(map[string]string)
	for key, value := range spec.Spec.AdditionalHeaders {
		additionalHeaders[key] = value
	}

	request := authRequest{
		endpoint:          tokenURL,
		formData:          formData,
		clientID:          spec.Spec.ClientID,
		clientSecret:      clientSecret,
		additionalHeaders: additionalHeaders,
		logMessage:        "Authenticating with password grant to OIDC provider",
	}

	return executeTokenRequest(ctx, g.httpClient, request)
}

func (g *Generator) authenticateWithTokenExchange(ctx context.Context, spec *genv1alpha1.OIDC, clientSecret string, scopes []string, kubeClient client.Client, targetNamespace string) (*TokenResponse, error) {
	log := logr.FromContextOrDiscard(ctx)

	subjectToken, err := g.getSubjectToken(ctx, spec, kubeClient, targetNamespace)
	if err != nil {
		return nil, err
	}

	log.V(1).Info("Starting token exchange", "tokenUrl", spec.Spec.TokenURL, "clientId", spec.Spec.ClientID)

	formData := g.buildTokenExchangeFormData(ctx, spec, scopes, subjectToken, kubeClient, targetNamespace)
	additionalHeaders := g.buildTokenExchangeHeaders(spec)

	request := authRequest{
		endpoint:          spec.Spec.TokenURL,
		formData:          formData,
		clientID:          spec.Spec.ClientID,
		clientSecret:      clientSecret,
		additionalHeaders: additionalHeaders,
		logMessage:        "Exchanging token with OIDC provider",
	}

	return executeTokenRequest(ctx, g.httpClient, request)
}

func (g *Generator) getSubjectToken(ctx context.Context, spec *genv1alpha1.OIDC, kubeClient client.Client, targetNamespace string) (string, error) {
	if spec.Spec.Grant.TokenExchange.ServiceAccountRef != nil {
		return fetchServiceAccountToken(ctx, *spec.Spec.Grant.TokenExchange.ServiceAccountRef, targetNamespace)
	}

	if spec.Spec.Grant.TokenExchange.SubjectTokenRef != nil {
		subjectToken, err := resolvers.SecretKeyRef(ctx, kubeClient, resolvers.EmptyStoreKind, targetNamespace, &esmeta.SecretKeySelector{
			Namespace: &targetNamespace,
			Name:      spec.Spec.Grant.TokenExchange.SubjectTokenRef.Name,
			Key:       spec.Spec.Grant.TokenExchange.SubjectTokenRef.Key,
		})
		if err != nil {
			return "", fmt.Errorf("failed to fetch subject token: %w", err)
		}
		return subjectToken, nil
	}

	return "", errors.New("either serviceAccountRef or subjectTokenRef must be specified for token exchange")
}

func (g *Generator) buildTokenExchangeFormData(ctx context.Context, spec *genv1alpha1.OIDC, scopes []string, subjectToken string, kubeClient client.Client, targetNamespace string) url.Values {
	formData := url.Values{}
	formData.Set("grant_type", grantTypeTokenExchange)
	formData.Set("scope", strings.Join(scopes, " "))
	formData.Set("subject_token", subjectToken)
	formData.Set("subject_token_type", spec.Spec.Grant.TokenExchange.SubjectTokenType)

	g.setOptionalTokenExchangeFields(formData, spec)
	g.addActorTokenIfPresent(ctx, formData, spec, kubeClient, targetNamespace)
	g.addAdditionalParameters(formData, spec)

	return formData
}

func (g *Generator) setOptionalTokenExchangeFields(formData url.Values, spec *genv1alpha1.OIDC) {
	if spec.Spec.Grant.TokenExchange.RequestedTokenType != "" {
		formData.Set("requested_token_type", spec.Spec.Grant.TokenExchange.RequestedTokenType)
	}
	if spec.Spec.Grant.TokenExchange.Audience != "" {
		formData.Set("audience", spec.Spec.Grant.TokenExchange.Audience)
	}
	if spec.Spec.Grant.TokenExchange.Resource != "" {
		formData.Set("resource", spec.Spec.Grant.TokenExchange.Resource)
	}
}

func (g *Generator) addActorTokenIfPresent(ctx context.Context, formData url.Values, spec *genv1alpha1.OIDC, kubeClient client.Client, targetNamespace string) {
	if spec.Spec.Grant.TokenExchange.ActorTokenRef == nil {
		return
	}

	actorToken, err := resolvers.SecretKeyRef(ctx, kubeClient, resolvers.EmptyStoreKind, targetNamespace, &esmeta.SecretKeySelector{
		Namespace: &targetNamespace,
		Name:      spec.Spec.Grant.TokenExchange.ActorTokenRef.Name,
		Key:       spec.Spec.Grant.TokenExchange.ActorTokenRef.Key,
	})
	if err != nil {
		return
	}

	formData.Set("actor_token", actorToken)
	if spec.Spec.Grant.TokenExchange.ActorTokenType != "" {
		formData.Set("actor_token_type", spec.Spec.Grant.TokenExchange.ActorTokenType)
	}
}

func (g *Generator) addAdditionalParameters(formData url.Values, spec *genv1alpha1.OIDC) {
	for key, value := range spec.Spec.Grant.TokenExchange.AdditionalParameters {
		formData.Set(key, value)
	}

	for key, value := range spec.Spec.AdditionalParameters {
		if formData.Get(key) == "" {
			formData.Set(key, value)
		}
	}
}

func (g *Generator) buildTokenExchangeHeaders(spec *genv1alpha1.OIDC) map[string]string {
	additionalHeaders := make(map[string]string)
	for key, value := range spec.Spec.AdditionalHeaders {
		additionalHeaders[key] = value
	}
	for key, value := range spec.Spec.Grant.TokenExchange.AdditionalHeaders {
		additionalHeaders[key] = value
	}
	return additionalHeaders
}

func executeTokenRequest(ctx context.Context, httpClient *http.Client, req authRequest) (*TokenResponse, error) {
	log := logr.FromContextOrDiscard(ctx)

	if httpClient == nil {
		httpClient = &http.Client{Timeout: httpClientTimeout}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.endpoint, strings.NewReader(req.formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf(errCreateTokenRequest, err)
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if req.clientID != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(req.clientID + ":" + req.clientSecret))
		httpReq.Header.Set("Authorization", "Basic "+auth)
	}

	for key, value := range req.additionalHeaders {
		httpReq.Header.Set(key, value)
	}

	log.Info(req.logMessage, "url", req.endpoint, "clientId", req.clientID)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Error(closeErr, "failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error(nil, "Authentication failed", "status", resp.Status, "body", string(body))
		return nil, fmt.Errorf(errUnexpectedStatus, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(errReadResponse, err)
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, fmt.Errorf(errUnmarshalResponse, err)
	}

	return &tokenResp, nil
}

func parseOIDCSpec(specData []byte) (*genv1alpha1.OIDC, error) {
	var spec genv1alpha1.OIDC
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
	genv1alpha1.Register(genv1alpha1.OIDCKind, &Generator{})
}
