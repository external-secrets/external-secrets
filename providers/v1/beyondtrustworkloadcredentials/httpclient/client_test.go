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

package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetHeadersServiceName(t *testing.T) {
	tests := []struct {
		name            string
		serviceName     string
		wantServiceName string
		wantPresent     bool
	}{
		{name: "api key auth sends no service name", serviceName: ""},
		{name: "oidc auth sends the service name", serviceName: "eso-cluster", wantServiceName: "eso-cluster", wantPresent: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got http.Header
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Clone()
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			c, err := NewClient(srv.URL, "the-token", tt.serviceName)
			if err != nil {
				t.Fatalf("NewClient() unexpected error: %v", err)
			}
			if err := c.CheckSession(context.Background()); err != nil {
				t.Fatalf("CheckSession() unexpected error: %v", err)
			}

			if auth := got.Get("Authorization"); auth != "Bearer the-token" {
				t.Errorf("Authorization = %q, want %q", auth, "Bearer the-token")
			}
			_, present := got[http.CanonicalHeaderKey(serviceNameHeader)]
			if present != tt.wantPresent {
				t.Errorf("%s present = %v, want %v", serviceNameHeader, present, tt.wantPresent)
			}
			if v := got.Get(serviceNameHeader); v != tt.wantServiceName {
				t.Errorf("%s = %q, want %q", serviceNameHeader, v, tt.wantServiceName)
			}
		})
	}
}
