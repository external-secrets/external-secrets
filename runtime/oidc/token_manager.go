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
	"fmt"
	"sync"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	DefaultTokenTTL = 600
	MinTokenBuffer  = 60
)

// ServiceAccountRef contains the reference to a Kubernetes ServiceAccount for OIDC authentication.
type ServiceAccountRef struct {
	Name       string
	Namespace  *string
	Audiences  []string
	Expiration *int64
}

// TokenExchanger is the interface that provider-specific token exchange implementations must satisfy.
type TokenExchanger interface {
	ExchangeToken(ctx context.Context, saToken string) (token string, expiry time.Time, err error)
}

// TokenManager manages OIDC token exchange with caching and automatic refresh.
type TokenManager struct {
	corev1         typedcorev1.CoreV1Interface
	namespace      string
	storeKind      string
	storeName      string
	baseURL        string
	saRef          ServiceAccountRef
	tokenExchanger TokenExchanger

	mu          sync.RWMutex
	cachedToken string
	tokenExpiry time.Time
}

// NewTokenManager creates a new TokenManager for handling OIDC authentication.
func NewTokenManager(
	corev1 typedcorev1.CoreV1Interface,
	namespace string,
	storeKind string,
	storeName string,
	baseURL string,
	saRef ServiceAccountRef,
	exchanger TokenExchanger,
) *TokenManager {
	return &TokenManager{
		corev1:         corev1,
		namespace:      namespace,
		storeKind:      storeKind,
		storeName:      storeName,
		baseURL:        baseURL,
		saRef:          saRef,
		tokenExchanger: exchanger,
	}
}

// Token returns a valid access token, refreshing it if necessary.
func (m *TokenManager) Token(ctx context.Context) (string, error) {
	m.mu.RLock()
	if m.isTokenValid() {
		token := m.cachedToken
		m.mu.RUnlock()
		return token, nil
	}
	m.mu.RUnlock()

	return m.refreshToken(ctx)
}

func (m *TokenManager) isTokenValid() bool {
	if m.cachedToken == "" {
		return false
	}
	return time.Until(m.tokenExpiry) > MinTokenBuffer*time.Second
}

func (m *TokenManager) refreshToken(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isTokenValid() {
		return m.cachedToken, nil
	}

	saToken, err := m.CreateServiceAccountToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create service account token: %w", err)
	}

	token, expiry, err := m.tokenExchanger.ExchangeToken(ctx, saToken)
	if err != nil {
		return "", err
	}

	m.cachedToken = token
	m.tokenExpiry = expiry

	return token, nil
}

// CreateServiceAccountToken creates a Kubernetes ServiceAccount token for OIDC authentication.
func (m *TokenManager) CreateServiceAccountToken(ctx context.Context) (string, error) {
	audiences := []string{m.baseURL}

	if len(m.saRef.Audiences) > 0 {
		audiences = append(audiences, m.saRef.Audiences...)
	}

	if m.storeKind == esv1.ClusterSecretStoreKind {
		audiences = append(audiences, fmt.Sprintf("clusterSecretStore:%s", m.storeName))
	} else {
		audiences = append(audiences, fmt.Sprintf("secretStore:%s:%s", m.namespace, m.storeName))
	}

	expirationSeconds := m.saRef.Expiration
	if expirationSeconds == nil {
		tmp := int64(DefaultTokenTTL)
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

	if m.storeKind == esv1.ClusterSecretStoreKind && m.saRef.Namespace != nil {
		tokenRequest.Namespace = *m.saRef.Namespace
	}

	tokenResponse, err := m.corev1.ServiceAccounts(tokenRequest.Namespace).
		CreateToken(ctx, m.saRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token for service account %s: %w",
			m.saRef.Name, err)
	}

	return tokenResponse.Status.Token, nil
}
