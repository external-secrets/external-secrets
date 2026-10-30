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
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestExecuteRequestLimitsResponseBody(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			name: "response at limit",
			size: DefaultMaxResponseBodySize,
		},
		{
			name:    "response over limit",
			size:    DefaultMaxResponseBodySize + 1,
			wantErr: true,
		},
	}

	restoreResponseBodySize(t)
	if err := setMaxResponseBodySize(DefaultMaxResponseBodySize); err != nil {
		t.Fatalf("setMaxResponseBodySize() error = %v", err)
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

func TestExecuteRequestUsesConfiguredResponseBodySize(t *testing.T) {
	restoreResponseBodySize(t)
	if err := setMaxResponseBodySize(4); err != nil {
		t.Fatalf("setMaxResponseBodySize() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("12345"))
	}))
	defer server.Close()

	webhook := &Webhook{HTTP: server.Client()}
	_, err := webhook.executeRequest(context.Background(), &Spec{}, nil, server.URL, http.MethodGet, nil)
	if err == nil || !strings.Contains(err.Error(), "response body exceeds 4 bytes") {
		t.Fatalf("executeRequest() error = %v, want configured response size error", err)
	}
}

func TestSetMaxResponseBodySizeRejectsInvalidValues(t *testing.T) {
	for _, size := range []int64{-1, 0, math.MaxInt64} {
		if err := setMaxResponseBodySize(size); err == nil {
			t.Errorf("setMaxResponseBodySize(%d) error = nil, want validation error", size)
		}
	}
}

func TestWebhookFeatureResponseSizeFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int64
	}{
		{
			name: "default",
			want: DefaultMaxResponseBodySize,
		},
		{
			name: "configured",
			args: []string{"--" + webhookProviderMaxResponseSizeFlagName, "4"},
			want: 4,
		},
	}

	restoreResponseBodySize(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhookFeature := newWebhookFeature(pflag.ContinueOnError)
			flag := webhookFeature.Flags.Lookup(webhookProviderMaxResponseSizeFlagName)
			if flag == nil {
				t.Fatalf("feature flag %q is not registered", webhookProviderMaxResponseSizeFlagName)
			}
			if flag.DefValue != strconv.FormatInt(DefaultMaxResponseBodySize, 10) {
				t.Fatalf("feature flag default = %q, want %d", flag.DefValue, DefaultMaxResponseBodySize)
			}
			if err := webhookFeature.Flags.Parse(tt.args); err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			webhookFeature.Initialize()
			if got := maxResponseBodySize.Load(); got != tt.want {
				t.Fatalf("maximum response body size = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWebhookFeatureRejectsInvalidResponseSize(t *testing.T) {
	for _, value := range []string{"-1", "0", strconv.FormatInt(math.MaxInt64, 10), "invalid"} {
		webhookFeature := newWebhookFeature(pflag.ContinueOnError)
		err := webhookFeature.Flags.Parse([]string{"--" + webhookProviderMaxResponseSizeFlagName, value})
		if err == nil {
			t.Errorf("Parse(%q) error = nil, want validation error", value)
		}
	}
}

func restoreResponseBodySize(t *testing.T) {
	t.Helper()
	original := maxResponseBodySize.Load()
	t.Cleanup(func() {
		maxResponseBodySize.Store(original)
	})
}
