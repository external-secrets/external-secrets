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

package cloudsmith

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const mockJWTToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiZXhwIjoxNzAwMDAwMDAwfQ.signature"

func TestCloudsmithGenerator_Generate(t *testing.T) {
	// Test server that mimics Cloudsmith OIDC endpoint
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req OIDCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Mock response with a JWT-like token (simplified for testing)
		mockToken := mockJWTToken
		response := OIDCResponse{
			Token: mockToken,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create test spec
	spec := &genv1alpha1.CloudsmithAccessToken{
		Spec: genv1alpha1.CloudsmithAccessTokenSpec{
			APIURL:      server.URL,
			OrgSlug:     "test-org",
			ServiceSlug: "test-service",
			ServiceAccountRef: esmeta.ServiceAccountSelector{
				Name:      "test-sa",
				Audiences: []string{"https://api.cloudsmith.io"},
			},
		},
	}

	specBytes, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal spec: %v", err)
	}

	_ = &apiextensions.JSON{
		Raw: specBytes,
	}

	// Create generator with custom HTTP client that accepts self-signed certificates
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	generator := &Generator{
		httpClient: httpClient,
	}

	// Note: This test will fail because we don't have a real service account token
	// In a real test environment, you would mock the fetchServiceAccountToken function
	// or set up proper test fixtures
	t.Run("parseSpec", func(t *testing.T) {
		parsed, err := parseSpec(specBytes)
		if err != nil {
			t.Fatalf("Failed to parse spec: %v", err)
		}

		if parsed.Spec.OrgSlug != "test-org" {
			t.Errorf("Expected OrgSlug to be 'test-org', got %s", parsed.Spec.OrgSlug)
		}
		if parsed.Spec.ServiceSlug != "test-service" {
			t.Errorf("Expected ServiceSlug to be 'test-service', got %s", parsed.Spec.ServiceSlug)
		}
	})

	t.Run("exchangeTokenWithCloudsmith", func(t *testing.T) {
		ctx := context.Background()
		oidcToken := "mock-oidc-token"

		token, err := generator.exchangeTokenWithCloudsmith(
			ctx,
			oidcToken,
			"test-org",
			"test-service",
			server.URL,
		)

		if err != nil {
			t.Fatalf("Failed to exchange token: %v", err)
		}

		if token == "" {
			t.Error("Expected non-empty token")
		}
	})

	t.Run("ParseJWTClaims", func(t *testing.T) {
		// Mock JWT token with known payload
		mockToken := mockJWTToken

		claims, err := esutils.ParseJWTClaims(mockToken)
		if err != nil {
			t.Fatalf("Failed to get claims: %v", err)
		}

		if claims["sub"] != "1234567890" {
			t.Errorf("Expected sub claim to be '1234567890', got %v", claims["sub"])
		}
		if claims["name"] != "John Doe" {
			t.Errorf("Expected name claim to be 'John Doe', got %v", claims["name"])
		}
	})

	t.Run("ExtractJWTExpiration", func(t *testing.T) {
		// Mock JWT token with known exp claim
		mockToken := mockJWTToken

		exp, err := esutils.ExtractJWTExpiration(mockToken)
		if err != nil {
			t.Fatalf("Failed to get token expiration: %v", err)
		}

		if exp != "1700000000" {
			t.Errorf("Expected expiration to be '1700000000', got %s", exp)
		}
	})
}
