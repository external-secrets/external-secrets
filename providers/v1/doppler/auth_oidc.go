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
	"time"

	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/oidc"
)

const dopplerOIDCPath = "/v3/auth/oidc"

// OIDCTokenManager manages OIDC token exchange with Doppler.
// It wraps the shared oidc.TokenManager with Doppler-specific token exchange logic.
type OIDCTokenManager struct {
	tokenManager *oidc.TokenManager
}

// dopplerTokenExchanger implements oidc.TokenExchanger for Doppler.
type dopplerTokenExchanger struct {
	baseURL   string
	identity  string
	verifyTLS bool
}

// NewOIDCTokenManager creates a new OIDCTokenManager for handling Doppler OIDC authentication.
func NewOIDCTokenManager(
	corev1 typedcorev1.CoreV1Interface,
	store *esv1.DopplerProvider,
	namespace string,
	storeKind string,
	storeName string,
) *OIDCTokenManager {
	if store == nil || store.Auth == nil || store.Auth.OIDCConfig == nil {
		return nil
	}

	oidcAuth := store.Auth.OIDCConfig

	baseURL := "https://api.doppler.com"
	if customURL := os.Getenv(customBaseURLEnvVar); customURL != "" {
		baseURL = customURL
	}

	verifyTLS := os.Getenv(verifyTLSOverrideEnvVar) != "false"

	saRef := oidc.ServiceAccountRef{
		Name:       oidcAuth.ServiceAccountRef.Name,
		Namespace:  oidcAuth.ServiceAccountRef.Namespace,
		Audiences:  oidcAuth.ServiceAccountRef.Audiences,
		Expiration: oidcAuth.ExpirationSeconds,
	}

	exchanger := &dopplerTokenExchanger{
		baseURL:   baseURL,
		identity:  oidcAuth.Identity,
		verifyTLS: verifyTLS,
	}

	return &OIDCTokenManager{
		tokenManager: oidc.NewTokenManager(
			corev1,
			namespace,
			storeKind,
			storeName,
			baseURL,
			saRef,
			exchanger,
		),
	}
}

// Token returns a valid Doppler API token, refreshing it if necessary.
func (m *OIDCTokenManager) Token(ctx context.Context) (string, error) {
	if m == nil || m.tokenManager == nil {
		return "", fmt.Errorf("OIDC token manager is not initialized")
	}
	return m.tokenManager.Token(ctx)
}

// ExchangeToken exchanges a ServiceAccount token for a Doppler API token.
func (e *dopplerTokenExchanger) ExchangeToken(ctx context.Context, saToken string) (string, time.Time, error) {
	url := e.baseURL + dopplerOIDCPath

	requestBody := map[string]string{
		"identity": e.identity,
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
	if !e.verifyTLS {
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
