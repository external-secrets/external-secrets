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

package secretmanager

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	testKeyEmail = "eso-test@example-project.iam.gserviceaccount.com"
	testKeyID    = "test-private-key-id"
)

// newTestKeyJSON returns a service account key JSON signed with a freshly
// generated RSA key. universeDomain and tokenURI are omitted when empty.
func newTestKeyJSON(t *testing.T, universeDomain, tokenURI string) string {
	t.Helper()
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	key := map[string]string{
		"type":           "service_account",
		"project_id":     "example-project",
		"private_key_id": testKeyID,
		"private_key":    string(pemKey),
		"client_email":   testKeyEmail,
		"client_id":      "1234567890",
	}
	if universeDomain != "" {
		key["universe_domain"] = universeDomain
	}
	if tokenURI != "" {
		key["token_uri"] = tokenURI
	}
	out, err := json.Marshal(key)
	require.NoError(t, err)
	return string(out)
}

func TestUsesNonDefaultUniverse(t *testing.T) {
	tests := []struct {
		name string
		json string
		want bool
	}{
		{name: "no universe_domain", json: `{"type":"service_account"}`, want: false},
		{name: "empty universe_domain", json: `{"universe_domain":""}`, want: false},
		{name: "default universe", json: `{"universe_domain":"googleapis.com"}`, want: false},
		{name: "dedicated universe", json: `{"universe_domain":"apis-berlin-build0.goog"}`, want: true},
		{name: "invalid json", json: `not json`, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, usesNonDefaultUniverse([]byte(tc.json)))
		})
	}
}

func newStaticKeyStoreAuth() esv1.GCPSMAuth {
	return esv1.GCPSMAuth{
		SecretRef: &esv1.GCPSMAuthSecretRef{
			SecretAccessKey: esmeta.SecretKeySelector{Name: "gcp-key", Key: "credentials"},
		},
	}
}

func newKeySecret(keyJSON string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "gcp-key", Namespace: "default"},
		Data:       map[string][]byte{"credentials": []byte(keyJSON)},
	}
}

// A key from a non-default universe must produce a token without any network
// call: its token_uri points at a host that does not resolve.
func TestServiceAccountTokenSourceNonDefaultUniverse(t *testing.T) {
	keyJSON := newTestKeyJSON(t, "apis-berlin-build0.goog", "https://oauth2.nonexistent.invalid/token")
	kube := clientfake.NewClientBuilder().WithObjects(newKeySecret(keyJSON)).Build()

	ts, err := serviceAccountTokenSource(t.Context(), newStaticKeyStoreAuth(), esv1.SecretStoreKind, kube, "default")
	require.NoError(t, err)
	require.NotNil(t, ts)

	token, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "Bearer", token.Type())

	parts := strings.Split(token.AccessToken, ".")
	require.Len(t, parts, 3, "access token must be a JWT")

	header := decodeJWTPart(t, parts[0])
	assert.Equal(t, "RS256", header["alg"])
	assert.Equal(t, testKeyID, header["kid"])

	claims := decodeJWTPart(t, parts[1])
	assert.Equal(t, testKeyEmail, claims["iss"])
	assert.Equal(t, testKeyEmail, claims["sub"])
	// Self-signed JWTs carry either the scopes or an audience.
	assert.True(t, claims["scope"] == CloudPlatformRole || claims["aud"] != nil,
		"expected a scope or aud claim, got %v", claims)
}

// A key without a non-default universe must keep using the OAuth 2.0 token
// exchange against its token_uri (unchanged behavior).
func TestServiceAccountTokenSourceDefaultUniverseUsesTokenURI(t *testing.T) {
	var gotRequest bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"exchanged-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer srv.Close()

	for _, universe := range []string{"", "googleapis.com"} {
		gotRequest = false
		keyJSON := newTestKeyJSON(t, universe, srv.URL)
		kube := clientfake.NewClientBuilder().WithObjects(newKeySecret(keyJSON)).Build()

		ts, err := serviceAccountTokenSource(t.Context(), newStaticKeyStoreAuth(), esv1.SecretStoreKind, kube, "default")
		require.NoError(t, err)

		token, err := ts.Token()
		require.NoError(t, err)
		assert.Equal(t, "exchanged-token", token.AccessToken)
		assert.True(t, gotRequest, "token_uri must be called for universe %q", universe)
	}
}

func TestServiceAccountTokenSourceInvalidKey(t *testing.T) {
	for _, keyJSON := range []string{
		`not json`,
		`{"type":"service_account","universe_domain":"apis-berlin-build0.goog","private_key":"broken"}`,
	} {
		kube := clientfake.NewClientBuilder().WithObjects(newKeySecret(keyJSON)).Build()
		ts, err := serviceAccountTokenSource(t.Context(), newStaticKeyStoreAuth(), esv1.SecretStoreKind, kube, "default")
		assert.Error(t, err)
		assert.Nil(t, ts)
	}
}

func TestServiceAccountTokenSourceNoSecretRef(t *testing.T) {
	kube := clientfake.NewClientBuilder().Build()
	ts, err := serviceAccountTokenSource(t.Context(), esv1.GCPSMAuth{}, esv1.SecretStoreKind, kube, "default")
	assert.NoError(t, err)
	assert.Nil(t, ts)
}

func decodeJWTPart(t *testing.T, part string) map[string]any {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(part)
	require.NoError(t, err)
	out := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}
