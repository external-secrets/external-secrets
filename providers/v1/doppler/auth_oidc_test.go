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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/util/fake"
)

func TestOIDCTokenManager_Token(t *testing.T) {
	// Mock Doppler OIDC endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v3/auth/oidc" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Return a token that expires in 1 hour
			expiresAt := time.Now().Add(time.Hour).Format(time.RFC3339)
			if _, err := w.Write([]byte(`{"success": true, "token": "doppler_token_123", "expires_at": "` + expiresAt + `"}`)); err != nil {
				t.Errorf("failed to write response: %v", err)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := &esv1.DopplerProvider{
		Auth: &esv1.DopplerAuth{
			OIDCConfig: &esv1.DopplerOIDCAuth{
				Identity:          "test-identity",
				ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "test-sa"},
				ExpirationSeconds: func() *int64 { v := int64(600); return &v }(),
			},
		},
	}

	manager := &OIDCTokenManager{
		corev1:    fake.NewCreateTokenMock().WithToken("k8s_jwt_token"),
		store:     store,
		namespace: "test-namespace",
		storeKind: "SecretStore",
		storeName: "test-store",
		baseURL:   server.URL,
		verifyTLS: false,
	}

	ctx := context.Background()

	// First call should fetch a new token
	token1, err := manager.Token(ctx)
	require.NoError(t, err)
	assert.Equal(t, "doppler_token_123", token1)

	// Second call should return cached token
	token2, err := manager.Token(ctx)
	require.NoError(t, err)
	assert.Equal(t, token1, token2)
}

func TestOIDCTokenManager_CreateServiceAccountToken(t *testing.T) {
	store := &esv1.DopplerProvider{
		Auth: &esv1.DopplerAuth{
			OIDCConfig: &esv1.DopplerOIDCAuth{
				Identity:          "test-identity",
				ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "test-sa", Namespace: func() *string { s := "custom-ns"; return &s }()},
				ExpirationSeconds: func() *int64 { v := int64(600); return &v }(),
			},
		},
	}

	manager := &OIDCTokenManager{
		corev1:    fake.NewCreateTokenMock().WithToken("k8s_jwt_token"),
		store:     store,
		namespace: "default-namespace",
		storeKind: "SecretStore",
		storeName: "test-store",
		baseURL:   "https://api.doppler.com",
		verifyTLS: true,
	}

	token, err := manager.createServiceAccountToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "k8s_jwt_token", token)
}

func TestOIDCTokenManager_TokenExpiry(t *testing.T) {
	manager := &OIDCTokenManager{
		cachedToken: "test_token",
		tokenExpiry: time.Now().Add(30 * time.Second), // Token expires in 30 seconds
	}

	// Token should be considered invalid (less than 60 second buffer)
	assert.False(t, manager.isTokenValid())

	// Token with more time should be valid
	manager.tokenExpiry = time.Now().Add(2 * time.Minute)
	assert.True(t, manager.isTokenValid())

	// Empty token should be invalid
	manager.cachedToken = ""
	assert.False(t, manager.isTokenValid())
}
