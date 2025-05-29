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
	"testing"

	infisical "github.com/infisical/go-sdk"
	"github.com/stretchr/testify/assert"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var errNoAccessToken = errors.New("unexpected error: no access token available to revoke")

const (
	fakeClientID        = "client-id"
	fakeClientSecret    = "client-secret"
	fakeToken           = "token"
	fakeProjectSlug     = "first-project"
	fakeEnvironmentSlug = "dev"
)

func TestSetTokenViaMachineIdentity(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(200, MachineIdentityDetailsResponse{
			AccessToken: "foobar",
		})
		defer closeFunc()

		_, err := apiClient.Auth().UniversalAuthLogin(fakeClientID, fakeClientSecret)
		assert.NoError(t, err)
		assert.Equal(t, apiClient.Auth().GetAccessToken(), "foobar")
	})

	t.Run("SetTokenViaMachineIdentity: Error when non-200 response received", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, InfisicalAPIErrorResponse{
			StatusCode: 401,
			Error:      "Unauthorized",
		})
		defer closeFunc()

		_, err := apiClient.Auth().UniversalAuthLogin(fakeClientID, fakeClientSecret)
		assert.Error(t, err)
		var apiErr *InfisicalAPIError
		assert.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 401, apiErr.StatusCode)
		assert.Equal(t, "Unauthorized", apiErr.Err)
	})
}

func TestRevokeAccessToken(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(200, RevokeMachineIdentityAccessTokenResponse{
			Message: "Success",
		})
		defer closeFunc()

		apiClient.Auth().SetAccessToken(fakeToken)

		err := apiClient.Auth().RevokeAccessToken()

		assert.NoError(t, err)
		// Verify that the access token was unset.
		assert.Equal(t, apiClient.Auth().GetAccessToken(), "")
	})

	t.Run("RevokeAccessToken: Error when non-200 response received", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, InfisicalAPIErrorResponse{
			StatusCode: 401,
			Error:      "Unauthorized",
		})
		defer closeFunc()

		apiClient.Auth().SetAccessToken(fakeToken)

		err := apiClient.Auth().RevokeAccessToken()
		assert.Error(t, err)
		var apiErr *InfisicalAPIError
		assert.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 401, apiErr.StatusCode)
		assert.Equal(t, "Unauthorized", apiErr.Err)
	})

	t.Run("Error when no access token is set", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, nil)
		defer closeFunc()

		err := apiClient.Auth().RevokeAccessToken()
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

		secrets, err := apiClient.Secrets().List(infisical.ListSecretsOptions{
			ProjectSlug: fakeProjectSlug,
			Environment: fakeEnvironmentSlug,
			SecretPath:  "/",
			Recursive:   true,
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

		secrets, err := apiClient.Secrets().List(infisical.ListSecretsOptions{
			ProjectSlug:    fakeProjectSlug,
			Environment:    fakeEnvironmentSlug,
			IncludeImports: true,
			SecretPath:     "/",
			Recursive:      true,
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

		_, err := apiClient.Secrets().List(infisical.ListSecretsOptions{
			ProjectSlug: fakeProjectSlug,
			Environment: fakeEnvironmentSlug,
			SecretPath:  "/",
			Recursive:   true,
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

		secret, err := apiClient.Secrets().Retrieve(infisical.RetrieveSecretOptions{
			ProjectSlug:    fakeProjectSlug,
			Environment:    fakeEnvironmentSlug,
			SecretPath:     "/",
			IncludeImports: true,
			SecretKey:      "foo",
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

		_, err := apiClient.Secrets().Retrieve(infisical.RetrieveSecretOptions{
			ProjectSlug:    fakeProjectSlug,
			Environment:    fakeEnvironmentSlug,
			SecretPath:     "/",
			IncludeImports: true,
			SecretKey:      "foo",
		})
		assert.Error(t, err)
		// Importantly, we return the standard error for no secrets found.
		assert.ErrorIs(t, err, esv1.NoSecretError{})
	})

	// Test case where the request is unauthorized
	t.Run("ErrorHandlingUnauthorized", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, InfisicalAPIErrorResponse{
			StatusCode: 401,
			Error:      "Unauthorized",
		})
		defer closeFunc()

		_, err := apiClient.Secrets().Retrieve(infisical.RetrieveSecretOptions{
			ProjectSlug:    fakeProjectSlug,
			Environment:    fakeEnvironmentSlug,
			SecretPath:     "/",
			IncludeImports: true,
			SecretKey:      "foo",
		})
		assert.Error(t, err)
		var apiErr *InfisicalAPIError
		assert.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 401, apiErr.StatusCode)
		assert.Equal(t, "Unauthorized", apiErr.Err)
	})
}
