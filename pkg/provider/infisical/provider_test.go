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

package infisical

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esv1meta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/api"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/fake"
)

type storeModifier func(*esv1beta1.SecretStore) *esv1beta1.SecretStore

var apiScope = InfisicalClientScope{
	SecretPath:      "/",
	ProjectSlug:     "first-project",
	EnvironmentSlug: "dev",
}

type TestCases struct {
	Name           string
	MockClient     *fake.MockInfisicalClient
	PropertyAccess string
	Error          error
	Output         any
}

func TestGetSecret(t *testing.T) {
	testCases := []TestCases{
		{
			Name: "Get_valid_key",
			MockClient: &fake.MockInfisicalClient{
				MockedGetSecretByKeyV3: func(data api.GetSecretByKeyV3Request) (string, error) {
					return "value", nil
				},
			},
			Error:  nil,
			Output: []byte("value"),
		},
		{
			Name: "Get_property_key",
			MockClient: &fake.MockInfisicalClient{
				MockedGetSecretByKeyV3: func(data api.GetSecretByKeyV3Request) (string, error) {
					return `{"key":"value"}`, nil
				},
			},
			Error:  nil,
			Output: []byte("value"),
		},
		{
			Name: "Key_not_found",
			MockClient: &fake.MockInfisicalClient{
				MockedGetSecretByKeyV3: func(data api.GetSecretByKeyV3Request) (string, error) {
					// from server
					return "", errors.New("Secret not found")
				},
			},
			Error:  errors.New("Secret not found"),
			Output: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			p := &Provider{
				apiClient: tc.MockClient,
				apiScope:  &apiScope,
			}
			var property string
			if tc.Name == "Get_property_key" {
				property = "key"
			}

			output, err := p.GetSecret(context.Background(), esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "key",
				Property: property,
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

func TestGetSecretMap(t *testing.T) {
	testCases := []TestCases{
		{
			Name: "Get_valid_key_map",
			MockClient: &fake.MockInfisicalClient{
				MockedGetSecretByKeyV3: func(data api.GetSecretByKeyV3Request) (string, error) {
					return `{"key":"value"}`, nil
				},
			},
			Error: nil,
			Output: map[string][]byte{
				"key": []byte("value"),
			},
		},
		{
			Name: "Get_invalid_map",
			MockClient: &fake.MockInfisicalClient{
				MockedGetSecretByKeyV3: func(data api.GetSecretByKeyV3Request) (string, error) {
					return ``, nil
				},
			},
			Error:  errors.New("unexpected end of JSON input"),
			Output: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			p := &Provider{
				apiClient: tc.MockClient,
				apiScope:  &apiScope,
			}
			output, err := p.GetSecretMap(context.Background(), esv1beta1.ExternalSecretDataRemoteRef{
				Key: "key",
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

func makeSecretStore(projectSlug, environment, secretPath string, fn ...storeModifier) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Infisical: &esv1beta1.InfisicalProvider{
					Auth: esv1beta1.InfisicalAuth{
						UniversalAuthCredentials: &esv1beta1.UniversalAuthCredentials{},
					},
					SecretsScope: esv1beta1.MachineIdentityScopeInWorkspace{
						SecretsPath:     secretPath,
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
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Infisical.Auth.UniversalAuthCredentials.ClientID = esv1meta.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}

func withClientSecret(name, key string, namespace *string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Infisical.Auth.UniversalAuthCredentials.ClientSecret = esv1meta.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}

type ValidateStoreTestCase struct {
	store       *esv1beta1.SecretStore
	assertError func(t *testing.T, err error)
}

func TestValidateStore(t *testing.T) {
	const randomID = "some-random-id"
	const authType = "universal-auth"
	var authCredMissingErr = errors.New("universalAuthCredentials.clientId and universalAuthCredentials.clientSecret cannot be empty")
	var authScopeMissingErr = errors.New("secretsScope.projectSlug and secretsScope.environmentSlug cannot be empty")

	testCases := []ValidateStoreTestCase{
		{
			store: makeSecretStore("", "", ""),
			assertError: func(t *testing.T, err error) {
				require.ErrorAs(t, err, &authScopeMissingErr)
			},
		},
		{
			store: makeSecretStore(apiScope.ProjectSlug, apiScope.EnvironmentSlug, apiScope.SecretPath, withClientID(authType, randomID, nil)),
			assertError: func(t *testing.T, err error) {
				require.ErrorAs(t, err, &authCredMissingErr)
			},
		},
		{
			store: makeSecretStore(apiScope.ProjectSlug, apiScope.EnvironmentSlug, apiScope.SecretPath, withClientSecret(authType, randomID, nil)),
			assertError: func(t *testing.T, err error) {
				require.ErrorAs(t, err, &authCredMissingErr)
			},
		},
		{
			store:       makeSecretStore(apiScope.ProjectSlug, apiScope.EnvironmentSlug, apiScope.SecretPath, withClientID(authType, randomID, nil), withClientSecret(authType, randomID, nil)),
			assertError: func(t *testing.T, err error) { require.NoError(t, err) },
		},
	}
	p := Provider{}
	for _, tc := range testCases {
		_, err := p.ValidateStore(tc.store)
		tc.assertError(t, err)
	}
}
