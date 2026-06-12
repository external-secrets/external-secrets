/*
Copyright © 2026 SSH Communications

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

package privx

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetServiceAccountNameFromJWT(t *testing.T) {
	makeJWT := func(payload map[string]any) string {
		header := `{"alg":"none","typ":"JWT"}`
		h := base64.RawURLEncoding.EncodeToString([]byte(header))

		pb, _ := json.Marshal(payload)
		p := base64.RawURLEncoding.EncodeToString(pb)

		// signature part can be empty for these tests
		return h + "." + p + "."
	}

	tests := map[string]struct {
		token   string
		want    string
		wantErr error
	}{
		"claim kubernetes service account name": {
			token: makeJWT(map[string]any{
				"kubernetes.io/serviceaccount/service-account.name": "my-sa",
			}),
			want: "my-sa",
		},
		"fallback to sub claim": {
			token: makeJWT(map[string]any{
				"sub": "system:serviceaccount:default:my-sa",
			}),
			want: "my-sa",
		},
		"invalid format": {
			token:   "invalid-token",
			wantErr: errInvalidJWTFormat,
		},
		"invalid base64 payload": {
			token:   "a.b!.c",
			wantErr: errDecodeJWTPayload,
		},
		"invalid json payload": {
			token: func() string {
				header := base64.RawURLEncoding.EncodeToString([]byte(`{}`))
				payload := base64.RawURLEncoding.EncodeToString([]byte(`{invalid json`))
				return header + "." + payload + "."
			}(),
			wantErr: errParseJWTPayload,
		},
		"missing service account": {
			token: makeJWT(map[string]any{
				"iss": "something",
			}),
			wantErr: errServiceAccountNameNotFound,
		},
		"sub malformed": {
			token: makeJWT(map[string]any{
				"sub": "system:serviceaccount:default",
			}),
			wantErr: errServiceAccountNameNotFound,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := getServiceAccountNameFromJWT(tc.token)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDecodeJWT(t *testing.T) {
	makeJWT := func(header string, payload string) string {
		h := base64.RawURLEncoding.EncodeToString([]byte(header))
		p := base64.RawURLEncoding.EncodeToString([]byte(payload))
		return h + "." + p + "."
	}

	tests := map[string]struct {
		token    string
		wantText []string
		wantErr  string
	}{
		"valid jwt": {
			token: makeJWT(
				`{"alg":"RS256","typ":"JWT"}`,
				`{"sub":"system:serviceaccount:default:my-sa","aud":"privx"}`,
			),
			wantText: []string{
				"HEADER:",
				`"alg": "RS256"`,
				`"typ": "JWT"`,
				"PAYLOAD:",
				`"sub": "system:serviceaccount:default:my-sa"`,
				`"aud": "privx"`,
			},
		},
		"invalid format": {
			token:   "not-a-jwt",
			wantErr: "invalid JWT format",
		},
		"invalid header base64": {
			token:   "%%%." + base64.RawURLEncoding.EncodeToString([]byte(`{}`)) + ".",
			wantErr: "decode header",
		},
		"invalid header json": {
			token: makeJWT(
				`{invalid json`,
				`{"sub":"x"}`,
			),
			wantErr: "unmarshal header",
		},
		"invalid payload base64": {
			token:   base64.RawURLEncoding.EncodeToString([]byte(`{}`)) + ".%%%.",
			wantErr: "decode payload",
		},
		"invalid payload json": {
			token: makeJWT(
				`{"alg":"RS256"}`,
				`{invalid json`,
			),
			wantErr: "unmarshal payload",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := decodeJWT(tc.token)

			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			for _, s := range tc.wantText {
				assert.Contains(t, got, s)
			}
		})
	}
}

func TestDetectJWTSigningKey(t *testing.T) {
	makeRSAPKCS1PEM := func(t *testing.T) []byte {
		t.Helper()

		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		block := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}

		return pem.EncodeToMemory(block)
	}

	makeRSAPKCS8PEM := func(t *testing.T) []byte {
		t.Helper()

		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		der, err := x509.MarshalPKCS8PrivateKey(key)
		require.NoError(t, err)

		block := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: der,
		}

		return pem.EncodeToMemory(block)
	}

	makeEd25519PKCS8PEM := func(t *testing.T) []byte {
		t.Helper()

		_, key, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		der, err := x509.MarshalPKCS8PrivateKey(key)
		require.NoError(t, err)

		block := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: der,
		}

		return pem.EncodeToMemory(block)
	}

	tests := map[string]struct {
		pemBytes        []byte
		wantErr         error
		wantMethod      jwt.SigningMethod
		assertKeyTypeFn func(t *testing.T, key any)
	}{
		"invalid pem block": {
			pemBytes: []byte("not a pem"),
			wantErr:  errInvalidPEMBlock,
		},
		"unsupported pem block type": {
			pemBytes: pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: []byte("dummy"),
			}),
			wantErr: errUnsupportedPEMBlockType,
		},
		"rsa pkcs1": {
			pemBytes:   makeRSAPKCS1PEM(t),
			wantMethod: jwt.SigningMethodRS256,
			assertKeyTypeFn: func(t *testing.T, key any) {
				t.Helper()
				_, ok := key.(*rsa.PrivateKey)
				assert.True(t, ok)
			},
		},
		"rsa pkcs8": {
			pemBytes:   makeRSAPKCS8PEM(t),
			wantMethod: jwt.SigningMethodRS256,
			assertKeyTypeFn: func(t *testing.T, key any) {
				t.Helper()
				_, ok := key.(*rsa.PrivateKey)
				assert.True(t, ok)
			},
		},
		"ed25519 pkcs8": {
			pemBytes:   makeEd25519PKCS8PEM(t),
			wantMethod: SigningMethodEd25519(),
			assertKeyTypeFn: func(t *testing.T, key any) {
				t.Helper()
				_, ok := key.(ed25519.PrivateKey)
				assert.True(t, ok)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotMethod, gotKey, err := detectJWTSigningKey(tc.pemBytes)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, gotMethod)
				assert.Nil(t, gotKey)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantMethod, gotMethod)
			require.NotNil(t, gotKey)

			if tc.assertKeyTypeFn != nil {
				tc.assertKeyTypeFn(t, gotKey)
			}
		})
	}
}

func TestExchangeToken(t *testing.T) {
	tests := map[string]struct {
		reqBody   exchangeTokenRequest
		server    func(t *testing.T) *httptest.Server
		client    *http.Client
		want      tokenResponse
		wantErr   error
		errSubstr string
	}{
		"empty token": {
			reqBody: exchangeTokenRequest{},
			wantErr: errTokenEmpty,
		},
		"successful response": {
			reqBody: exchangeTokenRequest{
				Token: "jwt-token",
			},
			server: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, "/auth/api/v1/token/login", r.URL.Path)
					require.Equal(t, "application/json", r.Header.Get("Content-Type"))

					w.WriteHeader(200)
					_, _ = w.Write([]byte(`{
						"access_token":"abc123",
						"token_type":"Bearer",
						"expires_in":3600
					}`))
				}))
			},
			want: tokenResponse{
				AccessToken: "abc123",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			},
		},
		"bad status code": {
			reqBody: exchangeTokenRequest{
				Token: "jwt-token",
			},
			server: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(400)
					_, _ = w.Write([]byte("bad request"))
				}))
			},
			wantErr: errPrivXTokenExchangeBadStatus,
		},
		"invalid json response": {
			reqBody: exchangeTokenRequest{
				Token: "jwt-token",
			},
			server: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					_, _ = w.Write([]byte(`{invalid json`))
				}))
			},
			errSubstr: "decode json",
		},
		"http client error": {
			reqBody: exchangeTokenRequest{
				Token: "jwt-token",
			},
			client: &http.Client{
				Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				}),
			},
			errSubstr: "do request",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var baseURL string
			if tc.server != nil {
				srv := tc.server(t)
				defer srv.Close()
				baseURL = srv.URL
			} else {
				baseURL = "http://example.com"
			}

			client := tc.client
			if client == nil {
				client = &http.Client{}
			}

			got, err := exchangeToken(context.Background(), client, baseURL, tc.reqBody)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}

			if tc.errSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
