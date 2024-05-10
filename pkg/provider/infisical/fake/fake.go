package fake

import (
	"time"

	"github.com/external-secrets/external-secrets/pkg/provider/infisical/api"
)

type MockInfisicalClient struct {}

func (a *MockInfisicalClient) RefreshMachineIdentityAccessToken(data api.MachineIdentityUniversalAuthRefreshRequest) (*api.MachineIdentityDetailsResponse, error) {
	return &api.MachineIdentityDetailsResponse{
		AccessToken:       "test-access-token",
		ExpiresIn:         int(time.Hour * 24),
		TokenType:         "bearer",
		AccessTokenMaxTTL: int(time.Hour * 24 * 2),
	}, nil
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
	return map[string]string{
		"key": "value",
	}, nil
}

func (a *MockInfisicalClient) GetSecretByKeyV3(data api.GetSecretByKeyV3Request) (string, error) {
	return "value", nil
}
