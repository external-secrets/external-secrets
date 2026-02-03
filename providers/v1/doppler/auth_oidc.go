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

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/oidc"
)

const dopplerOIDCPath = "/v3/auth/oidc"

// OIDCTokenManager manages OIDC token exchange with Doppler.
// It implements the oidc.TokenProvider interface.
type OIDCTokenManager struct {
	corev1    typedcorev1.CoreV1Interface
	namespace string
	storeKind string
	baseURL   string
	saRef     esmeta.ServiceAccountSelector
	identity  string
	cache     *oidc.TokenCache
}

// NewOIDCTokenManager creates a new OIDCTokenManager for handling Doppler OIDC authentication.
func NewOIDCTokenManager(
	corev1 typedcorev1.CoreV1Interface,
	store *esv1.DopplerProvider,
	namespace string,
	storeKind string,
) *OIDCTokenManager {
	if store == nil || store.Auth == nil || store.Auth.OIDCConfig == nil {
		return nil
	}

	oidcAuth := store.Auth.OIDCConfig

	baseURL := "https://api.doppler.com"
	if customURL := os.Getenv(customBaseURLEnvVar); customURL != "" {
		baseURL = customURL
	}

	return &OIDCTokenManager{
		corev1:    corev1,
		namespace: namespace,
		storeKind: storeKind,
		baseURL:   baseURL,
		saRef:     oidcAuth.ServiceAccountRef,
		identity:  oidcAuth.Identity,
		cache:     oidc.NewTokenCache(),
	}
}

// GetToken returns a valid Doppler API token, refreshing it if necessary.
// This implements the oidc.TokenProvider interface.
func (m *OIDCTokenManager) GetToken(ctx context.Context) (string, error) {
	if m == nil {
		return "", fmt.Errorf("OIDC token manager is not initialized")
	}

	// Check cache first
	if token, ok := m.cache.Get(); ok {
		return token, nil
	}

	// Create ServiceAccount token
	saToken, err := m.createServiceAccountToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create service account token: %w", err)
	}

	// Exchange for Doppler token
	token, expiry, err := m.exchangeToken(ctx, saToken)
	if err != nil {
		return "", err
	}

	// Cache the token
	m.cache.Set(token, expiry)

	return token, nil
}

// createServiceAccountToken creates a Kubernetes ServiceAccount token for OIDC authentication.
func (m *OIDCTokenManager) createServiceAccountToken(ctx context.Context) (string, error) {
	audiences := []string{m.baseURL}

	if len(m.saRef.Audiences) > 0 {
		audiences = append(audiences, m.saRef.Audiences...)
	}

	expirationSeconds := int64(oidc.DefaultTokenTTL)

	tokenRequest := &authv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: m.namespace,
		},
		Spec: authv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}

	// For ClusterSecretStore, use the namespace from the ServiceAccountRef if specified
	tokenNamespace := m.namespace
	if m.storeKind == esv1.ClusterSecretStoreKind && m.saRef.Namespace != nil {
		tokenNamespace = *m.saRef.Namespace
	}

	tokenResponse, err := m.corev1.ServiceAccounts(tokenNamespace).
		CreateToken(ctx, m.saRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token for service account %s: %w",
			m.saRef.Name, err)
	}

	return tokenResponse.Status.Token, nil
}

// exchangeToken exchanges a ServiceAccount token for a Doppler API token.
func (m *OIDCTokenManager) exchangeToken(ctx context.Context, saToken string) (string, time.Time, error) {
	url := m.baseURL + dopplerOIDCPath

	requestBody := map[string]string{
		"identity": m.identity,
		"token":    saToken,
	}

	body, err := oidc.PostJSONRequest(ctx, url, requestBody, "Doppler")
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
