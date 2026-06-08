/*
Copyright © The ESO Authors

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

package artifactory

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const mockJWTToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJleHAiOjE3MDAwMDAwMDB9.signature"

func TestParseSpec(t *testing.T) {
	spec := &genv1alpha1.ArtifactoryAccessToken{
		Spec: genv1alpha1.ArtifactoryAccessTokenSpec{
			URL: "https://acme.jfrog.io",
			Auth: genv1alpha1.ArtifactoryAccessTokenAuth{
				OIDC: &genv1alpha1.ArtifactoryOIDCAuth{
					ProviderName: "k8s-oidc",
					ServiceAccountRef: esmeta.ServiceAccountSelector{
						Name: "default",
					},
				},
			},
		},
	}

	specBytes, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}

	parsed, err := parseSpec(specBytes)
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	if parsed.Spec.URL != "https://acme.jfrog.io" {
		t.Errorf("expected URL, got %q", parsed.Spec.URL)
	}
	if parsed.Spec.Auth.OIDC.ProviderName != "k8s-oidc" {
		t.Errorf("expected provider name, got %q", parsed.Spec.Auth.OIDC.ProviderName)
	}
}

func TestExchangeOIDCToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasSuffix(r.URL.Path, oidcTokenPath) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var req oidcExchangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.ProviderName != "k8s-oidc" {
			http.Error(w, "invalid provider", http.StatusBadRequest)
			return
		}
		if req.ProviderType != defaultProviderType {
			http.Error(w, "invalid provider type", http.StatusBadRequest)
			return
		}

		resp := oidcExchangeResponse{
			AccessToken:    mockJWTToken,
			ReferenceToken: "ref-token",
			Username:       "test-user",
			ExpiresIn:      3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	generator := &Generator{httpClient: server.Client()}
	auth := &genv1alpha1.ArtifactoryOIDCAuth{
		ProviderName:          "k8s-oidc",
		IncludeReferenceToken: true,
	}

	resp, err := generator.exchangeOIDCToken(
		context.Background(), server.URL, auth, "mock-sa-token",
	)
	if err != nil {
		t.Fatalf("exchange OIDC token: %v", err)
	}
	if resp.Username != "test-user" {
		t.Errorf("expected username test-user, got %q", resp.Username)
	}
	if resp.ReferenceToken != "ref-token" {
		t.Errorf("expected reference token, got %q", resp.ReferenceToken)
	}
}

func TestExchangeOIDCTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	generator := &Generator{httpClient: server.Client()}
	auth := &genv1alpha1.ArtifactoryOIDCAuth{ProviderName: "k8s-oidc"}

	_, err := generator.exchangeOIDCToken(
		context.Background(), server.URL, auth, "mock-sa-token",
	)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestCreateScopedToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasSuffix(r.URL.Path, createTokenPath) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer bootstrap-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req createTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Scope != "applied-permissions/user" {
			http.Error(w, "invalid scope", http.StatusBadRequest)
			return
		}

		resp := createTokenResponse{
			AccessToken:    mockJWTToken,
			ReferenceToken: "short-ref",
			ExpiresIn:      1800,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	generator := &Generator{httpClient: server.Client()}
	auth := &genv1alpha1.ArtifactoryReferenceTokenAuth{
		Scope:                 "applied-permissions/user",
		IncludeReferenceToken: true,
		ExpiresIn:             1800,
	}

	resp, err := generator.createScopedToken(
		context.Background(), server.URL, auth, "bootstrap-token",
	)
	if err != nil {
		t.Fatalf("create scoped token: %v", err)
	}
	if resp.ReferenceToken != "short-ref" {
		t.Errorf("expected reference token, got %q", resp.ReferenceToken)
	}
}

func TestCreateScopedTokenAuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	generator := &Generator{httpClient: server.Client()}
	auth := &genv1alpha1.ArtifactoryReferenceTokenAuth{
		Scope: "applied-permissions/user",
	}

	_, err := generator.createScopedToken(
		context.Background(), server.URL, auth, "bootstrap-token",
	)
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
}

func TestBuildOutputPrefersReferenceToken(t *testing.T) {
	spec := &genv1alpha1.ArtifactoryAccessToken{
		Spec: genv1alpha1.ArtifactoryAccessTokenSpec{
			URL: "https://acme.jfrog.io",
		},
	}

	out, _, err := buildOutput(spec, tokenResult{
		accessToken:    "long-access-token",
		referenceToken: "short-ref",
		username:       "docker-user",
		expiresIn:      3600,
	})
	if err != nil {
		t.Fatalf("build output: %v", err)
	}

	auth, err := b64.StdEncoding.DecodeString(string(out["auth"]))
	if err != nil {
		t.Fatalf("decode auth: %v", err)
	}
	expected := "docker-user:short-ref"
	if string(auth) != expected {
		t.Errorf("expected auth %q, got %q", expected, string(auth))
	}
	if string(out["registry"]) != "acme.jfrog.io" {
		t.Errorf("expected registry host, got %q", out["registry"])
	}
}

func TestComputeExpiryFromExpiresIn(t *testing.T) {
	before := time.Now().Unix()
	expiry, err := computeExpiry(tokenResult{expiresIn: 3600})
	if err != nil {
		t.Fatalf("compute expiry: %v", err)
	}

	expiryInt, err := parseExpiry(expiry)
	if err != nil {
		t.Fatalf("parse expiry: %v", err)
	}
	if expiryInt < before+3600 || expiryInt > before+3601 {
		t.Errorf("expected expiry near %d, got %d", before+3600, expiryInt)
	}
}

func TestUsernameFromToken(t *testing.T) {
	username := usernameFromToken(mockJWTToken)
	if username != "test-user" {
		t.Errorf("expected test-user, got %q", username)
	}
}

func parseExpiry(expiry string) (int64, error) {
	return strconv.ParseInt(expiry, 10, 64)
}
