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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "with status",
			err:  &APIError{StatusCode: 401, Message: "Invalid Auth token"},
			want: "Doppler API Client Error (HTTP 401): Invalid Auth token",
		},
		{
			name: "no status stays backward compatible",
			err:  &APIError{Message: "secret 'FOO' not found"},
			want: "Doppler API Client Error: secret 'FOO' not found",
		},
		{
			name: "appends underlying error",
			err:  &APIError{StatusCode: 500, Message: "unable to load response", Err: errors.New("boom")},
			want: "Doppler API Client Error (HTTP 500): unable to load response\nboom",
		},
		{
			name: "appends data",
			err:  &APIError{Message: "unable to unmarshal secret payload", Data: "{bad json}"},
			want: "Doppler API Client Error: unable to unmarshal secret payload\nData: {bad json}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestPerformRequestSurfacesStatus exercises the real request path: a failing
// Doppler API response must yield an error naming the HTTP status, without
// leaking the request endpoint.
func TestPerformRequestSurfacesStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"messages":["Invalid Auth token"],"success":false}`))
	}))
	defer server.Close()

	c, err := NewDopplerClient("bad-token")
	if err != nil {
		t.Fatalf("NewDopplerClient: %v", err)
	}
	if err := c.SetBaseURL(server.URL); err != nil {
		t.Fatalf("SetBaseURL: %v", err)
	}

	err = c.Authenticate()
	if err == nil {
		t.Fatal("expected an authentication error, got nil")
	}

	got := err.Error()
	for _, want := range []string{"(HTTP 401)", "Invalid Auth token"} {
		if !strings.Contains(got, want) {
			t.Errorf("error %q does not contain %q", got, want)
		}
	}
	if strings.Contains(got, "/v3/projects") {
		t.Errorf("error %q should not surface the request endpoint", got)
	}
}
