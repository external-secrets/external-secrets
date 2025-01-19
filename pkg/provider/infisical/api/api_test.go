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
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func TestAPIClientDo(t *testing.T) {
	apiUrl := "foo"
	httpMethod := "bar"

	testCases := []struct {
		Name             string
		MockServer       *httptest.Server
		ExpectedResponse any
		ExpectedError    error
	}{
		{
			Name: "Success",
			MockServer: NewMockServer(200, MachineIdentityDetailsResponse{
				AccessToken: "foobar",
			}),
			ExpectedResponse: MachineIdentityDetailsResponse{
				AccessToken: "foobar",
			},
			ExpectedError: nil,
		},
		{
			Name:          "Error when response cannot be unmarshalled",
			MockServer:    NewMockServer(500, []byte("not-json")),
			ExpectedError: errors.New("API error (500), could not unmarshal error response: json: cannot unmarshal string into Go value of type api.InfisicalAPIErrorResponse"),
		},
		{
			Name:          "Error when non-Infisical error response received",
			MockServer:    NewMockServer(500, map[string]string{"foo": "bar"}),
			ExpectedError: errors.New("API error (500): {\"foo\":\"bar\"}"),
		},
		{
			Name: "Error when non-200 response received",
			MockServer: NewMockServer(401, InfisicalAPIErrorResponse{
				Message: "No authentication data provided",
				Error:   "Unauthorized",
			}),
			ExpectedError: &InfisicalAPIError{StatusCode: 401, Message: "No authentication data provided", Err: "Unauthorized"},
		},
		{
			Name: "Error when arbitrary details are returned",
			MockServer: NewMockServer(401, InfisicalAPIErrorResponse{
				Error:   "Unauthorized",
				Message: "No authentication data provided",
				Details: map[string]string{"foo": "details"},
			}),
			ExpectedError: &InfisicalAPIError{StatusCode: 401, Message: "No authentication data provided", Err: "Unauthorized", Details: map[string]string{"foo": "details"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			defer tc.MockServer.Close()

			apiClient, err := NewAPIClient(tc.MockServer.URL, tc.MockServer.Client())
			require.NoError(t, err)

			// Automatically pluck out the expected response type using reflection to create a new empty value for unmarshalling.
			var actualResponse any
			if tc.ExpectedResponse != nil {
				actualResponse = reflect.New(reflect.TypeOf(tc.ExpectedResponse)).Interface()
			}

			err = apiClient.do(apiUrl, httpMethod, nil, nil, actualResponse)
			if tc.ExpectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.ExpectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.ExpectedResponse, reflect.ValueOf(actualResponse).Elem().Interface())
			}
		})
	}
}

// TestAPIClientDoInvalidResponse tests the case where the response is a 200 but does not unmarshal
// correctly.
func TestAPIClientDoInvalidResponse(t *testing.T) {
	mockServer := NewMockServer(200, []byte("not-json"))
	defer mockServer.Close()

	apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
	require.NoError(t, err)

	err = apiClient.do("foo", "bar", nil, nil, nil)
	assert.ErrorIs(t, err, errJSONUnmarshal)
}

func TestSetTokenViaMachineIdentity(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockServer := NewMockServer(200, MachineIdentityDetailsResponse{
			AccessToken: "foobar",
		})
		defer mockServer.Close()

		apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
		require.NoError(t, err)

		err = apiClient.SetTokenViaMachineIdentity("client-id", "client-secret")
		assert.NoError(t, err)
		assert.Equal(t, apiClient.token, "foobar")
	})

	t.Run("Error when non-200 response received", func(t *testing.T) {
		mockServer := NewMockServer(401, InfisicalAPIErrorResponse{
			Error: "Unauthorized",
		})
		defer mockServer.Close()

		apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
		require.NoError(t, err)

		err = apiClient.SetTokenViaMachineIdentity("client-id", "client-secret")
		assert.Error(t, err)
		assert.IsType(t, &InfisicalAPIError{}, err)
		assert.Equal(t, 401, err.(*InfisicalAPIError).StatusCode)
		assert.Equal(t, "Unauthorized", err.(*InfisicalAPIError).Err)
	})

	t.Run("Error when token already set", func(t *testing.T) {
		mockServer := NewMockServer(401, nil)
		defer mockServer.Close()

		apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
		require.NoError(t, err)

		apiClient.token = "foobar"

		err = apiClient.SetTokenViaMachineIdentity("client-id", "client-secret")
		assert.ErrorIs(t, err, errAccessTokenAlreadyRetrieved)
	})
}

func TestRevokeAccessToken(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
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
	})

	t.Run("Error when non-200 response received", func(t *testing.T) {
		mockServer := NewMockServer(401, InfisicalAPIErrorResponse{
			Error: "Unauthorized",
		})
		defer mockServer.Close()

		apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
		require.NoError(t, err)

		apiClient.token = "foobar"

		err = apiClient.RevokeAccessToken()
		assert.Error(t, err)
		assert.IsType(t, &InfisicalAPIError{}, err)
		assert.Equal(t, 401, err.(*InfisicalAPIError).StatusCode)
		assert.Equal(t, "Unauthorized", err.(*InfisicalAPIError).Err)
	})

	t.Run("Error when no access token is set", func(t *testing.T) {
		mockServer := NewMockServer(401, nil)
		defer mockServer.Close()

		apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
		require.NoError(t, err)

		err = apiClient.RevokeAccessToken()
		assert.ErrorIs(t, err, errNoAccessToken)
	})
}

func TestGetSecretsV3(t *testing.T) {
	t.Run("Works with secrets", func(t *testing.T) {
		mockServer := NewMockServer(200, GetSecretsV3Response{
			Secrets: []SecretsV3{
				{SecretKey: "foo", SecretValue: "bar"},
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
		assert.NoError(t, err)
		assert.Equal(t, secrets, map[string]string{"foo": "bar"})
	})

	t.Run("Works with imported secrets", func(t *testing.T) {
		mockServer := NewMockServer(200, GetSecretsV3Response{
			ImportedSecrets: []ImportedSecretV3{{
				Secrets: []SecretsV3{{SecretKey: "foo", SecretValue: "bar"}},
			}},
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
		assert.NoError(t, err)
		assert.Equal(t, secrets, map[string]string{"foo": "bar"})
	})

	t.Run("Error when non-200 response received", func(t *testing.T) {
		mockServer := NewMockServer(401, InfisicalAPIErrorResponse{
			Error: "Unauthorized",
		})
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
		assert.IsType(t, &InfisicalAPIError{}, err)
		assert.Equal(t, 401, err.(*InfisicalAPIError).StatusCode)
		assert.Equal(t, "Unauthorized", err.(*InfisicalAPIError).Err)
	})
}
func TestGetSecretByKeyV3(t *testing.T) {
	t.Run("Works", func(t *testing.T) {
		mockServer := NewMockServer(200, GetSecretByKeyV3Response{
			Secret: SecretsV3{
				SecretKey:   "foo",
				SecretValue: "bar",
			},
		})
		defer mockServer.Close()

		apiClient, err := NewAPIClient(mockServer.URL, mockServer.Client())
		require.NoError(t, err)

		secret, err := apiClient.GetSecretByKeyV3(GetSecretByKeyV3Request{
			ProjectSlug:     "first-project",
			EnvironmentSlug: "dev",
			SecretPath:      "/",
			SecretKey:       "foo",
		})
		assert.NoError(t, err)
		assert.Equal(t, "bar", secret)
	})

	t.Run("Error when secret is not found", func(t *testing.T) {
		mockServer := NewMockServer(404, InfisicalAPIErrorResponse{
			Error: "Not Found",
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
		// Importantly, we return the standard error for no secrets found.
		assert.ErrorIs(t, err, esv1beta1.NoSecretError{})
	})

	// Test case where the request is unauthorized
	t.Run("ErrorHandlingUnauthorized", func(t *testing.T) {
		mockServer := NewMockServer(401, InfisicalAPIErrorResponse{
			Error: "Unauthorized",
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
		assert.IsType(t, &InfisicalAPIError{}, err)
		assert.Equal(t, 401, err.(*InfisicalAPIError).StatusCode)
		assert.Equal(t, "Unauthorized", err.(*InfisicalAPIError).Err)
	})
}
