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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestNewOIDCTokenManager_NilConfig(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Test with nil store
	manager := NewOIDCTokenManager(fakeClient.CoreV1(), nil, "default", esv1.SecretStoreKind)
	assert.Nil(t, manager)

	// Test with nil Auth
	store := &esv1.DopplerProvider{}
	manager = NewOIDCTokenManager(fakeClient.CoreV1(), store, "default", esv1.SecretStoreKind)
	assert.Nil(t, manager)

	// Test with nil OIDCConfig
	store.Auth = &esv1.DopplerAuth{}
	manager = NewOIDCTokenManager(fakeClient.CoreV1(), store, "default", esv1.SecretStoreKind)
	assert.Nil(t, manager)
}

func TestOIDCTokenManager_GetToken_NotInitialized(t *testing.T) {
	var manager *OIDCTokenManager
	_, err := manager.GetToken(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestOIDCTokenManager_ExchangeToken(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   map[string]interface{}
		responseStatus int
		wantError      bool
		errorContains  string
	}{
		{
			name: "successful exchange",
			responseBody: map[string]interface{}{
				"success":    true,
				"token":      "doppler_token_123",
				"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
			},
			responseStatus: http.StatusOK,
			wantError:      false,
		},
		{
			name: "failed exchange - success false",
			responseBody: map[string]interface{}{
				"success": false,
				"error":   "invalid identity",
			},
			responseStatus: http.StatusOK,
			wantError:      true,
			errorContains:  "Doppler OIDC auth failed",
		},
		{
			name: "unauthorized",
			responseBody: map[string]interface{}{
				"error": "invalid_token",
			},
			responseStatus: http.StatusUnauthorized,
			wantError:      true,
			errorContains:  "Doppler OIDC auth failed",
		},
		{
			name:           "server error",
			responseBody:   nil,
			responseStatus: http.StatusInternalServerError,
			wantError:      true,
			errorContains:  "Doppler OIDC auth failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				if tt.responseBody != nil {
					_ = json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			// Set the custom base URL env var to use the test server
			t.Setenv(customBaseURLEnvVar, server.URL)

			fakeClient := fake.NewSimpleClientset()
			store := &esv1.DopplerProvider{
				Auth: &esv1.DopplerAuth{
					OIDCConfig: &esv1.DopplerOIDCAuth{
						Identity: "test-identity",
						ServiceAccountRef: esmeta.ServiceAccountSelector{
							Name: "test-sa",
						},
					},
				},
			}

			manager := NewOIDCTokenManager(fakeClient.CoreV1(), store, "default", esv1.SecretStoreKind)
			require.NotNil(t, manager)

			token, _, err := manager.ExchangeToken(context.Background(), "k8s-token")

			if tt.wantError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestNewOIDCTokenManager_ValidConfig(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	expSec := int64(600)
	store := &esv1.DopplerProvider{
		Auth: &esv1.DopplerAuth{
			OIDCConfig: &esv1.DopplerOIDCAuth{
				Identity: "test-identity",
				ServiceAccountRef: esmeta.ServiceAccountSelector{
					Name: "test-sa",
				},
				ExpirationSeconds: &expSec,
			},
		},
	}

	manager := NewOIDCTokenManager(
		fakeClient.CoreV1(),
		store,
		"default",
		esv1.SecretStoreKind,
	)

	assert.NotNil(t, manager)
}
