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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)

	StatusCode int
	Body       []byte
	Err        error
}

func NewMockClient(status int, data any, err error) *MockClient {
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return &MockClient{
		StatusCode: status,
		Body:       body,
		Err:        err,
	}
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	if m.Err != nil {
		return nil, m.Err
	}

	return &http.Response{
		StatusCode: m.StatusCode,
		Body:       io.NopCloser(bytes.NewBufferString(string(m.Body))),
	}, nil
}

func TestSetTokenViaMachineIdentityWorks(t *testing.T) {
	body, err := json.Marshal(MachineIdentityDetailsResponse{
		AccessToken: "foobar",
	})
	require.NoError(t, err)

	apiClient, err := NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 200,
		Body:       body,
		Err:        nil,
	})
	require.NoError(t, err)

	err = apiClient.SetTokenViaMachineIdentity("client-id", "client-secret")
	assert.NoError(t, err)
	// Verify that the access token was set.
	assert.Equal(t, apiClient.token, "foobar")
}

func TestSetTokenViaMachineIdentityErrorHandling(t *testing.T) {
	apiClient, err := NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 401,
		Body:       []byte(`{"message":"Unauthorized"}`),
		Err:        nil,
	})
	require.NoError(t, err)

	err = apiClient.SetTokenViaMachineIdentity("client-id", "client-secret")
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "got error 401: Unauthorized")

	apiClient.token = "foobar"
	err = apiClient.SetTokenViaMachineIdentity("client-id", "client-secret")
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "access token already set")
}

func TestRevokeAccessTokenWorks(t *testing.T) {
	body, err := json.Marshal(RevokeMachineIdentityAccessTokenResponse{
		Message: "Success",
	})
	require.NoError(t, err)

	apiClient, err := NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 200,
		Body:       body,
		Err:        nil,
	})
	require.NoError(t, err)

	apiClient.token = "foobar"
	err = apiClient.RevokeAccessToken()
	assert.NoError(t, err)
}

func TestRevokeAccessTokenErrorHandling(t *testing.T) {
	apiClient, err := NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 401,
		Body:       []byte(`{"message":"Unauthorized"}`),
		Err:        nil,
	})
	require.NoError(t, err)

	err = apiClient.RevokeAccessToken()
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "no access token was set")

	apiClient.token = "foobar"
	err = apiClient.RevokeAccessToken()
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "got error 401: Unauthorized")
}

func TestGetSecretsV3Works(t *testing.T) {
	body, err := json.Marshal(GetSecretsV3Response{
		Secrets: []SecretsV3{
			{SecretKey: "foo", SecretValue: "bar"},
		},
		ImportedSecrets: []ImportedSecretV3{
			{
				Secrets: []SecretsV3{{SecretKey: "foo2", SecretValue: "bar2"}},
			},
		},
	})
	require.NoError(t, err)

	apiClient, err := NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 200,
		Body:       body,
		Err:        nil,
	})
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
	apiClient, err := NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 200,
		Body:       []byte("not-json"),
		Err:        nil,
	})
	require.NoError(t, err)

	_, err = apiClient.GetSecretsV3(GetSecretsV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		Recursive:       true,
	})
	assert.Error(t, err)

	apiClient, err = NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 401,
		Body:       []byte(`{"message":"Unauthorized"}`),
		Err:        nil,
	})
	require.NoError(t, err)

	_, err = apiClient.GetSecretsV3(GetSecretsV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		Recursive:       true,
	})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "got error 401: Unauthorized")

	apiClient, err = NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 404,
		Body:       []byte(`{"message":"Not Found"}`),
		Err:        nil,
	})
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
	body, err := json.Marshal(GetSecretByKeyV3Response{
		Secret: SecretsV3{
			SecretKey:   "foo",
			SecretValue: "bar",
		},
	})
	require.NoError(t, err)

	apiClient, err := NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 200,
		Body:       body,
		Err:        nil,
	})
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
	apiClient, err := NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 404,
		Body:       []byte(`{"message":"Not Found"}`),
		Err:        nil,
	})
	require.NoError(t, err)

	_, err = apiClient.GetSecretByKeyV3(GetSecretByKeyV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		SecretKey:       "foo",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, esv1beta1.NoSecretError{})

	apiClient, err = NewAPIClient("https://api.infisical.com", &MockClient{
		StatusCode: 401,
		Body:       []byte(`{"message":"Unauthorized"}`),
		Err:        nil,
	})
	require.NoError(t, err)

	_, err = apiClient.GetSecretByKeyV3(GetSecretByKeyV3Request{
		ProjectSlug:     "first-project",
		EnvironmentSlug: "dev",
		SecretPath:      "/",
		SecretKey:       "foo",
	})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "got error 401: Unauthorized")
}
