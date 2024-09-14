// Copyright External Secrets Inc. All Rights Reserved
package fake

import (
	"errors"
	"time"

	"github.com/external-secrets/external-secrets/pkg/provider/infisical/api"
)

var (
	ErrMissingMockImplementation = errors.New("missing mock implmentation")
)

type MockInfisicalClient struct {
	MockedGetSecretV3      func(data api.GetSecretsV3Request) (map[string]string, error)
	MockedGetSecretByKeyV3 func(data api.GetSecretByKeyV3Request) (string, error)
}

func (a *MockInfisicalClient) MachineIdentityLoginViaUniversalAuth(data api.MachineIdentityUniversalAuthLoginRequest) (*api.MachineIdentityDetailsResponse, error) {
	return &api.MachineIdentityDetailsResponse{
		AccessToken:       "test-access-token",
		ExpiresIn:         int(time.Hour * 24),
		TokenType:         "bearer",
		AccessTokenMaxTTL: int(time.Hour * 24 * 2),
	}, nil
}

func (a *MockInfisicalClient) GetSecretsV3(data api.GetSecretsV3Request) (map[string]string, error) {
	if a.MockedGetSecretV3 == nil {
		return nil, ErrMissingMockImplementation
	}

	return a.MockedGetSecretV3(data)
}

func (a *MockInfisicalClient) GetSecretByKeyV3(data api.GetSecretByKeyV3Request) (string, error) {
	if a.MockedGetSecretByKeyV3 == nil {
		return "", ErrMissingMockImplementation
	}
	return a.MockedGetSecretByKeyV3(data)
}

func (a *MockInfisicalClient) RevokeAccessToken() error {
	return nil
}
