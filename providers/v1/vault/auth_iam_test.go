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

package vault

import (
	"context"
	"errors"
	"net/http"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	vault "github.com/hashicorp/vault/api"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/vault/fake"
	vaultutil "github.com/external-secrets/external-secrets/providers/v1/vault/util"
)

// staticCreds returns a fixed set of AWS credentials for signing, standing in
// for whatever cfg.Credentials.Retrieve resolved at runtime.
func staticCreds() awssdk.Credentials {
	return awssdk.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "session-token",
	}
}

// iamTestClient bundles a client under test with the state its mocked Vault
// client records: the login write's path/data, the set token, and the
// request headers with real (append) semantics so the server-ID header guard
// can be exercised.
type iamTestClient struct {
	client  *client
	path    string
	data    map[string]any
	token   string
	headers http.Header
}

// newIamTestClient builds an iamTestClient whose login write returns writeErr,
// or a successful login otherwise.
func newIamTestClient(writeErr error) *iamTestClient {
	tc := &iamTestClient{headers: http.Header{}}

	logical := fake.Logical{
		WriteWithContextFn: func(_ context.Context, path string, d map[string]any) (*vault.Secret, error) {
			tc.path = path
			tc.data = d
			if writeErr != nil {
				return nil, writeErr
			}
			return &vault.Secret{Auth: &vault.SecretAuth{ClientToken: "hvs.token-abc"}}, nil
		},
	}
	vc := &vaultutil.VaultClient{
		SetTokenFunc: func(v string) { tc.token = v },
		AddHeaderFunc: func(key, value string) {
			tc.headers.Add(key, value)
		},
		HeadersFunc:  func() http.Header { return tc.headers },
		LogicalField: logical,
	}

	tc.client = &client{
		client:  vc,
		logical: logical,
	}
	return tc
}

func TestLoginWithIamCreds(t *testing.T) {
	tests := []struct {
		name          string
		iamAuth       *esv1.VaultIamAuth
		writeErr      error
		wantPath      string
		wantRole      string
		wantHeader    bool
		wantHeaderVal string
		wantToken     string
		wantErr       bool
	}{
		{
			name: "posts signed login data to the configured mount",
			iamAuth: &esv1.VaultIamAuth{
				Role: "my-role",
			},
			wantPath:  "auth/aws-mount/login",
			wantRole:  "my-role",
			wantToken: "hvs.token-abc",
		},
		{
			name: "adds the server-id header when configured",
			iamAuth: &esv1.VaultIamAuth{
				Role:                "my-role",
				VaultAWSIAMServerID: "vault.example.com",
			},
			wantPath:      "auth/aws-mount/login",
			wantRole:      "my-role",
			wantHeader:    true,
			wantHeaderVal: "vault.example.com",
			wantToken:     "hvs.token-abc",
		},
		{
			name: "returns error when the login write fails",
			iamAuth: &esv1.VaultIamAuth{
				Role: "my-role",
			},
			writeErr: errors.New("vault unreachable"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := newIamTestClient(tt.writeErr)

			err := tc.client.loginWithIamCreds(context.Background(), staticCreds(), tt.iamAuth, "aws-mount", "us-east-1")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.path != tt.wantPath {
				t.Errorf("login path: got %q, want %q", tc.path, tt.wantPath)
			}
			if role, _ := tc.data["role"].(string); role != tt.wantRole {
				t.Errorf("role: got %q, want %q", role, tt.wantRole)
			}
			// GenerateLoginData must have produced the signed STS request fields.
			for _, k := range []string{"iam_http_request_method", "iam_request_url", "iam_request_headers", "iam_request_body"} {
				if _, ok := tc.data[k]; !ok {
					t.Errorf("login data missing expected key %q", k)
				}
			}
			if tc.token != tt.wantToken {
				t.Errorf("token: got %q, want %q", tc.token, tt.wantToken)
			}
			if tt.wantHeader {
				if got := tc.headers.Get(iamServerIDHeader); got != tt.wantHeaderVal {
					t.Errorf("server-id header: got %q, want %q", got, tt.wantHeaderVal)
				}
			} else if got := tc.headers.Values(iamServerIDHeader); len(got) != 0 {
				t.Errorf("server-id header set unexpectedly: %q", got)
			}
		})
	}
}

// TestLoginWithIamCredsHeaderNotDuplicated verifies that re-logging in on the
// same (cached) Vault client does not accumulate duplicate server-id headers,
// which the underlying append-only AddHeader would otherwise cause.
func TestLoginWithIamCredsHeaderNotDuplicated(t *testing.T) {
	tc := newIamTestClient(nil)
	iamAuth := &esv1.VaultIamAuth{
		Role:                "my-role",
		VaultAWSIAMServerID: "vault.example.com",
	}

	for i := range 3 {
		if err := tc.client.loginWithIamCreds(context.Background(), staticCreds(), iamAuth, "aws-mount", "us-east-1"); err != nil {
			t.Fatalf("login %d: unexpected error: %v", i, err)
		}
	}

	if got := tc.headers.Values(iamServerIDHeader); len(got) != 1 {
		t.Errorf("server-id header count after repeated logins: got %d (%q), want 1", len(got), got)
	}
}
