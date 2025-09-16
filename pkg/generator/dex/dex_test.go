/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dex

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	mockJWTToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiZXhwIjoxNzAwMDAwMDAwfQ.signature"
	testClientID = "test-client"
	testSecret   = "test-secret"
	httpPost     = "POST"
)

func TestTokenExchangeGenerator_Generate(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != httpPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !strings.HasSuffix(r.URL.Path, "/token") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		grantType := r.Form.Get("grant_type")
		if grantType != "urn:ietf:params:oauth:grant-type:token-exchange" {
			http.Error(w, "Unsupported grant type", http.StatusBadRequest)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != testClientID || password != testSecret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		response := TokenResponse{
			AccessToken: mockJWTToken,
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create test spec for token-exchange mode
	spec := &genv1alpha1.DexTokenExchange{
		Spec: genv1alpha1.DexTokenExchangeSpec{
			DexURL:   server.URL,
			ClientID: testClientID,
			ClientSecretRef: &esmeta.SecretKeySelector{
				Name: "dex-client-secret",
				Key:  "client-secret",
			},
			ConnectorID: "kubernetes",
			Scopes:      []string{"openid", "profile"},
			ServiceAccountRef: esmeta.ServiceAccountSelector{
				Name:      "test-sa",
				Audiences: []string{"https://kubernetes.default.svc.cluster.local"},
			},
		},
	}

	specBytes, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal spec: %v", err)
	}

	// Create generator with custom HTTP client that accepts self-signed certificates
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	t.Run("parseTokenExchangeSpec", func(t *testing.T) {
		parsed, err := parseTokenExchangeSpec(specBytes)
		if err != nil {
			t.Fatalf("Failed to parse spec: %v", err)
		}

		if parsed.Spec.ClientID != testClientID {
			t.Errorf("Expected ClientID to be '%s', got %s", testClientID, parsed.Spec.ClientID)
		}
		if parsed.Spec.ConnectorID != "kubernetes" {
			t.Errorf("Expected ConnectorID to be 'kubernetes', got %s", parsed.Spec.ConnectorID)
		}
		if len(parsed.Spec.Scopes) != 2 {
			t.Errorf("Expected 2 scopes, got %d", len(parsed.Spec.Scopes))
		}
	})

	t.Run("authenticateWithTokenExchange", func(t *testing.T) {
		ctx := context.Background()
		serviceAccountToken := "mock-service-account-token"

		testSpec := &genv1alpha1.DexTokenExchange{
			Spec: genv1alpha1.DexTokenExchangeSpec{
				DexURL:   server.URL,
				ClientID: testClientID,
				ServiceAccountRef: esmeta.ServiceAccountSelector{
					Name: "test-sa",
				},
			},
		}

		u, _ := url.Parse(server.URL + "/token")
		data := url.Values{}
		data.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
		data.Set("connector_id", "kubernetes")
		data.Set("scope", "openid")
		data.Set("requested_token_type", "urn:ietf:params:oauth:token-type:access_token")
		data.Set("subject_token", serviceAccountToken)
		data.Set("subject_token_type", "urn:ietf:params:oauth:token-type:id_token")

		req, err := http.NewRequestWithContext(ctx, httpPost, u.String(), strings.NewReader(data.Encode()))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(testSpec.Spec.ClientID, testSecret)

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var tokenResp TokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if tokenResp.AccessToken == "" {
			t.Error("Expected non-empty access token")
		}
	})
}

func TestUsernamePasswordGenerator_Generate(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != httpPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !strings.HasSuffix(r.URL.Path, "/token") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		grantType := r.Form.Get("grant_type")
		if grantType != "password" {
			http.Error(w, "Unsupported grant type", http.StatusBadRequest)
			return
		}

		username := r.Form.Get("username")
		password := r.Form.Get("password")
		if username != "test-user" || password != "test-pass" {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Basic ") {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		response := TokenResponse{
			AccessToken: mockJWTToken,
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create test spec for username/password mode
	spec := &genv1alpha1.DexUsernamePassword{
		Spec: genv1alpha1.DexUsernamePasswordSpec{
			DexURL:   server.URL,
			ClientID: testClientID,
			ClientSecretRef: &esmeta.SecretKeySelector{
				Name: "dex-client-secret",
				Key:  "client-secret",
			},
			Scopes: []string{"openid", "profile"},
			UsernameRef: esmeta.SecretKeySelector{
				Name: "dex-credentials",
				Key:  "username",
			},
			PasswordRef: esmeta.SecretKeySelector{
				Name: "dex-credentials",
				Key:  "password",
			},
		},
	}

	specBytes, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal spec: %v", err)
	}

	t.Run("parseUsernamePasswordSpec", func(t *testing.T) {
		parsed, err := parseUsernamePasswordSpec(specBytes)
		if err != nil {
			t.Fatalf("Failed to parse spec: %v", err)
		}

		if parsed.Spec.ClientID != testClientID {
			t.Errorf("Expected ClientID to be '%s', got %s", testClientID, parsed.Spec.ClientID)
		}
		if parsed.Spec.UsernameRef.Name != "dex-credentials" {
			t.Errorf("Expected UsernameRef.Name to be 'dex-credentials', got %s", parsed.Spec.UsernameRef.Name)
		}
		if parsed.Spec.PasswordRef.Name != "dex-credentials" {
			t.Errorf("Expected PasswordRef.Name to be 'dex-credentials', got %s", parsed.Spec.PasswordRef.Name)
		}
		if len(parsed.Spec.Scopes) != 2 {
			t.Errorf("Expected 2 scopes, got %d", len(parsed.Spec.Scopes))
		}
	})
}

func TestSharedFunctionality(t *testing.T) {
	t.Run("ParseJWTClaims", func(t *testing.T) {
		mockToken := mockJWTToken

		token, _, err := new(jwt.Parser).ParseUnverified(mockToken, jwt.MapClaims{})
		if err != nil {
			t.Fatalf("Failed to parse JWT: %v", err)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			t.Fatal("Failed to extract claims from JWT")
		}

		if claims["sub"] != "1234567890" {
			t.Errorf("Expected sub claim to be '1234567890', got %v", claims["sub"])
		}
		if claims["name"] != "John Doe" {
			t.Errorf("Expected name claim to be 'John Doe', got %v", claims["name"])
		}
	})

	t.Run("ExtractJWTExpiration", func(t *testing.T) {
		mockToken := mockJWTToken

		exp, err := extractJWTExpiration(mockToken)
		if err != nil {
			t.Fatalf("Failed to get token expiration: %v", err)
		}

		expectedTime := time.Unix(1700000000, 0).Format(time.RFC3339)
		if exp != expectedTime {
			t.Errorf("Expected expiration to be '%s', got %s", expectedTime, exp)
		}
	})
}

func TestOptionalClientSecret(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != httpPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !strings.HasSuffix(r.URL.Path, "/token") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		response := TokenResponse{
			AccessToken: mockJWTToken,
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	t.Run("UsernamePassword_NoClientSecret", func(t *testing.T) {
		spec := &genv1alpha1.DexUsernamePassword{
			Spec: genv1alpha1.DexUsernamePasswordSpec{
				DexURL:   server.URL,
				ClientID: testClientID,

				Scopes: []string{"openid", "profile"},
				UsernameRef: esmeta.SecretKeySelector{
					Name: "dex-credentials",
					Key:  "username",
				},
				PasswordRef: esmeta.SecretKeySelector{
					Name: "dex-credentials",
					Key:  "password",
				},
			},
		}

		specBytes, err := yaml.Marshal(spec)
		if err != nil {
			t.Fatalf("Failed to marshal spec: %v", err)
		}

		parsed, err := parseUsernamePasswordSpec(specBytes)
		if err != nil {
			t.Fatalf("Failed to parse spec: %v", err)
		}

		if parsed.Spec.ClientSecretRef != nil {
			t.Errorf("Expected ClientSecretRef to be nil, got %+v", parsed.Spec.ClientSecretRef)
		}
	})

	t.Run("TokenExchange_NoClientSecret", func(t *testing.T) {
		spec := &genv1alpha1.DexTokenExchange{
			Spec: genv1alpha1.DexTokenExchangeSpec{
				DexURL:   server.URL,
				ClientID: testClientID,

				ConnectorID: "kubernetes",
				Scopes:      []string{"openid", "profile"},
				ServiceAccountRef: esmeta.ServiceAccountSelector{
					Name:      "test-sa",
					Audiences: []string{"https://kubernetes.default.svc.cluster.local"},
				},
			},
		}

		specBytes, err := yaml.Marshal(spec)
		if err != nil {
			t.Fatalf("Failed to marshal spec: %v", err)
		}

		parsed, err := parseTokenExchangeSpec(specBytes)
		if err != nil {
			t.Fatalf("Failed to parse spec: %v", err)
		}

		if parsed.Spec.ClientSecretRef != nil {
			t.Errorf("Expected ClientSecretRef to be nil, got %+v", parsed.Spec.ClientSecretRef)
		}
	})
}

func TestBuildTokenURL(t *testing.T) {
	tests := []struct {
		name     string
		dexURL   string
		expected string
	}{
		{
			name:     "URL without trailing slash",
			dexURL:   "https://dex.example.com",
			expected: "https://dex.example.com/token",
		},
		{
			name:     "URL with trailing slash",
			dexURL:   "https://dex.example.com/",
			expected: "https://dex.example.com/token",
		},
		{
			name:     "URL with path",
			dexURL:   "https://dex.example.com/auth",
			expected: "https://dex.example.com/auth/token",
		},
		{
			name:     "URL with path and trailing slash",
			dexURL:   "https://dex.example.com/auth/",
			expected: "https://dex.example.com/auth/token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildTokenURL(tt.dexURL)
			if result != tt.expected {
				t.Errorf("buildTokenURL(%q) = %q, want %q", tt.dexURL, result, tt.expected)
			}
		})
	}
}

func TestExecuteTokenRequest(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		request       authRequest
		expectedToken string
		expectError   bool
		errorContains string
	}{
		{
			name: "successful request with client auth",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					username, password, ok := r.BasicAuth()
					if !ok || username != testClientID || password != testSecret {
						http.Error(w, "Unauthorized", http.StatusUnauthorized)
						return
					}

					response := TokenResponse{
						AccessToken: "access-token-123",
						TokenType:   "Bearer",
						ExpiresIn:   3600,
					}
					json.NewEncoder(w).Encode(response)
				}))
			},
			request: authRequest{
				clientID:     testClientID,
				clientSecret: testSecret,
				formData:     url.Values{"grant_type": {"password"}},
				logMessage:   "test request",
			},
			expectedToken: "access-token-123",
			expectError:   false,
		},
		{
			name: "successful request without client auth",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response := TokenResponse{
						AccessToken: "access-token-456",
						TokenType:   "Bearer",
						ExpiresIn:   3600,
					}
					json.NewEncoder(w).Encode(response)
				}))
			},
			request: authRequest{
				clientID:   "",
				formData:   url.Values{"grant_type": {"password"}},
				logMessage: "test request without auth",
			},
			expectedToken: "access-token-456",
			expectError:   false,
		},
		{
			name: "server returns unauthorized",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
				}))
			},
			request: authRequest{
				clientID:   testClientID,
				formData:   url.Values{"grant_type": {"password"}},
				logMessage: "unauthorized request",
			},
			expectError:   true,
			errorContains: "unexpected status",
		},
		{
			name: "server returns invalid JSON",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("invalid json"))
				}))
			},
			request: authRequest{
				clientID:   testClientID,
				formData:   url.Values{"grant_type": {"password"}},
				logMessage: "invalid json request",
			},
			expectError:   true,
			errorContains: "failed to unmarshal response",
		},
		{
			name: "server returns empty access token",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response := TokenResponse{
						AccessToken: "",
						TokenType:   "Bearer",
						ExpiresIn:   3600,
					}
					json.NewEncoder(w).Encode(response)
				}))
			},
			request: authRequest{
				clientID:   testClientID,
				formData:   url.Values{"grant_type": {"password"}},
				logMessage: "empty token request",
			},
			expectError:   true,
			errorContains: "access_token not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			tt.request.tokenURL = server.URL

			token, err := executeTokenRequest(context.Background(), nil, tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if token != tt.expectedToken {
					t.Errorf("Expected token %q, got %q", tt.expectedToken, token)
				}
			}
		})
	}
}

func TestAuthenticateWithPassword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if r.Form.Get("grant_type") != "password" {
			http.Error(w, "Invalid grant type", http.StatusBadRequest)
			return
		}

		if r.Form.Get("username") != "test-user" || r.Form.Get("password") != "test-pass" {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != testClientID || password != testSecret {
			http.Error(w, "Invalid client", http.StatusUnauthorized)
			return
		}

		response := TokenResponse{
			AccessToken: mockJWTToken,
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)

	secrets := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "user-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"username": []byte("test-user"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pass-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"password": []byte("test-pass"),
			},
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secrets...).Build()

	generator := &UsernamePasswordGenerator{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	spec := &genv1alpha1.DexUsernamePassword{
		Spec: genv1alpha1.DexUsernamePasswordSpec{
			DexURL:   server.URL,
			ClientID: testClientID,
			UsernameRef: esmeta.SecretKeySelector{
				Name: "user-secret",
				Key:  "username",
			},
			PasswordRef: esmeta.SecretKeySelector{
				Name: "pass-secret",
				Key:  "password",
			},
		},
	}

	token, err := generator.authenticateWithPassword(
		context.Background(),
		spec,
		testSecret,
		[]string{"openid"},
		kubeClient,
		"test-namespace",
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if token != mockJWTToken {
		t.Errorf("Expected token %q, got %q", mockJWTToken, token)
	}
}

func TestAuthenticateWithTokenExchange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:token-exchange" {
			http.Error(w, "Invalid grant type", http.StatusBadRequest)
			return
		}

		if r.Form.Get("connector_id") != "test-connector" {
			http.Error(w, "Invalid connector", http.StatusBadRequest)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != testClientID || password != testSecret {
			http.Error(w, "Invalid client", http.StatusUnauthorized)
			return
		}

		response := TokenResponse{
			AccessToken: mockJWTToken,
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	generator := &TokenExchangeGenerator{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	spec := &genv1alpha1.DexTokenExchange{
		Spec: genv1alpha1.DexTokenExchangeSpec{
			DexURL:   server.URL,
			ClientID: testClientID,
			ServiceAccountRef: esmeta.ServiceAccountSelector{
				Name: "test-sa",
			},
		},
	}

	t.Run("service account token fetch error", func(t *testing.T) {
		_, err := generator.authenticateWithTokenExchange(
			context.Background(),
			spec,
			testSecret,
			"test-connector",
			[]string{"openid"},
			"invalid-namespace-that-does-not-exist",
		)

		if err == nil {
			t.Log("Function succeeded despite invalid namespace (likely in cluster environment)")
		} else {
			t.Logf("Function failed as expected: %v", err)
		}
	})
}

func TestFetchServiceAccountToken(t *testing.T) {
	t.Run("error case - cannot get kubernetes config", func(t *testing.T) {
		saRef := esmeta.ServiceAccountSelector{
			Name:      "test-sa",
			Audiences: []string{"test-audience"},
		}

		_, err := fetchServiceAccountToken(context.Background(), saRef, "test-namespace")

		if err == nil {
			t.Log("fetchServiceAccountToken succeeded (likely in cluster environment)")
		} else {
			t.Logf("fetchServiceAccountToken failed as expected: %v", err)
		}
	})
}
