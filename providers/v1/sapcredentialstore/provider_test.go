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
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// --- Test helpers ---

func generateTestJWEKeys(t *testing.T) (*JWEKeys, string, string) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	privB64 := base64.StdEncoding.EncodeToString(privDER)

	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pubB64 := base64.StdEncoding.EncodeToString(pubDER)

	keys := &JWEKeys{
		ClientPrivateKey: priv,
		ServerPublicKey:  &priv.PublicKey,
	}
	return keys, privB64, pubB64
}

// generateTestTLSPair generates a self-signed TLS certificate and private key in PEM format.
func generateTestTLSPair(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certBuf := &bytes.Buffer{}
	require.NoError(t, pem.Encode(certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))

	keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	keyBuf := &bytes.Buffer{}
	require.NoError(t, pem.Encode(keyBuf, &pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))

	return certBuf.String(), keyBuf.String()
}

// newMockServer creates a mock SAP Credential Store API server.
func newMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func newTestAPIClient(t *testing.T, server *httptest.Server, jweKeys *JWEKeys) *APIClient {
	t.Helper()
	return NewAPIClient(server.URL, server.Client(), jweKeys)
}

func newTestClient(t *testing.T, server *httptest.Server, namespace string, jweKeys *JWEKeys) *Client {
	t.Helper()
	return &Client{
		api:       newTestAPIClient(t, server, jweKeys),
		namespace: namespace,
	}
}

// makeSecretStore creates a minimal SecretStore with SAP CS provider config.
func makeSecretStore(cfg *esv1.SAPCredentialStoreProvider) *esv1.SecretStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-store",
			Namespace: "default",
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				SAPCredentialStore: cfg,
			},
		},
	}
}

func refSelector(name, namespace, key string) esmeta.SecretKeySelector {
	sel := esmeta.SecretKeySelector{
		Name: name,
		Key:  key,
	}
	if namespace != "" {
		sel.Namespace = &namespace
	}
	return sel
}

// --- ParseJWEKeys tests ---

func TestParseJWEKeys(t *testing.T) {
	_, privB64, pubB64 := generateTestJWEKeys(t)

	t.Run("valid keys", func(t *testing.T) {
		keys, err := ParseJWEKeys(privB64, pubB64)
		require.NoError(t, err)
		assert.NotNil(t, keys.ClientPrivateKey)
		assert.NotNil(t, keys.ServerPublicKey)
	})

	t.Run("invalid base64 private key", func(t *testing.T) {
		_, err := ParseJWEKeys("not-base64!!", pubB64)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decoding client private key")
	})

	t.Run("invalid base64 public key", func(t *testing.T) {
		_, err := ParseJWEKeys(privB64, "not-base64!!")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decoding server public key")
	})

	t.Run("invalid DER private key", func(t *testing.T) {
		badDER := base64.StdEncoding.EncodeToString([]byte("not a key"))
		_, err := ParseJWEKeys(badDER, pubB64)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parsing client private key")
	})

	t.Run("invalid DER public key", func(t *testing.T) {
		badDER := base64.StdEncoding.EncodeToString([]byte("not a key"))
		_, err := ParseJWEKeys(privB64, badDER)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parsing server public key")
	})
}

// --- JWE encrypt/decrypt round-trip ---

func TestJWERoundTrip(t *testing.T) {
	keys, _, _ := generateTestJWEKeys(t)

	plaintext := []byte(`{"name":"test","value":"secret123"}`)
	encrypted, err := encryptPayload(plaintext, keys.ServerPublicKey)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, string(plaintext), encrypted)

	decrypted, err := decryptPayload([]byte(encrypted), keys.ClientPrivateKey)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// --- API client tests ---

func TestAPIClient_GetCredential(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/password", r.URL.Path)
			assert.Equal(t, "my-secret", r.URL.Query().Get("name"))
			assert.Equal(t, "test-ns", r.Header.Get("sapcp-credstore-namespace"))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Credential{
				Name:  "my-secret",
				Value: "secret-value",
			})
		})

		client := newTestAPIClient(t, server, nil)
		cred, err := client.GetCredential(context.Background(), "test-ns", "password", "my-secret")
		require.NoError(t, err)
		assert.Equal(t, "my-secret", cred.Name)
		assert.Equal(t, "secret-value", cred.Value)
	})

	t.Run("not found includes credential context", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		client := newTestAPIClient(t, server, nil)
		_, err := client.GetCredential(context.Background(), "test-ns", "key", "missing-key")
		assert.Error(t, err)
		var nfe *NotFoundError
		assert.ErrorAs(t, err, &nfe)
		assert.Equal(t, "key", nfe.CredType)
		assert.Equal(t, "missing-key", nfe.Name)
		assert.Equal(t, "credential key/missing-key not found", nfe.Error())
	})

	t.Run("rate limit 429", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"errorCode":"rate_limit_exceeded"}`))
		})

		client := newTestAPIClient(t, server, nil)
		_, err := client.GetCredential(context.Background(), "test-ns", "password", "my-secret")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit exceeded")
		assert.Contains(t, err.Error(), "429")
		// Must not be a NotFoundError
		var nfe *NotFoundError
		assert.False(t, errors.As(err, &nfe))
	})

	t.Run("server error", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		})

		client := newTestAPIClient(t, server, nil)
		_, err := client.GetCredential(context.Background(), "test-ns", "password", "my-secret")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 500")
	})

	t.Run("with JWE encryption", func(t *testing.T) {
		keys, _, _ := generateTestJWEKeys(t)

		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			cred := Credential{Name: "enc-secret", Value: "encrypted-value"}
			plaintext, _ := json.Marshal(cred)
			encrypted, _ := encryptPayload(plaintext, keys.ServerPublicKey)
			w.Write([]byte(encrypted))
		})

		client := newTestAPIClient(t, server, keys)
		cred, err := client.GetCredential(context.Background(), "test-ns", "password", "enc-secret")
		require.NoError(t, err)
		assert.Equal(t, "enc-secret", cred.Name)
		assert.Equal(t, "encrypted-value", cred.Value)
	})
}

func TestAPIClient_ListCredentials(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/passwords", r.URL.Path) // pluralized
		assert.Equal(t, "test-ns", r.Header.Get("sapcp-credstore-namespace"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(credentialListResponse{
			Content: []CredentialMeta{
				{Name: "secret-a", Type: "password"},
				{Name: "secret-b", Type: "password"},
			},
			Last:          true,
			TotalElements: 2,
		})
	})

	client := newTestAPIClient(t, server, nil)
	metas, err := client.ListCredentials(context.Background(), "test-ns", "password")
	require.NoError(t, err)
	assert.Len(t, metas, 2)
	assert.Equal(t, "secret-a", metas[0].Name)
	assert.Equal(t, "secret-b", metas[1].Name)
}

func TestAPIClient_ListCredentials_Pagination(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/passwords", r.URL.Path)

		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")

		switch page {
		case "0":
			json.NewEncoder(w).Encode(credentialListResponse{
				Content: []CredentialMeta{
					{Name: "secret-1", Type: "password"},
					{Name: "secret-2", Type: "password"},
				},
				Last:          false,
				Number:        0,
				TotalElements: 3,
			})
		case "1":
			json.NewEncoder(w).Encode(credentialListResponse{
				Content: []CredentialMeta{
					{Name: "secret-3", Type: "password"},
				},
				Last:          true,
				Number:        1,
				TotalElements: 3,
			})
		default:
			t.Fatalf("unexpected page requested: %s", page)
		}
	})

	client := newTestAPIClient(t, server, nil)
	metas, err := client.ListCredentials(context.Background(), "test-ns", "password")
	require.NoError(t, err)
	assert.Len(t, metas, 3)
	assert.Equal(t, "secret-1", metas[0].Name)
	assert.Equal(t, "secret-2", metas[1].Name)
	assert.Equal(t, "secret-3", metas[2].Name)
}

func TestAPIClient_PutCredential(t *testing.T) {
	t.Run("without JWE", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/password", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "test-ns", r.Header.Get("sapcp-credstore-namespace"))

			body, _ := io.ReadAll(r.Body)
			var cb CredentialBody
			require.NoError(t, json.Unmarshal(body, &cb))
			assert.Equal(t, "my-cred", cb.Name)
			assert.Equal(t, "my-value", cb.Value)

			w.WriteHeader(http.StatusCreated)
		})

		client := newTestAPIClient(t, server, nil)
		err := client.PutCredential(context.Background(), "test-ns", "password", &CredentialBody{
			Name:  "my-cred",
			Value: "my-value",
		})
		require.NoError(t, err)
	})

	t.Run("with JWE", func(t *testing.T) {
		keys, _, _ := generateTestJWEKeys(t)

		server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/jose", r.Header.Get("Content-Type"))

			body, _ := io.ReadAll(r.Body)
			// Body should be a JWE compact serialization (5 dot-separated parts).
			parts := strings.Split(string(body), ".")
			assert.Len(t, parts, 5, "JWE compact serialization should have 5 parts")

			w.WriteHeader(http.StatusCreated)
		})

		client := newTestAPIClient(t, server, keys)
		err := client.PutCredential(context.Background(), "test-ns", "password", &CredentialBody{
			Name:  "enc-cred",
			Value: "enc-value",
		})
		require.NoError(t, err)
	})
}

func TestAPIClient_DeleteCredential(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/password", r.URL.Path)
		assert.Equal(t, "my-secret", r.URL.Query().Get("name"))
		assert.Equal(t, "test-ns", r.Header.Get("sapcp-credstore-namespace"))
		w.WriteHeader(http.StatusNoContent)
	})

	client := newTestAPIClient(t, server, nil)
	err := client.DeleteCredential(context.Background(), "test-ns", "password", "my-secret")
	require.NoError(t, err)
}

func TestAPIClient_CredentialExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Credential{Name: "my-secret", Value: "val"})
		})

		client := newTestAPIClient(t, server, nil)
		exists, err := client.CredentialExists(context.Background(), "test-ns", "password", "my-secret")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		client := newTestAPIClient(t, server, nil)
		exists, err := client.CredentialExists(context.Background(), "test-ns", "password", "missing")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// --- credTypeFromProperty tests ---

func TestCredTypeFromProperty(t *testing.T) {
	validTests := []struct {
		property string
		wantType string
	}{
		{"", "password"},
		{"password", "password"},
		{"Password", "password"},
		{"key", "key"},
		{"Key", "key"},
	}

	for _, tt := range validTests {
		t.Run(fmt.Sprintf("valid property=%q", tt.property), func(t *testing.T) {
			credType, err := credTypeFromProperty(tt.property)
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, credType)
		})
	}

	invalidTests := []string{
		"certificate",
		"Certificate",
		"certificate/key",
		"unknown",
		"keyring",
	}

	for _, prop := range invalidTests {
		t.Run(fmt.Sprintf("invalid property=%q", prop), func(t *testing.T) {
			_, err := credTypeFromProperty(prop)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported credential type")
		})
	}
}

// --- SecretsClient tests ---

func TestClient_GetSecret(t *testing.T) {
	t.Run("password", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode(Credential{Name: "my-pass", Value: "s3cret"})
		})

		c := newTestClient(t, server, "test-ns", nil)
		val, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key: "my-pass",
		})
		require.NoError(t, err)
		assert.Equal(t, "s3cret", string(val))
	})

	t.Run("key type base64 decoded", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("raw-key-data"))
		server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/key", r.URL.Path)
			assert.Equal(t, "my-key", r.URL.Query().Get("name"))
			json.NewEncoder(w).Encode(Credential{Name: "my-key", Value: encoded})
		})

		c := newTestClient(t, server, "test-ns", nil)
		val, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key:      "my-key",
			Property: "key",
		})
		require.NoError(t, err)
		assert.Equal(t, "raw-key-data", string(val))
	})

	t.Run("unsupported property returns error", func(t *testing.T) {
		c := newTestClient(t, newMockServer(t, func(http.ResponseWriter, *http.Request) {}), "test-ns", nil)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key:      "my-cert",
			Property: "certificate",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported credential type")
	})

	t.Run("not found returns NoSecretErr", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		c := newTestClient(t, server, "test-ns", nil)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key: "missing",
		})
		assert.ErrorIs(t, err, esv1.NoSecretErr)
	})
}

func TestClient_PushSecret(t *testing.T) {
	var receivedBody CredentialBody

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusCreated)
	})

	c := newTestClient(t, server, "test-ns", nil)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("push-value"),
		},
	}

	err := c.PushSecret(context.Background(), secret, &fakePushSecretData{
		secretKey: "my-key",
		remoteKey: "remote-name",
		property:  "password",
	})
	require.NoError(t, err)
	assert.Equal(t, "remote-name", receivedBody.Name)
	assert.Equal(t, "push-value", receivedBody.Value)
}

func TestClient_PushSecret_KeyType(t *testing.T) {
	var receivedBody CredentialBody
	var receivedPath string

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusCreated)
	})

	c := newTestClient(t, server, "test-ns", nil)

	rawData := []byte("raw-binary-key-data")
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": rawData,
		},
	}

	err := c.PushSecret(context.Background(), secret, &fakePushSecretData{
		secretKey: "my-key",
		remoteKey: "remote-key-name",
		property:  "key",
	})
	require.NoError(t, err)
	assert.Equal(t, "/key", receivedPath)
	assert.Equal(t, "remote-key-name", receivedBody.Name)
	// Value must be base64-encoded per SAP CS API spec for key credentials.
	assert.Equal(t, base64.StdEncoding.EncodeToString(rawData), receivedBody.Value)
}

func TestClient_DeleteSecret(t *testing.T) {
	var deletedPath string
	var deletedQuery string

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		deletedPath = r.URL.Path
		deletedQuery = r.URL.Query().Get("name")
		w.WriteHeader(http.StatusNoContent)
	})

	c := newTestClient(t, server, "test-ns", nil)
	err := c.DeleteSecret(context.Background(), &fakePushSecretRemoteRef{
		remoteKey: "to-delete",
		property:  "key",
	})
	require.NoError(t, err)
	assert.Equal(t, "/key", deletedPath)
	assert.Equal(t, "to-delete", deletedQuery)
}

func TestClient_SecretExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode(Credential{Name: "exists", Value: "val"})
		})

		c := newTestClient(t, server, "test-ns", nil)
		exists, err := c.SecretExists(context.Background(), &fakePushSecretRemoteRef{
			remoteKey: "exists",
			property:  "password",
		})
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		c := newTestClient(t, server, "test-ns", nil)
		exists, err := c.SecretExists(context.Background(), &fakePushSecretRemoteRef{
			remoteKey: "missing",
			property:  "password",
		})
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestClient_GetSecretMap(t *testing.T) {
	t.Run("password with username", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode(Credential{
				Name:     "my-cred",
				Value:    "pass123",
				Username: "admin",
			})
		})

		c := newTestClient(t, server, "test-ns", nil)
		m, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key: "my-cred",
		})
		require.NoError(t, err)
		assert.Equal(t, "my-cred", string(m["name"]))
		assert.Equal(t, "pass123", string(m["value"]))
		assert.Equal(t, "admin", string(m["username"]))
		_, hasKey := m["key"]
		assert.False(t, hasKey)
	})

	t.Run("unsupported property returns error", func(t *testing.T) {
		c := newTestClient(t, newMockServer(t, func(http.ResponseWriter, *http.Request) {}), "test-ns", nil)
		_, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key:      "my-cert",
			Property: "certificate",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported credential type")
	})

	t.Run("key type base64 decoded", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("decoded-value"))
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode(Credential{Name: "my-key", Value: encoded})
		})

		c := newTestClient(t, server, "test-ns", nil)
		m, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key:      "my-key",
			Property: "key",
		})
		require.NoError(t, err)
		assert.Equal(t, "decoded-value", string(m["value"]))
	})

	t.Run("not found returns NoSecretErr", func(t *testing.T) {
		server := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		c := newTestClient(t, server, "test-ns", nil)
		_, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
			Key: "missing",
		})
		assert.ErrorIs(t, err, esv1.NoSecretErr)
	})
}

func TestClient_GetAllSecrets(t *testing.T) {
	callCount := 0

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		path := r.URL.Path

		switch {
		// List endpoints (pluralized)
		case path == "/passwords":
			json.NewEncoder(w).Encode(credentialListResponse{
				Content: []CredentialMeta{
					{Name: "pass-a", Type: "password"},
					{Name: "pass-b", Type: "password"},
				},
				Last:          true,
				TotalElements: 2,
			})
		case path == "/keys":
			json.NewEncoder(w).Encode(credentialListResponse{
				Content: []CredentialMeta{
					{Name: "key-a", Type: "key"},
				},
				Last:          true,
				TotalElements: 1,
			})

		// Get endpoints
		case path == "/password" && r.URL.Query().Get("name") == "pass-a":
			json.NewEncoder(w).Encode(Credential{Name: "pass-a", Value: "val-a"})
		case path == "/password" && r.URL.Query().Get("name") == "pass-b":
			json.NewEncoder(w).Encode(Credential{Name: "pass-b", Value: "val-b"})
		case path == "/key" && r.URL.Query().Get("name") == "key-a":
			json.NewEncoder(w).Encode(Credential{
				Name:  "key-a",
				Value: base64.StdEncoding.EncodeToString([]byte("key-data")),
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	c := newTestClient(t, server, "test-ns", nil)
	result, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "val-a", string(result["password/pass-a"]))
	assert.Equal(t, "val-b", string(result["password/pass-b"]))
	assert.Equal(t, "key-data", string(result["key/key-a"]))
}

func TestClient_GetAllSecrets_WithRegexFilter(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case path == "/passwords":
			json.NewEncoder(w).Encode(credentialListResponse{
				Content: []CredentialMeta{
					{Name: "db-pass", Type: "password"},
					{Name: "api-pass", Type: "password"},
					{Name: "db-admin", Type: "password"},
				},
				Last:          true,
				TotalElements: 3,
			})
		case path == "/keys":
			json.NewEncoder(w).Encode(credentialListResponse{
				Content:       []CredentialMeta{},
				Last:          true,
				TotalElements: 0,
			})

		case path == "/password" && r.URL.Query().Get("name") == "db-pass":
			json.NewEncoder(w).Encode(Credential{Name: "db-pass", Value: "dbval"})
		case path == "/password" && r.URL.Query().Get("name") == "db-admin":
			json.NewEncoder(w).Encode(Credential{Name: "db-admin", Value: "adminval"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	c := newTestClient(t, server, "test-ns", nil)
	result, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
		Name: &esv1.FindName{RegExp: "^db-"},
	})
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "dbval", string(result["password/db-pass"]))
	assert.Equal(t, "adminval", string(result["password/db-admin"]))
}

func TestClient_GetAllSecrets_AllListsFail(t *testing.T) {
	// When every ListCredentials call fails (e.g. rate limit), GetAllSecrets
	// must return an error instead of silently returning an empty map.
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/passwords" || path == "/keys" {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error_code":"rate_limit_exceeded"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	c := newTestClient(t, server, "test-ns", nil)
	result, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list any credential types")
	assert.Contains(t, err.Error(), "password")
	assert.Contains(t, err.Error(), "key")
}

func TestClient_GetAllSecrets_PartialListFailure(t *testing.T) {
	// When only one credential type fails to list, GetAllSecrets should
	// still return results from the other type (partial degradation).
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/passwords":
			json.NewEncoder(w).Encode(credentialListResponse{
				Content: []CredentialMeta{
					{Name: "my-pass", Type: "password"},
				},
				Last:          true,
				TotalElements: 1,
			})
		case path == "/keys":
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error_code":"rate_limit_exceeded"}`))
		case path == "/password" && r.URL.Query().Get("name") == "my-pass":
			json.NewEncoder(w).Encode(Credential{Name: "my-pass", Value: "secret"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	c := newTestClient(t, server, "test-ns", nil)
	result, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "secret", string(result["password/my-pass"]))
}

func TestClient_Validate(t *testing.T) {
	c := &Client{}
	result, err := c.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultUnknown, result)
}

func TestClient_Close(t *testing.T) {
	c := &Client{}
	err := c.Close(context.Background())
	require.NoError(t, err)
}

// --- Provider tests ---

func TestProvider_Capabilities(t *testing.T) {
	p := &Provider{}
	assert.Equal(t, esv1.SecretStoreReadWrite, p.Capabilities())
}

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	assert.NotNil(t, p)
	assert.IsType(t, &Provider{}, p)
}

func TestProviderSpec(t *testing.T) {
	spec := ProviderSpec()
	assert.NotNil(t, spec)
	assert.NotNil(t, spec.SAPCredentialStore)
}

func TestMaintenanceStatus(t *testing.T) {
	assert.Equal(t, esv1.MaintenanceStatusMaintained, MaintenanceStatus())
}

// --- ValidateStore tests ---

func TestValidateStore(t *testing.T) {
	p := &Provider{}
	ns := "default"

	tests := []struct {
		name        string
		store       esv1.GenericStore
		wantErr     string
		wantWarning string
	}{
		{
			name:    "nil store",
			store:   nil,
			wantErr: errNilStore,
		},
		{
			name: "empty spec has nil provider",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
			},
			wantErr: errMissingProvider,
		},
		{
			name: "nil provider",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec:       esv1.SecretStoreSpec{},
			},
			wantErr: errMissingProvider,
		},
		{
			name: "nil SAP CS config",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{},
				},
			},
			wantErr: "invalid provider spec",
		},
		{
			name: "missing serviceURL",
			store: makeSecretStore(&esv1.SAPCredentialStoreProvider{
				Namespace: "ns",
				Auth: &esv1.SAPCSAuth{
					MTLS: &esv1.SAPCSMTLSAuth{
						Certificate: refSelector("secret", ns, "cert"),
						PrivateKey:  refSelector("secret", ns, "key"),
					},
				},
			}),
			wantErr: errMissingServiceURL,
		},
		{
			name: "missing namespace",
			store: makeSecretStore(&esv1.SAPCredentialStoreProvider{
				ServiceURL: "https://credstore.example.com",
				Auth: &esv1.SAPCSAuth{
					MTLS: &esv1.SAPCSMTLSAuth{
						Certificate: refSelector("secret", ns, "cert"),
						PrivateKey:  refSelector("secret", ns, "key"),
					},
				},
			}),
			wantErr: errMissingNamespace,
		},
		{
			name: "missing auth",
			store: makeSecretStore(&esv1.SAPCredentialStoreProvider{
				ServiceURL: "https://credstore.example.com",
				Namespace:  "my-ns",
			}),
			wantErr: errMissingAuth,
		},
		{
			name: "valid mtls config",
			store: makeSecretStore(&esv1.SAPCredentialStoreProvider{
				ServiceURL: "https://credstore.example.com",
				Namespace:  "my-ns",
				Auth: &esv1.SAPCSAuth{
					MTLS: &esv1.SAPCSMTLSAuth{
						Certificate: refSelector("secret", ns, "cert"),
						PrivateKey:  refSelector("secret", ns, "key"),
					},
				},
			}),
		},
		{
			name: "valid service binding ref",
			store: makeSecretStore(&esv1.SAPCredentialStoreProvider{
				Namespace: "cs-namespace",
				ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
					Name: "my-binding",
				},
			}),
		},
		{
			name: "service binding ref missing name",
			store: makeSecretStore(&esv1.SAPCredentialStoreProvider{
				ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{},
			}),
			wantErr: "serviceBindingSecretRef.name is required",
		},
		{
			name: "service binding missing namespace",
			store: makeSecretStore(&esv1.SAPCredentialStoreProvider{
				ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
					Name: "my-binding",
				},
			}),
			wantErr: errMissingNamespace,
		},
		{
			name: "service binding with auth warns",
			store: makeSecretStore(&esv1.SAPCredentialStoreProvider{
				Namespace: "cs-namespace",
				ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
					Name: "my-binding",
				},
				Auth: &esv1.SAPCSAuth{
					MTLS: &esv1.SAPCSMTLSAuth{
						Certificate: refSelector("secret", ns, "cert"),
						PrivateKey:  refSelector("secret", ns, "key"),
					},
				},
			}),
			wantWarning: "serviceBindingSecretRef takes precedence",
		},
		{
			name: "valid encryption config",
			store: makeSecretStore(&esv1.SAPCredentialStoreProvider{
				ServiceURL: "https://credstore.example.com",
				Namespace:  "my-ns",
				Auth: &esv1.SAPCSAuth{
					MTLS: &esv1.SAPCSMTLSAuth{
						Certificate: refSelector("secret", ns, "cert"),
						PrivateKey:  refSelector("secret", ns, "key"),
					},
				},
				Encryption: &esv1.SAPCSEncryption{
					ClientPrivateKey: refSelector("enc-secret", ns, "privkey"),
					ServerPublicKey:  refSelector("enc-secret", ns, "pubkey"),
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := p.ValidateStore(tt.store)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantWarning != "" {
				require.Len(t, warnings, 1)
				assert.Contains(t, warnings[0], tt.wantWarning)
			}
		})
	}
}

// --- NewClient tests ---

func TestNewClient_ServiceBinding(t *testing.T) {
	certPEM, keyPEM := generateTestTLSPair(t)

	bindingJSON := []byte(fmt.Sprintf(`{
		"url": "https://credstore.example.com/api/v1/credentials",
		"certificate": %s,
		"key": %s,
		"parameters": {
			"authentication": {"type": "mtls"},
			"encryption": {"payload": "disabled"}
		}
	}`, jsonString(certPEM), jsonString(keyPEM)))

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-binding",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"credentials": bindingJSON,
			},
		}).
		Build()

	store := makeSecretStore(&esv1.SAPCredentialStoreProvider{
		Namespace: "cs-namespace",
		ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
			Name: "my-binding",
		},
	})

	p := &Provider{}
	client, err := p.NewClient(context.Background(), store, kubeClient, "default")
	require.NoError(t, err)
	assert.NotNil(t, client)

	sc, ok := client.(*Client)
	require.True(t, ok)
	assert.Equal(t, "cs-namespace", sc.namespace)
	// No JWE keys since encryption is disabled.
	assert.Nil(t, sc.api.jweKeys)
}

func TestNewClient_ServiceBindingWithEncryption(t *testing.T) {
	certPEM, keyPEM := generateTestTLSPair(t)
	_, privB64, pubB64 := generateTestJWEKeys(t)

	bindingJSON := []byte(fmt.Sprintf(`{
		"url": "https://credstore.example.com/api/v1/credentials",
		"certificate": %s,
		"key": %s,
		"encryption": {
			"client_private_key": %q,
			"server_public_key": %q
		},
		"parameters": {
			"authentication": {"type": "mtls"},
			"encryption": {"payload": "enabled"}
		}
	}`, jsonString(certPEM), jsonString(keyPEM), privB64, pubB64))

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-binding",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"credentials": bindingJSON,
			},
		}).
		Build()

	store := makeSecretStore(&esv1.SAPCredentialStoreProvider{
		Namespace: "cs-namespace",
		ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
			Name: "my-binding",
		},
	})

	p := &Provider{}
	client, err := p.NewClient(context.Background(), store, kubeClient, "default")
	require.NoError(t, err)
	assert.NotNil(t, client)

	sc, ok := client.(*Client)
	require.True(t, ok)
	// JWE keys should have been auto-derived from the binding.
	assert.NotNil(t, sc.api.jweKeys)
}

func TestNewClient_ServiceBindingMissingCert(t *testing.T) {
	// Binding has mtls type but is missing the certificate field.
	bindingJSON := []byte(`{
		"url": "https://credstore.example.com/api/v1/credentials",
		"key": "-----BEGIN PRIVATE KEY-----\nfake\n-----END PRIVATE KEY-----\n",
		"parameters": {"authentication": {"type": "mtls"}}
	}`)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bad-binding",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"credentials": bindingJSON,
			},
		}).
		Build()

	store := makeSecretStore(&esv1.SAPCredentialStoreProvider{
		Namespace: "cs-namespace",
		ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
			Name: "bad-binding",
		},
	})

	p := &Provider{}
	_, err := p.NewClient(context.Background(), store, kubeClient, "default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field for mtls auth: certificate")
}

func TestNewClient_ServiceBindingMissingURL(t *testing.T) {
	certPEM, keyPEM := generateTestTLSPair(t)

	bindingJSON := []byte(fmt.Sprintf(`{
		"certificate": %s,
		"key": %s,
		"parameters": {"authentication": {"type": "mtls"}}
	}`, jsonString(certPEM), jsonString(keyPEM)))

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bad-binding",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"credentials": bindingJSON,
			},
		}).
		Build()

	store := makeSecretStore(&esv1.SAPCredentialStoreProvider{
		Namespace: "cs-namespace",
		ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
			Name: "bad-binding",
		},
	})

	p := &Provider{}
	_, err := p.NewClient(context.Background(), store, kubeClient, "default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field: url")
}

func TestNewClient_ServiceBindingUnsupportedType(t *testing.T) {
	tests := []struct {
		name    string
		binding string
		wantErr string
	}{
		{
			name: "oauth:mtls not supported",
			binding: `{
				"url": "https://credstore.example.com",
				"parameters": {"authentication": {"type": "oauth:mtls"}}
			}`,
			wantErr: "not yet supported",
		},
		{
			name: "oauth:key not supported",
			binding: `{
				"url": "https://credstore.example.com",
				"parameters": {"authentication": {"type": "oauth:key"}}
			}`,
			wantErr: "not yet supported",
		},
		{
			name: "basic not supported",
			binding: `{
				"url": "https://credstore.example.com",
				"parameters": {"authentication": {"type": "basic"}}
			}`,
			wantErr: "not yet supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)

			kubeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "binding",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"credentials": []byte(tt.binding),
					},
				}).
				Build()

			store := makeSecretStore(&esv1.SAPCredentialStoreProvider{
				Namespace: "cs-namespace",
				ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
					Name: "binding",
				},
			})

			p := &Provider{}
			_, err := p.NewClient(context.Background(), store, kubeClient, "default")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestNewClient_ServiceBindingMissingSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	store := makeSecretStore(&esv1.SAPCredentialStoreProvider{
		Namespace: "cs-namespace",
		ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
			Name: "nonexistent",
		},
	})

	p := &Provider{}
	_, err := p.NewClient(context.Background(), store, kubeClient, "default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetching service binding secret")
}

func TestNewClient_NoAuth(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	store := makeSecretStore(&esv1.SAPCredentialStoreProvider{
		ServiceURL: "https://credstore.example.com",
		Namespace:  "my-ns",
	})

	p := &Provider{}
	_, err := p.NewClient(context.Background(), store, kubeClient, "default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), errMissingAuth)
}

func TestNewClient_NilProvider(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{},
		},
	}

	p := &Provider{}
	_, err := p.NewClient(context.Background(), store, kubeClient, "default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid provider spec")
}

// --- detectBindingAuthType tests ---

func TestDetectBindingAuthType(t *testing.T) {
	tests := []struct {
		name    string
		binding serviceBinding
		want    string
	}{
		{
			name: "explicit mtls type",
			binding: serviceBinding{
				Certificate: "cert",
				Key:         "key",
				Parameters:  &serviceBindingParameters{Authentication: &serviceBindingAuthParams{Type: "mtls"}},
			},
			want: "mtls",
		},
		{
			name: "explicit oauth:mtls type",
			binding: serviceBinding{
				Certificate:   "cert",
				Key:           "key",
				OAuthTokenURL: "https://oauth.example.com",
				Parameters:    &serviceBindingParameters{Authentication: &serviceBindingAuthParams{Type: "oauth:mtls"}},
			},
			want: "oauth:mtls",
		},
		{
			name: "explicit oauth:key type",
			binding: serviceBinding{
				Key:           "key",
				OAuthTokenURL: "https://oauth.example.com",
				Parameters:    &serviceBindingParameters{Authentication: &serviceBindingAuthParams{Type: "oauth:key"}},
			},
			want: "oauth:key",
		},
		{
			name: "heuristic: cert+key without oauth → mtls",
			binding: serviceBinding{
				Certificate: "cert",
				Key:         "key",
			},
			want: "mtls",
		},
		{
			name: "heuristic: key only without oauth → mtls",
			binding: serviceBinding{
				Key: "key",
			},
			want: "mtls",
		},
		{
			name: "heuristic: oauth_token_url present → oauth:unknown",
			binding: serviceBinding{
				Certificate:   "cert",
				Key:           "key",
				OAuthTokenURL: "https://oauth.example.com",
			},
			want: "oauth:unknown",
		},
		{
			name:    "no fields at all → empty",
			binding: serviceBinding{},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectBindingAuthType(&tt.binding)
			assert.Equal(t, tt.want, got)
		})
	}
}

// jsonString marshals a string to a JSON string literal (with proper escaping).
// Useful for embedding PEM values in JSON templates.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// --- Fake implementations for testing ---

type fakePushSecretData struct {
	secretKey string
	remoteKey string
	property  string
}

func (f *fakePushSecretData) GetMetadata() *apiextensionsv1.JSON { return nil }
func (f *fakePushSecretData) GetSecretKey() string               { return f.secretKey }
func (f *fakePushSecretData) GetRemoteKey() string               { return f.remoteKey }
func (f *fakePushSecretData) GetProperty() string                { return f.property }

type fakePushSecretRemoteRef struct {
	remoteKey string
	property  string
}

func (f *fakePushSecretRemoteRef) GetRemoteKey() string { return f.remoteKey }
func (f *fakePushSecretRemoteRef) GetProperty() string  { return f.property }
