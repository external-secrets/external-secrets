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
	"time"

	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/oidc"
)

const pulumiOIDCPath = "/api/oauth/oidc/token"

// OIDCTokenManager manages OIDC token exchange with Pulumi.
// It wraps the shared oidc.TokenManager with Pulumi-specific token exchange logic.
type OIDCTokenManager struct {
	tokenManager *oidc.TokenManager
}

// pulumiTokenExchanger implements oidc.TokenExchanger for Pulumi.
type pulumiTokenExchanger struct {
	baseURL      string
	organization string
}

// NewOIDCTokenManager creates a new OIDCTokenManager for handling Pulumi OIDC authentication.
func NewOIDCTokenManager(
	corev1 typedcorev1.CoreV1Interface,
	store *esv1.PulumiProvider,
	namespace string,
	storeKind string,
	storeName string,
) *OIDCTokenManager {
	if store == nil || store.Auth == nil || store.Auth.OIDCConfig == nil {
		return nil
	}

	oidcAuth := store.Auth.OIDCConfig

	// Normalize the URL first by trimming trailing slash, then remove /api/esc suffix
	apiURL := strings.TrimSuffix(store.APIURL, "/")
	baseURL := strings.TrimSuffix(apiURL, "/api/esc")
	if baseURL == "" {
		baseURL = "https://api.pulumi.com"
	}

	saRef := oidc.ServiceAccountRef{
		Name:       oidcAuth.ServiceAccountRef.Name,
		Namespace:  oidcAuth.ServiceAccountRef.Namespace,
		Audiences:  oidcAuth.ServiceAccountRef.Audiences,
		Expiration: oidcAuth.ExpirationSeconds,
	}

	exchanger := &pulumiTokenExchanger{
		baseURL:      baseURL,
		organization: oidcAuth.Organization,
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

// Token returns a valid Pulumi access token, refreshing it if necessary.
func (m *OIDCTokenManager) Token(ctx context.Context) (string, error) {
	if m == nil || m.tokenManager == nil {
		return "", fmt.Errorf("OIDC token manager is not initialized")
	}
	return m.tokenManager.Token(ctx)
}

// ExchangeToken exchanges a ServiceAccount token for a Pulumi access token.
func (e *pulumiTokenExchanger) ExchangeToken(ctx context.Context, saToken string) (string, time.Time, error) {
	url := e.baseURL + pulumiOIDCPath

	requestBody := map[string]string{
		"organization": e.organization,
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
