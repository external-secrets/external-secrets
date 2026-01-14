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

package doppler

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	dopplerOIDCPath = "/v3/auth/oidc"
)

// OIDCTokenManager manages OIDC token exchange with Doppler.
type OIDCTokenManager struct {
	corev1    typedcorev1.CoreV1Interface
	store     *esv1.DopplerProvider
	namespace string
	storeKind string
	storeName string
	baseURL   string
	verifyTLS bool

	mu          sync.RWMutex
	cachedToken string
	tokenExpiry time.Time
}

// NewOIDCTokenManager creates a new OIDCTokenManager for handling Doppler OIDC authentication.
func NewOIDCTokenManager(
	corev1 typedcorev1.CoreV1Interface,
	store *esv1.DopplerProvider,
	namespace string,
	storeKind string,
	storeName string,
) *OIDCTokenManager {
	baseURL := "https://api.doppler.com"
	if customURL := os.Getenv(customBaseURLEnvVar); customURL != "" {
		baseURL = customURL
	}

	verifyTLS := os.Getenv(verifyTLSOverrideEnvVar) != "false"

	return &OIDCTokenManager{
		corev1:    corev1,
		store:     store,
		namespace: namespace,
		storeKind: storeKind,
		storeName: storeName,
		baseURL:   baseURL,
		verifyTLS: verifyTLS,
	}
}

// Token returns a valid Doppler API token, refreshing it if necessary.
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

	dopplerToken, expiry, err := m.exchangeTokenWithDoppler(ctx, saToken)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token with Doppler: %w", err)
	}

	m.cachedToken = dopplerToken
	m.tokenExpiry = expiry

	return dopplerToken, nil
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

func (m *OIDCTokenManager) exchangeTokenWithDoppler(ctx context.Context, saToken string) (string, time.Time, error) {
	oidcAuth := m.store.Auth.OIDCConfig
	url := m.baseURL + dopplerOIDCPath

	requestBody := map[string]string{
		"identity": oidcAuth.Identity,
		"token":    saToken,
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

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if !m.verifyTLS {
		tlsConfig.InsecureSkipVerify = true
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to make request to Doppler: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("Doppler OIDC auth failed with status %d: %s",
			resp.StatusCode, string(body))
	}

	var response struct {
		Success   bool   `json:"success"`
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.Success {
		return "", time.Time{}, fmt.Errorf("Doppler OIDC auth failed: %s", string(body))
	}

	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse expiration time: %w", err)
	}

	return response.Token, expiresAt, nil
}
