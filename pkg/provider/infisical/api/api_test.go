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

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func NewMockServer(status int, data any) *httptest.Server {
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write(body)
	}))
}

func TestSetTokenViaMachineIdentityWorks(t *testing.T) {
	mockServer := NewMockServer(200, MachineIdentityDetailsResponse{
		AccessToken: "foobar",
	})
	defer mockServer.Close()

	apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	err = apiClient.SetTokenViaMachineIdentity("client-id", "client-secret")
	assert.NoError(t, err)
	// Verify that the access token was set.
	assert.Equal(t, apiClient.token, "foobar")
}

func TestSetTokenViaMachineIdentityErrorHandling(t *testing.T) {
	mockServer := NewMockServer(401, RevokeMachineIdentityAccessTokenResponse{
		Message: "Unauthorized",
	})
	defer mockServer.Close()

	apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	err = apiClient.SetTokenViaMachineIdentity("client-id", "client-secret")
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "API error (401): Unauthorized")

	apiClient.token = "foobar"
	err = apiClient.SetTokenViaMachineIdentity("client-id", "client-secret")
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "access token already set")
}

func TestRevokeAccessTokenWorks(t *testing.T) {
	mockServer := NewMockServer(200, RevokeMachineIdentityAccessTokenResponse{
		Message: "Success",
	})
	defer mockServer.Close()

	apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	apiClient.token = "foobar"
	err = apiClient.RevokeAccessToken()
	assert.NoError(t, err)

	// Verify that the access token was unset.
	assert.Equal(t, apiClient.token, "")
}

func TestRevokeAccessTokenErrorHandling(t *testing.T) {
	mockServer := NewMockServer(401, RevokeMachineIdentityAccessTokenResponse{
		Message: "Unauthorized",
	})
	defer mockServer.Close()

	apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	err = apiClient.RevokeAccessToken()
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "no access token was set")

	apiClient.token = "foobar"
	err = apiClient.RevokeAccessToken()
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "API error (401): Unauthorized")
}

func TestGetSecretsV3Works(t *testing.T) {
	mockServer := NewMockServer(200, GetSecretsV3Response{
		Secrets: []SecretsV3{
			{SecretKey: "foo", SecretValue: "bar"},
		},
		ImportedSecrets: []ImportedSecretV3{
			{
				Secrets: []SecretsV3{{SecretKey: "foo2", SecretValue: "bar2"}},
			},
		},
	})
	defer mockServer.Close()

	apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	secrets, err := apiClient.GetSecretsV3(GetSecretsV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		Recursive:       true,
	})
	require.NoError(t, err)
	assert.Equal(t, secrets, map[string]string{"foo": "bar", "foo2": "bar2"})
}

func TestGetSecretsV3ErrorHandling(t *testing.T) {
	mockServer := NewMockServer(401, []byte("not-json"))
	defer mockServer.Close()

	apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	_, err = apiClient.GetSecretsV3(GetSecretsV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		Recursive:       true,
	})
	assert.Error(t, err)

	mockServer = NewMockServer(401, InfisicalAPIErrorResponse{
		Message: "Unauthorized",
	})
	defer mockServer.Close()

	apiClient, err = NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	_, err = apiClient.GetSecretsV3(GetSecretsV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		Recursive:       true,
	})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "API error (401): Unauthorized")

	mockServer = NewMockServer(404, InfisicalAPIErrorResponse{
		Message: "Not Found",
	})
	defer mockServer.Close()

	apiClient, err = NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	_, err = apiClient.GetSecretsV3(GetSecretsV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		Recursive:       true,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, esv1beta1.NoSecretError{})
}

func TestGetSecretByKeyV3Works(t *testing.T) {
	mockServer := NewMockServer(200, GetSecretByKeyV3Response{
		Secret: SecretsV3{
			SecretKey:   "foo",
			SecretValue: "bar",
		},
	})
	defer mockServer.Close()

	apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	_, err = apiClient.GetSecretByKeyV3(GetSecretByKeyV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		SecretKey:       "foo",
	})
	assert.NoError(t, err)
}

func TestGetSecretByKeyV3ErrorHandling(t *testing.T) {
	mockServer := NewMockServer(404, InfisicalAPIErrorResponse{
		Message: "Not Found",
	})
	defer mockServer.Close()

	apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	_, err = apiClient.GetSecretByKeyV3(GetSecretByKeyV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		SecretKey:       "foo",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, esv1beta1.NoSecretError{})

	mockServer = NewMockServer(401, InfisicalAPIErrorResponse{
		Message: "Unauthorized",
	})
	defer mockServer.Close()

	apiClient, err = NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	_, err = apiClient.GetSecretByKeyV3(GetSecretByKeyV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		SecretKey:       "foo",
	})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "API error (401): Unauthorized")
}
