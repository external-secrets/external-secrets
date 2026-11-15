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

	"github.com/google/go-cmp/cmp"

	btwcutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustworkloadcredentials/util"
)

func TestGenerateDynamicSecret(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       btwcutil.GeneratedSecret
		wantErr    string
	}{
		{
			name:       "AWS credentials",
			statusCode: http.StatusCreated,
			body:       `{"secret":{"accessKeyId":"AKIAIOSFODNN7EXAMPLE","secretAccessKey":"wJalrXUtnFEMI","sessionToken":"FwoGZXIvYXdzEBYaD","leaseId":"aws/creds/example/abc123","expiration":"2025-12-08T12:00:00Z"}}`,
			want: btwcutil.GeneratedSecret{
				"accessKeyId":     "AKIAIOSFODNN7EXAMPLE",
				"secretAccessKey": "wJalrXUtnFEMI",
				"sessionToken":    "FwoGZXIvYXdzEBYaD",
				"leaseId":         "aws/creds/example/abc123",
				"expiration":      "2025-12-08T12:00:00Z",
			},
		},
		{
			name:       "Arbitrary fields",
			statusCode: http.StatusOK,
			body:       `{"secret":{"username":"app_user_x1","password":"s3cr3t","ttl":3600,"connection":{"host":"db1"}}}`,
			want: btwcutil.GeneratedSecret{
				"username":   "app_user_x1",
				"password":   "s3cr3t",
				"ttl":        float64(3600),
				"connection": map[string]any{"host": "db1"},
			},
		},
		{
			name:       "Response without a secret",
			statusCode: http.StatusCreated,
			body:       `{}`,
			wantErr:    "generated secret response contains no secret",
		},
		{
			name:       "Invalid response body",
			statusCode: http.StatusCreated,
			body:       `not json`,
			wantErr:    "failed to unmarshal generated secret response: invalid character 'o' in literal null (expecting 'u')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/dynamic/aws-creds/generate" || r.URL.Query().Get("folder") != "team-a" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test-token")
			if err != nil {
				t.Fatalf("NewClient() unexpected error: %v", err)
			}

			got, err := client.GenerateDynamicSecret(context.Background(), "aws-creds", new("team-a"))
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("GenerateDynamicSecret() expected error %q, got nil", tt.wantErr)
				}
				if diff := cmp.Diff(tt.wantErr, err.Error()); diff != "" {
					t.Errorf("GenerateDynamicSecret() error mismatch (-want +got):\n%s", diff)
				}
				return
			}
			if err != nil {
				t.Fatalf("GenerateDynamicSecret() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GenerateDynamicSecret() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
