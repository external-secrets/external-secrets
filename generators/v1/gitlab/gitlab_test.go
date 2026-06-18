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

package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

const (
	testNamespace = "foo"
	testSecret    = "gitlab-token"
	testKey       = "token"
)

func newKube() client.Client {
	return clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecret,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			testKey: []byte("glpat-secret-access-token"),
		},
	}).Build()
}

// captured records the create requests the mock GitLab server received.
type captured struct {
	method      string
	path        string
	privateTok  string
	contentType string
	body        map[string]any
}

func newServer(t *testing.T, status int, response []byte, sink *captured) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		sink.method = req.Method
		// EscapedPath() preserves the on-the-wire encoding (e.g. group%2Fproject),
		// whereas req.URL.Path would be decoded back and hide double/non-encoding.
		sink.path = req.URL.EscapedPath()
		sink.privateTok = req.Header.Get("PRIVATE-TOKEN")
		sink.contentType = req.Header.Get("Content-Type")
		if req.Body != nil && req.Method == http.MethodPost {
			_ = json.NewDecoder(req.Body).Decode(&sink.body)
		}
		rw.WriteHeader(status)
		_, _ = rw.Write(response)
	}))
}

func specJSON(t *testing.T, url, projectID, groupID string) *apiextensions.JSON {
	t.Helper()
	target := ""
	if projectID != "" {
		target = fmt.Sprintf("  projectID: %q\n", projectID)
	}
	if groupID != "" {
		target += fmt.Sprintf("  groupID: %q\n", groupID)
	}
	raw := fmt.Sprintf(`apiVersion: generators.external-secrets.io/v1alpha1
kind: GitlabDeployToken
spec:
  url: %q
%s  name: "eso-token"
  scopes:
  - read_repository
  - read_registry
  username: "custom-user"
  auth:
    token:
      secretRef:
        name: %q
        key: %q
`, url, target, testSecret, testKey)
	return &apiextensions.JSON{Raw: []byte(raw)}
}

func TestGenerate(t *testing.T) {
	createResp := []byte(`{
		"id": 42,
		"name": "eso-token",
		"username": "custom-user",
		"expires_at": null,
		"token": "gitlab-deploy-token-value",
		"revoked": false,
		"expired": false,
		"scopes": ["read_repository", "read_registry"]
	}`)

	t.Run("nil spec", func(t *testing.T) {
		g := &Generator{}
		_, _, err := g.generate(context.Background(), nil, newKube(), testNamespace)
		require.Error(t, err)
	})

	t.Run("project deploy token", func(t *testing.T) {
		sink := &captured{}
		srv := newServer(t, http.StatusCreated, createResp, sink)
		defer srv.Close()

		g := &Generator{httpClient: srv.Client()}
		got, state, err := g.generate(context.Background(), specJSON(t, srv.URL, "group/project", ""), newKube(), testNamespace)
		require.NoError(t, err)

		assert.Equal(t, http.MethodPost, sink.method)
		// Unescaped path input is URL-escaped on the wire.
		assert.Equal(t, "/api/v4/projects/group%2Fproject/deploy_tokens", sink.path)
		assert.Equal(t, "glpat-secret-access-token", sink.privateTok)
		assert.Equal(t, "application/json", sink.contentType)
		assert.Equal(t, "eso-token", sink.body["name"])
		assert.Equal(t, "custom-user", sink.body["username"])
		assert.ElementsMatch(t, []any{"read_repository", "read_registry"}, sink.body["scopes"])

		assert.Equal(t, map[string][]byte{
			"username": []byte("custom-user"),
			"token":    []byte("gitlab-deploy-token-value"),
		}, got)

		require.NotNil(t, state)
		var st deployTokenState
		require.NoError(t, json.Unmarshal(state.Raw, &st))
		assert.Equal(t, 42, st.TokenID)
		assert.Equal(t, "group/project", st.ProjectID)
	})

	t.Run("group deploy token", func(t *testing.T) {
		sink := &captured{}
		srv := newServer(t, http.StatusCreated, createResp, sink)
		defer srv.Close()

		g := &Generator{httpClient: srv.Client()}
		_, state, err := g.generate(context.Background(), specJSON(t, srv.URL, "", "42"), newKube(), testNamespace)
		require.NoError(t, err)
		assert.Equal(t, "/api/v4/groups/42/deploy_tokens", sink.path)

		var st deployTokenState
		require.NoError(t, json.Unmarshal(state.Raw, &st))
		assert.Equal(t, "42", st.GroupID)
	})

	t.Run("error when both project and group set", func(t *testing.T) {
		g := &Generator{}
		_, _, err := g.generate(context.Background(), specJSON(t, "https://gitlab.com", "1", "2"), newKube(), testNamespace)
		require.ErrorContains(t, err, "exactly one of projectID or groupID")
	})

	t.Run("error on non-2xx response", func(t *testing.T) {
		sink := &captured{}
		srv := newServer(t, http.StatusForbidden, []byte(`{"message":"403 Forbidden"}`), sink)
		defer srv.Close()

		g := &Generator{httpClient: srv.Client()}
		_, _, err := g.generate(context.Background(), specJSON(t, srv.URL, "1", ""), newKube(), testNamespace)
		require.ErrorContains(t, err, "403 Forbidden")
	})

	t.Run("error when token missing in secret", func(t *testing.T) {
		kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: testSecret, Namespace: testNamespace},
			Data:       map[string][]byte{},
		}).Build()
		g := &Generator{}
		_, _, err := g.generate(context.Background(), specJSON(t, "https://gitlab.com", "1", ""), kube, testNamespace)
		require.ErrorContains(t, err, "cannot find secret data for key")
	})
}

func TestCleanup(t *testing.T) {
	t.Run("nil state is a no-op", func(t *testing.T) {
		g := &Generator{}
		require.NoError(t, g.cleanup(context.Background(), specJSON(t, "https://gitlab.com", "1", ""), nil, newKube(), testNamespace))
	})

	t.Run("revokes project deploy token", func(t *testing.T) {
		sink := &captured{}
		srv := newServer(t, http.StatusNoContent, nil, sink)
		defer srv.Close()

		state := &apiextensions.JSON{Raw: []byte(`{"url":"` + srv.URL + `","projectID":"1","tokenID":42}`)}
		g := &Generator{httpClient: srv.Client()}
		require.NoError(t, g.cleanup(context.Background(), specJSON(t, srv.URL, "1", ""), state, newKube(), testNamespace))
		assert.Equal(t, http.MethodDelete, sink.method)
		assert.Equal(t, "/api/v4/projects/1/deploy_tokens/42", sink.path)
		assert.Equal(t, "glpat-secret-access-token", sink.privateTok)
	})

	t.Run("idempotent on 404", func(t *testing.T) {
		sink := &captured{}
		srv := newServer(t, http.StatusNotFound, []byte(`{"message":"404 Not Found"}`), sink)
		defer srv.Close()

		state := &apiextensions.JSON{Raw: []byte(`{"url":"` + srv.URL + `","groupID":"7","tokenID":99}`)}
		g := &Generator{httpClient: srv.Client()}
		require.NoError(t, g.cleanup(context.Background(), specJSON(t, srv.URL, "", "7"), state, newKube(), testNamespace))
		assert.Equal(t, "/api/v4/groups/7/deploy_tokens/99", sink.path)
	})

	t.Run("error on 500", func(t *testing.T) {
		sink := &captured{}
		srv := newServer(t, http.StatusInternalServerError, []byte(`{"message":"boom"}`), sink)
		defer srv.Close()

		state := &apiextensions.JSON{Raw: []byte(`{"url":"` + srv.URL + `","projectID":"1","tokenID":42}`)}
		g := &Generator{httpClient: srv.Client()}
		require.ErrorContains(t, g.cleanup(context.Background(), specJSON(t, srv.URL, "1", ""), state, newKube(), testNamespace), "boom")
	})
}

func TestDeployTokensURL(t *testing.T) {
	tests := []struct {
		name    string
		spec    genv1alpha1.GitlabDeployTokenSpec
		want    string
		wantErr bool
	}{
		{name: "project default url", spec: genv1alpha1.GitlabDeployTokenSpec{ProjectID: "10"}, want: "https://gitlab.com/api/v4/projects/10/deploy_tokens"},
		{name: "group custom url", spec: genv1alpha1.GitlabDeployTokenSpec{URL: "https://gl.example.com", GroupID: "5"}, want: "https://gl.example.com/api/v4/groups/5/deploy_tokens"},
		{name: "path project encoded", spec: genv1alpha1.GitlabDeployTokenSpec{ProjectID: "grp/proj"}, want: "https://gitlab.com/api/v4/projects/grp%2Fproj/deploy_tokens"},
		{name: "both set", spec: genv1alpha1.GitlabDeployTokenSpec{ProjectID: "1", GroupID: "2"}, wantErr: true},
		{name: "neither set", spec: genv1alpha1.GitlabDeployTokenSpec{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deployTokensURL(&tt.spec)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
