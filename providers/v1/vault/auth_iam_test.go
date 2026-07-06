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
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	vault "github.com/hashicorp/vault/api"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/vault/fake"
	vaultutil "github.com/external-secrets/external-secrets/providers/v1/vault/util"
)

const (
	testLoginMountPath = "aws-mount"
	testLoginPath      = "auth/aws-mount/login"
	testLoginRole      = "my-role"
	testLoginToken     = "hvs.token-abc"
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
// client records: the login write's path/data and the token set on the client.
type iamTestClient struct {
	client *client
	path   string
	data   map[string]any
	token  string
}

// newIamTestClient builds an iamTestClient whose login write returns writeErr,
// or a successful login otherwise.
func newIamTestClient(writeErr error) *iamTestClient {
	tc := &iamTestClient{}

	logical := fake.Logical{
		WriteWithContextFn: func(_ context.Context, path string, d map[string]any) (*vault.Secret, error) {
			tc.path = path
			tc.data = d
			if writeErr != nil {
				return nil, writeErr
			}
			return &vault.Secret{Auth: &vault.SecretAuth{ClientToken: testLoginToken}}, nil
		},
	}
	vc := &vaultutil.VaultClient{
		SetTokenFunc: func(v string) { tc.token = v },
		LogicalField: logical,
	}

	tc.client = &client{
		client:  vc,
		logical: logical,
	}
	return tc
}

// assertLoginRequest verifies the login write hit the expected mount path with
// the signed GetCallerIdentity fields and the role, and that the returned
// client token was set on the Vault client.
func assertLoginRequest(t *testing.T, tc *iamTestClient) {
	t.Helper()
	if tc.path != testLoginPath {
		t.Errorf("login path: got %q, want %q", tc.path, testLoginPath)
	}
	if role, _ := tc.data["role"].(string); role != testLoginRole {
		t.Errorf("role: got %q, want %q", role, testLoginRole)
	}
	// GenerateLoginData must have produced the signed STS request fields.
	for _, k := range []string{"iam_http_request_method", "iam_request_url", "iam_request_headers", "iam_request_body"} {
		if _, ok := tc.data[k]; !ok {
			t.Errorf("login data missing expected key %q", k)
		}
	}
	if tc.token != testLoginToken {
		t.Errorf("token: got %q, want %q", tc.token, testLoginToken)
	}
}

// stsRequestHeaders decodes the signed STS request headers that
// GenerateLoginData embedded in the login data. Vault validates the
// X-Vault-AWS-IAM-Server-ID value against these, not against the login
// request's own HTTP headers.
func stsRequestHeaders(t *testing.T, data map[string]any) http.Header {
	t.Helper()
	enc, ok := data["iam_request_headers"].(string)
	if !ok {
		t.Fatalf("login data iam_request_headers is %T, want base64 string", data["iam_request_headers"])
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("decoding iam_request_headers: %v", err)
	}
	var h http.Header
	if err := json.Unmarshal(raw, &h); err != nil {
		t.Fatalf("unmarshaling iam_request_headers: %v", err)
	}
	return h
}

func TestLoginWithIamCreds(t *testing.T) {
	tests := []struct {
		name         string
		iamAuth      *esv1.VaultIamAuth
		writeErr     error
		wantServerID string
		wantErr      bool
	}{
		{
			name: "posts signed login data to the configured mount",
			iamAuth: &esv1.VaultIamAuth{
				Role: testLoginRole,
			},
		},
		{
			name: "signs the server-id into the STS request headers when configured",
			iamAuth: &esv1.VaultIamAuth{
				Role:                testLoginRole,
				VaultAWSIAMServerID: "vault.example.com",
			},
			wantServerID: "vault.example.com",
		},
		{
			name: "returns error when the login write fails",
			iamAuth: &esv1.VaultIamAuth{
				Role: testLoginRole,
			},
			writeErr: errors.New("vault unreachable"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := newIamTestClient(tt.writeErr)

			err := tc.client.loginWithIamCreds(context.Background(), staticCreds(), tt.iamAuth, testLoginMountPath, "us-east-1")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertLoginRequest(t, tc)
			if got := stsRequestHeaders(t, tc.data).Get("X-Vault-AWS-IAM-Server-ID"); got != tt.wantServerID {
				t.Errorf("signed server-id header: got %q, want %q", got, tt.wantServerID)
			}
		})
	}
}
