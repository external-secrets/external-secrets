/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliec.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
