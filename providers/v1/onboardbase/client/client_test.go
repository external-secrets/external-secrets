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
	"testing"
)

func TestNewOnboardbaseClientUsesAPIHost(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c, err := NewOnboardbaseClient("key", "passcode", server.URL+"/api/v1/")
	if err != nil {
		t.Fatalf("NewOnboardbaseClient: %v", err)
	}
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
