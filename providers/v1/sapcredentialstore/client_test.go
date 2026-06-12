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

package sapcredentialstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/sapcredentialstore/api"
)

const testNamespace = "test-ns"

// newHTTPTestClient builds a Client backed by a test HTTP server.
// The server URL is used as baseURL; the namespace header is checked by tests.
func newHTTPTestClient(serverURL string) *Client {
	return &Client{
		sapClient: api.NewOAuth2Client(serverURL, http.DefaultTransport, nil),
		namespace: testNamespace,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// assertNS verifies the namespace header is present on every request.
func assertNS(t *testing.T, r *http.Request) {
	t.Helper()
	assert.Equal(t, testNamespace, r.Header.Get("sapcp-credstore-namespace"))
}

// --- US1: GetSecret ---

func TestGetSecret_PasswordDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/password", r.URL.Path)
		assert.Equal(t, "db-pass", r.URL.Query().Get("name"))
		writeJSON(w, api.Credential{Name: "db-pass", Value: "secret123", Username: "admin"})
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "db-pass",
		Property: "",
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("secret123"), got)
}

func TestGetSecret_PasswordExplicit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		assert.Equal(t, "/password", r.URL.Path)
		assert.Equal(t, "my-pass", r.URL.Query().Get("name"))
		writeJSON(w, api.Credential{Name: "my-pass", Value: "pass-value"})
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "my-pass",
		Property: "password",
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("pass-value"), got)
}

func TestGetSecret_Key(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		assert.Equal(t, "/key", r.URL.Path)
		assert.Equal(t, "api-key", r.URL.Query().Get("name"))
		writeJSON(w, api.Credential{Name: "api-key", Value: "key-value-abc"})
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "api-key",
		Property: "key",
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("key-value-abc"), got)
}

func TestGetSecret_Certificate(t *testing.T) {
	certPEM := "-----BEGIN CERTIFICATE-----\nMIIBxx\n-----END CERTIFICATE-----"
	keyPEM := "-----BEGIN RSA PRIVATE KEY-----\nMIIEyy\n-----END RSA PRIVATE KEY-----"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		assert.Equal(t, "/certificate", r.URL.Path)
		assert.Equal(t, "my-cert", r.URL.Query().Get("name"))
		writeJSON(w, api.Credential{Name: "my-cert", Value: certPEM, Key: keyPEM})
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "my-cert",
		Property: "certificate",
	})
	require.NoError(t, err)
	assert.Equal(t, []byte(certPEM), got)
}

func TestGetSecret_CertificateKey(t *testing.T) {
	keyPEM := "-----BEGIN RSA PRIVATE KEY-----\nMIIEyy\n-----END RSA PRIVATE KEY-----"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		assert.Equal(t, "/certificate", r.URL.Path)
		assert.Equal(t, "my-cert", r.URL.Query().Get("name"))
		writeJSON(w, api.Credential{Name: "my-cert", Value: "CERT_PEM", Key: keyPEM})
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "my-cert",
		Property: "certificate/key",
	})
	require.NoError(t, err)
	assert.Equal(t, []byte(keyPEM), got)
}

func TestGetSecret_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "missing",
		Property: "password",
	})
	require.Error(t, err)
	assert.ErrorAs(t, err, &esv1.NoSecretError{})
}

func TestGetSecretMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		writeJSON(w, api.Credential{Name: "db-pass", Value: "secret", Username: "admin"})
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	got, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "db-pass",
		Property: "password",
	})
	require.NoError(t, err)
	assert.Equal(t, map[string][]byte{
		"name":     []byte("db-pass"),
		"value":    []byte("secret"),
		"username": []byte("admin"),
	}, got)
}

func TestGetSecretMap_Certificate(t *testing.T) {
	certPEM := "-----BEGIN CERTIFICATE-----\nMIIBxx\n-----END CERTIFICATE-----"
	keyPEM := "-----BEGIN RSA PRIVATE KEY-----\nMIIEyy\n-----END RSA PRIVATE KEY-----"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		writeJSON(w, api.Credential{Name: "my-cert", Value: certPEM, Key: keyPEM})
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	got, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "my-cert",
		Property: "certificate",
	})
	require.NoError(t, err)
	assert.Equal(t, map[string][]byte{
		"name":  []byte("my-cert"),
		"value": []byte(certPEM),
		"key":   []byte(keyPEM),
	}, got)
}

// --- US2: PushSecret, DeleteSecret, SecretExists ---

type fakePushData struct {
	secretKey string
	remoteKey string
	property  string
}

func (f *fakePushData) GetMetadata() *apiextensionsv1.JSON { return nil }
func (f *fakePushData) GetSecretKey() string               { return f.secretKey }
func (f *fakePushData) GetRemoteKey() string               { return f.remoteKey }
func (f *fakePushData) GetProperty() string                { return f.property }

type fakePushRemoteRef struct {
	remoteKey string
	property  string
}

func (f *fakePushRemoteRef) GetRemoteKey() string { return f.remoteKey }
func (f *fakePushRemoteRef) GetProperty() string  { return f.property }

func TestPushSecret_Create(t *testing.T) {
	type createBody struct {
		Name     string `json:"name"`
		Value    string `json:"value"`
		Username string `json:"username,omitempty"`
	}
	var receivedBody createBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/password", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&receivedBody))
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-k8s-secret"},
		Data:       map[string][]byte{"password": []byte("super-secret")},
	}
	err := c.PushSecret(context.Background(), secret, &fakePushData{
		secretKey: "password",
		remoteKey: "db-pass",
		property:  "password",
	})
	require.NoError(t, err)
	assert.Equal(t, "db-pass", receivedBody.Name)
	assert.Equal(t, "super-secret", receivedBody.Value)
}

func TestPushSecret_KeyType(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	secret := &corev1.Secret{
		Data: map[string][]byte{"apikey": []byte("my-api-key")},
	}
	err := c.PushSecret(context.Background(), secret, &fakePushData{
		secretKey: "apikey",
		remoteKey: "my-key",
		property:  "key",
	})
	require.NoError(t, err)
	assert.Equal(t, "/key", receivedPath)
}

func TestDeleteSecret(t *testing.T) {
	var receivedPath, receivedName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		assert.Equal(t, http.MethodDelete, r.Method)
		receivedPath = r.URL.Path
		receivedName = r.URL.Query().Get("name")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	err := c.DeleteSecret(context.Background(), &fakePushRemoteRef{
		remoteKey: "db-pass",
		property:  "password",
	})
	require.NoError(t, err)
	assert.Equal(t, "/password", receivedPath)
	assert.Equal(t, "db-pass", receivedName)
}

func TestSecretExists_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		writeJSON(w, api.Credential{Name: "my-pass", Value: "val"})
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	exists, err := c.SecretExists(context.Background(), &fakePushRemoteRef{
		remoteKey: "my-pass",
		property:  "password",
	})
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSecretExists_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	exists, err := c.SecretExists(context.Background(), &fakePushRemoteRef{
		remoteKey: "missing",
		property:  "key",
	})
	require.NoError(t, err)
	assert.False(t, exists)
}

// --- US3: GetAllSecrets ---

func TestGetAllSecrets(t *testing.T) {
	credentials := map[string]api.Credential{
		"password/db-pass": {Name: "db-pass", Value: "pass-val"},
		"key/api-key":      {Name: "api-key", Value: "key-val"},
		"certificate/cert": {Name: "cert", Value: "cert-val", Key: "key-pem"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNS(t, r)
		// List request: GET /{type}s  e.g. /passwords /keys /certificates
		// Item request: GET /{type}?name={name}
		path := strings.TrimPrefix(r.URL.Path, "/")
		name := r.URL.Query().Get("name")

		if name != "" {
			// Single credential GET
			for key, cred := range credentials {
				parts := strings.SplitN(key, "/", 2)
				if parts[0] == path && parts[1] == name {
					writeJSON(w, cred)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// List: path is plural e.g. "passwords" → type is "password"
		credType := strings.TrimSuffix(path, "s")
		var items []api.CredentialMeta
		for key := range credentials {
			parts := strings.SplitN(key, "/", 2)
			if parts[0] == credType {
				items = append(items, api.CredentialMeta{Name: parts[1], Type: credType})
			}
		}
		writeJSON(w, map[string]any{"content": items})
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	require.NoError(t, err)
	assert.Equal(t, map[string][]byte{
		"password/db-pass": []byte("pass-val"),
		"key/api-key":      []byte("key-val"),
		"certificate/cert": []byte("cert-val"),
	}, got)
}

func TestGetAllSecrets_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"content": []api.CredentialMeta{}})
	}))
	defer srv.Close()

	c := newHTTPTestClient(srv.URL)
	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	require.NoError(t, err)
	assert.Empty(t, got)
}
