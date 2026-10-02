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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	infisicalSdk "github.com/infisical/go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1meta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func credential(token string, expiresIn int64) infisicalSdk.MachineIdentityCredential {
	return infisicalSdk.MachineIdentityCredential{AccessToken: token, ExpiresIn: expiresIn}
}

func TestTokenCacheRefreshMargin(t *testing.T) {
	tests := []struct {
		expiresIn int64
		refreshAt time.Duration
	}{
		{expiresIn: 3600, refreshAt: 55 * time.Minute},
		{expiresIn: 60, refreshAt: 48 * time.Second},
	}
	for _, tt := range tests {
		clock := &fakeClock{t: time.Unix(0, 0)}
		cache := newTokenCache(clock.now)
		require.True(t, cache.store("key", credential("token", tt.expiresIn), clock.t))

		clock.t = time.Unix(0, 0).Add(tt.refreshAt - time.Second)
		_, ok := cache.get("key")
		assert.True(t, ok)
		clock.t = time.Unix(0, 0).Add(tt.refreshAt)
		_, ok = cache.get("key")
		assert.False(t, ok)

		require.True(t, cache.store("other", credential("token", tt.expiresIn), clock.t))
		assert.NotContains(t, cache.entries, "key", "a write drops the expired entries")
	}
}

func TestTokenCacheDoesNotCacheLoginError(t *testing.T) {
	cache := newTokenCache(time.Now)
	_, _, err := cache.login("key", func() (infisicalSdk.MachineIdentityCredential, error) {
		return infisicalSdk.MachineIdentityCredential{}, errors.New("rate limited")
	})
	require.Error(t, err)

	_, cached, err := cache.login("key", func() (infisicalSdk.MachineIdentityCredential, error) {
		return credential("token", 3600), nil
	})
	require.NoError(t, err)
	assert.True(t, cached)
}

// concurrentLogins runs 10 logins for one key. The first login blocks until
// all callers wait inside the cache.
func concurrentLogins(t *testing.T, expiresIn int64) (logins int32, tokens []string, cache *tokenCache) {
	tokens = make([]string, 10)
	synctest.Test(t, func(t *testing.T) {
		cache = newTokenCache(time.Now)
		var n atomic.Int32
		release := make(chan struct{})
		loginFn := func() (infisicalSdk.MachineIdentityCredential, error) {
			id := n.Add(1)
			if id == 1 {
				<-release
			}
			return credential(fmt.Sprintf("token-%d", id), expiresIn), nil
		}

		var wg sync.WaitGroup
		for i := range tokens {
			wg.Go(func() {
				token, cached, err := cache.login("key", loginFn)
				assert.NoError(t, err)
				assert.Equal(t, expiresIn > 0, cached)
				tokens[i] = token
			})
		}
		synctest.Wait()
		close(release)
		wg.Wait()
		logins = n.Load()
	})
	return logins, tokens, cache
}

func TestTokenCacheConcurrentLoginsShareOneLogin(t *testing.T) {
	logins, tokens, _ := concurrentLogins(t, 3600)
	assert.Equal(t, int32(1), logins)
	for _, token := range tokens {
		assert.Equal(t, "token-1", token)
	}
}

func TestTokenCacheConcurrentLoginsWithoutTTL(t *testing.T) {
	// Each caller revokes its token on Close, so no two callers may share one.
	logins, tokens, cache := concurrentLogins(t, 0)
	assert.Equal(t, int32(10), logins)
	for _, token := range tokens {
		assert.True(t, cache.releaseOwned(token), token)
	}
}

func TestTokenCacheInvalidateToken(t *testing.T) {
	cache := newTokenCache(time.Now)
	require.True(t, cache.store("key", credential("new", 3600), time.Now()))

	cache.invalidateToken("old")
	_, ok := cache.get("key")
	assert.True(t, ok)

	cache.invalidateToken("new")
	_, ok = cache.get("key")
	assert.False(t, ok)
}

// infisicalAuthMock counts logins and revokes and serves one secret. A
// rejected token gets a 401 on the secret read.
type infisicalAuthMock struct {
	expiresIn int
	logins    atomic.Int32
	revokes   atomic.Int32
	rejected  atomic.Value
}

func (m *infisicalAuthMock) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/universal-auth/login", func(w http.ResponseWriter, _ *http.Request) {
		n := m.logins.Add(1)
		writeJSON(w, http.StatusOK, map[string]any{
			"accessToken": fmt.Sprintf("token-%d", n),
			"expiresIn":   m.expiresIn,
			"tokenType":   "Bearer",
		})
	})
	mux.HandleFunc("/api/v1/auth/token/revoke", func(w http.ResponseWriter, _ *http.Request) {
		m.revokes.Add(1)
		writeJSON(w, http.StatusOK, map[string]any{})
	})
	mux.HandleFunc("/api/v3/secrets/raw/", func(w http.ResponseWriter, r *http.Request) {
		if rejected, _ := m.rejected.Load().(string); r.Header.Get("Authorization") == "Bearer "+rejected {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "Token revoked"})
			return
		}
		writeSecret(w, "foo", "bar")
	})
	return mux
}

func newUniversalAuthStore(hostAPI string) (*esv1.SecretStore, *corev1.Secret) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "infisical-creds", Namespace: "default"},
		Data:       map[string][]byte{"clientId": []byte("client"), "clientSecret": []byte("secret")},
	}
	store := &esv1.SecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		ObjectMeta: metav1.ObjectMeta{Name: "infisical", Namespace: "default"},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Infisical: &esv1.InfisicalProvider{
					HostAPI: hostAPI,
					Auth: esv1.InfisicalAuth{
						UniversalAuthCredentials: &esv1.UniversalAuthCredentials{
							ClientID:     esv1meta.SecretKeySelector{Name: "infisical-creds", Key: "clientId"},
							ClientSecret: esv1meta.SecretKeySelector{Name: "infisical-creds", Key: "clientSecret"},
						},
					},
					SecretsScope: esv1.MachineIdentityScopeInWorkspace{
						ProjectSlug:     "first-project",
						EnvironmentSlug: "dev",
					},
				},
			},
		},
	}
	return store, secret
}

func withTokenCache(t *testing.T) {
	enableTokenCache = true
	t.Cleanup(func() { enableTokenCache = false })
}

func syncOnce(t *testing.T, store *esv1.SecretStore, secret *corev1.Secret) error {
	t.Helper()
	ctx := context.Background()
	kube := fake.NewClientBuilder().WithObjects(secret).Build()

	client, err := (&Provider{}).NewClient(ctx, store, kube, "default")
	require.NoError(t, err)
	_, getErr := client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "foo"})
	require.NoError(t, client.Close(ctx))
	return getErr
}

func TestNewClientReusesCachedToken(t *testing.T) {
	withTokenCache(t)
	mock := &infisicalAuthMock{expiresIn: 3600}
	server := httptest.NewServer(mock.handler())
	t.Cleanup(server.Close)
	store, secret := newUniversalAuthStore(server.URL)

	for range 3 {
		require.NoError(t, syncOnce(t, store, secret))
	}
	assert.Equal(t, int32(1), mock.logins.Load())
	assert.Equal(t, int32(0), mock.revokes.Load(), "a cached token is never revoked")

	// A revoked token drops the cache entry, and the next sync logs in again.
	mock.rejected.Store("token-1")
	require.Error(t, syncOnce(t, store, secret))
	require.NoError(t, syncOnce(t, store, secret))
	assert.Equal(t, int32(2), mock.logins.Load())
}

func TestNewClientDoesNotShareTokenAcrossCredentials(t *testing.T) {
	withTokenCache(t)
	mock := &infisicalAuthMock{expiresIn: 3600}
	server := httptest.NewServer(mock.handler())
	t.Cleanup(server.Close)
	store, secret := newUniversalAuthStore(server.URL)

	require.NoError(t, syncOnce(t, store, secret))
	secret.Data["clientSecret"] = []byte("other-secret")
	require.NoError(t, syncOnce(t, store, secret))
	assert.Equal(t, int32(2), mock.logins.Load())
}

func TestNewClientRevokesTokenWithoutTTL(t *testing.T) {
	withTokenCache(t)
	mock := &infisicalAuthMock{expiresIn: 0}
	server := httptest.NewServer(mock.handler())
	t.Cleanup(server.Close)
	store, secret := newUniversalAuthStore(server.URL)

	for range 2 {
		require.NoError(t, syncOnce(t, store, secret))
	}
	assert.Equal(t, int32(2), mock.logins.Load())
	assert.Equal(t, int32(2), mock.revokes.Load())
}

func TestNewClientWithoutTokenCache(t *testing.T) {
	mock := &infisicalAuthMock{expiresIn: 3600}
	server := httptest.NewServer(mock.handler())
	t.Cleanup(server.Close)
	store, secret := newUniversalAuthStore(server.URL)

	for range 2 {
		require.NoError(t, syncOnce(t, store, secret))
	}
	assert.Equal(t, int32(2), mock.logins.Load())
	assert.Equal(t, int32(2), mock.revokes.Load())
}
