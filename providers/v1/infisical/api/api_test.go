/*
Copyright © 2025 ESO Maintainer Team

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

package api

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	infisical "github.com/infisical/go-sdk"
	"github.com/stretchr/testify/assert"
)

func parseInfisicalAPIError(err error, t *testing.T) (int, string, error) {
	var apiErr *infisical.APIError
	assert.True(t, errors.As(err, &apiErr))

	// Regex to extract status-code
	statusRegex := regexp.MustCompile(`\[status-code=(\d+)\]`)
	statusMatch := statusRegex.FindStringSubmatch(apiErr.Error())

	// Regex to extract message (handles quoted content)
	messageRegex := regexp.MustCompile(`\[message="([^"]*)"\]`)
	messageMatch := messageRegex.FindStringSubmatch(apiErr.Error())

	if len(statusMatch) < 2 {
		return 0, "", fmt.Errorf("status-code not found in error string")
	}

	if len(messageMatch) < 2 {
		return 0, "", fmt.Errorf("message not found in error string")
	}

	statusCode, err := strconv.Atoi(statusMatch[1])
	if err != nil {
		return 0, "", fmt.Errorf("invalid status code: %w", err)
	}

	return statusCode, messageMatch[1], nil
}

const errNoAccessToken = "sdk client is not authenticated, cannot revoke access token"

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
			AccessToken:       "foobar",
			ExpiresIn:         2592000,
			AccessTokenMaxTTL: 2592000,
			TokenType:         "Bearer",
		})
		defer closeFunc()

		_, err := apiClient.Auth().UniversalAuthLogin(fakeClientID, fakeClientSecret)

		assert.NoError(t, err)
		assert.Equal(t, apiClient.Auth().GetAccessToken(), "foobar")
	})

	t.Run("SetTokenViaMachineIdentity: Error when non-200 response received", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, InfisicalAPIErrorResponse{
			StatusCode: 401,
			Message:    "Unauthorized",
		})
		defer closeFunc()

		_, err := apiClient.Auth().UniversalAuthLogin(fakeClientID, fakeClientSecret)
		assert.Error(t, err)

		apiErrorStatusCode, apiErrorMessage, err := parseInfisicalAPIError(err, t)
		if err != nil {
			t.Fatalf("Error parsing infisical API error: %v", err)
		}

		assert.Equal(t, 401, apiErrorStatusCode)
		assert.Equal(t, "Unauthorized", apiErrorMessage)
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
			Message:    "Unauthorized",
		})
		defer closeFunc()

		apiClient.Auth().SetAccessToken(fakeToken)

		err := apiClient.Auth().RevokeAccessToken()
		assert.Error(t, err)

		apiErrorStatusCode, apiErrorMessage, err := parseInfisicalAPIError(err, t)
		if err != nil {
			t.Fatalf("Error parsing infisical API error: %v", err)
		}
		assert.Equal(t, 401, apiErrorStatusCode)
		assert.Equal(t, "Unauthorized", apiErrorMessage)
	})

	t.Run("Error when no access token is set", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, nil)
		defer closeFunc()

		err := apiClient.Auth().RevokeAccessToken()

		assert.EqualError(t, err, errNoAccessToken)
	})
}

func TestGetSecretsV3(t *testing.T) {
	t.Run("Works with secrets", func(t *testing.T) {
		secrets := []SecretsV3{
			{SecretKey: "foo", SecretValue: "bar"},
		}

		apiClient, closeFunc := NewMockClient(200, GetSecretsV3Response{
			Secrets: secrets,
		})

		var sdkFormattedSecrets []infisical.Secret

		for _, secret := range secrets {
			sdkFormattedSecrets = append(sdkFormattedSecrets, infisical.Secret{
				SecretKey:   secret.SecretKey,
				SecretValue: secret.SecretValue,
			})
		}

		defer closeFunc()

		sdkSecrets, err := apiClient.Secrets().List(infisical.ListSecretsOptions{
			ProjectSlug: fakeProjectSlug,
			Environment: fakeEnvironmentSlug,
			SecretPath:  "/",
			Recursive:   true,
		})
		assert.NoError(t, err)
		assert.Equal(t, sdkSecrets, sdkFormattedSecrets)
	})

	t.Run("Works with imported secrets", func(t *testing.T) {
		secrets := []SecretsV3{
			{SecretKey: "foo", SecretValue: "bar"},
		}

		apiClient, closeFunc := NewMockClient(200, GetSecretsV3Response{
			ImportedSecrets: []ImportedSecretV3{{
				Secrets: secrets,
			}},
		})
		defer closeFunc()

		var sdkFormattedSecrets []infisical.Secret

		for _, secret := range secrets {
			sdkFormattedSecrets = append(sdkFormattedSecrets, infisical.Secret{
				SecretKey:   secret.SecretKey,
				SecretValue: secret.SecretValue,
			})
		}

		sdkSecrets, err := apiClient.Secrets().List(infisical.ListSecretsOptions{
			ProjectSlug:    fakeProjectSlug,
			Environment:    fakeEnvironmentSlug,
			IncludeImports: true,
			SecretPath:     "/",
			Recursive:      true,
		})
		assert.NoError(t, err)
		assert.Equal(t, sdkSecrets, sdkFormattedSecrets)
	})

	t.Run("GetSecretsV3: Error when non-200 response received", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, InfisicalAPIErrorResponse{
			StatusCode: 401,
			Message:    "Unauthorized",
		})
		defer closeFunc()

		_, err := apiClient.Secrets().List(infisical.ListSecretsOptions{
			ProjectSlug: fakeProjectSlug,
			Environment: fakeEnvironmentSlug,
			SecretPath:  "/",
			Recursive:   true,
		})
		assert.Error(t, err)

		apiErrorStatusCode, apiErrorMessage, err := parseInfisicalAPIError(err, t)
		if err != nil {
			t.Fatalf("Error parsing infisical API error: %v", err)
		}

		assert.Equal(t, 401, apiErrorStatusCode)
		assert.Equal(t, "Unauthorized", apiErrorMessage)
	})
}
func TestGetSecretByKeyV3(t *testing.T) {
	t.Run("Works", func(t *testing.T) {
		secret := SecretsV3{
			SecretKey:   "foo",
			SecretValue: "bar",
		}

		sdkFormattedSecret := infisical.Secret{
			SecretKey:   secret.SecretKey,
			SecretValue: secret.SecretValue,
		}

		apiClient, closeFunc := NewMockClient(200, GetSecretByKeyV3Response{
			Secret: secret,
		})
		defer closeFunc()

		sdkSecret, err := apiClient.Secrets().Retrieve(infisical.RetrieveSecretOptions{
			ProjectSlug:    fakeProjectSlug,
			Environment:    fakeEnvironmentSlug,
			SecretPath:     "/",
			IncludeImports: true,
			SecretKey:      "foo",
		})
		assert.NoError(t, err)
		assert.Equal(t, sdkSecret, sdkFormattedSecret)
	})

	t.Run("Error when secret is not found", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(404, InfisicalAPIErrorResponse{
			StatusCode: 404,
			Message:    "Not Found",
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

		apiErrorStatusCode, apiErrorMessage, err := parseInfisicalAPIError(err, t)
		if err != nil {
			t.Fatalf("Error parsing infisical API error: %v", err)
		}

		assert.Equal(t, 404, apiErrorStatusCode)
		assert.Equal(t, "Not Found", apiErrorMessage)
	})

	// Test case where the request is unauthorized
	t.Run("ErrorHandlingUnauthorized", func(t *testing.T) {
		apiClient, closeFunc := NewMockClient(401, InfisicalAPIErrorResponse{
			StatusCode: 401,
			Message:    "Unauthorized",
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

		apiErrorStatusCode, apiErrorMessage, err := parseInfisicalAPIError(err, t)
		if err != nil {
			t.Fatalf("Error parsing infisical API error: %v", err)
		}

		assert.Equal(t, 401, apiErrorStatusCode)
		assert.Equal(t, "Unauthorized", apiErrorMessage)
	})
}
