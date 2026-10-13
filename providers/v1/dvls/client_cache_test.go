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

package dvls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Devolutions/go-dvls"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	cacheTestVaultName   = "cache-test-vault"
	cacheTestVaultID     = "00000000-0000-0000-0000-0000000000aa"
	cacheTestOtherNS     = "other-namespace"
	cacheTestOtherSecret = "other-secret"
	cacheTestEntryID     = "00000000-0000-0000-0000-0000000000bb"
)

// requestKind groups the requests the fake DVLS can hold.
type requestKind string

const (
	loginRequest requestKind = "login"
	sessionCheck requestKind = "session"
	apiRequest   requestKind = "request"
)

func kindOf(r *http.Request) requestKind {
	switch r.URL.Path {
	case "/api/v1/login":
		return loginRequest
	case "/api/is-logged":
		return sessionCheck
	default:
		return apiRequest
	}
}

// fakeDVLS implements the login, session check, server information, vault list and entry
// endpoints, and records every login so tests can assert how many sessions were opened.
type fakeDVLS struct {
	t   *testing.T
	srv *httptest.Server

	mu               sync.Mutex
	logins           int
	loginAttempts    int
	failNextLogins   int
	staleTokenLogin  int
	lastAppSecret    string
	nextToken        int
	validTokens      map[string]bool
	expireAfterCheck bool
	rejectVault      bool
	vaultRequests    int
	rejectEntry      map[string]bool
	entryRequests    map[string]int
	loginRequests    int
	unavailable      bool
	held             map[requestKind]int
	requestHeld      chan struct{}
	release          chan struct{}
	releaseOnce      sync.Once
}

func newFakeDVLS(t *testing.T) *fakeDVLS {
	t.Helper()
	f := &fakeDVLS{
		t:             t,
		validTokens:   map[string]bool{},
		rejectEntry:   map[string]bool{},
		entryRequests: map[string]int{},
		held:          map[requestKind]int{},
		requestHeld:   make(chan struct{}, 8),
		release:       make(chan struct{}),
	}
	f.srv = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.srv.Close)
	// Cleanups run last in, first out: held requests are released before the server waits for them.
	t.Cleanup(f.releaseRequests)
	return f
}

func (f *fakeDVLS) handle(w http.ResponseWriter, r *http.Request) {
	if !f.waitIfHeld(r) {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.unavailable {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	switch r.URL.Path {
	case "/api/v1/login":
		f.handleLogin(w, r)
	case "/api/is-logged":
		f.handleSessionCheck(w, r)
	case "/api/public-instance-information":
		_ = json.NewEncoder(w).Encode(map[string]any{"result": dvls.SaveResultSuccess, "data": map[string]string{"version": "test"}})
	case "/api/v1/vault":
		f.handleVaultList(w, r)
	default:
		f.handleEntry(w, r)
	}
}

func (f *fakeDVLS) handleLogin(w http.ResponseWriter, r *http.Request) {
	f.loginAttempts++
	if r.Header.Get("tokenId") != "" {
		f.staleTokenLogin++
	}
	if f.failNextLogins > 0 {
		f.failNextLogins--
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	f.logins++
	f.lastAppSecret = r.PostForm.Get("AppSecret")
	f.nextToken++
	token := fmt.Sprintf("token-%d", f.nextToken)
	f.validTokens[token] = true
	_ = json.NewEncoder(w).Encode(map[string]string{"tokenId": token})
}

func (f *fakeDVLS) handleSessionCheck(w http.ResponseWriter, r *http.Request) {
	if !f.validTokens[r.Header.Get("tokenId")] {
		_, _ = w.Write([]byte("false"))
		return
	}
	_, _ = w.Write([]byte("true"))
	if f.expireAfterCheck {
		f.expireAfterCheck = false
		f.validTokens = map[string]bool{}
	}
}

func (f *fakeDVLS) handleVaultList(w http.ResponseWriter, r *http.Request) {
	f.vaultRequests++
	if f.rejectVault || !f.validTokens[r.Header.Get("tokenId")] {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":        []map[string]string{{"id": cacheTestVaultID, "name": cacheTestVaultName}},
		"currentPage": 1,
		"pageSize":    100,
		"totalCount":  1,
		"totalPage":   1,
	})
}

func (f *fakeDVLS) handleEntry(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/v1/vault/"+cacheTestVaultID+"/entry") {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	f.entryRequests[r.Method]++
	if f.rejectEntry[r.Method] || !f.validTokens[r.Header.Get("tokenId")] {
		w.WriteHeader(http.StatusUnauthorized)
	}
}

// expireSessions invalidates every token, as DVLS does when a session times out.
func (f *fakeDVLS) expireSessions() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.validTokens = map[string]bool{}
}

// expireSessionsAfterNextCheck lets the next session check succeed and then invalidates
// every token, so the request that follows the check is rejected.
func (f *fakeDVLS) expireSessionsAfterNextCheck() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.expireAfterCheck = true
}

// setUnavailable makes every request fail with 503, as when DVLS is down.
func (f *fakeDVLS) setUnavailable(unavailable bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unavailable = unavailable
}

// rejectVaultRequests makes every vault request fail with 401 while sessions stay valid.
func (f *fakeDVLS) rejectVaultRequests() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rejectVault = true
}

// rejectEntryRequests makes entry requests with the given method fail with 401 while sessions stay valid.
func (f *fakeDVLS) rejectEntryRequests(method string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rejectEntry[method] = true
}

func (f *fakeDVLS) entryRequestCount(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.entryRequests[method]
}

func (f *fakeDVLS) loginCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logins
}

func (f *fakeDVLS) vaultRequestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.vaultRequests
}

// holdNext makes the next n requests of a kind wait until releaseRequests, or until the client gives up.
func (f *fakeDVLS) holdNext(kind requestKind, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.held[kind] = n
}

// waitIfHeld counts login requests and holds a request if asked. It reports false when the client gave up first.
func (f *fakeDVLS) waitIfHeld(r *http.Request) bool {
	kind := kindOf(r)
	f.mu.Lock()
	if kind == loginRequest {
		f.loginRequests++
	}
	held := f.held[kind] > 0
	if held {
		f.held[kind]--
	}
	f.mu.Unlock()
	if !held {
		return true
	}

	f.requestHeld <- struct{}{}
	select {
	case <-f.release:
		return true
	case <-r.Context().Done():
		return false
	}
}

func (f *fakeDVLS) waitForHeldRequest(t *testing.T) {
	t.Helper()
	select {
	case <-f.requestHeld:
	case <-time.After(5 * time.Second):
		t.Fatal("the held request never reached the server")
	}
}

func (f *fakeDVLS) releaseRequests() {
	f.releaseOnce.Do(func() { close(f.release) })
}

func (f *fakeDVLS) loginRequestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.loginRequests
}

func withCache(t *testing.T, enabled bool, size int) {
	t.Helper()
	prevEnabled, prevCache := enableCache, clientCache
	enableCache = enabled
	initCache(size)
	t.Cleanup(func() {
		enableCache, clientCache = prevEnabled, prevCache
	})
}

func withLoginTimeout(t *testing.T, timeout time.Duration) {
	t.Helper()
	prev := loginTimeout
	loginTimeout = timeout
	t.Cleanup(func() {
		loginTimeout = prev
	})
}

func withCallTimeout(t *testing.T, timeout time.Duration) {
	t.Helper()
	prev := callTimeout
	callTimeout = timeout
	t.Cleanup(func() {
		callTimeout = prev
	})
}

func credentialSecret(namespace, appSecret string) *corev1.Secret {
	return &corev1.Secret{
		Name:      secretName,
		Namespace: namespace,
		Data: map[string][]byte{
			appIDKey:     []byte(testAppID),
			appSecretKey: []byte(appSecret),
		},
	}
}

func newKube(t *testing.T, objs ...kclient.Object) kclient.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func clusterStore(serverURL, name, resourceVersion string, credentialsNamespace *string) *esv1.ClusterSecretStore {
	return &esv1.ClusterSecretStore{
		Kind:            esv1.ClusterSecretStoreKind,
		Name:            name,
		ResourceVersion: resourceVersion,
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				DVLS: &esv1.DVLSProvider{
					ServerURL: serverURL,
					Vault:     cacheTestVaultName,
					Auth: esv1.DVLSAuth{
						SecretRef: esv1.DVLSAuthSecretRef{
							AppID:     esmeta.SecretKeySelector{Name: secretName, Key: appIDKey, Namespace: credentialsNamespace},
							AppSecret: esmeta.SecretKeySelector{Name: secretName, Key: appSecretKey, Namespace: credentialsNamespace},
						},
					},
				},
			},
		},
	}
}

func reconcile(t *testing.T, store esv1.GenericStore, kube kclient.Client, namespace string) {
	t.Helper()
	_, err := (&Provider{}).NewClient(context.Background(), store, kube, namespace)
	require.NoError(t, err)
}

func TestClientCache_DisabledByDefaultLogsInOnEveryReconcile(t *testing.T) {
	withCache(t, false, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	for range 3 {
		reconcile(t, store, kube, testNamespace)
	}

	assert.Equal(t, 3, f.loginCount())
}

func TestClientCache_ReusesSessionAcrossReconciles(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	for range 10 {
		reconcile(t, store, kube, testNamespace)
	}

	assert.Equal(t, 1, f.loginCount())
}

func TestClientCache_StoresWithSameCredentialsShareOneSession(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	a := clusterStore(f.srv.URL, "store-a", "1", new(testNamespace))
	b := clusterStore(f.srv.URL, "store-b", "1", new(testNamespace))

	for range 3 {
		reconcile(t, a, kube, testNamespace)
		reconcile(t, b, kube, testNamespace)
	}

	assert.Equal(t, 1, f.loginCount())
}

func TestClientCache_StoresWithDifferentCredentialsDoNotShareSessions(t *testing.T) {
	tests := map[string]struct {
		appID     string
		appSecret string
	}{
		"different appId":     {appID: "other-app-id", appSecret: testAppSecret},
		"different appSecret": {appID: testAppID, appSecret: cacheTestOtherSecret},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			withCache(t, true, defaultCacheSize)
			f := newFakeDVLS(t)
			other := credentialSecret(cacheTestOtherNS, tc.appSecret)
			other.Data[appIDKey] = []byte(tc.appID)
			kube := newKube(t, credentialSecret(testNamespace, testAppSecret), other)
			a := clusterStore(f.srv.URL, "store-a", "1", new(testNamespace))
			b := clusterStore(f.srv.URL, "store-b", "1", new(cacheTestOtherNS))

			for range 3 {
				reconcile(t, a, kube, testNamespace)
				reconcile(t, b, kube, testNamespace)
			}

			assert.Equal(t, 2, f.loginCount())
		})
	}
}

func TestClientCache_StoresWithoutPinnedVaultShareOneSession(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	pinned := clusterStore(f.srv.URL, "pinned", "1", new(testNamespace))
	unpinned := clusterStore(f.srv.URL, "unpinned", "1", new(testNamespace))
	unpinned.Spec.Provider.DVLS.Vault = ""

	for range 3 {
		reconcile(t, unpinned, kube, testNamespace)
		reconcile(t, pinned, kube, testNamespace)
	}

	assert.Equal(t, 1, f.loginCount())
}

func TestClientCache_ServersDoNotShareSessions(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	first, second := newFakeDVLS(t), newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	a := clusterStore(first.srv.URL, "store-a", "1", new(testNamespace))
	b := clusterStore(second.srv.URL, "store-b", "1", new(testNamespace))

	for range 3 {
		reconcile(t, a, kube, testNamespace)
		reconcile(t, b, kube, testNamespace)
	}

	assert.Equal(t, 1, first.loginCount())
	assert.Equal(t, 1, second.loginCount())
}

func TestClientCache_StoreUpdateKeepsSession(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))

	reconcile(t, clusterStore(f.srv.URL, "store", "1", new(testNamespace)), kube, testNamespace)
	reconcile(t, clusterStore(f.srv.URL, "store", "2", new(testNamespace)), kube, testNamespace)

	assert.Equal(t, 1, f.loginCount())
}

func TestClientCache_CredentialRotationOpensNewSession(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	secret := credentialSecret(testNamespace, testAppSecret)
	kube := newKube(t, secret)
	a := clusterStore(f.srv.URL, "store-a", "1", new(testNamespace))
	b := clusterStore(f.srv.URL, "store-b", "1", new(testNamespace))

	reconcile(t, a, kube, testNamespace)
	reconcile(t, b, kube, testNamespace)

	secret.Data[appSecretKey] = []byte(cacheTestOtherSecret)
	require.NoError(t, kube.Update(context.Background(), secret))

	reconcile(t, a, kube, testNamespace)
	reconcile(t, b, kube, testNamespace)

	assert.Equal(t, 2, f.loginCount())
	assert.Equal(t, cacheTestOtherSecret, f.lastAppSecret)
}

func TestClientCache_ReferentClusterStoreSharesSessionWhenCredentialsMatch(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t,
		credentialSecret(testNamespace, testAppSecret),
		credentialSecret(cacheTestOtherNS, testAppSecret),
	)
	store := clusterStore(f.srv.URL, "store", "1", nil)

	for range 3 {
		reconcile(t, store, kube, testNamespace)
		reconcile(t, store, kube, cacheTestOtherNS)
	}

	assert.Equal(t, 1, f.loginCount())
}

func TestClientCache_ReferentClusterStoreKeepsOneSessionPerCredentials(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t,
		credentialSecret(testNamespace, testAppSecret),
		credentialSecret(cacheTestOtherNS, cacheTestOtherSecret),
	)
	store := clusterStore(f.srv.URL, "store", "1", nil)

	for range 3 {
		reconcile(t, store, kube, testNamespace)
		reconcile(t, store, kube, cacheTestOtherNS)
	}

	assert.Equal(t, 2, f.loginCount())
}

func TestClientCache_ExplicitCredentialNamespaceSharesSessionAcrossNamespaces(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	reconcile(t, store, kube, testNamespace)
	reconcile(t, store, kube, cacheTestOtherNS)

	assert.Equal(t, 1, f.loginCount())
}

func TestClientCache_ExpiredSessionIsRenewedOnce(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	reconcile(t, store, kube, testNamespace)
	f.expireSessions()

	reconcile(t, store, kube, testNamespace)
	reconcile(t, store, kube, testNamespace)

	assert.Equal(t, 2, f.loginCount())
	assert.Zero(t, f.staleTokenLogin, "a renewal login must not send the expired token")
}

func TestClientCache_SessionExpiringAfterCheckIsRetried(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	reconcile(t, store, kube, testNamespace)
	f.expireSessionsAfterNextCheck()

	reconcile(t, store, kube, testNamespace)

	assert.Equal(t, 2, f.loginCount())
	assert.Equal(t, 3, f.vaultRequestCount(), "the rejected request must be sent again exactly once")
}

func TestClientCache_PersistentUnauthorizedIsRetriedOnce(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	reconcile(t, store, kube, testNamespace)
	f.rejectVaultRequests()

	_, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)

	require.Error(t, err)
	assert.Equal(t, 3, f.vaultRequestCount(), "a request that stays unauthorized must not be retried more than once")
	assert.Equal(t, 1, f.loginCount())
}

func TestClientCache_DisabledDoesNotRetryUnauthorized(t *testing.T) {
	withCache(t, false, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))
	f.rejectVaultRequests()

	_, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)

	require.Error(t, err)
	assert.Equal(t, 1, f.vaultRequestCount())
}

func TestClientCache_ClientsRetryOnlyWhenCached(t *testing.T) {
	for _, cached := range []bool{true, false} {
		t.Run(fmt.Sprintf("cache enabled %t", cached), func(t *testing.T) {
			withCache(t, cached, defaultCacheSize)
			f := newFakeDVLS(t)
			kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
			store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

			clients, err := newStoreClients(context.Background(), kube, store.GetKind(), testNamespace, store.Spec.Provider.DVLS)
			require.NoError(t, err)

			cred, ok := clients.cred.(*realCredentialClient)
			require.True(t, ok)
			vaults, ok := clients.vaults.(*realVaultClient)
			require.True(t, ok)
			assert.Equal(t, cached, cred.cached)
			assert.Equal(t, cached, vaults.cached)
			assert.Equal(t, cached, clients.checkSession != nil, "only a cached client checks its session on validation")
			require.NotNil(t, cred.lock)
			assert.Equal(t, cred.lock, vaults.lock, "both services of a client must share its lock")
		})
	}
}

func TestClientCache_OnlyReadsAreRetried(t *testing.T) {
	entry := dvls.Entry{
		Id:      cacheTestEntryID,
		VaultId: cacheTestVaultID,
		Type:    dvls.EntryCredentialType,
		SubType: dvls.EntryCredentialSubTypeDefault,
	}

	tests := map[string]struct {
		reject string
		call   func(cred credentialClient) error
		want   map[string]int
	}{
		"GetByID is sent again once": {
			reject: http.MethodGet,
			call: func(cred credentialClient) error {
				_, err := cred.GetByID(context.Background(), cacheTestVaultID, cacheTestEntryID)
				return err
			},
			want: map[string]int{http.MethodGet: 2},
		},
		"GetEntries is sent again once": {
			reject: http.MethodGet,
			call: func(cred credentialClient) error {
				_, err := cred.GetEntries(context.Background(), cacheTestVaultID, dvls.GetEntriesOptions{})
				return err
			},
			want: map[string]int{http.MethodGet: 2},
		},
		"Update does not repeat a write whose read back is rejected": {
			reject: http.MethodGet,
			call: func(cred credentialClient) error {
				_, err := cred.Update(context.Background(), entry)
				return err
			},
			want: map[string]int{http.MethodPut: 1, http.MethodGet: 1},
		},
		"Update is sent once": {
			reject: http.MethodPut,
			call: func(cred credentialClient) error {
				_, err := cred.Update(context.Background(), entry)
				return err
			},
			want: map[string]int{http.MethodPut: 1},
		},
		"DeleteByID is sent once": {
			reject: http.MethodDelete,
			call: func(cred credentialClient) error {
				return cred.DeleteByID(context.Background(), cacheTestVaultID, cacheTestEntryID)
			},
			want: map[string]int{http.MethodDelete: 1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			withCache(t, true, defaultCacheSize)
			f := newFakeDVLS(t)
			kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
			store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))
			clients, err := newStoreClients(context.Background(), kube, store.GetKind(), testNamespace, store.Spec.Provider.DVLS)
			require.NoError(t, err)
			f.rejectEntryRequests(tc.reject)

			err = tc.call(clients.cred)

			require.Error(t, err)
			assert.True(t, isUnauthorized(err))
			for method, want := range tc.want {
				assert.Equal(t, want, f.entryRequestCount(method), method)
			}
			assert.Equal(t, 1, f.loginCount())
		})
	}
}

func TestClientCache_DisabledDoesNotRetryEntryReads(t *testing.T) {
	withCache(t, false, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))
	clients, err := newStoreClients(context.Background(), kube, store.GetKind(), testNamespace, store.Spec.Provider.DVLS)
	require.NoError(t, err)
	f.rejectEntryRequests(http.MethodGet)

	_, err = clients.cred.GetByID(context.Background(), cacheTestVaultID, cacheTestEntryID)

	require.Error(t, err)
	assert.Equal(t, 1, f.entryRequestCount(http.MethodGet))
}

func TestRetryUnauthorized(t *testing.T) {
	// go-dvls returns *RequestError and wraps it, so the helper must see through both.
	var rejected error = &dvls.RequestError{StatusCode: http.StatusUnauthorized, Err: errors.New("unauthorized")}
	unauthorized := fmt.Errorf("wrapped: %w", rejected)
	unauthorizedAgain := fmt.Errorf("wrapped again: %w", rejected)
	var notFound error = &dvls.RequestError{StatusCode: http.StatusNotFound, Err: errors.New("not found")}

	tests := []struct {
		name      string
		enabled   bool
		errs      []error
		wantCalls int
		wantErr   error
	}{
		{name: "success is not retried", enabled: true, errs: []error{nil}, wantCalls: 1},
		{name: "unauthorized then success", enabled: true, errs: []error{unauthorized, nil}, wantCalls: 2},
		{name: "unauthorized twice returns the second error", enabled: true, errs: []error{unauthorized, unauthorizedAgain}, wantCalls: 2, wantErr: unauthorizedAgain},
		{name: "other errors are not retried", enabled: true, errs: []error{notFound, nil}, wantCalls: 1, wantErr: notFound},
		{name: "disabled does not retry", enabled: false, errs: []error{unauthorized, nil}, wantCalls: 1, wantErr: unauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			got, err := retryUnauthorized(tt.enabled, func() (int, error) {
				calls++
				return calls, tt.errs[calls-1]
			})

			assert.Equal(t, tt.wantCalls, calls)
			assert.Equal(t, tt.wantCalls, got, "the value of the last attempt must be returned")
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestClientCache_FailedLoginIsNotCached(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	f.failNextLogins = 1
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	_, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)
	require.Error(t, err)

	reconcile(t, store, kube, testNamespace)
	reconcile(t, store, kube, testNamespace)

	assert.Equal(t, 1, f.loginCount())
	assert.Equal(t, 2, f.loginAttempts)
}

func TestClientCache_EvictionOpensNewSession(t *testing.T) {
	withCache(t, true, 1)
	f := newFakeDVLS(t)
	kube := newKube(t,
		credentialSecret(testNamespace, testAppSecret),
		credentialSecret(cacheTestOtherNS, cacheTestOtherSecret),
	)
	a := clusterStore(f.srv.URL, "store-a", "1", new(testNamespace))
	b := clusterStore(f.srv.URL, "store-b", "1", new(cacheTestOtherNS))

	reconcile(t, a, kube, testNamespace)
	reconcile(t, b, kube, testNamespace)
	reconcile(t, a, kube, testNamespace)

	assert.Equal(t, 3, f.loginCount())
}

func TestClientCache_ConcurrentReconcilesShareOneSession(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	reconcile(t, store, kube, testNamespace)
	f.expireSessions()

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			_, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)
			assert.NoError(t, err)
		})
	}
	wg.Wait()

	assert.Equal(t, 2, f.loginCount(), "the expired session must be renewed by exactly one caller")
	assert.Zero(t, f.staleTokenLogin)
}

func TestClientCache_ConcurrentColdStartLogsInOnce(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			_, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)
			assert.NoError(t, err)
		})
	}
	wg.Wait()

	assert.Equal(t, 1, f.loginCount(), "concurrent cache misses must wait for a single login")
}

func TestClientCache_UnansweredLoginTimesOutAndNextReconcileLogsIn(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	withLoginTimeout(t, 100*time.Millisecond)
	f := newFakeDVLS(t)
	f.holdNext(loginRequest, 1)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := (&Provider{}).NewClient(ctx, store, kube, testNamespace)
	require.ErrorContains(t, err, "login failed")
	require.ErrorContains(t, err, context.DeadlineExceeded.Error())
	require.NoError(t, ctx.Err(), "the shared login must give up on its own deadline, not the caller's")

	_, err = (&Provider{}).NewClient(ctx, store, kube, testNamespace)
	require.NoError(t, err, "a later reconcile must start a new login")

	assert.Equal(t, 2, f.loginRequestCount())
	assert.Equal(t, 1, f.loginCount())
}

func TestClientCache_CanceledCallerDoesNotCancelTheSharedLogin(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	f.holdNext(loginRequest, 1)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))

	first, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()
	firstErr := make(chan error, 1)
	go func() {
		_, err := (&Provider{}).NewClient(first, store, kube, testNamespace)
		firstErr <- err
	}()
	f.waitForHeldRequest(t)

	secondErr := make(chan error, 1)
	go func() {
		_, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)
		secondErr <- err
	}()

	cancelFirst()
	select {
	case err := <-firstErr:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("the canceled caller kept waiting for the held login")
	}

	f.releaseRequests()
	select {
	case err := <-secondErr:
		require.NoError(t, err, "the login must survive the cancellation of the caller that started it")
	case <-time.After(5 * time.Second):
		t.Fatal("the second caller never got a client")
	}

	assert.Equal(t, 1, f.loginRequestCount(), "the second caller must reuse the first login, not start another")
	assert.Equal(t, 1, f.loginCount())
}

func TestClientCache_UnansweredCallReleasesTheSharedClient(t *testing.T) {
	entry := dvls.Entry{
		Id:      cacheTestEntryID,
		VaultId: cacheTestVaultID,
		Type:    dvls.EntryCredentialType,
		SubType: dvls.EntryCredentialSubTypeDefault,
	}

	tests := map[string]func(ctx context.Context, c *storeClients) error{
		"vault GetByName": func(ctx context.Context, c *storeClients) error {
			_, err := c.vaults.GetByName(ctx, cacheTestVaultName)
			return err
		},
		"vault Get": func(ctx context.Context, c *storeClients) error {
			_, err := c.vaults.Get(ctx, cacheTestVaultID)
			return err
		},
		"vault List": func(ctx context.Context, c *storeClients) error {
			_, err := c.vaults.List(ctx)
			return err
		},
		"entry GetByID": func(ctx context.Context, c *storeClients) error {
			_, err := c.cred.GetByID(ctx, cacheTestVaultID, cacheTestEntryID)
			return err
		},
		"entry GetEntries": func(ctx context.Context, c *storeClients) error {
			_, err := c.cred.GetEntries(ctx, cacheTestVaultID, dvls.GetEntriesOptions{})
			return err
		},
		"entry Update": func(ctx context.Context, c *storeClients) error {
			_, err := c.cred.Update(ctx, entry)
			return err
		},
		"entry DeleteByID": func(ctx context.Context, c *storeClients) error {
			return c.cred.DeleteByID(ctx, cacheTestVaultID, cacheTestEntryID)
		},
	}

	for name, call := range tests {
		t.Run(name, func(t *testing.T) {
			withCache(t, true, defaultCacheSize)
			withCallTimeout(t, 100*time.Millisecond)
			f := newFakeDVLS(t)
			kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
			store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))
			clients, err := newStoreClients(context.Background(), kube, store.GetKind(), testNamespace, store.Spec.Provider.DVLS)
			require.NoError(t, err)
			f.holdNext(apiRequest, 1)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = call(ctx, clients)
			require.ErrorContains(t, err, context.DeadlineExceeded.Error())
			require.NoError(t, ctx.Err(), "the call must give up on its own deadline, not the caller's")

			_, err = clients.vaults.List(ctx)
			require.NoError(t, err, "the shared client must be free again once the call gave up")
			assert.Equal(t, 1, f.loginCount())
		})
	}
}

func TestClientCache_ValidationReportsAnUnavailableServer(t *testing.T) {
	// A store that pins a vault by UUID, or none, sends no request when its client comes from the cache.
	for name, vault := range map[string]string{"vault UUID": cacheTestVaultID, "no vault": ""} {
		t.Run(name, func(t *testing.T) {
			withCache(t, true, defaultCacheSize)
			f := newFakeDVLS(t)
			kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
			store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))
			store.Spec.Provider.DVLS.Vault = vault
			reconcile(t, store, kube, testNamespace)
			f.setUnavailable(true)

			client, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)
			require.NoError(t, err)
			result, err := client.Validate()
			require.ErrorContains(t, err, "unexpected status code 503")
			assert.Equal(t, esv1.ValidationResultError, result)

			f.setUnavailable(false)
			result, err = client.Validate()
			require.NoError(t, err)
			assert.Equal(t, esv1.ValidationResultReady, result)
			assert.Equal(t, 1, f.loginCount())
		})
	}
}

func TestClientCache_ValidationRenewsAnExpiredSession(t *testing.T) {
	tests := map[string]struct {
		refuseLogin bool
		wantResult  esv1.ValidationResult
		wantLogins  int
	}{
		"renewed":         {wantResult: esv1.ValidationResultReady, wantLogins: 2},
		"renewal refused": {refuseLogin: true, wantResult: esv1.ValidationResultError, wantLogins: 1},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			withCache(t, true, defaultCacheSize)
			f := newFakeDVLS(t)
			kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
			store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))
			client, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)
			require.NoError(t, err)
			f.expireSessions()
			if tc.refuseLogin {
				f.mu.Lock()
				f.failNextLogins = 1
				f.mu.Unlock()
			}

			result, err := client.Validate()
			assert.Equal(t, tc.wantResult, result)
			assert.Equal(t, tc.refuseLogin, err != nil)
			assert.Equal(t, tc.wantLogins, f.loginCount())
		})
	}
}

func TestClientCache_ValidationAndRequestsShareTheSession(t *testing.T) {
	withCache(t, true, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))
	client, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)
	require.NoError(t, err)
	f.expireSessions()

	_, err = client.Validate()
	require.NoError(t, err)
	reconcile(t, store, kube, testNamespace)

	assert.Equal(t, 2, f.loginCount(), "the session renewed by the validation must be the one requests use")
}

func TestClientCache_DisabledValidationSendsNoRequest(t *testing.T) {
	withCache(t, false, defaultCacheSize)
	f := newFakeDVLS(t)
	kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
	store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))
	client, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)
	require.NoError(t, err)
	f.setUnavailable(true)

	result, err := client.Validate()
	require.NoError(t, err, "without the cache, the login in NewClient already reached DVLS")
	assert.Equal(t, esv1.ValidationResultReady, result)
}

func TestClientCache_ValidationGivesUpOnItsDeadline(t *testing.T) {
	tests := map[string]func(t *testing.T, f *fakeDVLS, shared *sharedClient){
		"DVLS does not answer": func(_ *testing.T, f *fakeDVLS, _ *sharedClient) {
			f.holdNext(sessionCheck, 1)
		},
		"the shared client stays busy": func(t *testing.T, _ *fakeDVLS, shared *sharedClient) {
			unlock, err := shared.lock.acquire(context.Background())
			require.NoError(t, err)
			t.Cleanup(unlock)
		},
	}

	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			withCache(t, true, defaultCacheSize)
			withCallTimeout(t, 100*time.Millisecond)
			f := newFakeDVLS(t)
			kube := newKube(t, credentialSecret(testNamespace, testAppSecret))
			store := clusterStore(f.srv.URL, "store", "1", new(testNamespace))
			client, err := (&Provider{}).NewClient(context.Background(), store, kube, testNamespace)
			require.NoError(t, err)
			shared, ok := clientCache.Get("", credentialsKey(f.srv.URL, testAppID, testAppSecret))
			require.True(t, ok)
			setup(t, f, shared)

			var result esv1.ValidationResult
			done := make(chan struct{})
			go func() {
				defer close(done)
				result, err = client.Validate()
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("the validation never gave up")
			}
			require.ErrorContains(t, err, context.DeadlineExceeded.Error())
			assert.Equal(t, esv1.ValidationResultError, result)
		})
	}
}

func TestCallLock(t *testing.T) {
	t.Run("waiting stops when the context is done", func(t *testing.T) {
		lock := newCallLock()
		unlock, err := lock.acquire(context.Background())
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, err = lock.acquire(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		unlock()
		unlock, err = lock.acquire(context.Background())
		require.NoError(t, err, "the lock must be free again once released")
		unlock()
	})

	t.Run("a done context never gets a free lock", func(t *testing.T) {
		lock := newCallLock()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		for range 100 {
			_, err := lock.acquire(ctx)
			require.ErrorIs(t, err, context.Canceled)
		}
	})

	t.Run("a nil lock does not block", func(t *testing.T) {
		var lock callLock
		unlock, err := lock.acquire(context.Background())
		require.NoError(t, err)
		unlock()
	})
}

func TestRun_SkipsTheCallWhenTheLockIsNotAcquired(t *testing.T) {
	lock := newCallLock()
	unlock, err := lock.acquire(context.Background())
	require.NoError(t, err)
	defer unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	calls := 0
	got, err := run(ctx, lock, true, func(context.Context) (int, error) {
		calls++
		return 1, nil
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Zero(t, calls)
	assert.Zero(t, got)
}

func TestRun_HoldsTheLockDuringTheCall(t *testing.T) {
	lock := newCallLock()

	_, err := run(context.Background(), lock, false, func(context.Context) (int, error) {
		assert.Len(t, lock, 1, "the lock must be held while the call runs")
		return 0, nil
	})

	require.NoError(t, err)
	assert.Empty(t, lock, "the lock must be released after the call")
}

func TestRun_OnlyCachedCallsGetADeadline(t *testing.T) {
	for _, cached := range []bool{true, false} {
		t.Run(fmt.Sprintf("cached %t", cached), func(t *testing.T) {
			_, err := run(context.Background(), newCallLock(), cached, func(callCtx context.Context) (int, error) {
				_, hasDeadline := callCtx.Deadline()
				assert.Equal(t, cached, hasDeadline)
				return 0, nil
			})
			require.NoError(t, err)
		})
	}
}

func TestCacheFlags_Apply(t *testing.T) {
	prevEnabled, prevCache := enableCache, clientCache
	t.Cleanup(func() {
		enableCache, clientCache = prevEnabled, prevCache
	})
	clientCache = nil

	(&cacheFlags{enable: false, size: 0}).apply()
	assert.False(t, enableCache)
	assert.Nil(t, clientCache, "a disabled cache must not be allocated, whatever its size")

	(&cacheFlags{enable: true, size: 5}).apply()
	assert.True(t, enableCache)
	assert.NotNil(t, clientCache)

	for _, size := range []int{0, -1} {
		assert.PanicsWithValue(t, fmt.Sprintf("--dvls-client-cache-size must be greater than 0, got %d", size), func() {
			(&cacheFlags{enable: true, size: size}).apply()
		})
	}
}

func TestRegisterCacheFlags(t *testing.T) {
	fs, flags := registerCacheFlags(pflag.ContinueOnError)
	assert.False(t, flags.enable)
	assert.Equal(t, defaultCacheSize, flags.size)

	require.NoError(t, fs.Parse([]string{"--enable-dvls-client-cache", "--dvls-client-cache-size=5"}))
	assert.True(t, flags.enable)
	assert.Equal(t, 5, flags.size)
}
