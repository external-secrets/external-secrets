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

package webhook

import (
	"context"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestExecuteRequestLimitsResponseBody(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			name: "response at limit",
			size: maxResponseBodySize,
		},
		{
			name:    "response over limit",
			size:    maxResponseBodySize + 1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(strings.Repeat("a", tt.size)))
			}))
			defer server.Close()

			webhook := &Webhook{HTTP: server.Client()}
			body, err := webhook.executeRequest(context.Background(), &Spec{}, nil, server.URL, http.MethodGet, nil)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "response body exceeds") {
					t.Fatalf("executeRequest() error = %v, want response size error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("executeRequest() error = %v", err)
			}
			if len(body) != tt.size {
				t.Fatalf("executeRequest() returned %d bytes, want %d", len(body), tt.size)
			}
		})
	}
}

func TestExecuteRequestRejectsUnsupportedScheme(t *testing.T) {
	webhook := &Webhook{HTTP: &http.Client{}}

	_, err := webhook.executeRequest(context.Background(), &Spec{}, nil, "file:///etc/passwd", http.MethodGet, nil)
	if err == nil || !strings.Contains(err.Error(), `unsupported webhook URL scheme "file"`) {
		t.Fatalf("executeRequest() error = %v, want unsupported scheme error", err)
	}
}

func TestGetHTTPClientLimitsAndValidatesRedirects(t *testing.T) {
	t.Run("redirect limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			redirects, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/"))
			http.Redirect(w, r, fmt.Sprintf("/%d", redirects+1), http.StatusFound)
		}))
		defer server.Close()

		webhook := &Webhook{}
		client, err := webhook.GetHTTPClient(context.Background(), &Spec{})
		if err != nil {
			t.Fatalf("GetHTTPClient() error = %v", err)
		}
		response, err := client.Get(server.URL + "/0")
		if response != nil {
			_ = response.Body.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "stopped after 10 redirects") {
			t.Fatalf("client.Get() error = %v, want redirect limit error", err)
		}
	})

	t.Run("unsupported redirect scheme", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "file:///etc/passwd")
			w.WriteHeader(http.StatusFound)
		}))
		defer server.Close()

		webhook := &Webhook{}
		client, err := webhook.GetHTTPClient(context.Background(), &Spec{})
		if err != nil {
			t.Fatalf("GetHTTPClient() error = %v", err)
		}
		response, err := client.Get(server.URL)
		if response != nil {
			_ = response.Body.Close()
		}
		if err == nil || !strings.Contains(err.Error(), `unsupported webhook URL scheme "file"`) {
			t.Fatalf("client.Get() error = %v, want unsupported scheme error", err)
		}
	})
}

func TestGetHTTPClientPreservesCustomCAWithNTLM(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	caBundle := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	provider := &Spec{
		CABundle: caBundle,
		Auth: &AuthorizationProtocol{
			NTLM: &NTLMProtocol{},
		},
	}

	webhook := &Webhook{}
	client, err := webhook.GetHTTPClient(context.Background(), provider)
	if err != nil {
		t.Fatalf("GetHTTPClient() error = %v", err)
	}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("custom CA was not preserved with NTLM: %v", err)
	}
	_ = response.Body.Close()
}
