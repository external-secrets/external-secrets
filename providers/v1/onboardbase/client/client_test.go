/*
Copyright © The ESO Authors

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

package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewOnboardbaseClientUsesAPIHost(t *testing.T) {
	var gotPath string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c, err := NewOnboardbaseClient("key", "passcode", server.URL+"/api/v1/")
	if err != nil {
		t.Fatalf("NewOnboardbaseClient: %v", err)
	}
	// trust the test server certificate
	c.httpClient = server.Client()
	if err := c.Authenticate(); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if gotPath != "/api/v1/team/members" {
		t.Errorf("request went to path %q, want /api/v1/team/members on the apiHost server", gotPath)
	}
}

func TestNewOnboardbaseClientDefaultsToPublicAPI(t *testing.T) {
	c, err := NewOnboardbaseClient("key", "passcode", "")
	if err != nil {
		t.Fatalf("NewOnboardbaseClient: %v", err)
	}
	if got, want := c.BaseURL().String(), "https://public.onboardbase.com/api/v1"; got != want {
		t.Errorf("BaseURL() = %q, want %q", got, want)
	}
}

func TestNewOnboardbaseClientRejectsCleartextAPIHost(t *testing.T) {
	_, err := NewOnboardbaseClient("key", "passcode", "http://onboardbase.internal.example.com/api/v1/")
	if err == nil {
		t.Fatal("NewOnboardbaseClient accepted an http apiHost, want an error")
	}
	if !strings.Contains(err.Error(), "apiHost must use https") {
		t.Errorf("error = %q, want it to mention that apiHost must use https", err)
	}
}

func TestSameHostHTTPSRedirect(t *testing.T) {
	first, _ := http.NewRequest(http.MethodGet, "https://onboardbase.example.com/api/v1/team/members", http.NoBody)
	tests := []struct {
		name    string
		target  string
		allowed bool
	}{
		{"same host https", "https://onboardbase.example.com/api/v2/team/members", true},
		{"other host", "https://attacker.example.com/api/v1/team/members", false},
		{"same host cleartext", "http://onboardbase.example.com/api/v1/team/members", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, tt.target, http.NoBody)
			err := sameHostHTTPSRedirect(req, []*http.Request{first})
			if tt.allowed && err != nil {
				t.Errorf("redirect to %s refused: %v", tt.target, err)
			}
			if !tt.allowed && err == nil {
				t.Errorf("redirect to %s followed, want it refused", tt.target)
			}
		})
	}
}

func TestClientDoesNotSendAPIKeyOnCrossHostRedirect(t *testing.T) {
	var leaked string
	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaked = r.Header.Get("api_key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer other.Close()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+r.URL.Path, http.StatusFound)
	}))
	defer server.Close()

	c, err := NewOnboardbaseClient("secret-key", "passcode", server.URL+"/api/v1/")
	if err != nil {
		t.Fatalf("NewOnboardbaseClient: %v", err)
	}
	// trust the test server certificate but keep the client redirect policy
	c.httpClient.Transport = server.Client().Transport
	_ = c.Authenticate()
	if leaked != "" {
		t.Errorf("api_key %q reached the redirect target on another host", leaked)
	}
}
