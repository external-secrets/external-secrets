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

// Package gitea — HTTP-server-based tests for the org and repo implementation
// functions. These replace the previous integration_test.go and run with plain
// `go test` (no external Gitea instance required).
package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	giteasdk "code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	mockOrg   = "test-org"
	mockRepo  = "test-repo"
	mockToken = "test-pat-token"
)

// mockGiteaServer is a stateful httptest.Server that simulates the Gitea
// Actions REST API for secrets and variables at org and repo scope.
type mockGiteaServer struct {
	*httptest.Server
	mu        sync.Mutex
	secrets   map[string]struct{} // "org/<name>" or "repo/<name>"
	variables map[string]string   // "orgvar/<name>" or "repovar/<name>"
}

func newMockGiteaServer(t *testing.T) *mockGiteaServer {
	t.Helper()
	ms := &mockGiteaServer{
		secrets:   make(map[string]struct{}),
		variables: make(map[string]string),
	}
	ms.Server = httptest.NewServer(ms)
	t.Cleanup(ms.Server.Close)
	return ms
}

func (ms *mockGiteaServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "token "+mockToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	p := r.URL.Path
	orgSecretBase := "/api/v1/orgs/" + mockOrg + "/actions/secrets"
	orgVarBase := "/api/v1/orgs/" + mockOrg + "/actions/variables"
	repoSecretBase := fmt.Sprintf("/api/v1/repos/%s/%s/actions/secrets", mockOrg, mockRepo)
	repoVarBase := fmt.Sprintf("/api/v1/repos/%s/%s/actions/variables", mockOrg, mockRepo)

	switch {
	case p == "/api/v1/version":
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"1.21.0"}`))
	case p == "/api/v1/settings/api":
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"max_response_items":50,"default_paging_num":20,"allowed_host_list_pattern":[]}`))
	case strings.HasPrefix(p, orgSecretBase):
		ms.handleSecrets(w, r, orgSecretBase, "org/")
	case strings.HasPrefix(p, orgVarBase+"/"):
		ms.handleVarSingle(w, r, orgVarBase+"/", "orgvar/")
	case p == orgVarBase:
		ms.handleVarList(w, "orgvar/")
	case strings.HasPrefix(p, repoSecretBase):
		ms.handleSecrets(w, r, repoSecretBase, "repo/")
	case strings.HasPrefix(p, repoVarBase+"/"):
		ms.handleVarSingle(w, r, repoVarBase+"/", "repovar/")
	case p == repoVarBase:
		ms.handleVarList(w, "repovar/")
	default:
		http.Error(w, "not found: "+p, http.StatusNotFound)
	}
}

func (ms *mockGiteaServer) handleSecrets(w http.ResponseWriter, r *http.Request, base, scope string) {
	name := strings.TrimPrefix(r.URL.Path, base+"/")
	key := scope + name

	ms.mu.Lock()
	defer ms.mu.Unlock()

	switch r.Method {
	case http.MethodPut:
		ms.secrets[key] = struct{}{}
		w.WriteHeader(http.StatusCreated)

	case http.MethodGet:
		type secretRow struct {
			Name string `json:"name"`
		}
		var rows []secretRow
		for k := range ms.secrets {
			if strings.HasPrefix(k, scope) {
				rows = append(rows, secretRow{Name: strings.TrimPrefix(k, scope)})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Page", "1")
		w.Header().Set("X-Total-Pages", "1")
		w.Header().Set("X-Total-Count", fmt.Sprint(len(rows)))
		_ = json.NewEncoder(w).Encode(rows)

	case http.MethodDelete:
		if _, ok := ms.secrets[key]; !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		delete(ms.secrets, key)
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleVarSingle serves GET for a single variable.
// Both the direct HTTP path (giteaVariableResponse) and the SDK path (RepoActionVariable)
// use json:"data" for the value field, so we always return {"name":…,"data":…}.
func (ms *mockGiteaServer) handleVarSingle(w http.ResponseWriter, r *http.Request, prefix, scope string) {
	varName := strings.TrimPrefix(r.URL.Path, prefix)
	key := scope + varName

	ms.mu.Lock()
	val, ok := ms.variables[key]
	ms.mu.Unlock()

	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(giteaVariableResponse{Name: varName, Value: val})
}

func (ms *mockGiteaServer) handleVarList(w http.ResponseWriter, scope string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var list []giteaVariableResponse
	for k, v := range ms.variables {
		if strings.HasPrefix(k, scope) {
			list = append(list, giteaVariableResponse{Name: strings.TrimPrefix(k, scope), Value: v})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(giteaVariablesListResponse{Data: list, TotalCount: len(list)})
}

func (ms *mockGiteaServer) seedOrgVar(name, value string) {
	ms.mu.Lock()
	ms.variables["orgvar/"+name] = value
	ms.mu.Unlock()
}

func (ms *mockGiteaServer) seedRepoVar(name, value string) {
	ms.mu.Lock()
	ms.variables["repovar/"+name] = value
	ms.mu.Unlock()
}

// newMockK8sClient creates a fake controller-runtime client pre-loaded with a
// Kubernetes Secret holding the mock PAT token.
func newTestOrgClient(t *testing.T, ms *mockGiteaServer) *Client {
	t.Helper()

	gc, err := giteasdk.NewClient(ms.URL, giteasdk.SetToken(mockToken))
	require.NoError(t, err)

	k8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "gitea-pat", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte(mockToken)},
	}
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(k8sSecret).Build()

	provider := &esv1.GiteaProvider{
		URL:          ms.URL,
		Organization: mockOrg,
		Auth:         esv1.GiteaAuth{SecretRef: esmeta.SecretKeySelector{Name: "gitea-pat", Key: "token"}},
	}
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: "test-store", Namespace: "default"},
		Spec:       esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{Gitea: provider}},
	}

	c := &Client{
		provider:   provider,
		baseClient: gc,
		store:      store,
		crClient:   fakeClient,
		namespace:  "default",
		storeKind:  "SecretStore",
	}
	c.createOrUpdateFn = c.orgCreateOrUpdateSecret
	c.listSecretsFn = c.orgListSecretsFn
	c.deleteSecretFn = c.orgDeleteSecretsFn
	c.getSecretFn = c.orgGetSecretFn
	c.getVariableFn = c.orgGetVariableFn
	c.listVariablesFn = c.orgListVariablesFn
	return c
}

func newTestRepoClient(t *testing.T, ms *mockGiteaServer) *Client {
	t.Helper()

	gc, err := giteasdk.NewClient(ms.URL, giteasdk.SetToken(mockToken))
	require.NoError(t, err)

	k8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "gitea-pat", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte(mockToken)},
	}
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(k8sSecret).Build()

	provider := &esv1.GiteaProvider{
		URL:          ms.URL,
		Organization: mockOrg,
		Repository:   mockRepo,
		Auth:         esv1.GiteaAuth{SecretRef: esmeta.SecretKeySelector{Name: "gitea-pat", Key: "token"}},
	}
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: "test-store", Namespace: "default"},
		Spec:       esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{Gitea: provider}},
	}

	c := &Client{
		provider:   provider,
		baseClient: gc,
		store:      store,
		crClient:   fakeClient,
		namespace:  "default",
		storeKind:  "SecretStore",
	}
	c.createOrUpdateFn = c.repoCreateOrUpdateSecret
	c.listSecretsFn = c.repoListSecretsFn
	c.deleteSecretFn = c.repoDeleteSecretsFn
	c.getSecretFn = c.repoGetSecretFn
	c.getVariableFn = c.repoGetVariableFn
	c.listVariablesFn = c.repoListVariablesFn
	return c
}

func pushSecretRef(secretKey, remoteKey string) esv1alpha1.PushSecretData {
	return esv1alpha1.PushSecretData{
		Match: esv1alpha1.PushSecretMatch{
			SecretKey: secretKey,
			RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: remoteKey},
		},
	}
}

// --- Org scope ---------------------------------------------------------------

func TestHTTP_OrgScope_PushExistsDelete(t *testing.T) {
	ms := newMockGiteaServer(t)
	c := newTestOrgClient(t, ms)
	ctx := context.Background()

	ref := pushSecretRef("password", "ORG_SECRET")
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("s3cr3t")}}

	require.NoError(t, c.PushSecret(ctx, secret, ref))

	exists, err := c.SecretExists(ctx, ref)
	require.NoError(t, err)
	assert.True(t, exists, "secret should exist after push")

	require.NoError(t, c.DeleteSecret(ctx, ref))

	exists, err = c.SecretExists(ctx, ref)
	require.NoError(t, err)
	assert.False(t, exists, "secret should be gone after delete")
}

func TestHTTP_OrgScope_PushUpdate(t *testing.T) {
	ms := newMockGiteaServer(t)
	c := newTestOrgClient(t, ms)
	ctx := context.Background()

	ref := pushSecretRef("val", "ORG_UPDATE")

	for _, v := range []string{"first", "second"} {
		secret := &corev1.Secret{Data: map[string][]byte{"val": []byte(v)}}
		require.NoError(t, c.PushSecret(ctx, secret, ref), "PushSecret(%s)", v)

		exists, err := c.SecretExists(ctx, ref)
		require.NoError(t, err)
		assert.True(t, exists, "secret should exist after push(%s)", v)
	}
}

func TestHTTP_OrgScope_GetVariable(t *testing.T) {
	ms := newMockGiteaServer(t)
	ms.seedOrgVar("MY_VAR", "hello-org")
	c := newTestOrgClient(t, ms)

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "MY_VAR"})
	require.NoError(t, err)
	assert.Equal(t, []byte("hello-org"), got)
}

func TestHTTP_OrgScope_GetVariable_NotFound(t *testing.T) {
	ms := newMockGiteaServer(t)
	c := newTestOrgClient(t, ms)

	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "MISSING"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "not found")
}

func TestHTTP_OrgScope_GetAllSecrets(t *testing.T) {
	ms := newMockGiteaServer(t)
	ms.seedOrgVar("APP_SECRET", "val1")
	ms.seedOrgVar("APP_TOKEN", "val2")
	ms.seedOrgVar("OTHER", "val3")
	c := newTestOrgClient(t, ms)

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
		Name: &esv1.FindName{RegExp: "^APP_"},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"APP_SECRET", "APP_TOKEN"}, keys(got))
}

// --- Repo scope --------------------------------------------------------------

func TestHTTP_RepoScope_PushExistsDelete(t *testing.T) {
	ms := newMockGiteaServer(t)
	c := newTestRepoClient(t, ms)
	ctx := context.Background()

	ref := pushSecretRef("password", "REPO_SECRET")
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("s3cr3t")}}

	require.NoError(t, c.PushSecret(ctx, secret, ref))

	exists, err := c.SecretExists(ctx, ref)
	require.NoError(t, err)
	assert.True(t, exists, "secret should exist after push")

	require.NoError(t, c.DeleteSecret(ctx, ref))

	exists, err = c.SecretExists(ctx, ref)
	require.NoError(t, err)
	assert.False(t, exists, "secret should be gone after delete")
}

func TestHTTP_RepoScope_GetVariable(t *testing.T) {
	ms := newMockGiteaServer(t)
	ms.seedRepoVar("REPO_VAR", "hello-repo")
	c := newTestRepoClient(t, ms)

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "REPO_VAR"})
	require.NoError(t, err)
	assert.Equal(t, []byte("hello-repo"), got)
}

func TestHTTP_RepoScope_GetVariable_NotFound(t *testing.T) {
	ms := newMockGiteaServer(t)
	c := newTestRepoClient(t, ms)

	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "MISSING"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "not found")
}

func TestHTTP_RepoScope_GetAllSecrets(t *testing.T) {
	ms := newMockGiteaServer(t)
	ms.seedRepoVar("CI_TOKEN", "v1")
	ms.seedRepoVar("CI_SECRET", "v2")
	ms.seedRepoVar("DEPLOY_KEY", "v3")
	c := newTestRepoClient(t, ms)

	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
		Name: &esv1.FindName{RegExp: "^CI_"},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"CI_TOKEN", "CI_SECRET"}, keys(got))
}

// --- Validate ----------------------------------------------------------------

func TestHTTP_Validate_OrgScope(t *testing.T) {
	ms := newMockGiteaServer(t)
	ms.mu.Lock()
	ms.secrets["org/DUMMY"] = struct{}{}
	ms.mu.Unlock()
	c := newTestOrgClient(t, ms)

	result, err := c.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)
}

func TestHTTP_Validate_RepoScope(t *testing.T) {
	ms := newMockGiteaServer(t)
	ms.mu.Lock()
	ms.secrets["repo/DUMMY"] = struct{}{}
	ms.mu.Unlock()
	c := newTestRepoClient(t, ms)

	result, err := c.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)
}

// --- helpers -----------------------------------------------------------------

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
