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

package pulumi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	defaultTokenTTL = 600
	minTokenBuffer  = 60
	pulumiOIDCPath  = "/api/oauth/oidc/token"
)

// OIDCTokenManager manages OIDC token exchange with Pulumi.
type OIDCTokenManager struct {
	corev1    typedcorev1.CoreV1Interface
	store     *esv1.PulumiProvider
	namespace string
	storeKind string
	storeName string
	baseURL   string

	mu          sync.RWMutex
	cachedToken string
	tokenExpiry time.Time
}

// NewOIDCTokenManager creates a new OIDCTokenManager for handling Pulumi OIDC authentication.
func NewOIDCTokenManager(
	corev1 typedcorev1.CoreV1Interface,
	store *esv1.PulumiProvider,
	namespace string,
	storeKind string,
	storeName string,
) *OIDCTokenManager {
	baseURL := strings.TrimSuffix(store.APIURL, "/api/esc")
	if baseURL == store.APIURL {
		// APIURL doesn't end with /api/esc, assume it's a base URL
		baseURL = strings.TrimSuffix(store.APIURL, "/")
	}
	if baseURL == "" {
		baseURL = "https://api.pulumi.com"
	}

	return &OIDCTokenManager{
		corev1:    corev1,
		store:     store,
		namespace: namespace,
		storeKind: storeKind,
		storeName: storeName,
		baseURL:   baseURL,
	}
}

// Token returns a valid Pulumi access token, refreshing it if necessary.
func (m *OIDCTokenManager) Token(ctx context.Context) (string, error) {
	m.mu.RLock()
	if m.isTokenValid() {
		token := m.cachedToken
		m.mu.RUnlock()
		return token, nil
	}
	m.mu.RUnlock()

	return m.refreshToken(ctx)
}

func (m *OIDCTokenManager) isTokenValid() bool {
	if m.cachedToken == "" {
		return false
	}
	return time.Until(m.tokenExpiry) > minTokenBuffer*time.Second
}

func (m *OIDCTokenManager) refreshToken(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isTokenValid() {
		return m.cachedToken, nil
	}

	saToken, err := m.createServiceAccountToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create service account token: %w", err)
	}

	pulumiToken, expiry, err := m.exchangeTokenWithPulumi(ctx, saToken)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token with Pulumi: %w", err)
	}

	m.cachedToken = pulumiToken
	m.tokenExpiry = expiry

	return pulumiToken, nil
}

func (m *OIDCTokenManager) createServiceAccountToken(ctx context.Context) (string, error) {
	oidcAuth := m.store.Auth.OIDCConfig

	audiences := []string{m.baseURL}

	// Add custom audiences from serviceAccountRef
	if len(oidcAuth.ServiceAccountRef.Audiences) > 0 {
		audiences = append(audiences, oidcAuth.ServiceAccountRef.Audiences...)
	}

	// Add resource-specific audience for cryptographic binding
	if m.storeKind == esv1.ClusterSecretStoreKind {
		audiences = append(audiences, fmt.Sprintf("clusterSecretStore:%s", m.storeName))
	} else {
		audiences = append(audiences, fmt.Sprintf("secretStore:%s:%s", m.namespace, m.storeName))
	}

	expirationSeconds := oidcAuth.ExpirationSeconds
	if expirationSeconds == nil {
		tmp := int64(defaultTokenTTL)
		expirationSeconds = &tmp
	}

	tokenRequest := &authv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: m.namespace,
		},
		Spec: authv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: expirationSeconds,
		},
	}

	// For ClusterSecretStores, we use the ServiceAccountRef.Namespace if specified
	if m.storeKind == esv1.ClusterSecretStoreKind && oidcAuth.ServiceAccountRef.Namespace != nil {
		tokenRequest.Namespace = *oidcAuth.ServiceAccountRef.Namespace
	}

	tokenResponse, err := m.corev1.ServiceAccounts(tokenRequest.Namespace).
		CreateToken(ctx, oidcAuth.ServiceAccountRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token for service account %s: %w",
			oidcAuth.ServiceAccountRef.Name, err)
	}

	return tokenResponse.Status.Token, nil
}

func (m *OIDCTokenManager) exchangeTokenWithPulumi(ctx context.Context, saToken string) (string, time.Time, error) {
	oidcAuth := m.store.Auth.OIDCConfig
	url := m.baseURL + pulumiOIDCPath

	requestBody := map[string]string{
		"organization": oidcAuth.Organization,
		"token":        saToken,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Clone http.DefaultTransport to preserve proxy settings, connection pooling, and other defaults
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to make request to Pulumi: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("Pulumi OIDC auth failed with status %d: %s",
			resp.StatusCode, string(body))
	}

	var response struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse response: %w", err)
	}

	if response.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("Pulumi OIDC auth failed: no access_token in response")
	}

	// Validate expires_in to prevent rapid refresh loops
	if response.ExpiresIn <= 0 {
		return "", time.Time{}, fmt.Errorf("Pulumi OIDC auth failed: invalid expires_in value %d", response.ExpiresIn)
	}

	// Calculate expiry time based on expires_in (in seconds)
	expiresAt := time.Now().Add(time.Duration(response.ExpiresIn) * time.Second)

	return response.AccessToken, expiresAt, nil
}
