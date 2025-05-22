/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1meta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/api"
)

type storeModifier func(*esv1.SecretStore) *esv1.SecretStore

var apiScope = InfisicalClientScope{
	SecretPath:      "/",
	ProjectSlug:     "first-project",
	EnvironmentSlug: "dev",
}

type TestCases struct {
	Name           string
	MockStatusCode int
	MockResponse   any
	Key            string
	Property       string
	Error          error
	Output         any
}

func TestGetSecret(t *testing.T) {
	key := "foo"

	testCases := []TestCases{
		{
			Name:           "Get_valid_key",
			MockStatusCode: 200,
			MockResponse: api.GetSecretByKeyV3Response{
				Secret: api.SecretsV3{
					SecretKey:   key,
					SecretValue: "bar",
				},
			},
			Key:    key,
			Output: []byte("bar"),
		},
		{
			Name:           "Get_property_key",
			MockStatusCode: 200,
			MockResponse: api.GetSecretByKeyV3Response{
				Secret: api.SecretsV3{
					SecretKey:   key,
					SecretValue: `{"bar": "value"}`,
				},
			},
			Key:      key,
			Property: "bar",
			Output:   []byte("value"),
		},
		{
			Name:           "Key_not_found",
			MockStatusCode: 404,
			MockResponse: api.InfisicalAPIError{
				StatusCode: 404,
				Err:        "Not Found",
				Message:    "Secret not found",
			},
			Key:    "key",
			Error:  esv1.NoSecretErr,
			Output: "",
		},
		{
			Name:           "Key_with_slash",
			MockStatusCode: 200,
			MockResponse: api.GetSecretByKeyV3Response{
				Secret: api.SecretsV3{
					SecretKey:   "bar",
					SecretValue: "value",
				},
			},
			Key:    "/foo/bar",
			Output: []byte("value"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			apiClient, closeFunc := api.NewMockClient(tc.MockStatusCode, tc.MockResponse)
			defer closeFunc()
			p := &Provider{
				apiClient: apiClient,
				apiScope:  &apiScope,
			}

			output, err := p.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
				Key:      tc.Key,
				Property: tc.Property,
			})

			if tc.Error == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.Output, output)
			} else {
				assert.ErrorAs(t, err, &tc.Error)
			}
		})
	}
}

// TestGetSecretWithPath verifies that request is translated from a key
// `/foo/bar` to a secret `bar` with `secretPath` of `/foo`.
func TestGetSecretWithPath(t *testing.T) {
	requestedKey := "/foo/bar"
	expectedSecretPath := "/foo"
	expectedSecretKey := "bar"

	// Prepare the mock response.
	data := api.GetSecretByKeyV3Response{
		Secret: api.SecretsV3{
			SecretKey:   expectedSecretKey,
			SecretValue: `value`,
		},
	}
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	// Prepare the mock server, which asserts the request translation is correct.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf("/api/v3/secrets/raw/%s", expectedSecretKey), r.URL.Path)
		assert.Equal(t, expectedSecretPath, r.URL.Query().Get("secretPath"))
		w.WriteHeader(200)
		_, err := w.Write(body)
		if err != nil {
			panic(err)
		}
	}))
	defer server.Close()

	client, err := api.NewAPIClient(server.URL, server.Client())
	require.NoError(t, err)
	p := &Provider{
		apiClient: client,
		apiScope:  &apiScope,
	}

	// Retrieve the secret.
	output, err := p.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      requestedKey,
		Property: "",
	})
	// And, we should get back the expected secret value.
	require.NoError(t, err)
	assert.Equal(t, []byte("value"), output)
}

func TestGetSecretMap(t *testing.T) {
	key := "foo"
	testCases := []TestCases{
		{
			Name:           "Get_valid_key_map",
			MockStatusCode: 200,
			MockResponse: api.GetSecretByKeyV3Response{
				Secret: api.SecretsV3{
					SecretKey:   key,
					SecretValue: `{"bar": "value"}`,
				},
			},
			Key: key,
			Output: map[string][]byte{
				"bar": []byte("value"),
			},
		},
		{
			Name:           "Get_invalid_map",
			MockStatusCode: 200,
			MockResponse:   []byte(``),
			Key:            key,
			Error:          errors.New("unable to unmarshal secret foo"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			apiClient, closeFunc := api.NewMockClient(tc.MockStatusCode, tc.MockResponse)
			defer closeFunc()
			p := &Provider{
				apiClient: apiClient,
				apiScope:  &apiScope,
			}
			output, err := p.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
				Key:      tc.Key,
				Property: tc.Property,
			})
			if tc.Error == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.Output, output)
			} else {
				assert.ErrorAs(t, err, &tc.Error)
			}
		})
	}
}

func makeSecretStore(projectSlug, environment, secretsPath string, fn ...storeModifier) *esv1.SecretStore {
	store := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Infisical: &esv1.InfisicalProvider{
					Auth: esv1.InfisicalAuth{
						UniversalAuthCredentials: &esv1.UniversalAuthCredentials{},
					},
					SecretsScope: esv1.MachineIdentityScopeInWorkspace{
						SecretsPath:     secretsPath,
						EnvironmentSlug: environment,
						ProjectSlug:     projectSlug,
					},
				},
			},
		},
	}
	for _, f := range fn {
		store = f(store)
	}
	return store
}

func withClientID(name, key string, namespace *string) storeModifier {
	return func(store *esv1.SecretStore) *esv1.SecretStore {
		store.Spec.Provider.Infisical.Auth.UniversalAuthCredentials.ClientID = esv1meta.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}

func withClientSecret(name, key string, namespace *string) storeModifier {
	return func(store *esv1.SecretStore) *esv1.SecretStore {
		store.Spec.Provider.Infisical.Auth.UniversalAuthCredentials.ClientSecret = esv1meta.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}

type ValidateStoreTestCase struct {
	name        string
	store       *esv1.SecretStore
	assertError func(t *testing.T, err error)
}

func TestValidateStore(t *testing.T) {
	const randomID = "some-random-id"
	const authType = "universal-auth"
	var authCredMissingErr = errors.New("universalAuthCredentials.clientId and universalAuthCredentials.clientSecret cannot be empty")
	var authScopeMissingErr = errors.New("secretsScope.projectSlug and secretsScope.environmentSlug cannot be empty")

	testCases := []ValidateStoreTestCase{
		{
			name:  "Missing projectSlug",
			store: makeSecretStore("", "", ""),
			assertError: func(t *testing.T, err error) {
				require.ErrorAs(t, err, &authScopeMissingErr)
			},
		},
		{
			name:  "Missing clientID",
			store: makeSecretStore(apiScope.ProjectSlug, apiScope.EnvironmentSlug, apiScope.SecretPath, withClientID(authType, randomID, nil)),
			assertError: func(t *testing.T, err error) {
				require.ErrorAs(t, err, &authCredMissingErr)
			},
		},
		{
			name:  "Missing clientSecret",
			store: makeSecretStore(apiScope.ProjectSlug, apiScope.EnvironmentSlug, apiScope.SecretPath, withClientSecret(authType, randomID, nil)),
			assertError: func(t *testing.T, err error) {
				require.ErrorAs(t, err, &authCredMissingErr)
			},
		},
		{
			name:        "Success",
			store:       makeSecretStore(apiScope.ProjectSlug, apiScope.EnvironmentSlug, apiScope.SecretPath, withClientID(authType, randomID, nil), withClientSecret(authType, randomID, nil)),
			assertError: func(t *testing.T, err error) { require.NoError(t, err) },
		},
	}
	p := Provider{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := p.ValidateStore(tc.store)
			tc.assertError(t, err)
		})
	}
}
