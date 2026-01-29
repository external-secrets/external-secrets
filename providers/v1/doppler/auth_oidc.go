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
	"context"
	"encoding/json"
	"fmt"
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

	config := &oidc.HTTPClientConfig{
		VerifyTLS: e.verifyTLS,
	}

	body, err := oidc.PostJSONRequest(ctx, url, requestBody, "Doppler", config)
	if err != nil {
		return "", time.Time{}, err
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
