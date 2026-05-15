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

package truefoundry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	tBaseURL = "http://stub.local" // placeholder; overridden by httptest server URL
	tTenant  = "my-tenant"
	tAPIKey  = "pat-1234"
)

func newTestClient(serverURL string) *Client {
	return newClient(serverURL, tTenant, tAPIKey, &http.Client{Timeout: 5 * time.Second})
}

func TestParseRef(t *testing.T) {
	cases := []struct {
		in        string
		wantGroup string
		wantKey   string
	}{
		{in: "", wantGroup: "", wantKey: ""},
		{in: "  ", wantGroup: "", wantKey: ""},
		{in: "group-a", wantGroup: "group-a", wantKey: ""},
		{in: "group-a/KEY", wantGroup: "group-a", wantKey: "KEY"},
		{in: "group-a/", wantGroup: "group-a", wantKey: ""},
		{in: "/KEY", wantGroup: "", wantKey: "KEY"},
		{in: " group-a / KEY ", wantGroup: "group-a", wantKey: "KEY"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q", tc.in), func(t *testing.T) {
			g, k := parseRef(tc.in)
			require.Equal(t, tc.wantGroup, g)
			require.Equal(t, tc.wantKey, k)
		})
	}
}

func TestFQN(t *testing.T) {
	c := newClient("https://example", "my-tenant", "k", nil)
	require.Equal(t, "my-tenant:group", c.fqn("group"))
	require.Equal(t, "my-tenant:group:with:colons", c.fqn("group:with:colons"))
}

func TestParseRetryAfter(t *testing.T) {
	require.Equal(t, time.Duration(0), parseRetryAfter(""))
	require.Equal(t, time.Duration(0), parseRetryAfter("garbage"))
	require.Equal(t, 5*time.Second, parseRetryAfter("5"))
	// Past HTTP date → 0 (no wait).
	require.Equal(t, time.Duration(0), parseRetryAfter("Mon, 01 Jan 2001 00:00:00 GMT"))
}

func TestDoRequest_AuthHeadersOnEveryAttempt(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		require.Equal(t, "Bearer "+tAPIKey, r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, status, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestDoRequest_RetryOn5xxThenSuccess(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			http.Error(w, "boom", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	body, status, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
	require.Contains(t, string(body), "ok")
}

func TestDoRequest_RetryExhausted(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "still down", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, status, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	require.Error(t, err)
	require.Equal(t, http.StatusInternalServerError, status)
	require.Equal(t, int32(maxRetryAttempts), atomic.LoadInt32(&calls))
	require.Contains(t, err.Error(), "server error 500")
}

func TestDoRequest_NonRetryable4xxFailsFast(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, status, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, status)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "401 must not retry")
	require.Contains(t, err.Error(), "status 401")
}

func TestDoRequest_RetryAfter429(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	start := time.Now()
	_, status, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls))
	require.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "Retry-After must be honored")
}

func TestDoRequest_TransportError(t *testing.T) {
	// Closed server URL → connection refused, retried then fails.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()

	c := newTestClient(srv.URL)
	_, _, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	require.Error(t, err)
}

func TestDoRequest_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := c.doRequest(ctx, "test", srv.URL+"/x")
	require.Error(t, err)
}

// --- searchGroupByFQN -------------------------------------------------------

func TestSearchGroupByFQN_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/svc/v1/secret-groups", r.URL.Path)
		require.Equal(t, tTenant+":app", r.URL.Query().Get("fqn"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"id":"g-1","fqn":"my-tenant:app","associatedSecrets":[{"id":"s-1","name":"DB"}]}]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	g, err := c.searchGroupByFQN(context.Background(), "my-tenant:app")
	require.NoError(t, err)
	require.Equal(t, "g-1", g.ID)
	require.Len(t, g.AssociatedSecrets, 1)
	require.Equal(t, "DB", g.AssociatedSecrets[0].Name)
}

func TestSearchGroupByFQN_EmptyDataIsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.searchGroupByFQN(context.Background(), "my-tenant:missing")
	require.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestSearchGroupByFQN_404IsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.searchGroupByFQN(context.Background(), "my-tenant:missing")
	require.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestSearchGroupByFQN_401Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"err":"bad token"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.searchGroupByFQN(context.Background(), "my-tenant:app")
	require.Error(t, err)
	require.False(t, errors.Is(err, esv1.NoSecretErr))
	require.Contains(t, err.Error(), "status 401")
}

func TestSearchGroupByFQN_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<<<not json>>>`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.searchGroupByFQN(context.Background(), "my-tenant:app")
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode search response")
	require.Contains(t, err.Error(), "my-tenant:app")
}

// --- fetchSecretValue -------------------------------------------------------

func TestFetchSecretValue_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasPrefix(r.URL.Path, "/api/svc/v1/secrets/"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"value":"hunter2"}}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	val, err := c.fetchSecretValue(context.Background(), "s-1")
	require.NoError(t, err)
	require.Equal(t, []byte("hunter2"), val)
}

func TestFetchSecretValue_404IsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.fetchSecretValue(context.Background(), "s-1")
	require.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestFetchSecretValue_401Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"err":"forbidden"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.fetchSecretValue(context.Background(), "s-1")
	require.Error(t, err)
	require.False(t, errors.Is(err, esv1.NoSecretErr))
	require.Contains(t, err.Error(), "status 403")
}

func TestFetchSecretValue_IDIsPathEscaped(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"value":"x"}}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.fetchSecretValue(context.Background(), "id/with/slashes")
	require.NoError(t, err)
	require.Contains(t, capturedPath, "id%2Fwith%2Fslashes")
}

// --- Two-step fan-out: GetSecret / GetSecretMap / Validate ------------------

// stubAPI is a minimal httptest server that mimics the TrueFoundry secrets
// API for tests. It serves a single group lookup by FQN plus per-secret value
// lookup by ID. Optional hooks let individual tests inject failures.
type stubAPI struct {
	srv *httptest.Server

	tenant string
	group  string // expected group name (the part after tenant: in FQN)

	// secrets: id → value
	secrets map[string]string
	// nameToID: secret name → id (used to build the group's associatedSecrets)
	nameToID map[string]string

	// optional overrides
	groupStatus   int                                     // when > 0, group endpoint returns this status
	secretStatus  map[string]int                          // per-id status overrides
	secretHandler func(w http.ResponseWriter, id string)  // full custom secret handler
	groupBody     []byte                                  // raw override for group body
	groupHandler  func(w http.ResponseWriter, fqn string) // full custom group handler

	groupCalls  atomic.Int32
	secretCalls atomic.Int32

	// inFlight tracks the current number of concurrent secret-fetch handlers.
	curInFlight atomic.Int32
	maxInFlight atomic.Int32
}

func newStubAPI(t *testing.T, tenant, group string, secrets map[string]string) *stubAPI {
	t.Helper()
	s := &stubAPI{
		tenant:       tenant,
		group:        group,
		secrets:      secrets,
		nameToID:     map[string]string{},
		secretStatus: map[string]int{},
	}
	i := 0
	for name := range secrets {
		i++
		s.nameToID[name] = fmt.Sprintf("id-%d-%s", i, name)
	}
	s.srv = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *stubAPI) serve(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/svc/v1/secret-groups":
		s.serveGroups(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/svc/v1/secrets/"):
		s.serveSecret(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *stubAPI) serveGroups(w http.ResponseWriter, r *http.Request) {
	s.groupCalls.Add(1)
	fqn := r.URL.Query().Get("fqn")
	if s.groupHandler != nil {
		s.groupHandler(w, fqn)
		return
	}
	if s.groupStatus > 0 {
		http.Error(w, "stub", s.groupStatus)
		return
	}
	if s.groupBody != nil {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(s.groupBody)
		return
	}
	if fqn != s.tenant+":"+s.group {
		_, _ = w.Write([]byte(`{"data":[]}`))
		return
	}
	// Build a deterministic associatedSecrets array sorted by name.
	names := make([]string, 0, len(s.nameToID))
	for n := range s.nameToID {
		names = append(names, n)
	}
	sort.Strings(names)
	type pair struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type group struct {
		ID                string `json:"id"`
		FQN               string `json:"fqn"`
		AssociatedSecrets []pair `json:"associatedSecrets"`
	}
	g := group{ID: "g-1", FQN: fqn}
	for _, n := range names {
		g.AssociatedSecrets = append(g.AssociatedSecrets, pair{ID: s.nameToID[n], Name: n})
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": []group{g}})
}

func (s *stubAPI) serveSecret(w http.ResponseWriter, r *http.Request) {
	cur := s.curInFlight.Add(1)
	if cur > s.maxInFlight.Load() {
		s.maxInFlight.Store(cur)
	}
	defer s.curInFlight.Add(-1)
	s.secretCalls.Add(1)

	id := strings.TrimPrefix(r.URL.Path, "/api/svc/v1/secrets/")
	id, _ = url.PathUnescape(id)
	if s.secretHandler != nil {
		s.secretHandler(w, id)
		return
	}
	if st, ok := s.secretStatus[id]; ok {
		http.Error(w, "stub", st)
		return
	}
	val, ok := s.secrets[idToName(s.nameToID, id)]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"data":{"value":%q}}`, val)
}

func idToName(nameToID map[string]string, id string) string {
	for n, i := range nameToID {
		if i == id {
			return n
		}
	}
	return ""
}

// TestGetSecret covers the eleven scenarios listed in the plan.
func TestGetSecret(t *testing.T) {
	t.Run("whole group returns sorted JSON map", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"B": "2", "A": "1"})
		c := newTestClient(stub.srv.URL)
		out, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app"})
		require.NoError(t, err)
		require.JSONEq(t, `{"A":"1","B":"2"}`, string(out))
		require.Equal(t, int32(1), stub.groupCalls.Load())
		require.Equal(t, int32(2), stub.secretCalls.Load())
	})

	t.Run("group/key returns single value with exactly 2 calls", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"DB": "hunter2"})
		c := newTestClient(stub.srv.URL)
		out, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app/DB"})
		require.NoError(t, err)
		require.Equal(t, []byte("hunter2"), out)
		require.Equal(t, int32(1), stub.groupCalls.Load())
		require.Equal(t, int32(1), stub.secretCalls.Load())
	})

	t.Run("group/key with property selector via gjson", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"BLOB": `{"nested":{"x":"yes"}}`})
		c := newTestClient(stub.srv.URL)
		out, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app/BLOB", Property: "nested.x"})
		require.NoError(t, err)
		require.Equal(t, []byte("yes"), out)
	})

	t.Run("group missing returns NoSecretErr without fetching values", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"DB": "v"})
		stub.groupBody = []byte(`{"data":[]}`)
		c := newTestClient(stub.srv.URL)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "missing/DB"})
		require.ErrorIs(t, err, esv1.NoSecretErr)
		require.Equal(t, int32(0), stub.secretCalls.Load(), "must not fetch any secret value")
	})

	t.Run("group 404 returns NoSecretErr", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", nil)
		stub.groupStatus = http.StatusNotFound
		c := newTestClient(stub.srv.URL)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app/DB"})
		require.ErrorIs(t, err, esv1.NoSecretErr)
	})

	t.Run("group ok but key not in associated returns NoSecretErr", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"OTHER": "v"})
		c := newTestClient(stub.srv.URL)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app/MISSING"})
		require.ErrorIs(t, err, esv1.NoSecretErr)
		require.Equal(t, int32(0), stub.secretCalls.Load())
	})

	t.Run("secret-id 404 returns NoSecretErr", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"DB": "v"})
		stub.secretStatus[stub.nameToID["DB"]] = http.StatusNotFound
		c := newTestClient(stub.srv.URL)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app/DB"})
		require.ErrorIs(t, err, esv1.NoSecretErr)
	})

	t.Run("401 on group is wrapped and not NoSecretErr", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"DB": "v"})
		stub.groupStatus = http.StatusUnauthorized
		c := newTestClient(stub.srv.URL)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app/DB"})
		require.Error(t, err)
		require.False(t, errors.Is(err, esv1.NoSecretErr))
		require.Contains(t, err.Error(), "status 401")
		require.Equal(t, int32(0), stub.secretCalls.Load())
	})

	t.Run("malformed group JSON wraps fqn", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", nil)
		stub.groupBody = []byte(`<not json>`)
		c := newTestClient(stub.srv.URL)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app/DB"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "decode search response")
		require.Contains(t, err.Error(), "my-tenant:app")
	})

	t.Run("property selector missing returns error", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"BLOB": `{"x":"yes"}`})
		c := newTestClient(stub.srv.URL)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app/BLOB", Property: "missing.path"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "property")
	})

	t.Run("empty key is rejected", func(t *testing.T) {
		c := newTestClient("http://unused")
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: ""})
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing group")
	})
}

func TestGetSecretMap(t *testing.T) {
	t.Run("whole group fans out", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"A": "1", "B": "2"})
		c := newTestClient(stub.srv.URL)
		m, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app"})
		require.NoError(t, err)
		require.Equal(t, []byte("1"), m["A"])
		require.Equal(t, []byte("2"), m["B"])
	})

	t.Run("group/key JSON object → map", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"BLOB": `{"a":"x","b":42}`})
		c := newTestClient(stub.srv.URL)
		m, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app/BLOB"})
		require.NoError(t, err)
		require.Equal(t, []byte("x"), m["a"])
		require.Equal(t, []byte("42"), m["b"])
	})

	t.Run("group/key non-object → unmarshal error", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"BLOB": `"just a string"`})
		c := newTestClient(stub.srv.URL)
		_, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app/BLOB"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "unable to unmarshal")
	})

	t.Run("group missing → NoSecretErr", func(t *testing.T) {
		stub := newStubAPI(t, tTenant, "app", map[string]string{"A": "1"})
		stub.groupBody = []byte(`{"data":[]}`)
		c := newTestClient(stub.srv.URL)
		_, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "missing"})
		require.ErrorIs(t, err, esv1.NoSecretErr)
	})
}

func TestConcurrencyBound(t *testing.T) {
	// 50 secrets, server measures peak concurrent secret-fetch handlers.
	secrets := map[string]string{}
	for i := range 50 {
		secrets[fmt.Sprintf("k%02d", i)] = fmt.Sprintf("v%02d", i)
	}
	stub := newStubAPI(t, tTenant, "app", secrets)
	// Add a small sleep so concurrency is observable.
	stub.secretHandler = func(w http.ResponseWriter, id string) {
		name := idToName(stub.nameToID, id)
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"data":{"value":%q}}`, secrets[name])
	}

	c := newTestClient(stub.srv.URL)
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app"})
	require.NoError(t, err)
	require.LessOrEqual(t, int(stub.maxInFlight.Load()), defaultMaxConcurrent,
		"max in-flight requests must not exceed %d", defaultMaxConcurrent)
	require.Greater(t, int(stub.maxInFlight.Load()), 1, "expected actual concurrency")
}

func TestPartialFailureCancels(t *testing.T) {
	secrets := map[string]string{}
	for i := range 20 {
		secrets[fmt.Sprintf("k%02d", i)] = fmt.Sprintf("v%02d", i)
	}
	stub := newStubAPI(t, tTenant, "app", secrets)

	stub.secretHandler = func(w http.ResponseWriter, id string) {
		name := idToName(stub.nameToID, id)
		if name == "k05" {
			// Fail one secret hard so the errgroup cancels the rest.
			http.Error(w, "boom", http.StatusUnauthorized)
			return
		}
		// Slow handler so context cancellation is observable.
		<-time.After(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"data":{"value":%q}}`, secrets[name])
	}

	c := newTestClient(stub.srv.URL)
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "app"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 401")
	require.Less(t, int(stub.secretCalls.Load()), len(secrets),
		"errgroup must cancel in-flight fetches; got %d/%d", stub.secretCalls.Load(), len(secrets))
}

func TestGetAllSecrets_NotSupported(t *testing.T) {
	c := newTestClient("http://unused")
	_, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
	require.Contains(t, err.Error(), "FQN")
}

func TestWriteOpsNotImplemented(t *testing.T) {
	c := newTestClient("http://unused")
	require.ErrorIs(t, c.PushSecret(context.Background(), nil, nil), errNotImplemented)
	require.ErrorIs(t, c.DeleteSecret(context.Background(), nil), errNotImplemented)
	ok, err := c.SecretExists(context.Background(), nil)
	require.ErrorIs(t, err, errNotImplemented)
	require.False(t, ok)
}

func TestClose(t *testing.T) {
	c := newTestClient("http://unused")
	require.NoError(t, c.Close(context.Background()))
}

func TestValidate(t *testing.T) {
	t.Run("ready on 200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		}))
		defer srv.Close()
		c := newTestClient(srv.URL)
		r, err := c.Validate()
		require.NoError(t, err)
		require.Equal(t, esv1.ValidationResultReady, r)
	})

	t.Run("error on 401", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "no", http.StatusUnauthorized)
		}))
		defer srv.Close()
		c := newTestClient(srv.URL)
		r, err := c.Validate()
		require.Error(t, err)
		require.Equal(t, esv1.ValidationResultError, r)
	})

	t.Run("unknown on transport error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		srv.Close()
		c := newTestClient(srv.URL)
		r, err := c.Validate()
		require.Error(t, err)
		require.Equal(t, esv1.ValidationResultUnknown, r)
	})
}
