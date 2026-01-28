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

func TestOIDCTokenManager_IsTokenValid(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	expSec := int64(600)
	store := &esv1.PulumiProvider{
		APIURL:       "https://api.pulumi.com/api/esc",
		Organization: "test-org",
		Auth: &esv1.PulumiAuth{
			OIDCConfig: &esv1.PulumiOIDCAuth{
				Organization: "test-org",
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
		"test-store",
	)

	// Test with no cached token
	assert.False(t, manager.isTokenValid())

	// Test with cached token that's expired
	manager.cachedToken = "test-token"
	manager.tokenExpiry = time.Now().Add(-1 * time.Hour)
	assert.False(t, manager.isTokenValid())

	// Test with cached token that's within buffer time
	manager.tokenExpiry = time.Now().Add(30 * time.Second)
	assert.False(t, manager.isTokenValid())

	// Test with cached token that's valid
	manager.tokenExpiry = time.Now().Add(10 * time.Minute)
	assert.True(t, manager.isTokenValid())
}

func TestOIDCTokenManager_ExchangeResponse(t *testing.T) {
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
				"access_token": "pul-test-token",
				"expires_in":   3600,
			},
			responseStatus: http.StatusOK,
			wantError:      false,
		},
		{
			name: "missing access_token",
			responseBody: map[string]interface{}{
				"expires_in": 3600,
			},
			responseStatus: http.StatusOK,
			wantError:      true,
			errorContains:  "no access_token",
		},
		{
			name: "unauthorized",
			responseBody: map[string]interface{}{
				"error": "invalid_token",
			},
			responseStatus: http.StatusUnauthorized,
			wantError:      true,
			errorContains:  "Pulumi OIDC auth failed",
		},
		{
			name:           "server error",
			responseBody:   nil,
			responseStatus: http.StatusInternalServerError,
			wantError:      true,
			errorContains:  "Pulumi OIDC auth failed",
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

			fakeClient := fake.NewSimpleClientset()
			expSec := int64(600)
			store := &esv1.PulumiProvider{
				APIURL:       server.URL + "/api/esc",
				Organization: "test-org",
				Auth: &esv1.PulumiAuth{
					OIDCConfig: &esv1.PulumiOIDCAuth{
						Organization: "test-org",
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
				"test-store",
			)

			token, _, err := manager.exchangeTokenWithPulumi(context.Background(), "k8s-token")

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

func TestNewOIDCTokenManager_BaseURLParsing(t *testing.T) {
	tests := []struct {
		name            string
		apiURL          string
		expectedBaseURL string
	}{
		{
			name:            "standard API URL",
			apiURL:          "https://api.pulumi.com/api/esc",
			expectedBaseURL: "https://api.pulumi.com",
		},
		{
			name:            "custom API URL",
			apiURL:          "https://custom.pulumi.io/api/esc",
			expectedBaseURL: "https://custom.pulumi.io",
		},
		{
			name:            "base URL without /api/esc",
			apiURL:          "https://api.pulumi.com",
			expectedBaseURL: "https://api.pulumi.com",
		},
		{
			name:            "empty URL",
			apiURL:          "",
			expectedBaseURL: "https://api.pulumi.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset()
			expSec := int64(600)
			store := &esv1.PulumiProvider{
				APIURL:       tt.apiURL,
				Organization: "test-org",
				Auth: &esv1.PulumiAuth{
					OIDCConfig: &esv1.PulumiOIDCAuth{
						Organization: "test-org",
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
				"test-store",
			)

			assert.Equal(t, tt.expectedBaseURL, manager.baseURL)
		})
	}
}
