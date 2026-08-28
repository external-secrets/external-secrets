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

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	github "github.com/google/go-github/v56/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

func TestAdaptDependabotEncryptedSecret(t *testing.T) {
	actionsSecret := &github.EncryptedSecret{
		Name:                  "TOKEN",
		KeyID:                 "key-id",
		EncryptedValue:        "encrypted-value",
		Visibility:            "selected",
		SelectedRepositoryIDs: github.SelectedRepoIDs{12, 34},
	}

	got := adaptDependabotEncryptedSecret(actionsSecret)

	assert.Equal(t, "TOKEN", got.Name)
	assert.Equal(t, "key-id", got.KeyID)
	assert.Equal(t, "encrypted-value", got.EncryptedValue)
	assert.Equal(t, "selected", got.Visibility)
	assert.Equal(t, github.DependabotSecretsSelectedRepoIDs{12, 34}, got.SelectedRepositoryIDs)
}

func TestDependabotSecretLifecycle(t *testing.T) {
	tests := []struct {
		name       string
		provider   *esv1.GithubProvider
		pathPrefix string
	}{
		{
			name: "organization",
			provider: &esv1.GithubProvider{
				SecretType:   esv1.GithubSecretTypeDependabot,
				Organization: "acme",
			},
			pathPrefix: "/orgs/acme/dependabot/secrets",
		},
		{
			name: "repository",
			provider: &esv1.GithubProvider{
				SecretType:   esv1.GithubSecretTypeDependabot,
				Organization: "acme",
				Repository:   "widgets",
			},
			pathPrefix: "/repos/acme/widgets/dependabot/secrets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			var putBody map[string]any
			var putDecodeErr error
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.Path)
				switch {
				case r.Method == http.MethodGet && r.URL.Path == tt.pathPrefix+"/public-key":
					_, _ = fmt.Fprint(w, `{"key_id":"key-id","key":"a2V5"}`)
				case r.Method == http.MethodGet && r.URL.Path == tt.pathPrefix+"/TOKEN":
					_, _ = fmt.Fprint(w, `{"name":"TOKEN","visibility":"selected"}`)
				case r.Method == http.MethodGet && r.URL.Path == tt.pathPrefix:
					_, _ = fmt.Fprint(w, `{"total_count":1,"secrets":[{"name":"TOKEN"}]}`)
				case r.Method == http.MethodPut && r.URL.Path == tt.pathPrefix+"/TOKEN":
					putDecodeErr = json.NewDecoder(r.Body).Decode(&putBody)
					if putDecodeErr != nil {
						http.Error(w, putDecodeErr.Error(), http.StatusBadRequest)
						return
					}
					w.WriteHeader(http.StatusCreated)
				case r.Method == http.MethodDelete && r.URL.Path == tt.pathPrefix+"/TOKEN":
					w.WriteHeader(http.StatusNoContent)
				default:
					http.Error(w, "unexpected request", http.StatusNotFound)
				}
			}))
			t.Cleanup(server.Close)

			g := &Client{provider: tt.provider}
			ghClient := newGithubTestClient(t, server)
			require.NoError(t, g.configureSecretClient(context.Background(), ghClient, esv1.GithubSecretTypeDependabot))

			ref := esv1alpha1.PushSecretData{
				Match: esv1alpha1.PushSecretMatch{
					RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: "TOKEN"},
				},
			}
			secret, _, err := g.getSecretFn(context.Background(), ref)
			require.NoError(t, err)
			assert.Equal(t, "TOKEN", secret.Name)

			secrets, _, err := g.listSecretsFn(context.Background())
			require.NoError(t, err)
			assert.Equal(t, 1, secrets.TotalCount)

			key, _, err := g.getPublicKeyFn(context.Background())
			require.NoError(t, err)
			assert.Equal(t, "key-id", key.GetKeyID())

			_, err = g.createOrUpdateFn(context.Background(), &github.EncryptedSecret{
				Name:           "TOKEN",
				KeyID:          "key-id",
				EncryptedValue: "encrypted-value",
				Visibility:     "selected",
			})
			require.NoError(t, err)
			_, err = g.deleteSecretFn(context.Background(), ref)
			require.NoError(t, err)
			require.NoError(t, putDecodeErr)

			assert.Equal(t, []string{
				http.MethodGet + " " + tt.pathPrefix + "/TOKEN",
				http.MethodGet + " " + tt.pathPrefix,
				http.MethodGet + " " + tt.pathPrefix + "/public-key",
				http.MethodPut + " " + tt.pathPrefix + "/TOKEN",
				http.MethodDelete + " " + tt.pathPrefix + "/TOKEN",
			}, requests)
			assert.Equal(t, "key-id", putBody["key_id"])
			assert.Equal(t, "encrypted-value", putBody["encrypted_value"])
			assert.Equal(t, "selected", putBody["visibility"])
		})
	}
}

func TestDependabotPushSecretPreservesSelectedRepositories(t *testing.T) {
	var putBody map[string]json.RawMessage
	var putDecodeErr error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/acme/dependabot/secrets/TOKEN":
			_, _ = fmt.Fprint(w, `{"name":"TOKEN","visibility":"selected"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/acme/dependabot/secrets/public-key":
			_, _ = fmt.Fprint(w, `{"key_id":"key-id","key":"Zm9vYmFyCg=="}`)
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/acme/dependabot/secrets/TOKEN/repositories":
			_, _ = fmt.Fprint(w, `{"total_count":2,"repositories":[{"id":12},{"id":34}]}`)
		case r.Method == http.MethodPut && r.URL.Path == "/orgs/acme/dependabot/secrets/TOKEN":
			putDecodeErr = json.NewDecoder(r.Body).Decode(&putBody)
			w.WriteHeader(http.StatusCreated)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	provider := &esv1.GithubProvider{
		SecretType:   esv1.GithubSecretTypeDependabot,
		Organization: "acme",
	}
	g := &Client{provider: provider}
	require.NoError(t, g.configureSecretClient(context.Background(), newGithubTestClient(t, server), esv1.GithubSecretTypeDependabot))

	remoteRef := esv1alpha1.PushSecretData{
		Match: esv1alpha1.PushSecretMatch{
			SecretKey: "value",
			RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: "TOKEN"},
		},
	}
	err := g.PushSecret(context.Background(), &corev1.Secret{Data: map[string][]byte{"value": []byte("secret")}}, remoteRef)
	require.NoError(t, err)
	require.NoError(t, putDecodeErr)
	assert.JSONEq(t, `"selected"`, string(putBody["visibility"]))
	assert.JSONEq(t, `["12","34"]`, string(putBody["selected_repository_ids"]))
}

func TestDependabotOrgSelectedRepositoriesPagination(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/orgs/acme/dependabot/secrets/TOKEN/repositories", r.URL.Path)
		switch r.URL.Query().Get("page") {
		case "":
			w.Header().Set("Link", "<"+server.URL+r.URL.Path+"?page=2>; rel=\"next\"")
			_, _ = fmt.Fprint(w, `{"total_count":2,"repositories":[{"id":12}]}`)
		case "2":
			_, _ = fmt.Fprint(w, `{"total_count":2,"repositories":[{"id":34}]}`)
		default:
			http.Error(w, "unexpected page", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	ghClient := newGithubTestClient(t, server)
	g := &Client{
		provider:         &esv1.GithubProvider{Organization: "acme"},
		dependabotClient: *ghClient.Dependabot,
	}

	ids, err := g.dependabotOrgListSelectedRepoIDs(context.Background(), "TOKEN")
	require.NoError(t, err)
	assert.Equal(t, github.SelectedRepoIDs{12, 34}, ids)
}

func TestDependabotErrorsPropagate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	ghClient := newGithubTestClient(t, server)
	g := &Client{
		provider:         &esv1.GithubProvider{Organization: "acme"},
		dependabotClient: *ghClient.Dependabot,
	}

	ref := esv1alpha1.PushSecretData{
		Match: esv1alpha1.PushSecretMatch{
			RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: "TOKEN"},
		},
	}
	_, _, err := g.dependabotOrgGetSecretFn(context.Background(), ref)
	assert.Error(t, err)
	_, err = g.dependabotOrgListSelectedRepoIDs(context.Background(), "TOKEN")
	assert.Error(t, err)
}

func newGithubTestClient(t *testing.T, server *httptest.Server) *github.Client {
	t.Helper()
	client := github.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL
	client.UploadURL = baseURL
	return client
}
