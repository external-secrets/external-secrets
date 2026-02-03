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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	DefaultTokenTTL = 600
	MinTokenBuffer  = 60
)

// TokenProvider is the interface that provider-specific OIDC implementations must satisfy.
// Providers implement this interface to handle their own ServiceAccount token creation
// and token exchange logic.
type TokenProvider interface {
	// GetToken returns a valid access token, refreshing it if necessary.
	GetToken(ctx context.Context) (string, error)
}

// TokenExchanger is the interface that provider-specific token exchange implementations must satisfy.
type TokenExchanger interface {
	ExchangeToken(ctx context.Context, saToken string) (token string, expiry time.Time, err error)
}

// BaseTokenManager provides common OIDC token management functionality.
// Provider-specific implementations embed this struct and provide their own TokenExchanger.
type BaseTokenManager struct {
	Corev1    typedcorev1.CoreV1Interface
	Namespace string
	StoreKind string
	BaseURL   string
	SaRef     esmeta.ServiceAccountSelector
	Cache     *TokenCache
	Exchanger TokenExchanger
}

// GetToken returns a valid access token, refreshing it if necessary.
// This is the common implementation used by all OIDC providers.
func (m *BaseTokenManager) GetToken(ctx context.Context) (string, error) {
	if m == nil {
		return "", fmt.Errorf("OIDC token manager is not initialized")
	}

	// Check cache first
	if token, ok := m.Cache.Get(); ok {
		return token, nil
	}

	// Create ServiceAccount token
	saToken, err := m.CreateServiceAccountToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create service account token: %w", err)
	}

	// Exchange for provider-specific token
	token, expiry, err := m.Exchanger.ExchangeToken(ctx, saToken)
	if err != nil {
		return "", err
	}

	// Cache the token
	m.Cache.Set(token, expiry)

	return token, nil
}

// CreateServiceAccountToken creates a Kubernetes ServiceAccount token for OIDC authentication.
// This is the common implementation used by all OIDC providers.
func (m *BaseTokenManager) CreateServiceAccountToken(ctx context.Context) (string, error) {
	audiences := m.BuildAudiences()

	expirationSeconds := int64(DefaultTokenTTL)

	tokenRequest := &authv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: m.Namespace,
		},
		Spec: authv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}

	// For ClusterSecretStore, use the namespace from the ServiceAccountRef if specified
	tokenNamespace := m.Namespace
	if m.StoreKind == esv1.ClusterSecretStoreKind && m.SaRef.Namespace != nil {
		tokenNamespace = *m.SaRef.Namespace
	}

	tokenResponse, err := m.Corev1.ServiceAccounts(tokenNamespace).
		CreateToken(ctx, m.SaRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token for service account %s: %w",
			m.SaRef.Name, err)
	}

	return tokenResponse.Status.Token, nil
}

// BuildAudiences builds the audiences list for the ServiceAccount token.
// It starts with baseURL and adds custom audiences, deduplicating against baseURL.
func (m *BaseTokenManager) BuildAudiences() []string {
	audiences := []string{m.BaseURL}

	for _, aud := range m.SaRef.Audiences {
		if aud != m.BaseURL {
			audiences = append(audiences, aud)
		}
	}

	return audiences
}

// TokenCache provides thread-safe caching for OIDC tokens.
type TokenCache struct {
	mu          sync.RWMutex
	cachedToken string
	tokenExpiry time.Time
}

// NewTokenCache creates a new TokenCache.
func NewTokenCache() *TokenCache {
	return &TokenCache{}
}

// Get returns the cached token if it's still valid, otherwise returns empty string.
func (c *TokenCache) Get() (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cachedToken == "" {
		return "", false
	}
	if time.Until(c.tokenExpiry) <= MinTokenBuffer*time.Second {
		return "", false
	}
	return c.cachedToken, true
}

// Set stores a token with its expiry time.
func (c *TokenCache) Set(token string, expiry time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cachedToken = token
	c.tokenExpiry = expiry
}

// PostJSONRequest sends a POST request with JSON body and returns the response body.
// This is a shared utility for OIDC token exchange implementations.
func PostJSONRequest(ctx context.Context, url string, requestBody map[string]string, providerName string) ([]byte, error) {
	return postJSONRequestInternal(ctx, url, requestBody, providerName)
}

// PostJSONRequestInterface sends a POST request with JSON body (supporting interface{} values) and returns the response body.
// This is a shared utility for OIDC token exchange implementations that need non-string values in the request body.
func PostJSONRequestInterface(ctx context.Context, url string, requestBody map[string]interface{}, providerName string) ([]byte, error) {
	return postJSONRequestInternal(ctx, url, requestBody, providerName)
}

func postJSONRequestInternal(ctx context.Context, url string, requestBody interface{}, providerName string) ([]byte, error) {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Clone the default transport if possible, otherwise create a new one
	var transport *http.Transport
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = t.Clone()
	} else {
		transport = &http.Transport{}
	}
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to %s: %w", providerName, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s OIDC auth failed with status %d: %s",
			providerName, resp.StatusCode, string(body))
	}

	return body, nil
}
