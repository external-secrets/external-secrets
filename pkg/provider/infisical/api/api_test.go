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
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	fakeClientID        = "client-id"
	fakeClientSecret    = "client-secret"
	fakeToken           = "token"
	fakeProjectSlug     = "first-project"
	fakeEnvironmentSlug = "dev"
)

func TestAPIClientDo(t *testing.T) {
	apiURL := "foo"
	httpMethod := "bar"

	testCases := []struct {
		Name             string
		MockStatusCode   int
		MockResponse     any
		ExpectedResponse any
		ExpectedError    error
	}{
		{
			Name:           "Success",
			MockStatusCode: 200,
			MockResponse: MachineIdentityDetailsResponse{
				AccessToken: "foobar",
			},
			ExpectedResponse: MachineIdentityDetailsResponse{
				AccessToken: "foobar",
			},
			ExpectedError: nil,
		},
		{
			Name:           "Error when response cannot be unmarshalled",
			MockStatusCode: 500,
			MockResponse:   []byte("not-json"),
			ExpectedError:  errors.New("API error (500), could not unmarshal error response: json: cannot unmarshal string into Go value of type api.InfisicalAPIErrorResponse"),
		},
		{
			Name:           "Error when non-Infisical error response received",
			MockStatusCode: 500,
			MockResponse:   map[string]string{"foo": "bar"},
			ExpectedError:  errors.New("API error (500): {\"foo\":\"bar\"}"),
		},
		{
			Name:           "Do: Error when non-200 response received",
			MockStatusCode: 401,
			MockResponse: InfisicalAPIErrorResponse{
				StatusCode: 401,
				Error:      "Unauthorized",
			},
			ExpectedError: &InfisicalAPIError{StatusCode: 401, Err: "Unauthorized", Message: ""},
		},
		{
			Name:           "Error when arbitrary details are returned",
			MockStatusCode: 401,
			MockResponse: InfisicalAPIErrorResponse{
				StatusCode: 401,
				Error:      "Unauthorized",
				Details:    map[string]string{"foo": "details"},
			},
			ExpectedError: &InfisicalAPIError{StatusCode: 401, Err: "Unauthorized", Message: "", Details: map[string]string{"foo": "details"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			apiClient, closeFunc := NewMockClient(tc.MockStatusCode, tc.MockResponse)
			defer closeFunc()

			// Automatically pluck out the expected response type using reflection to create a new empty value for unmarshalling.
			var actualResponse any
			if tc.ExpectedResponse != nil {
				actualResponse = reflect.New(reflect.TypeOf(tc.ExpectedResponse)).Interface()
			}

			err := apiClient.do(apiURL, httpMethod, nil, nil, actualResponse)
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
	apiClient, closeFunc := NewMockClient(200, []byte("not-json"))
	defer closeFunc()

	err := apiClient.do("foo", "bar", nil, nil, nil)
	assert.ErrorIs(t, err, errJSONUnmarshal)
}

func TestSetTokenViaMachineIdentity(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(200, MachineIdentityDetailsResponse{
			AccessToken: "foobar",
		})
		defer closeFunc()

		err := apiClient.SetTokenViaMachineIdentity(fakeClientID, fakeClientSecret)
		assert.NoError(t, err)
		assert.Equal(t, apiClient.token, "foobar")
	})

	t.Run("SetTokenViaMachineIdentity: Error when non-200 response received", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, InfisicalAPIErrorResponse{
			StatusCode: 401,
			Error:      "Unauthorized",
		})
		defer closeFunc()

		err := apiClient.SetTokenViaMachineIdentity(fakeClientID, fakeClientSecret)
		assert.Error(t, err)
		var apiErr *InfisicalAPIError
		assert.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 401, apiErr.StatusCode)
		assert.Equal(t, "Unauthorized", apiErr.Err)
	})

	t.Run("Error when token already set", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, nil)
		defer closeFunc()

		apiClient.token = fakeToken

		err := apiClient.SetTokenViaMachineIdentity(fakeClientID, fakeClientSecret)
		assert.ErrorIs(t, err, errAccessTokenAlreadyRetrieved)
	})
}

func TestRevokeAccessToken(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(200, RevokeMachineIdentityAccessTokenResponse{
			Message: "Success",
		})
		defer closeFunc()

		apiClient.token = fakeToken

		err := apiClient.RevokeAccessToken()
		assert.NoError(t, err)
		// Verify that the access token was unset.
		assert.Equal(t, apiClient.token, "")
	})

	t.Run("RevokeAccessToken: Error when non-200 response received", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, InfisicalAPIErrorResponse{
			StatusCode: 401,
			Error:      "Unauthorized",
		})
		defer closeFunc()

		apiClient.token = fakeToken

		err := apiClient.RevokeAccessToken()
		assert.Error(t, err)
		var apiErr *InfisicalAPIError
		assert.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 401, apiErr.StatusCode)
		assert.Equal(t, "Unauthorized", apiErr.Err)
	})

	t.Run("Error when no access token is set", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, nil)
		defer closeFunc()

		err := apiClient.RevokeAccessToken()
		assert.ErrorIs(t, err, errNoAccessToken)
	})
}

func TestGetSecretsV3(t *testing.T) {
	t.Run("Works with secrets", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(200, GetSecretsV3Response{
			Secrets: []SecretsV3{
				{SecretKey: "foo", SecretValue: "bar"},
			},
		})
		defer closeFunc()

		secrets, err := apiClient.GetSecretsV3(GetSecretsV3Request{
			ProjectSlug:     fakeProjectSlug,
			EnvironmentSlug: fakeEnvironmentSlug,
			SecretPath:      "/",
			Recursive:       true,
		})
		assert.NoError(t, err)
		assert.Equal(t, secrets, map[string]string{"foo": "bar"})
	})

	t.Run("Works with imported secrets", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(200, GetSecretsV3Response{
			ImportedSecrets: []ImportedSecretV3{{
				Secrets: []SecretsV3{{SecretKey: "foo", SecretValue: "bar"}},
			}},
		})
		defer closeFunc()

		secrets, err := apiClient.GetSecretsV3(GetSecretsV3Request{
			ProjectSlug:     fakeProjectSlug,
			EnvironmentSlug: fakeEnvironmentSlug,
			SecretPath:      "/",
			Recursive:       true,
		})
		assert.NoError(t, err)
		assert.Equal(t, secrets, map[string]string{"foo": "bar"})
	})

	t.Run("GetSecretsV3: Error when non-200 response received", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, InfisicalAPIErrorResponse{
			StatusCode: 401,
			Error:      "Unauthorized",
		})
		defer closeFunc()

		_, err := apiClient.GetSecretsV3(GetSecretsV3Request{
			ProjectSlug:     fakeProjectSlug,
			EnvironmentSlug: fakeEnvironmentSlug,
			SecretPath:      "/",
			Recursive:       true,
		})
		assert.Error(t, err)
		var apiErr *InfisicalAPIError
		assert.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 401, apiErr.StatusCode)
		assert.Equal(t, "Unauthorized", apiErr.Err)
	})
}
func TestGetSecretByKeyV3(t *testing.T) {
	t.Run("Works", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(200, GetSecretByKeyV3Response{
			Secret: SecretsV3{
				SecretKey:   "foo",
				SecretValue: "bar",
			},
		})
		defer closeFunc()

		secret, err := apiClient.GetSecretByKeyV3(GetSecretByKeyV3Request{
			ProjectSlug:     fakeProjectSlug,
			EnvironmentSlug: fakeEnvironmentSlug,
			SecretPath:      "/",
			SecretKey:       "foo",
		})
		assert.NoError(t, err)
		assert.Equal(t, "bar", secret)
	})

	t.Run("Error when secret is not found", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(404, InfisicalAPIErrorResponse{
			StatusCode: 404,
			Error:      "Not Found",
		})
		defer closeFunc()

		_, err := apiClient.GetSecretByKeyV3(GetSecretByKeyV3Request{
			ProjectSlug:     fakeProjectSlug,
			EnvironmentSlug: fakeEnvironmentSlug,
			SecretPath:      "/",
			SecretKey:       "foo",
		})
		assert.Error(t, err)
		// Importantly, we return the standard error for no secrets found.
		assert.ErrorIs(t, err, esv1beta1.NoSecretError{})
	})

	// Test case where the request is unauthorized
	t.Run("ErrorHandlingUnauthorized", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, InfisicalAPIErrorResponse{
			StatusCode: 401,
			Error:      "Unauthorized",
		})
		defer closeFunc()

		_, err := apiClient.GetSecretByKeyV3(GetSecretByKeyV3Request{
			ProjectSlug:     fakeProjectSlug,
			EnvironmentSlug: fakeEnvironmentSlug,
			SecretPath:      "/",
			SecretKey:       "foo",
		})
		assert.Error(t, err)
		var apiErr *InfisicalAPIError
		assert.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 401, apiErr.StatusCode)
		assert.Equal(t, "Unauthorized", apiErr.Err)
	})
}
