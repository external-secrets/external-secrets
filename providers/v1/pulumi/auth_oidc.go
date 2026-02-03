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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/oidc"
)

// Pulumi OAuth token exchange endpoint and constants per:
// https://www.pulumi.com/docs/reference/cloud-rest-api/oauth-token-exchange/
const (
	pulumiOAuthPath             = "/api/oauth/token"
	pulumiGrantType             = "urn:ietf:params:oauth:grant-type:token-exchange"
	pulumiSubjectTokenType      = "urn:ietf:params:oauth:token-type:id_token"
	pulumiRequestedTokenTypeOrg = "urn:pulumi:token-type:access_token:organization"
)

// OIDCTokenManager manages OIDC token exchange with Pulumi.
// It implements the oidc.TokenProvider interface.
type OIDCTokenManager struct {
	corev1       typedcorev1.CoreV1Interface
	namespace    string
	storeKind    string
	baseURL      string
	saRef        esmeta.ServiceAccountSelector
	organization string
	expiration   int64
	cache        *oidc.TokenCache
}

// NewOIDCTokenManager creates a new OIDCTokenManager for handling Pulumi OIDC authentication.
func NewOIDCTokenManager(
	corev1 typedcorev1.CoreV1Interface,
	store *esv1.PulumiProvider,
	namespace string,
	storeKind string,
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

	// Get expiration from config, default to 3600 seconds (1 hour) if not set
	expiration := int64(3600)
	if oidcAuth.ExpirationSeconds != nil && *oidcAuth.ExpirationSeconds > 0 {
		expiration = *oidcAuth.ExpirationSeconds
	}

	return &OIDCTokenManager{
		corev1:       corev1,
		namespace:    namespace,
		storeKind:    storeKind,
		baseURL:      baseURL,
		saRef:        oidcAuth.ServiceAccountRef,
		organization: oidcAuth.Organization,
		expiration:   expiration,
		cache:        oidc.NewTokenCache(),
	}
}

// GetToken returns a valid Pulumi access token, refreshing it if necessary.
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

	// Exchange for Pulumi token
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

// exchangeToken exchanges a ServiceAccount token for a Pulumi access token using the
// OAuth 2.0 Token Exchange flow per RFC 8693.
// See: https://www.pulumi.com/docs/reference/cloud-rest-api/oauth-token-exchange/
func (m *OIDCTokenManager) exchangeToken(ctx context.Context, saToken string) (string, time.Time, error) {
	url := m.baseURL + pulumiOAuthPath

	// Build the OAuth 2.0 Token Exchange request per Pulumi's API specification
	requestBody := map[string]interface{}{
		"audience":             fmt.Sprintf("urn:pulumi:org:%s", m.organization),
		"grant_type":           pulumiGrantType,
		"subject_token_type":   pulumiSubjectTokenType,
		"requested_token_type": pulumiRequestedTokenTypeOrg,
		"subject_token":        saToken,
		"expiration":           m.expiration,
		"scope":                "",
	}

	body, err := oidc.PostJSONRequestInterface(ctx, url, requestBody, "Pulumi")
	if err != nil {
		return "", time.Time{}, err
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

	if response.ExpiresIn <= 0 {
		return "", time.Time{}, fmt.Errorf("Pulumi OIDC auth failed: invalid expires_in value %d", response.ExpiresIn)
	}

	expiresAt := time.Now().Add(time.Duration(response.ExpiresIn) * time.Second)

	return response.AccessToken, expiresAt, nil
}
