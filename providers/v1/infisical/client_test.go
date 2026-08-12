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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	infisical "github.com/infisical/go-sdk"
	"github.com/stretchr/testify/assert"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestGetSecretAddress(t *testing.T) {
	t.Run("when the key is not addressing a path and uses the default path", func(t *testing.T) {
		path, key := getSecretAddress("/", "foo")
		assert.Equal(t, "/", path)
		assert.Equal(t, "foo", key)

		path, key = getSecretAddress("/foo", "bar")
		assert.Equal(t, "/foo", path)
		assert.Equal(t, "bar", key)
	})

	t.Run("when the key is addressing a path", func(t *testing.T) {
		path, key := getSecretAddress("/", "/foo/bar")
		assert.Equal(t, path, "/foo")
		assert.Equal(t, key, "bar")
	})

	t.Run("when the key is addressing a path and ignores the default path", func(t *testing.T) {
		path, key := getSecretAddress("/foo", "/bar/baz")
		assert.Equal(t, "/bar", path)
		assert.Equal(t, "baz", key)
	})

	t.Run("works with a nested directory", func(t *testing.T) {
		path, key := getSecretAddress("/", "/foo/bar/baz")
		assert.Equal(t, "/foo/bar", path)
		assert.Equal(t, "baz", key, "baz")
	})

	t.Run("relative key joins onto the default path", func(t *testing.T) {
		path, key := getSecretAddress("/secrets/mysql-core", "azure/admin-users")
		assert.Equal(t, "/secrets/mysql-core/azure", path)
		assert.Equal(t, "admin-users", key)
	})

	t.Run("relative key with default root path", func(t *testing.T) {
		path, key := getSecretAddress("/", "azure/admin-users")
		assert.Equal(t, "/azure", path)
		assert.Equal(t, "admin-users", key)
	})

	t.Run("relative key with nested folders", func(t *testing.T) {
		path, key := getSecretAddress("/scope", "a/b/c/name")
		assert.Equal(t, "/scope/a/b/c", path)
		assert.Equal(t, "name", key)
	})
}

// listStub answers the Infisical list endpoint with body and keeps the query it
// was called with, so tests can assert on the request instead of the response.
type listStub struct {
	query url.Values
	calls int
}

func (s *listStub) provider(t *testing.T, scope *ClientScope, body string, status int) *Provider {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.query = r.URL.Query()
		s.calls++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	return &Provider{
		sdkClient: infisical.NewInfisicalClient(context.Background(), infisical.Config{SiteUrl: srv.URL}),
		apiScope:  scope,
	}
}

func (s *listStub) get(key string) string {
	return s.query.Get(key)
}

func TestGetAllSecretsListRequest(t *testing.T) {
	const emptyList = `{"secrets":[],"imports":[]}`

	storeScope := func(path string, recursive bool) *ClientScope {
		return &ClientScope{
			EnvironmentSlug:        "dev",
			ProjectSlug:            "proj",
			SecretPath:             path,
			Recursive:              recursive,
			ExpandSecretReferences: true,
		}
	}

	tests := []struct {
		name          string
		scope         *ClientScope
		find          esv1.ExternalSecretFind
		wantPath      string
		wantRecursive string
	}{
		{
			name:          "find path becomes the request root",
			scope:         storeScope("/store-default", false),
			find:          esv1.ExternalSecretFind{Path: new("/app")},
			wantPath:      "/app",
			wantRecursive: "false",
		},
		{
			name:          "without a find path the store scope is left alone",
			scope:         storeScope("/store-default", false),
			find:          esv1.ExternalSecretFind{},
			wantPath:      "/store-default",
			wantRecursive: "false",
		},
		{
			name:          "a recursive store stays recursive",
			scope:         storeScope("/store-default", true),
			find:          esv1.ExternalSecretFind{},
			wantPath:      "/store-default",
			wantRecursive: "true",
		},
		{
			name:          "a recursive store stays recursive under a find path",
			scope:         storeScope("/store-default", true),
			find:          esv1.ExternalSecretFind{Path: new("/app")},
			wantPath:      "/app",
			wantRecursive: "true",
		},
		{
			name:          "the project root is a valid find path",
			scope:         storeScope("/store-default", false),
			find:          esv1.ExternalSecretFind{Path: new("/")},
			wantPath:      "/",
			wantRecursive: "false",
		},
		{
			// The SDK turns an empty SecretPath into "/", so an empty find path
			// would widen the request to the whole project if it were passed on.
			name:          "an empty find path keeps the store scope",
			scope:         storeScope("/store-default", false),
			find:          esv1.ExternalSecretFind{Path: new("")},
			wantPath:      "/store-default",
			wantRecursive: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &listStub{}
			p := stub.provider(t, tt.scope, emptyList, http.StatusOK)

			_, err := p.GetAllSecrets(context.Background(), tt.find)
			assert.NoError(t, err)

			assert.Equal(t, tt.wantPath, stub.get("secretPath"))
			assert.Equal(t, tt.wantRecursive, stub.get("recursive"))
			assert.Equal(t, "dev", stub.get("environment"))
			assert.Equal(t, "proj", stub.get("workspaceSlug"))
			assert.Equal(t, "true", stub.get("include_imports"))
			assert.Equal(t, "true", stub.get("expandSecretReferences"))
		})
	}
}

func TestGetAllSecretsResults(t *testing.T) {
	scope := &ClientScope{EnvironmentSlug: "dev", ProjectSlug: "proj", SecretPath: "/store-default"}

	t.Run("the path scopes the request and the name filters what comes back", func(t *testing.T) {
		body := `{"secrets":[
			{"secretKey":"DB_HOST","secretValue":"db","secretPath":"/app"},
			{"secretKey":"API_KEY","secretValue":"key","secretPath":"/app/nested"}
		],"imports":[]}`

		stub := &listStub{}
		p := stub.provider(t, scope, body, http.StatusOK)

		got, err := p.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
			Path: new("/app"),
			Name: &esv1.FindName{RegExp: "^DB_"},
		})
		assert.NoError(t, err)

		assert.Equal(t, "/app", stub.get("secretPath"))
		assert.Equal(t, map[string][]byte{"DB_HOST": []byte("db")}, got)
	})

	t.Run("imported secrets are returned even without a path of their own", func(t *testing.T) {
		body := `{"secrets":[
			{"secretKey":"LOCAL","secretValue":"local","secretPath":"/app"}
		],"imports":[
			{"secretPath":"/shared","environment":"dev","folderId":"f1","secrets":[
				{"secretKey":"IMPORTED","secretValue":"imported"}
			]}
		]}`

		stub := &listStub{}
		p := stub.provider(t, scope, body, http.StatusOK)

		got, err := p.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{Path: new("/app")})
		assert.NoError(t, err)
		assert.Equal(t, map[string][]byte{
			"LOCAL":    []byte("local"),
			"IMPORTED": []byte("imported"),
		}, got)
	})

	t.Run("an api error is not swallowed", func(t *testing.T) {
		stub := &listStub{}
		p := stub.provider(t, scope, `{"message":"forbidden"}`, http.StatusForbidden)

		_, err := p.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{Path: new("/app")})
		assert.Error(t, err)
	})

	t.Run("an invalid name regexp fails before anything is selected", func(t *testing.T) {
		stub := &listStub{}
		p := stub.provider(t, scope, `{"secrets":[],"imports":[]}`, http.StatusOK)

		_, err := p.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
			Name: &esv1.FindName{RegExp: "("},
		})
		assert.Error(t, err)
	})

	t.Run("finding by tags is still unsupported", func(t *testing.T) {
		stub := &listStub{}
		p := stub.provider(t, scope, `{"secrets":[],"imports":[]}`, http.StatusOK)

		_, err := p.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{Tags: map[string]string{"env": "dev"}})
		assert.ErrorIs(t, err, errTagsNotImplemented)
		assert.Zero(t, stub.calls)
	})
}
