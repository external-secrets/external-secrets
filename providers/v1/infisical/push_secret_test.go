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

package infisical

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/infisical/api"
	"github.com/external-secrets/external-secrets/runtime/testing/fake"
)

const testProjectID = "11111111-1111-1111-1111-111111111111"

// infisicalMock is a stateful fake of the Infisical raw-secret and project
// endpoints the write path exercises. It records call counts so tests can
// assert create-vs-update behavior and project-ID caching.
type infisicalMock struct {
	mu           sync.Mutex
	secrets      map[string]string // name -> value
	projectID    string            // ID returned by the slug lookup (mutable)
	creates      int
	updates      int
	deletes      int
	slugLookups  int
	failSlug     bool // when true, the slug -> ID lookup returns 500
	slugNotFound bool // when true, every slug -> ID route returns 404
	// fallbackToWorkspace makes the /projects/slug route 404 and only the
	// legacy /workspace/slug route resolve, exercising the resolver fallback.
	fallbackToWorkspace bool
}

func newInfisicalMock() *infisicalMock {
	return &infisicalMock{secrets: map[string]string{}, projectID: testProjectID}
}

func (m *infisicalMock) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()

		switch {
		case r.Method == http.MethodGet && (strings.HasPrefix(r.URL.Path, "/api/v1/projects/slug/") || strings.HasPrefix(r.URL.Path, "/api/v1/workspace/slug/")):
			m.slugLookups++
			if m.failSlug {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
				return
			}
			if m.slugNotFound {
				writeJSON(w, http.StatusNotFound, map[string]string{"message": "project not found"})
				return
			}
			onProjects := strings.HasPrefix(r.URL.Path, "/api/v1/projects/slug/")
			if m.fallbackToWorkspace && onProjects {
				// Simulate an older Infisical that lacks the /projects route.
				writeJSON(w, http.StatusNotFound, map[string]string{"message": "Route not found"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"id": m.projectID})

		case strings.HasPrefix(r.URL.Path, "/api/v3/secrets/raw/"):
			name := strings.TrimPrefix(r.URL.Path, "/api/v3/secrets/raw/")
			m.handleSecret(w, r, name)

		default:
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}
	}
}

func (m *infisicalMock) handleSecret(w http.ResponseWriter, r *http.Request, name string) {
	switch r.Method {
	case http.MethodGet:
		// Reads resolve the project server-side from the slug, so they do not
		// carry (or depend on) a workspace ID.
		value, ok := m.secrets[name]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "secret not found"})
			return
		}
		writeSecret(w, name, value)
	case http.MethodPost:
		value, wsID := readSecretBody(r)
		if m.staleWorkspace(w, wsID) {
			return
		}
		m.creates++
		m.secrets[name] = value
		writeSecret(w, name, value)
	case http.MethodPatch:
		value, wsID := readSecretBody(r)
		if m.staleWorkspace(w, wsID) {
			return
		}
		m.updates++
		m.secrets[name] = value
		writeSecret(w, name, value)
	case http.MethodDelete:
		_, wsID := readSecretBody(r)
		if m.staleWorkspace(w, wsID) {
			return
		}
		if _, ok := m.secrets[name]; !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "secret not found"})
			return
		}
		m.deletes++
		value := m.secrets[name]
		delete(m.secrets, name)
		writeSecret(w, name, value)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method not allowed"})
	}
}

// staleWorkspace mirrors the real API: a write whose workspaceId does not match
// the current project returns 404, which the provider treats as a stale cached
// project ID.
func (m *infisicalMock) staleWorkspace(w http.ResponseWriter, wsID string) bool {
	if wsID != m.projectID {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "project not found"})
		return true
	}
	return false
}

func readSecretBody(r *http.Request) (value, workspaceID string) {
	var body struct {
		SecretValue string `json:"secretValue"`
		WorkspaceID string `json:"workspaceId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	return body.SecretValue, body.WorkspaceID
}

func writeSecret(w http.ResponseWriter, name, value string) {
	writeJSON(w, http.StatusOK, map[string]any{
		"secret": map[string]any{"secretKey": name, "secretValue": value},
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// newPushTestProvider wires a Provider to a fresh plain-HTTP mock server. Each
// server gets a unique URL, so the package-level project-ID cache never
// collides between tests.
func newPushTestProvider(t *testing.T) (*Provider, *infisicalMock) {
	return buildPushTestProvider(t, newInfisicalMock(), false)
}

func buildPushTestProvider(t *testing.T, mock *infisicalMock, useTLS bool) (*Provider, *infisicalMock) {
	t.Helper()

	var server *httptest.Server
	if useTLS {
		server = httptest.NewTLSServer(mock.handler())
	} else {
		server = httptest.NewServer(mock.handler())
	}
	t.Cleanup(server.Close)

	sdkClient, cancel, err := api.NewAPIClient(server.URL, server.Certificate())
	require.NoError(t, err)
	t.Cleanup(cancel)

	p := &Provider{
		sdkClient: sdkClient,
		apiScope: &ClientScope{
			SecretPath:      "/",
			ProjectSlug:     "first-project",
			EnvironmentSlug: "dev",
		},
		hostAPI:      server.URL,
		authIdentity: "test-identity",
	}
	if useTLS && server.Certificate() != nil {
		p.caCertificate = string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: server.Certificate().Raw,
		}))
	}
	return p, mock
}

func TestPushSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("creates a missing secret", func(t *testing.T) {
		p, mock := newPushTestProvider(t)
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("value")}}

		err := p.PushSecret(ctx, secret, fake.PushSecretData{SecretKey: "key", RemoteKey: "remote"})
		require.NoError(t, err)
		assert.Equal(t, 1, mock.creates)
		assert.Equal(t, 0, mock.updates)
		assert.Equal(t, "value", mock.secrets["remote"])
	})

	t.Run("updates an existing secret with a new value", func(t *testing.T) {
		p, mock := newPushTestProvider(t)
		mock.secrets["remote"] = "old"
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("new")}}

		err := p.PushSecret(ctx, secret, fake.PushSecretData{SecretKey: "key", RemoteKey: "remote"})
		require.NoError(t, err)
		assert.Equal(t, 0, mock.creates)
		assert.Equal(t, 1, mock.updates)
		assert.Equal(t, "new", mock.secrets["remote"])
	})

	t.Run("is a no-op when the value already matches", func(t *testing.T) {
		p, mock := newPushTestProvider(t)
		mock.secrets["remote"] = "same"
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("same")}}

		err := p.PushSecret(ctx, secret, fake.PushSecretData{SecretKey: "key", RemoteKey: "remote"})
		require.NoError(t, err)
		assert.Equal(t, 0, mock.creates)
		assert.Equal(t, 0, mock.updates)
	})

	t.Run("pushes the whole secret as JSON when no secretKey is set", func(t *testing.T) {
		p, mock := newPushTestProvider(t)
		secret := &corev1.Secret{Data: map[string][]byte{"a": []byte("1"), "b": []byte("2")}}

		err := p.PushSecret(ctx, secret, fake.PushSecretData{RemoteKey: "remote"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":"1","b":"2"}`, mock.secrets["remote"])
	})

	t.Run("merges into a JSON property", func(t *testing.T) {
		p, mock := newPushTestProvider(t)
		mock.secrets["remote"] = `{"existing":"keep"}`
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("bar")}}

		err := p.PushSecret(ctx, secret, fake.PushSecretData{SecretKey: "key", RemoteKey: "remote", Property: "added"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"existing":"keep","added":"bar"}`, mock.secrets["remote"])
	})

	t.Run("errors when the source key is absent", func(t *testing.T) {
		p, _ := newPushTestProvider(t)
		secret := &corev1.Secret{Data: map[string][]byte{"other": []byte("v")}}

		err := p.PushSecret(ctx, secret, fake.PushSecretData{SecretKey: "missing", RemoteKey: "remote"})
		require.Error(t, err)
	})

	t.Run("errors when remoteKey is empty", func(t *testing.T) {
		p, _ := newPushTestProvider(t)
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("v")}}

		err := p.PushSecret(ctx, secret, fake.PushSecretData{SecretKey: "key"})
		require.ErrorIs(t, err, errMissingRemoteKey)
	})
}

func TestDeleteSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes an existing secret", func(t *testing.T) {
		p, mock := newPushTestProvider(t)
		mock.secrets["remote"] = "value"

		err := p.DeleteSecret(ctx, fake.PushSecretData{RemoteKey: "remote"})
		require.NoError(t, err)
		assert.Equal(t, 1, mock.deletes)
		_, exists := mock.secrets["remote"]
		assert.False(t, exists)
	})

	t.Run("is idempotent on a missing secret", func(t *testing.T) {
		p, mock := newPushTestProvider(t)

		err := p.DeleteSecret(ctx, fake.PushSecretData{RemoteKey: "remote"})
		require.NoError(t, err)
		assert.Equal(t, 0, mock.deletes)
	})

	t.Run("removes only the property, keeping the secret", func(t *testing.T) {
		p, mock := newPushTestProvider(t)
		mock.secrets["remote"] = `{"a":"1","b":"2"}`

		err := p.DeleteSecret(ctx, fake.PushSecretData{RemoteKey: "remote", Property: "a"})
		require.NoError(t, err)
		assert.Equal(t, 0, mock.deletes)
		assert.JSONEq(t, `{"b":"2"}`, mock.secrets["remote"])
	})

	t.Run("deletes the secret when the last property is removed", func(t *testing.T) {
		p, mock := newPushTestProvider(t)
		mock.secrets["remote"] = `{"a":"1"}`

		err := p.DeleteSecret(ctx, fake.PushSecretData{RemoteKey: "remote", Property: "a"})
		require.NoError(t, err)
		assert.Equal(t, 1, mock.deletes)
	})

	t.Run("errors when the existing value is not a JSON object and a property is set", func(t *testing.T) {
		for _, nonObject := range []string{"plain-string", `"valid-json-string"`, `[1,2,3]`} {
			p, mock := newPushTestProvider(t)
			mock.secrets["remote"] = nonObject

			err := p.DeleteSecret(ctx, fake.PushSecretData{RemoteKey: "remote", Property: "key"})
			require.Errorf(t, err, "expected error for value %q", nonObject)
			assert.Contains(t, err.Error(), "not a JSON object")
			assert.Equal(t, 0, mock.deletes)
		}
	})
}

func TestSecretExists(t *testing.T) {
	ctx := context.Background()

	t.Run("true when present", func(t *testing.T) {
		p, mock := newPushTestProvider(t)
		mock.secrets["remote"] = "value"

		exists, err := p.SecretExists(ctx, fake.PushSecretData{RemoteKey: "remote"})
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("false when absent", func(t *testing.T) {
		p, _ := newPushTestProvider(t)

		exists, err := p.SecretExists(ctx, fake.PushSecretData{RemoteKey: "remote"})
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("property presence", func(t *testing.T) {
		p, mock := newPushTestProvider(t)
		mock.secrets["remote"] = `{"a":"1"}`

		got, err := p.SecretExists(ctx, fake.PushSecretData{RemoteKey: "remote", Property: "a"})
		require.NoError(t, err)
		assert.True(t, got)

		got, err = p.SecretExists(ctx, fake.PushSecretData{RemoteKey: "remote", Property: "missing"})
		require.NoError(t, err)
		assert.False(t, got)
	})
}

func TestResolveProjectIDCaching(t *testing.T) {
	ctx := context.Background()
	p, mock := newPushTestProvider(t)
	secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("value")}}

	for i := range 3 {
		err := p.PushSecret(ctx, secret, fake.PushSecretData{SecretKey: "key", RemoteKey: fmt.Sprintf("remote-%d", i)})
		require.NoError(t, err)
	}

	// Three pushes, but the slug -> project ID lookup must happen only once.
	assert.Equal(t, 1, mock.slugLookups)
}

func TestResolveProjectIDLookupFailure(t *testing.T) {
	mock := newInfisicalMock()
	mock.failSlug = true
	p, mock := buildPushTestProvider(t, mock, false)
	secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("value")}}

	err := p.PushSecret(context.Background(), secret, fake.PushSecretData{SecretKey: "key", RemoteKey: "remote"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project id")
	// The push must abort before any write when the project cannot be resolved.
	assert.Equal(t, 0, mock.creates)
	assert.Equal(t, 0, mock.updates)
}

func TestPushSecretOverTLSWithCABundle(t *testing.T) {
	p, mock := buildPushTestProvider(t, newInfisicalMock(), true)
	// The CA bundle is required: the resolver's HTTP client must trust the
	// test server's self-signed certificate to perform the slug lookup.
	require.NotEmpty(t, p.caCertificate)
	secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("value")}}

	err := p.PushSecret(context.Background(), secret, fake.PushSecretData{SecretKey: "key", RemoteKey: "remote"})
	require.NoError(t, err)
	assert.Equal(t, 1, mock.slugLookups)
	assert.Equal(t, "value", mock.secrets["remote"])
}

func TestResolveProjectIDFallsBackToWorkspaceRoute(t *testing.T) {
	mock := newInfisicalMock()
	mock.fallbackToWorkspace = true
	p, mock := buildPushTestProvider(t, mock, false)
	secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("value")}}

	err := p.PushSecret(context.Background(), secret, fake.PushSecretData{SecretKey: "key", RemoteKey: "remote"})
	require.NoError(t, err)
	// Both the /projects (404) and the legacy /workspace (200) routes are hit.
	assert.Equal(t, 2, mock.slugLookups)
	assert.Equal(t, "value", mock.secrets["remote"])
}

func TestPushSecretReResolvesStaleProjectID(t *testing.T) {
	ctx := context.Background()
	mock := newInfisicalMock()
	mock.projectID = "project-A"
	p, mock := buildPushTestProvider(t, mock, false)
	secret := func(v string) *corev1.Secret {
		return &corev1.Secret{Data: map[string][]byte{"key": []byte(v)}}
	}

	// First push resolves and caches project-A.
	require.NoError(t, p.PushSecret(ctx, secret("v1"), fake.PushSecretData{SecretKey: "key", RemoteKey: "remote"}))
	assert.Equal(t, "v1", mock.secrets["remote"])
	assert.Equal(t, 1, mock.slugLookups)

	// Simulate the project being deleted and recreated under the same slug with
	// a new ID. The cached project-A is now stale.
	mock.projectID = "project-B"

	// Pushing a new secret must hit the stale 404, invalidate, re-resolve to
	// project-B, and retry the write.
	require.NoError(t, p.PushSecret(ctx, secret("v2"), fake.PushSecretData{SecretKey: "key", RemoteKey: "remote-after"}))
	assert.Equal(t, "v2", mock.secrets["remote-after"])
	assert.Equal(t, 2, mock.slugLookups, "expected one re-resolve after the stale 404")

	// The freshly cached project-B is reused without another lookup.
	require.NoError(t, p.PushSecret(ctx, secret("v3"), fake.PushSecretData{SecretKey: "key", RemoteKey: "remote-third"}))
	assert.Equal(t, "v3", mock.secrets["remote-third"])
	assert.Equal(t, 2, mock.slugLookups)
}

func TestPushSecretStaleProjectGoneReturnsError(t *testing.T) {
	ctx := context.Background()
	mock := newInfisicalMock()
	mock.projectID = "project-A"
	p, mock := buildPushTestProvider(t, mock, false)
	secret := func(v string) *corev1.Secret {
		return &corev1.Secret{Data: map[string][]byte{"key": []byte(v)}}
	}

	// Prime the cache with project-A.
	require.NoError(t, p.PushSecret(ctx, secret("v1"), fake.PushSecretData{SecretKey: "key", RemoteKey: "remote"}))

	// The project is deleted: writes 404 on the stale ID and the slug no longer
	// resolves to any project.
	mock.projectID = "gone"
	mock.slugNotFound = true

	// The write 404s, the cache is invalidated, the re-resolve also 404s, so a
	// clear "no such project" error surfaces (not the raw write 404).
	err := p.PushSecret(ctx, secret("v2"), fake.PushSecretData{SecretKey: "key", RemoteKey: "remote-new"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Infisical project found")
}

func TestAppendAPIEndpoint(t *testing.T) {
	assert.Equal(t, "https://h/api", appendAPIEndpoint("https://h"))
	assert.Equal(t, "https://h/api", appendAPIEndpoint("https://h/"))
	assert.Equal(t, "https://h/api", appendAPIEndpoint("https://h/api"))
}

var _ esv1.PushSecretData = fake.PushSecretData{}
