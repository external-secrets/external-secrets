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

package oidc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	httpGet      = "GET"
)

func setupOIDCServer(handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewTLSServer(handler)
	return server
}

func TestOIDCGenerator_PasswordGrant(t *testing.T) {
	server := setupOIDCServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if username != "testuser" || password != "testpass" {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Verify client authentication
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Basic ") {
			http.Error(w, "Missing client authentication", http.StatusUnauthorized)
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

	spec := &genv1alpha1.OIDC{
		Spec: genv1alpha1.OIDCSpec{
			TokenURL: server.URL + "/token",
			ClientID: testClientID,
			ClientSecretRef: &esmeta.SecretKeySelector{
				Name: "oidc-client-secret",
				Key:  "client-secret",
			},
			Scopes: []string{"openid", "profile"},
			Grant: genv1alpha1.GrantSpec{
				Password: &genv1alpha1.PasswordGrantSpec{
					Username: "testuser",
					PasswordRef: esmeta.SecretKeySelector{
						Name: "user-credentials",
						Key:  "password",
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)

	secrets := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "user-credentials",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"username": []byte("testuser"),
				"password": []byte("testpass"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oidc-client-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"client-secret": []byte(testSecret),
			},
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secrets...).Build()

	generator := &Generator{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}

	specBytes, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal spec: %v", err)
	}

	result, _, err := generator.generate(
		context.Background(),
		&apiextensions.JSON{Raw: specBytes},
		kubeClient,
		"test-namespace",
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	token := string(result["token"])
	if token != mockJWTToken {
		t.Errorf("Expected token %q, got %q", mockJWTToken, token)
	}
}

func TestOIDCGenerator_TokenExchange(t *testing.T) {
	server := setupOIDCServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		subjectToken := r.Form.Get("subject_token")
		if subjectToken != "subject-token-value" {
			http.Error(w, "Invalid subject token", http.StatusUnauthorized)
			return
		}

		// Check for the audience parameter (now a first-class field)
		audience := r.Form.Get("audience")
		if audience != "https://api.example.com" {
			http.Error(w, "Invalid audience", http.StatusBadRequest)
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

	spec := &genv1alpha1.OIDC{
		Spec: genv1alpha1.OIDCSpec{
			TokenURL: server.URL + "/token",
			ClientID: testClientID,
			ClientSecretRef: &esmeta.SecretKeySelector{
				Name: "oidc-client-secret",
				Key:  "client-secret",
			},
			Scopes: []string{"openid"},
			Grant: genv1alpha1.GrantSpec{
				TokenExchange: &genv1alpha1.TokenExchangeGrantSpec{
					SubjectTokenRef: &esmeta.SecretKeySelector{
						Name: "subject-token",
						Key:  "token",
					},
					SubjectTokenType:   "urn:ietf:params:oauth:token-type:access_token",
					RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
					// Use first-class field for RFC 8693 audience parameter
					Audience: "https://api.example.com",
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)

	secrets := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subject-token",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"token": []byte("subject-token-value"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oidc-client-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"client-secret": []byte(testSecret),
			},
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secrets...).Build()

	generator := &Generator{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}

	specBytes, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal spec: %v", err)
	}

	result, _, err := generator.generate(
		context.Background(),
		&apiextensions.JSON{Raw: specBytes},
		kubeClient,
		"test-namespace",
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	token := string(result["token"])
	if token != mockJWTToken {
		t.Errorf("Expected token %q, got %q", mockJWTToken, token)
	}
}

func TestOIDCGenerator_GenericParameters(t *testing.T) {
	server := setupOIDCServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Check for Dex-specific connector_id from grant-level additionalParameters
		connectorID := r.Form.Get("connector_id")
		if connectorID != "ldap" {
			http.Error(w, "Invalid connector_id", http.StatusBadRequest)
			return
		}

		// Check for resource parameter (now a first-class field)
		resource := r.Form.Get("resource")
		if resource != "api://myapp" {
			http.Error(w, "Invalid resource", http.StatusBadRequest)
			return
		}

		// Check for custom header
		customHeader := r.Header.Get("X-Provider-Hint")
		if customHeader != "dex" {
			http.Error(w, "Missing custom header", http.StatusBadRequest)
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

	spec := &genv1alpha1.OIDC{
		Spec: genv1alpha1.OIDCSpec{
			TokenURL: server.URL + "/token",
			ClientID: testClientID,
			ClientSecretRef: &esmeta.SecretKeySelector{
				Name: "oidc-client-secret",
				Key:  "client-secret",
			},
			Scopes: []string{"openid"},
			// Global additional headers (provider-specific only)
			// Note: RFC 8693 standard parameters should use dedicated fields
			AdditionalHeaders: map[string]string{
				"X-Provider-Hint": "dex",
			},
			Grant: genv1alpha1.GrantSpec{
				TokenExchange: &genv1alpha1.TokenExchangeGrantSpec{
					SubjectTokenRef: &esmeta.SecretKeySelector{
						Name: "subject-token",
						Key:  "token",
					},
					SubjectTokenType:   "urn:ietf:params:oauth:token-type:access_token",
					RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
					// Use first-class field for RFC 8693 resource parameter
					Resource: "api://myapp",
					// Use additionalParameters only for truly provider-specific extensions
					AdditionalParameters: map[string]string{
						"connector_id": "ldap", // Dex-specific parameter
					},
					AdditionalHeaders: map[string]string{
						"X-Token-Exchange-Hint": "enabled",
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)

	secrets := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subject-token",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"token": []byte("subject-token-value"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oidc-client-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"client-secret": []byte(testSecret),
			},
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secrets...).Build()

	generator := &Generator{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}

	specBytes, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal spec: %v", err)
	}

	result, _, err := generator.generate(
		context.Background(),
		&apiextensions.JSON{Raw: specBytes},
		kubeClient,
		"test-namespace",
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	token := string(result["token"])
	if token != mockJWTToken {
		t.Errorf("Expected token %q, got %q", mockJWTToken, token)
	}
}

func TestOIDCGenerator_RFC8693Compliance(t *testing.T) {
	server := setupOIDCServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Verify all RFC 8693 standard parameters are present as expected
		expectedParams := map[string]string{
			"grant_type":           "urn:ietf:params:oauth:grant-type:token-exchange",
			"subject_token":        "subject-token-value",
			"subject_token_type":   "urn:ietf:params:oauth:token-type:access_token",
			"requested_token_type": "urn:ietf:params:oauth:token-type:access_token",
			"actor_token":          "actor-token-value",
			"actor_token_type":     "urn:ietf:params:oauth:token-type:jwt",
			"audience":             "https://api.example.com",
			"resource":             "api://myapp",
			"scope":                "openid profile",
		}

		for param, expected := range expectedParams {
			actual := r.Form.Get(param)
			if actual != expected {
				http.Error(w, "Invalid "+param+": expected "+expected+", got "+actual, http.StatusBadRequest)
				return
			}
		}

		// Verify provider-specific parameter is still passed through additionalParameters
		connectorID := r.Form.Get("connector_id")
		if connectorID != "ldap" {
			http.Error(w, "Invalid connector_id", http.StatusBadRequest)
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

	spec := &genv1alpha1.OIDC{
		Spec: genv1alpha1.OIDCSpec{
			TokenURL: server.URL + "/token",
			ClientID: testClientID,
			ClientSecretRef: &esmeta.SecretKeySelector{
				Name: "oidc-client-secret",
				Key:  "client-secret",
			},
			Scopes: []string{"openid", "profile"}, // RFC 8693 scope parameter
			Grant: genv1alpha1.GrantSpec{
				TokenExchange: &genv1alpha1.TokenExchangeGrantSpec{
					SubjectTokenRef: &esmeta.SecretKeySelector{
						Name: "subject-token",
						Key:  "token",
					},
					// All RFC 8693 standard parameters as first-class fields
					SubjectTokenType:   "urn:ietf:params:oauth:token-type:access_token",
					RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
					ActorTokenRef: &esmeta.SecretKeySelector{
						Name: "actor-token",
						Key:  "token",
					},
					ActorTokenType: "urn:ietf:params:oauth:token-type:jwt",
					Audience:       "https://api.example.com",
					Resource:       "api://myapp",
					// Only truly provider-specific parameters in additionalParameters
					AdditionalParameters: map[string]string{
						"connector_id": "ldap", // Dex-specific parameter
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)

	secrets := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subject-token",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"token": []byte("subject-token-value"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "actor-token",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"token": []byte("actor-token-value"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oidc-client-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"client-secret": []byte(testSecret),
			},
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secrets...).Build()

	generator := &Generator{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}

	specBytes, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal spec: %v", err)
	}

	result, _, err := generator.generate(
		context.Background(),
		&apiextensions.JSON{Raw: specBytes},
		kubeClient,
		"test-namespace",
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	token := string(result["token"])
	if token != mockJWTToken {
		t.Errorf("Expected token %q, got %q", mockJWTToken, token)
	}
}

func TestParseRequestTimeout(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected time.Duration
	}{
		{
			name:     "nil timeout uses default",
			input:    nil,
			expected: defaultHTTPTimeout,
		},
		{
			name:     "empty timeout uses default",
			input:    stringPtr(""),
			expected: defaultHTTPTimeout,
		},
		{
			name:     "valid timeout in seconds",
			input:    stringPtr("45s"),
			expected: 45 * time.Second,
		},
		{
			name:     "valid timeout in minutes",
			input:    stringPtr("2m"),
			expected: 2 * time.Minute,
		},
		{
			name:     "valid timeout in milliseconds",
			input:    stringPtr("500ms"),
			expected: 500 * time.Millisecond,
		},
		{
			name:     "invalid timeout format uses default",
			input:    stringPtr("invalid"),
			expected: defaultHTTPTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRequestTimeout(tt.input)
			if result != tt.expected {
				t.Errorf("parseRequestTimeout() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestOIDCGenerator_RequestTimeout(t *testing.T) {
	// Test that the timeout is properly configured in the HTTP client
	spec := &genv1alpha1.OIDC{
		Spec: genv1alpha1.OIDCSpec{
			TokenURL:       "https://example.com/token",
			ClientID:       testClientID,
			Scopes:         []string{"openid"},
			RequestTimeout: stringPtr("45s"),
			Grant: genv1alpha1.GrantSpec{
				Password: &genv1alpha1.PasswordGrantSpec{
					Username: "testuser",
					PasswordRef: esmeta.SecretKeySelector{
						Name: "user-credentials",
						Key:  "password",
					},
				},
			},
		},
	}

	generator := &Generator{}

	specBytes, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal spec: %v", err)
	}

	// Call Generate to trigger HTTP client creation with timeout
	_, _, err = generator.Generate(
		context.Background(),
		&apiextensions.JSON{Raw: specBytes},
		fake.NewClientBuilder().Build(),
		"test-namespace",
	)

	// We expect an error since the secrets don't exist, but the HTTP client should be created
	if err == nil {
		t.Fatal("Expected error due to missing secrets")
	}

	// Verify the HTTP client was created with the correct timeout
	if generator.httpClient == nil {
		t.Fatal("Expected HTTP client to be created")
	}

	if generator.httpClient.Timeout != 45*time.Second {
		t.Errorf("Expected timeout to be 45s, got %v", generator.httpClient.Timeout)
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestParseOIDCSpec(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid password grant spec",
			input: `
spec:
  tokenUrl: "https://dex.example.com/token"
  clientId: "test-client"
  scopes: ["openid"]
  grant:
    password:
      usernameRef:
        name: "user-creds"
        key: "username"
      passwordRef:
        name: "user-creds"
        key: "password"
`,
			wantErr: false,
		},
		{
			name: "valid token exchange grant spec with RFC 8693 compliance",
			input: `
spec:
  tokenUrl: "https://dex.example.com/token"
  clientId: "test-client"
  scopes: ["openid"]
  additionalHeaders:
    X-Provider-Hint: "dex"
  grant:
    tokenExchange:
      subjectTokenRef:
        name: "subject-token"
        key: "token"
      subjectTokenType: "urn:ietf:params:oauth:token-type:access_token"
      requestedTokenType: "urn:ietf:params:oauth:token-type:access_token"
      audience: "https://api.example.com"
      resource: "api://myapp"
      additionalParameters:
        connector_id: "ldap"
      additionalHeaders:
        X-Token-Exchange-Hint: "enabled"
`,
			wantErr: false,
		},
		{
			name: "invalid YAML",
			input: `
invalid: yaml: content:
  - missing
    - proper: structure
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseOIDCSpec([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOIDCSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
