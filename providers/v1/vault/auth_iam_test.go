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
	"slices"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	vault "github.com/hashicorp/vault/api"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/vault/fake"
	vaultiamauth "github.com/external-secrets/external-secrets/providers/v1/vault/iamauth"
	vaultutil "github.com/external-secrets/external-secrets/providers/v1/vault/util"
)

const (
	testLoginMountPath = "aws-mount"
	testLoginPath      = "auth/aws-mount/login"
	testLoginRole      = "my-role"
	testLoginToken     = "hvs.token-abc"
	testLoginRegion    = "us-east-1"
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

// testCreds returns the signing credentials, optionally clearing the session
// token and applying the AWS_STS_ENDPOINT override for the duration of the
// test. The env var is always set (an empty value clears it) so an ambient
// AWS_STS_ENDPOINT in the developer's shell cannot leak into the test.
func testCreds(t *testing.T, noSessionToken bool, stsEndpoint string) awssdk.Credentials {
	t.Helper()
	creds := staticCreds()
	if noSessionToken {
		creds.SessionToken = ""
	}
	t.Setenv(vaultiamauth.STSEndpointEnv, stsEndpoint)
	return creds
}

// iamTestClient bundles a client under test with the state its mocked Vault
// client records: the login write's path/data and the token set on the client.
type iamTestClient struct {
	client *client
	path   string
	data   map[string]any
	token  string
}

// newIamTestClient builds an iamTestClient whose login write returns the
// result of response, or a successful login when response is nil.
func newIamTestClient(response func() (*vault.Secret, error)) *iamTestClient {
	tc := &iamTestClient{}
	if response == nil {
		response = func() (*vault.Secret, error) {
			return &vault.Secret{Auth: &vault.SecretAuth{ClientToken: testLoginToken}}, nil
		}
	}

	logical := fake.Logical{
		WriteWithContextFn: func(_ context.Context, path string, d map[string]any) (*vault.Secret, error) {
			tc.path = path
			tc.data = d
			return response()
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

// decodeLoginField base64-decodes a login-data field.
func decodeLoginField(t *testing.T, data map[string]any, key string) string {
	t.Helper()
	enc, ok := data[key].(string)
	if !ok {
		t.Fatalf("login data %s is %T, want base64 string", key, data[key])
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("decoding %s: %v", key, err)
	}
	return string(raw)
}

// stsRequestHeaders decodes the signed STS request headers embedded in the
// login data. Vault validates the X-Vault-AWS-IAM-Server-ID value against
// these, not against the login request's own HTTP headers.
func stsRequestHeaders(t *testing.T, data map[string]any) http.Header {
	t.Helper()
	var h http.Header
	if err := json.Unmarshal([]byte(decodeLoginField(t, data, "iam_request_headers")), &h); err != nil {
		t.Fatalf("unmarshaling iam_request_headers: %v", err)
	}
	return h
}

// assertLoginRequest verifies the login write hit the expected mount path
// with a SigV4-signed GetCallerIdentity request against the expected STS
// endpoint host and signing region, and that the returned client token was
// set on the Vault client.
func assertLoginRequest(t *testing.T, tc *iamTestClient, wantSessionToken, wantEndpointHost, wantSigningRegion string) {
	t.Helper()
	if tc.path != testLoginPath {
		t.Errorf("login path: got %q, want %q", tc.path, testLoginPath)
	}
	if role, _ := tc.data["role"].(string); role != testLoginRole {
		t.Errorf("role: got %q, want %q", role, testLoginRole)
	}
	if method, _ := tc.data["iam_http_request_method"].(string); method != http.MethodPost {
		t.Errorf("method: got %q, want %q", method, http.MethodPost)
	}
	if url := decodeLoginField(t, tc.data, "iam_request_url"); !strings.Contains(url, wantEndpointHost) {
		t.Errorf("request url %q does not target STS endpoint host %q", url, wantEndpointHost)
	}
	if body := decodeLoginField(t, tc.data, "iam_request_body"); body != getCallerIdentityBody {
		t.Errorf("request body: got %q, want %q", body, getCallerIdentityBody)
	}
	if tc.token != testLoginToken {
		t.Errorf("token: got %q, want %q", tc.token, testLoginToken)
	}
	assertSignature(t, stsRequestHeaders(t, tc.data), wantSessionToken, wantSigningRegion)
}

// assertSignature verifies the SigV4 signature artifacts on the signed STS
// request headers: the credential scope's signing region (us-east-1 for the
// legacy global endpoint, the store's region otherwise) and the session
// token that carries the (rotating) Pod Identity credential — or its absence
// for static credentials that have no session token (e.g. secretRef IAM user
// keys).
func assertSignature(t *testing.T, headers http.Header, wantSessionToken, wantSigningRegion string) {
	t.Helper()
	auth := headers.Get("Authorization")
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") || !strings.Contains(auth, "Signature=") {
		t.Errorf("Authorization header is not a SigV4 signature: %q", auth)
	}
	if !strings.Contains(auth, "/"+wantSigningRegion+"/sts/aws4_request") {
		t.Errorf("Authorization credential scope is not signed for region %q: %q", wantSigningRegion, auth)
	}
	if headers.Get("X-Amz-Date") == "" {
		t.Error("X-Amz-Date header missing from signed request")
	}
	if got := headers.Get("X-Amz-Security-Token"); got != wantSessionToken {
		t.Errorf("X-Amz-Security-Token: got %q, want %q", got, wantSessionToken)
	}
}

func TestLoginWithIamCreds(t *testing.T) {
	tests := []struct {
		name              string
		iamAuth           *esv1.VaultIamAuth
		response          func() (*vault.Secret, error)
		wantServerID      string
		stsEndpoint       string
		region            string
		wantHost          string
		wantSigningRegion string
		noSessionToken    bool
		wantErr           bool
	}{
		{
			// The default (Legacy) policy signs classic regions against the
			// global endpoint with a us-east-1 scope — the request shape this
			// provider has always produced, accepted by default Vault mounts.
			name: "posts signed login data to the configured mount",
			iamAuth: &esv1.VaultIamAuth{
				Role: testLoginRole,
			},
		},
		{
			name: "signs regionally when the Regional endpoint policy is set",
			iamAuth: &esv1.VaultIamAuth{
				Role:              testLoginRole,
				STSEndpointPolicy: esv1.VaultIAMSTSEndpointPolicyRegional,
			},
			region:            "us-west-2",
			wantHost:          "sts.us-west-2.amazonaws.com",
			wantSigningRegion: "us-west-2",
		},
		{
			// The global endpoint does not exist for post-2019 regions, so the
			// Legacy policy falls back to regional signing there on its own.
			name: "legacy policy signs a post-2019 region regionally",
			iamAuth: &esv1.VaultIamAuth{
				Role: testLoginRole,
			},
			region:            "ap-east-1",
			wantHost:          "sts.ap-east-1.amazonaws.com",
			wantSigningRegion: "ap-east-1",
		},
		{
			name: "signs against the AWS_STS_ENDPOINT override when set",
			iamAuth: &esv1.VaultIamAuth{
				Role: testLoginRole,
			},
			stsEndpoint:       "https://sts.internal.example.com",
			wantHost:          "sts.internal.example.com",
			wantSigningRegion: testLoginRegion,
		},
		{
			name: "omits the session token for static credentials without one",
			iamAuth: &esv1.VaultIamAuth{
				Role: testLoginRole,
			},
			noSessionToken: true,
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
			response: func() (*vault.Secret, error) { return nil, errors.New("vault unreachable") },
			wantErr:  true,
		},
		{
			name: "returns error when the login response is nil",
			iamAuth: &esv1.VaultIamAuth{
				Role: testLoginRole,
			},
			response: func() (*vault.Secret, error) { return nil, nil },
			wantErr:  true,
		},
		{
			name: "returns error when the login response has no auth stanza",
			iamAuth: &esv1.VaultIamAuth{
				Role: testLoginRole,
			},
			response: func() (*vault.Secret, error) { return &vault.Secret{}, nil },
			wantErr:  true,
		},
		{
			name: "returns error when the login response has an empty client token",
			iamAuth: &esv1.VaultIamAuth{
				Role: testLoginRole,
			},
			response: func() (*vault.Secret, error) {
				return &vault.Secret{Auth: &vault.SecretAuth{}}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := newIamTestClient(tt.response)
			creds := testCreds(t, tt.noSessionToken, tt.stsEndpoint)
			region := tt.region
			if region == "" {
				region = testLoginRegion
			}
			// The Legacy default signs classic regions against the global
			// endpoint with a us-east-1 scope.
			wantHost := tt.wantHost
			if wantHost == "" {
				wantHost = "sts.amazonaws.com"
			}
			wantSigningRegion := tt.wantSigningRegion
			if wantSigningRegion == "" {
				wantSigningRegion = "us-east-1"
			}

			err := tc.client.loginWithIamCreds(t.Context(), creds, tt.iamAuth, testLoginMountPath, region)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.token != "" {
					t.Errorf("token set to %q on failed login, want unset", tc.token)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertLoginRequest(t, tc, creds.SessionToken, wantHost, wantSigningRegion)
			assertServerID(t, tc, tt.wantServerID)
		})
	}
}

// TestLoginWithIamCredsUsesFreshCredentials pins the Pod Identity fix: every
// login signs with the credentials passed to that call, so a rotated session
// token is picked up on the next login without a pod restart.
func TestLoginWithIamCredsUsesFreshCredentials(t *testing.T) {
	tc := newIamTestClient(nil)
	iamAuth := &esv1.VaultIamAuth{Role: testLoginRole}
	creds := testCreds(t, false, "")

	if err := tc.client.loginWithIamCreds(t.Context(), creds, iamAuth, testLoginMountPath, testLoginRegion); err != nil {
		t.Fatalf("unexpected error on first login: %v", err)
	}
	if got := stsRequestHeaders(t, tc.data).Get("X-Amz-Security-Token"); got != creds.SessionToken {
		t.Errorf("first login session token: got %q, want %q", got, creds.SessionToken)
	}

	rotated := creds
	rotated.SessionToken = "rotated-session-token"
	if err := tc.client.loginWithIamCreds(t.Context(), rotated, iamAuth, testLoginMountPath, testLoginRegion); err != nil {
		t.Fatalf("unexpected error on second login: %v", err)
	}
	if got := stsRequestHeaders(t, tc.data).Get("X-Amz-Security-Token"); got != rotated.SessionToken {
		t.Errorf("second login did not sign with the rotated session token: got %q, want %q", got, rotated.SessionToken)
	}
}

// TestGenerateLoginDataEmptyCreds verifies keyless credentials are rejected
// before signing: the v2 signer would happily sign with empty keys, unlike
// the v1 creds.Get() fail-fast in the flow this replaced.
func TestGenerateLoginDataEmptyCreds(t *testing.T) {
	if _, err := generateLoginData(t.Context(), awssdk.Credentials{}, "", testLoginRegion, ""); err == nil {
		t.Fatal("expected error for credentials without keys, got nil")
	}
}

// assertServerID verifies the X-Vault-AWS-IAM-Server-ID header: present with
// the configured value AND covered by the SigV4 SignedHeaders list (Vault
// rejects logins whose server-id header is not signed), or absent entirely
// when not configured.
func assertServerID(t *testing.T, tc *iamTestClient, want string) {
	t.Helper()
	headers := stsRequestHeaders(t, tc.data)
	if got := headers.Get(iamServerIDHeader); got != want {
		t.Errorf("signed server-id header: got %q, want %q", got, want)
	}
	signed := slices.Contains(signedHeaders(t, headers), strings.ToLower(iamServerIDHeader))
	if want != "" && !signed {
		t.Errorf("server-id header not covered by SignedHeaders in %q", headers.Get("Authorization"))
	}
	if want == "" && signed {
		t.Errorf("server-id header unexpectedly signed in %q", headers.Get("Authorization"))
	}
}

// signedHeaders extracts the header names from the SignedHeaders= component
// of the SigV4 Authorization header.
func signedHeaders(t *testing.T, headers http.Header) []string {
	t.Helper()
	for field := range strings.SplitSeq(strings.TrimPrefix(headers.Get("Authorization"), "AWS4-HMAC-SHA256 "), ", ") {
		if names, ok := strings.CutPrefix(field, "SignedHeaders="); ok {
			return strings.Split(names, ";")
		}
	}
	t.Fatalf("no SignedHeaders component in Authorization header %q", headers.Get("Authorization"))
	return nil
}
